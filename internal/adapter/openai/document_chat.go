package openai

import (
	"context"

	"rag-api/internal/usecase/document"
)

type DocumentChatService struct {
	client *ChatClient
}

func NewDocumentChatService(client *ChatClient) *DocumentChatService {
	return &DocumentChatService{client: client}
}

func (s *DocumentChatService) GenerateAnswer(
	ctx context.Context,
	query string,
	docContext string,
	history []document.ChatMessage,
) (string, error) {
	converted := make([]HistoryMessage, len(history))
	for i, message := range history {
		converted[i] = HistoryMessage{
			Role:    message.Role,
			Content: message.Content,
		}
	}

	return s.client.GenerateAnswer(ctx, query, docContext, converted)
}
