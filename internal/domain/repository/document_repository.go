package repository

import (
	"context"
	"rag-api/internal/domain/docaccess"
	"rag-api/internal/domain/entity"
)

type DocumentRepository interface {
	Create(ctx context.Context, doc *entity.Document) error
	FindByID(ctx context.Context, id string) (*entity.Document, error)
	FindByIDAndUserID(ctx context.Context, id, userID string) (*entity.Document, error)
	FindByIDWithAccess(ctx context.Context, id string, access docaccess.Context) (*entity.Document, error)
	List(ctx context.Context, access docaccess.Context, page, limit int, status *entity.DocumentStatus) ([]entity.Document, int, error)
	UpdateStatus(ctx context.Context, id string, status entity.DocumentStatus) error
	UpdateTotalChunks(ctx context.Context, id string, totalChunks int) error
	UpdateStoragePath(ctx context.Context, id string, storagePath string) error
	UpdateVisibility(ctx context.Context, id string, visibility entity.DocumentVisibility) error
	Delete(ctx context.Context, id string) error
}








