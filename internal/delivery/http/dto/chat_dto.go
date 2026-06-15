package dto

type CreateConversationRequest struct {
	Message string `json:"message"`
}

type SendMessageRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	ConversationID   string              `json:"conversationId"`
	UserMessage      ChatMessageResponse `json:"userMessage"`
	AssistantMessage ChatMessageResponse `json:"assistantMessage"`
}

type ChatMessageResponse struct {
	ID        string        `json:"id"`
	Role      string        `json:"role"`
	Content   string        `json:"content"`
	Sources   []ChunkSource `json:"sources,omitempty"`
	CreatedAt string        `json:"createdAt,omitempty"`
}

type ConversationInfo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type ConversationDetail struct {
	Conversation ConversationInfo  `json:"conversation"`
	Messages     []ChatMessageResponse `json:"messages"`
}
