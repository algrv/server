package tui

import (
	"fmt"
	"time"
)

const (
	typeAgentRequest  = "agent_request"
	typeAgentResponse = "agent_response"
	typeError         = "error"
)

const (
	agentRequestTimeout = 60 * time.Second
	reconnectDelay      = 2 * time.Second
	pongWait            = 60 * time.Second
	pingPeriod          = (pongWait * 9) / 10
)

func formatAgentResponse(result agentResponsePayload) (string, string) {
	code := result.Code

	metadata := fmt.Sprintf("retrieved: %d docs, %d examples | model: %s",
		result.DocsRetrieved,
		result.ExamplesRetrieved,
		result.Model)

	return code, metadata
}
