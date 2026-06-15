package repository

import (
	"context"
	"rag-api/internal/domain/docaccess"
	"rag-api/internal/domain/entity"

	"github.com/pgvector/pgvector-go"
)

type ChunkRepository interface {
	Create(ctx context.Context, chunk *entity.DocumentChunk) error
	CreateBatch(ctx context.Context, chunks []entity.DocumentChunk) error
	SearchSimilar(ctx context.Context, embedding pgvector.Vector, topK int, threshold float64) ([]entity.SimilarChunk, error)
	SearchSimilarWithAccess(ctx context.Context, embedding pgvector.Vector, access docaccess.Context, topK int, threshold float64) ([]entity.SimilarChunk, error)
	HybridSearchWithAccess(ctx context.Context, query string, embedding pgvector.Vector, access docaccess.Context, topK int, threshold float64) ([]entity.SimilarChunk, error)
	SearchByKeywords(ctx context.Context, terms []string, access docaccess.Context, topK int) ([]entity.SimilarChunk, error)
	FindByDocumentID(ctx context.Context, documentID string) ([]entity.DocumentChunk, error)
	DeleteByDocumentID(ctx context.Context, documentID string) error
}
