package repository

import (
	"context"
	"rag-api/internal/domain/entity"

	"github.com/pgvector/pgvector-go"
)

type ChunkRepository interface {
	Create(ctx context.Context, chunk *entity.DocumentChunk) error
	CreateBatch(ctx context.Context, chunks []entity.DocumentChunk) error
	SearchSimilar(ctx context.Context, embedding pgvector.Vector, topK int, threshold float64) ([]entity.SimilarChunk, error)
	FindByDocumentID(ctx context.Context, documentID string) ([]entity.DocumentChunk, error)
	DeleteByDocumentID(ctx context.Context, documentID string) error
}
