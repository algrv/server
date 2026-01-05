package agent

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	agentcore "github.com/algrv/server/internal/agent"
	"github.com/algrv/server/internal/errors"
	"github.com/algrv/server/internal/llm"
)

const (
	defaultAnthropicModel = "claude-sonnet-4-20250514"
	defaultOpenAIModel    = "gpt-4o"
	defaultMaxTokens      = 4096
	defaultTemperature    = 0.7
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
func GenerateHandler(agentClient *agentcore.Agent, _ llm.LLM) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GenerateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		// convert conversation history to agent format
		conversationHistory := make([]agentcore.Message, 0, len(req.ConversationHistory))
		for _, msg := range req.ConversationHistory {
			conversationHistory = append(conversationHistory, agentcore.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
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
