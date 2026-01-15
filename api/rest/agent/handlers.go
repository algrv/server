package agent

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"codeberg.org/algorave/server/algorave/strudels"
	agentcore "codeberg.org/algorave/server/internal/agent"
	"codeberg.org/algorave/server/internal/attribution"
	"codeberg.org/algorave/server/internal/buffer"
	"codeberg.org/algorave/server/internal/errors"
	"codeberg.org/algorave/server/internal/llm"
)

const (
	defaultAnthropicModel = "claude-sonnet-4-20250514"
	defaultOpenAIModel    = "gpt-4o"
	defaultMaxTokens      = 4096
	defaultTemperature    = 0.7
	maxHistoryMessages    = 50
)

// GenerateHandler godoc
// @Summary Generate code with AI
// @Description Generate Strudel code using AI with optional BYOK support
// @Tags agent
// @Accept json
// @Produce json
// @Param request body GenerateRequest true "Generation request"
// @Success 200 {object} GenerateResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/agent/generate [post]
func GenerateHandler(agentClient *agentcore.Agent, _ llm.LLM, strudelRepo *strudels.Repository, attrService *attribution.Service, sessionBuffer *buffer.SessionBuffer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GenerateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		// paste lock validation (if session_id provided)
		// decoupled from WebSocket - just check Redis directly
		if req.SessionID != "" {
			ctx := c.Request.Context()
			locked, err := sessionBuffer.IsPasteLocked(ctx, req.SessionID)
			if err != nil {
				// fail open on Redis errors - log but allow request
				log.Printf("failed to check paste lock for session %s: %v", req.SessionID, err)
			} else if locked {
				errors.Forbidden(c, "AI assistant temporarily disabled - please make significant edits to the pasted code before using AI. This helps protect code shared with 'no-ai' restrictions.")
				return
			}
		}

		// block AI for forks from strudels with 'no-ai' signal
		// also block if parent can't be verified (deleted or fake fork ID)
		// check client-provided forked_from_id (for drafts)
		if req.ForkedFromID != "" {
			parentCCSignal, err := strudelRepo.GetStrudelCCSignal(c.Request.Context(), req.ForkedFromID)
			if err != nil {
				// parent strudel doesn't exist - block AI since we can't verify CC signal
				log.Printf("blocking AI: parent strudel %s not found: %v", req.ForkedFromID, err)
				errors.Forbidden(c, "AI assistant disabled - the original strudel no longer exists or is invalid")
				return
			}
			if parentCCSignal != nil && *parentCCSignal == strudels.CCSignalNoAI {
				errors.Forbidden(c, "AI assistant disabled - original author restricted AI use for this strudel")
				return
			}
		}

		// also check server-side for saved strudels (can't be bypassed)
		if req.StrudelID != "" {
			forkedFromID, err := strudelRepo.GetStrudelForkedFrom(c.Request.Context(), req.StrudelID)
			if err != nil {
				log.Printf("could not retrieve forked_from for strudel %s: %v", req.StrudelID, err)
			} else if forkedFromID != nil {
				parentCCSignal, err := strudelRepo.GetStrudelCCSignal(c.Request.Context(), *forkedFromID)
				if err != nil {
					// parent strudel doesn't exist - block AI since we can't verify CC signal
					log.Printf("blocking AI: parent strudel %s not found: %v", *forkedFromID, err)
					errors.Forbidden(c, "AI assistant disabled - the original strudel no longer exists")
					return
				}
				if parentCCSignal != nil && *parentCCSignal == strudels.CCSignalNoAI {
					errors.Forbidden(c, "AI assistant disabled - original author restricted AI use for this strudel")
					return
				}
			}
		}

		var conversationHistory []agentcore.Message

		// for saved strudels: load history from DB if not provided
		// for drafts: use history from request
		if req.StrudelID != "" && len(req.ConversationHistory) == 0 {
			// load from database
			messages, err := strudelRepo.GetStrudelMessages(c.Request.Context(), req.StrudelID, maxHistoryMessages)
			if err != nil {
				// log error but continue with empty history (non-fatal)
				conversationHistory = []agentcore.Message{}
			} else {
				conversationHistory = make([]agentcore.Message, 0, len(messages))
				for _, msg := range messages {
					// only include messages with content
					if msg.Content != "" {
						conversationHistory = append(conversationHistory, agentcore.Message{
							Role:    msg.Role,
							Content: msg.Content,
						})
					}
				}
			}
		} else {
			// use history from request (drafts)
			conversationHistory = make([]agentcore.Message, 0, len(req.ConversationHistory))
			for _, msg := range req.ConversationHistory {
				if msg.Content != "" {
					conversationHistory = append(conversationHistory, agentcore.Message{
						Role:    msg.Role,
						Content: msg.Content,
					})
				}
			}
		}

		// build generate request
		generateReq := agentcore.GenerateRequest{
			UserQuery:           req.UserQuery,
			EditorState:         req.EditorState,
			ConversationHistory: conversationHistory,
		}

		// create custom generator if BYOK key provided
		if req.ProviderAPIKey != "" {
			customGenerator, err := createBYOKGenerator(req.Provider, req.ProviderAPIKey)
			if err != nil {
				errors.BadRequest(c, "invalid provider configuration", err)
				return
			}
			generateReq.CustomGenerator = customGenerator
		}

		// generate response
		resp, err := agentClient.Generate(c.Request.Context(), generateReq)
		if err != nil {
			errors.InternalError(c, "failed to generate code", err)
			return
		}

		// record attributions if examples were used (runs async)
		if attrService != nil && len(resp.Examples) > 0 {
			userID, _ := c.Get("user_id")
			userIDStr, ok := userID.(string)
			if !ok {
				userIDStr = ""
			}

			var targetStrudelID *string
			if req.StrudelID != "" {
				targetStrudelID = &req.StrudelID
			}

			attrService.RecordAttributions(c.Request.Context(), resp.Examples, userIDStr, targetStrudelID)
		}

		// for saved strudels: persist messages to DB (non-fatal errors)
		if req.StrudelID != "" {
			ctx := c.Request.Context()

			// save user message
			if _, err := strudelRepo.AddStrudelMessage(ctx, &strudels.AddStrudelMessageRequest{
				StrudelID:      req.StrudelID,
				Role:           "user",
				Content:        req.UserQuery,
				IsActionable:   false,
				IsCodeResponse: false,
			}); err != nil {
				log.Printf("failed to persist user message for strudel %s: %v", req.StrudelID, err)
			}

			hasContent := resp.Code != "" || len(resp.ClarifyingQuestions) > 0
			if hasContent {
				// convert agent references to strudels package types for persistence
				strudelRefsForDB := make([]strudels.StrudelReference, len(resp.StrudelReferences))
				for i, ref := range resp.StrudelReferences {
					strudelRefsForDB[i] = strudels.StrudelReference{
						ID:         ref.ID,
						Title:      ref.Title,
						AuthorName: ref.AuthorName,
						URL:        ref.URL,
					}
				}
				docRefsForDB := make([]strudels.DocReference, len(resp.DocReferences))
				for i, ref := range resp.DocReferences {
					docRefsForDB[i] = strudels.DocReference{
						PageName:     ref.PageName,
						SectionTitle: ref.SectionTitle,
						URL:          ref.URL,
					}
				}

				if _, err := strudelRepo.AddStrudelMessage(ctx, &strudels.AddStrudelMessageRequest{
					StrudelID:           req.StrudelID,
					Role:                "assistant",
					Content:             resp.Code,
					IsActionable:        resp.IsActionable,
					IsCodeResponse:      resp.IsCodeResponse,
					ClarifyingQuestions: resp.ClarifyingQuestions,
					StrudelReferences:   strudelRefsForDB,
					DocReferences:       docRefsForDB,
				}); err != nil {
					log.Printf("failed to persist assistant message for strudel %s: %v", req.StrudelID, err)
				}
			}
		}

		// map internal references to API types
		strudelRefs := make([]StrudelReference, len(resp.StrudelReferences))
		for i, ref := range resp.StrudelReferences {
			strudelRefs[i] = StrudelReference{
				ID:         ref.ID,
				Title:      ref.Title,
				AuthorName: ref.AuthorName,
				URL:        ref.URL,
			}
		}

		docRefs := make([]DocReference, len(resp.DocReferences))
		for i, ref := range resp.DocReferences {
			docRefs[i] = DocReference{
				PageName:     ref.PageName,
				SectionTitle: ref.SectionTitle,
				URL:          ref.URL,
			}
		}

		c.JSON(http.StatusOK, GenerateResponse{
			Code:                resp.Code,
			IsActionable:        resp.IsActionable,
			IsCodeResponse:      resp.IsCodeResponse,
			ClarifyingQuestions: resp.ClarifyingQuestions,
			DocsRetrieved:       resp.DocsRetrieved,
			ExamplesRetrieved:   resp.ExamplesRetrieved,
			StrudelReferences:   strudelRefs,
			DocReferences:       docRefs,
			Model:               resp.Model,
		})
	}
}

// creates a BYOK generator based on provider
func createBYOKGenerator(provider, apiKey string) (llm.TextGenerator, error) {
	switch provider {
	case "anthropic", "":
		return llm.NewAnthropicTransformer(llm.AnthropicConfig{
			APIKey:      apiKey,
			Model:       defaultAnthropicModel,
			MaxTokens:   defaultMaxTokens,
			Temperature: defaultTemperature,
		}), nil
	case "openai":
		return llm.NewOpenAIGenerator(llm.OpenAIConfig{
			APIKey: apiKey,
			Model:  defaultOpenAIModel,
		}), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
