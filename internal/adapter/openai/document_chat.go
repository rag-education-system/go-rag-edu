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
	return s.client.GenerateAnswer(ctx, query, docContext, toHistory(history))
}

func (s *DocumentChatService) GenerateAnswerStream(
	ctx context.Context,
	query string,
	docContext string,
	history []document.ChatMessage,
) (<-chan string, <-chan error) {
	return s.client.GenerateAnswerStream(ctx, query, docContext, toHistory(history))
}

func toHistory(history []document.ChatMessage) []HistoryMessage {
	converted := make([]HistoryMessage, len(history))
	for i, message := range history {
		converted[i] = HistoryMessage{Role: message.Role, Content: message.Content}
	}
	return converted
}

// ReformulatorAdapter adapts QueryReformulator to document.QueryReformulator interface.
type ReformulatorAdapter struct {
	inner *QueryReformulator
}

func NewReformulatorAdapter(inner *QueryReformulator) *ReformulatorAdapter {
	return &ReformulatorAdapter{inner: inner}
}

func (a *ReformulatorAdapter) Enabled() bool {
	if a.inner == nil {
		return false
	}
	return a.inner.Enabled()
}

func (a *ReformulatorAdapter) ReformulateQuery(ctx context.Context, query string, history []document.ChatMessage) (string, error) {
	if a.inner == nil {
		return query, nil
	}
	return a.inner.ReformulateQuery(ctx, query, history)
}
