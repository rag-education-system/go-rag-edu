package repository

import (
	"context"
	"rag-api/internal/domain/entity"
)

type ConversationRepository interface {
	Create(ctx context.Context, conv *entity.Conversation) error
	FindByID(ctx context.Context, id string) (*entity.Conversation, error)
	FindByIDAndUserID(ctx context.Context, id, userID string) (*entity.Conversation, error)
	List(ctx context.Context, userID string, page, limit int) ([]entity.Conversation, int, error)
	Update(ctx context.Context, conv *entity.Conversation) error
	Delete(ctx context.Context, id string) error
}
