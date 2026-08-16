package model

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

type CompletionTokensDetails struct {
	ReasoningTokens          int `json:"reasoning_tokens"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
}

type Error struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    any    `json:"code"`
}

type ErrorWithStatusCode struct {
	Error
	StatusCode int `json:"status_code"`
}

// ResponsesAPIResponse represents the response from Responses API
type ResponsesAPIResponse struct {
	Id                 string              `json:"id"`
	Object             string              `json:"object"`
	Created            int64               `json:"created"`
	Model              string              `json:"model"`
	ConversationId     string              `json:"conversation_id,omitempty"`
	OutputItems        []ResponseOutputItem `json:"output_items,omitempty"`
	Usage              *Usage              `json:"usage,omitempty"`
	Status             string              `json:"status,omitempty"`           // "completed", "in_progress", etc.
	IncompleteDetails  *IncompleteDetails  `json:"incomplete_details,omitempty"`
}

// ResponseOutputItem represents an output item in Responses API response
type ResponseOutputItem struct {
	Type    string  `json:"type"`              // "message", "function_call", etc.
	Role    string  `json:"role,omitempty"`    // "assistant"
	Content []Content `json:"content,omitempty"` // Content array
	ToolCalls []Tool  `json:"tool_calls,omitempty"`
}

// IncompleteDetails provides details when response is incomplete
type IncompleteDetails struct {
	Reason string `json:"reason,omitempty"` // "max_tokens", "stop_sequence", etc.
}
