package dto

type StreamChunk struct {
	Type           string        `json:"type"`
	Content        string        `json:"content,omitempty"`
	ConversationID string        `json:"conversationId,omitempty"`
	Sources        []ChunkSource `json:"sources,omitempty"`
	Error          string        `json:"error,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type StreamChatRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversationId,omitempty"`
}
