package agent

// request payload for AI code generation
type GenerateRequest struct {
	UserQuery           string    `json:"user_query" binding:"required"`
	EditorState         string    `json:"editor_state"`
	ConversationHistory []Message `json:"conversation_history"`
	Provider            string    `json:"provider,omitempty"`         // "anthropic" or "openai"
	ProviderAPIKey      string    `json:"provider_api_key,omitempty"` // BYOK key
	StrudelID           string    `json:"strudel_id,omitempty"`       // optional: for persisting conversation
}

// conversation message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// response payload for AI code generation
type GenerateResponse struct {
	Code                string   `json:"code,omitempty"`
	IsActionable        bool     `json:"is_actionable"`
	IsCodeResponse      bool     `json:"is_code_response"`
	ClarifyingQuestions []string `json:"clarifying_questions,omitempty"`
	DocsRetrieved       int      `json:"docs_retrieved"`
	ExamplesRetrieved   int      `json:"examples_retrieved"`
	Model               string   `json:"model"`
}
