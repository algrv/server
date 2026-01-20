package agent

import (
	"context"
	"fmt"
	"log"

	"codeberg.org/algopatterns/server/internal/buffer"
	"codeberg.org/algopatterns/server/internal/llm"
	"codeberg.org/algopatterns/server/internal/retriever"
	"codeberg.org/algopatterns/server/internal/strudel"
)

func New(ret Retriever, llmClient llm.LLM) *Agent {
	return &Agent{
		retriever: ret,
		generator: llmClient,
	}
}

// creates an agent with code validation enabled
func NewWithValidator(ret Retriever, llmClient llm.LLM, validator *strudel.Validator) *Agent {
	return &Agent{
		retriever: ret,
		generator: llmClient,
		validator: validator,
	}
}

// sets the validator for the agent
func (a *Agent) SetValidator(v *strudel.Validator) {
	a.validator = v
}

func (a *Agent) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	textGenerator := llm.TextGenerator(a.generator)
	isBYOK := req.CustomGenerator != nil

	if isBYOK {
		textGenerator = req.CustomGenerator
	}

	// for byok users: skip AnalyzeQuery to save ~1-3s latency
	// the main llm will naturally determine if it's a code request or question
	var analysis *llm.QueryAnalysis
	if !isBYOK {
		var err error
		analysis, err = a.generator.AnalyzeQuery(ctx, req.UserQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze query: %w", err)
		}
	}

	// rag retrieval with caching for byok users
	var docs []retriever.SearchResult
	var examples []retriever.ExampleResult
	usedCache := false
	canUseCache := isBYOK && req.SessionID != "" && req.RAGCache != nil

	// skip rag for purely conversational queries (greetings, thanks, etc)
	skipRAG := isConversationalQuery(req.UserQuery)

	if canUseCache && !skipRAG {
		// try to get cached rag results
		cached, err := req.RAGCache.GetRAGCache(ctx, req.SessionID)
		if err != nil {
			log.Printf("rag cache get failed (will fetch fresh): %v", err)
		} else if cached != nil {
			// cache hit - use cached docs and examples
			docs = cacheToDocs(cached.Docs)
			examples = cacheToExamples(cached.Examples)
			usedCache = true
		}
	}

	if !usedCache && !skipRAG {
		// cache miss or caching disabled - fetch from retriever
		var err error
		docs, err = a.retriever.HybridSearchDocs(ctx, req.UserQuery, req.EditorState, 3)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve docs: %w", err)
		}

		examples, err = a.retriever.HybridSearchExamples(ctx, req.UserQuery, req.EditorState, 2)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve examples: %w", err)
		}

		// cache the results for follow-up messages
		if canUseCache && !skipRAG {
			cacheData := &buffer.CachedRAGResult{
				Docs:     docsToCache(docs),
				Examples: examplesToCache(examples),
				Query:    req.UserQuery,
			}
			if err := req.RAGCache.SetRAGCache(ctx, req.SessionID, cacheData); err != nil {
				log.Printf("rag cache set failed: %v", err)
			}
		}
	}

	systemPrompt := buildSystemPrompt(SystemPromptContext{
		Cheatsheet:    getCheatsheet(),
		EditorState:   req.EditorState,
		Docs:          docs,
		Examples:      examples,
		Conversations: req.ConversationHistory,
		QueryAnalysis: analysis,
		UsedRAGCache:  usedCache, // tell prompt builder to add "need docs" instruction
	})

	// call llm for code generation (uses custom generator if byok)
	response, err := a.callGeneratorWithClient(ctx, textGenerator, systemPrompt, req.UserQuery, req.ConversationHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	totalInputTokens := response.Usage.InputTokens
	totalOutputTokens := response.Usage.OutputTokens

	// check if llm requested more documentation (only when using cached docs)
	if usedCache {
		needDocsTopic := parseNeedDocsMarker(response.Text)
		if needDocsTopic != "" {
			// clear old cache and fetch fresh docs for the new topic
			if req.RAGCache != nil {
				_ = req.RAGCache.ClearRAGCache(ctx, req.SessionID)
			}

			// fetch docs for the requested topic
			freshDocs, err := a.retriever.HybridSearchDocs(ctx, needDocsTopic, req.EditorState, 3)
			if err != nil {
				log.Printf("failed to fetch docs for topic '%s': %v", needDocsTopic, err)
			} else {
				docs = freshDocs
			}

			freshExamples, err := a.retriever.HybridSearchExamples(ctx, needDocsTopic, req.EditorState, 2)
			if err != nil {
				log.Printf("failed to fetch examples for topic '%s': %v", needDocsTopic, err)
			} else {
				examples = freshExamples
			}

			// cache the new results
			if canUseCache {
				cacheData := &buffer.CachedRAGResult{
					Docs:     docsToCache(docs),
					Examples: examplesToCache(examples),
					Query:    needDocsTopic,
				}
				_ = req.RAGCache.SetRAGCache(ctx, req.SessionID, cacheData)
			}

			// rebuild prompt with fresh docs and regenerate
			systemPrompt = buildSystemPrompt(SystemPromptContext{
				Cheatsheet:    getCheatsheet(),
				EditorState:   req.EditorState,
				Docs:          docs,
				Examples:      examples,
				Conversations: req.ConversationHistory,
				QueryAnalysis: analysis,
				UsedRAGCache:  false, // fresh docs, no need for "need docs" instruction
			})

			response, err = a.callGeneratorWithClient(ctx, textGenerator, systemPrompt, req.UserQuery, req.ConversationHistory)
			if err != nil {
				return nil, fmt.Errorf("failed to regenerate with fresh docs: %w", err)
			}
			totalInputTokens += response.Usage.InputTokens
			totalOutputTokens += response.Usage.OutputTokens
		}
	}

	didRetry := false
	var validationError string

	// analyze response to determine if it's code and extract from markdown if needed
	content, isCode := analyzeResponse(response.Text)

	// validate and retry only for code responses
	if a.validator != nil && isCode && content != "" {
		result, err := a.validator.Validate(ctx, content)
		if err == nil && !result.Valid {
			retryResponse, retryErr := a.retryWithValidationError(
				ctx, textGenerator, systemPrompt, req.UserQuery,
				req.ConversationHistory, content, result,
			)
			if retryErr == nil {
				// re-analyze the retry response
				content, isCode = analyzeResponse(retryResponse.Text)
				totalInputTokens += retryResponse.Usage.InputTokens
				totalOutputTokens += retryResponse.Usage.OutputTokens
				didRetry = true
			}

			validationError = result.Error
		}
	}

	// build references for frontend display
	strudelRefs := make([]StrudelReference, 0, len(examples))
	for _, ex := range examples {
		strudelRefs = append(strudelRefs, StrudelReference{
			ID:         ex.ID,
			Title:      ex.Title,
			AuthorName: ex.AuthorName,
			URL:        fmt.Sprintf("/strudel/%s", ex.ID),
		})
	}

	docRefs := make([]DocReference, 0, len(docs))
	seen := make(map[string]bool) // dedupe by page URL
	for _, doc := range docs {
		if seen[doc.PageURL] {
			continue
		}
		seen[doc.PageURL] = true
		docRefs = append(docRefs, DocReference{
			PageName:     doc.PageName,
			SectionTitle: doc.SectionTitle,
			URL:          doc.PageURL,
		})
	}

	return &GenerateResponse{
		Code:              content,
		DocsRetrieved:     len(docs),
		ExamplesRetrieved: len(examples),
		Examples:          examples,
		Docs:              docs,
		StrudelReferences: strudelRefs,
		DocReferences:     docRefs,
		Model:             textGenerator.Model(),
		IsActionable:      true,
		IsCodeResponse:    isCode,
		InputTokens:       totalInputTokens,
		OutputTokens:      totalOutputTokens,
		DidRetry:          didRetry,
		ValidationError:   validationError,
	}, nil
}

