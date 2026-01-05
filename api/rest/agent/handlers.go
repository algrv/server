package agent

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/algrv/server/algorave/strudels"
	agentcore "github.com/algrv/server/internal/agent"
	"github.com/algrv/server/internal/errors"
	"github.com/algrv/server/internal/llm"
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
func GenerateHandler(agentClient *agentcore.Agent, _ llm.LLM, strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GenerateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
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
				if _, err := strudelRepo.AddStrudelMessage(ctx, &strudels.AddStrudelMessageRequest{
					StrudelID:           req.StrudelID,
					Role:                "assistant",
					Content:             resp.Code,
					IsActionable:        resp.IsActionable,
					IsCodeResponse:      resp.IsCodeResponse,
					ClarifyingQuestions: resp.ClarifyingQuestions,
				}); err != nil {
					log.Printf("failed to persist assistant message for strudel %s: %v", req.StrudelID, err)
				}
			}
		}

		c.JSON(http.StatusOK, GenerateResponse{
			Code:                resp.Code,
			IsActionable:        resp.IsActionable,
			IsCodeResponse:      resp.IsCodeResponse,
			ClarifyingQuestions: resp.ClarifyingQuestions,
			DocsRetrieved:       resp.DocsRetrieved,
			ExamplesRetrieved:   resp.ExamplesRetrieved,
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
