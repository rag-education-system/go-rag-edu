package repository

import (
	"context"
	"rag-api/internal/domain/entity"
)

type DocumentRepository interface {
	Create(ctx context.Context, doc *entity.Document) error
	FindByID(ctx context.Context, id string) (*entity.Document, error)
	FindByIDAndUserID(ctx context.Context, id, userID string) (*entity.Document, error)
	List(ctx context.Context, userID string, page, limit int, status *entity.DocumentStatus) ([]entity.Document, int, error)
	UpdateStatus(ctx context.Context, id string, status entity.DocumentStatus) error
	UpdateTotalChunks(ctx context.Context, id string, totalChunks int) error
	Delete(ctx context.Context, id string) error
}