// generates code with streaming response chunks.
// the onEvent callback is called for each chunk and final metadata.
// note: streaming skips validation retry to maintain real-time delivery.
func (a *Agent) GenerateStream(ctx context.Context, req GenerateRequest, onEvent func(event StreamEvent) error) error {
	textGenerator := llm.TextGenerator(a.generator)
	isBYOK := req.CustomGenerator != nil

	if isBYOK {
		textGenerator = req.CustomGenerator
	}

	// rag retrieval with caching (same as non-streaming)
	var docs []retriever.SearchResult
	var examples []retriever.ExampleResult
	usedCache := false
	canUseCache := isBYOK && req.SessionID != "" && req.RAGCache != nil

	// skip rag for purely conversational queries (greetings, thanks, etc)
	skipRAG := isConversationalQuery(req.UserQuery)

	if canUseCache && !skipRAG {
		cached, err := req.RAGCache.GetRAGCache(ctx, req.SessionID)
		if err != nil {
			log.Printf("rag cache get failed (will fetch fresh): %v", err)
		} else if cached != nil {
			docs = cacheToDocs(cached.Docs)
			examples = cacheToExamples(cached.Examples)
			usedCache = true
		}
	}

	if !usedCache && !skipRAG {
		var err error
		docs, err = a.retriever.HybridSearchDocs(ctx, req.UserQuery, req.EditorState, 3)
		if err != nil {
			return fmt.Errorf("failed to retrieve docs: %w", err)
		}

		examples, err = a.retriever.HybridSearchExamples(ctx, req.UserQuery, req.EditorState, 2)
		if err != nil {
			return fmt.Errorf("failed to retrieve examples: %w", err)
		}

		if canUseCache && !skipRAG {
			cacheData := &buffer.CachedRAGResult{
				Docs:     docsToCache(docs),
				Examples: examplesToCache(examples),
				Query:    req.UserQuery,
			}
			_ = req.RAGCache.SetRAGCache(ctx, req.SessionID, cacheData)
		}
	}

	// send references early so frontend can display them while streaming
	strudelRefs := make([]StrudelReference, 0, len(examples))
	for _, ex := range examples {
		strudelRefs = append(strudelRefs, StrudelReference{
			ID:         ex.ID,
			Title:      ex.Title,
			AuthorName: ex.AuthorName,
			URL:        fmt.Sprintf("/strudel/%s", ex.ID),
		})
	}

	docRefs := make([]DocReference, 0, len(docs))
	seen := make(map[string]bool)
	for _, doc := range docs {
		if seen[doc.PageURL] {
			continue
		}
		seen[doc.PageURL] = true
		docRefs = append(docRefs, DocReference{
			PageName:     doc.PageName,
			SectionTitle: doc.SectionTitle,
			URL:          doc.PageURL,
		})
	}

	// send refs event first
	if err := onEvent(StreamEvent{
		Type:              "refs",
		StrudelReferences: strudelRefs,
		DocReferences:     docRefs,
	}); err != nil {
		return err
	}

	systemPrompt := buildSystemPrompt(SystemPromptContext{
		Cheatsheet:    getCheatsheet(),
		EditorState:   req.EditorState,
		Docs:          docs,
		Examples:      examples,
		Conversations: req.ConversationHistory,
		UsedRAGCache:  usedCache,
	})

	// prepare messages for LLM
	llmMessages := make([]llm.Message, 0, len(req.ConversationHistory)+1)
	for _, msg := range req.ConversationHistory {
		llmMessages = append(llmMessages, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	llmMessages = append(llmMessages, llm.Message{
		Role:    "user",
		Content: req.UserQuery,
	})

	// stream llm response
	llmReq := llm.TextGenerationRequest{
		SystemPrompt: systemPrompt,
		Messages:     llmMessages,
		MaxTokens:    4096,
	}

	response, err := textGenerator.GenerateTextStream(ctx, llmReq, func(chunk string) error {
		return onEvent(StreamEvent{
			Type:    "chunk",
			Content: chunk,
		})
	})
	if err != nil {
		return fmt.Errorf("failed to stream generation: %w", err)
	}

	// analyze final response
	content, isCode := analyzeResponse(response.Text)

	// send done event with final metadata
	return onEvent(StreamEvent{
		Type:              "done",
		Content:           content, // processed content (extracted from markdown if needed)
		Model:             textGenerator.Model(),
		IsCodeResponse:    isCode,
		InputTokens:       response.Usage.InputTokens,
		OutputTokens:      response.Usage.OutputTokens,
		StrudelReferences: strudelRefs,
		DocReferences:     docRefs,
	})
}
