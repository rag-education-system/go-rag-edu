package repository

import (
	"context"
	"rag-api/internal/domain/entity"
)

type MessageRepository interface {
	Create(ctx context.Context, msg *entity.Message) error
	ListByConversation(ctx context.Context, conversationID string, limit int) ([]entity.Message, error)
}
