package postgres

import (
	"context"
	"strconv"
	"strings"
	"time"

	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pgvector/pgvector-go"
)

type chunkRepository struct {
	db *sqlx.DB
}

func NewChunkRepository(db *sqlx.DB) repository.ChunkRepository {
	return &chunkRepository{db: db}
}

// create chunk
func (r *chunkRepository) Create(ctx context.Context, chunk *entity.DocumentChunk) error {
	chunk.ID = uuid.New().String()
	chunk.CreatedAt = time.Now()

	query := `
		INSERT INTO "document_chunks" ("id", "documentId", "chunkIndex", "content", "embedding", "metadata", "createdAt")
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		chunk.ID,
		chunk.DocumentID,
		chunk.ChunkIndex,
		chunk.Content,
		chunk.Embedding,
		chunk.Metadata,
		chunk.CreatedAt,
	)
	return err
}

// CreateBatch creates multiple chunks
func (r *chunkRepository) CreateBatch(ctx context.Context, chunks []entity.DocumentChunk) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO "document_chunks" ("id", "documentId", "chunkIndex", "content", "embedding", "metadata", "createdAt")
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	for i := range chunks {
		chunks[i].ID = uuid.New().String()
		chunks[i].CreatedAt = time.Now()

		_, err := tx.ExecContext(ctx, query,
			chunks[i].ID,
			chunks[i].DocumentID,
			chunks[i].ChunkIndex,
			chunks[i].Content,
			chunks[i].Embedding,
			chunks[i].Metadata,
			chunks[i].CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()

}

// SearchSimilar searches for similar chunks using vector similarity
func (r *chunkRepository) SearchSimilar(ctx context.Context, embedding pgvector.Vector, topK int, threshold float64) ([]entity.SimilarChunk, error) {
	query := `
		SELECT 
			dc."id",
			dc."documentId",
			dc."chunkIndex",
			dc."content",
			dc."metadata",
			dc."createdAt",		
		1 - (dc."embedding" <=> $1) AS similarity
		FROM "document_chunks" dc
		INNER JOIN "documents" d ON dc."documentId" = d."id"
		WHERE d."status" = 'COMPLETED'
		AND (1 - (dc."embedding" <=> $1)) >= $2
		ORDER BY dc."embedding" <=> $1
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, embedding, threshold, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []entity.SimilarChunk
	for rows.Next() {
		var chunk entity.SimilarChunk
		err := rows.Scan(
			&chunk.ID,
			&chunk.DocumentID,
			&chunk.ChunkIndex,
			&chunk.Content,
			&chunk.Metadata,
			&chunk.CreatedAt,
			&chunk.Similarity,
		)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

func (r *chunkRepository) SearchByKeywords(ctx context.Context, terms []string, topK int) ([]entity.SimilarChunk, error) {
	if len(terms) == 0 {
		return nil, nil
	}

	conditions := make([]string, 0, len(terms))
	args := make([]any, 0, len(terms)+1)
	for i, term := range terms {
		conditions = append(conditions, `dc."content" ILIKE $`+strconv.Itoa(i+1))
		args = append(args, "%"+term+"%")
	}
	args = append(args, topK)

	query := `
		SELECT
			dc."id",
			dc."documentId",
			dc."chunkIndex",
			dc."content",
			dc."metadata",
			dc."createdAt",
			0.55::float8 AS similarity
		FROM "document_chunks" dc
		INNER JOIN "documents" d ON dc."documentId" = d."id"
		WHERE d."status" = 'COMPLETED'
		AND (` + strings.Join(conditions, " OR ") + `)
		ORDER BY dc."chunkIndex" ASC
		LIMIT $` + strconv.Itoa(len(terms)+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []entity.SimilarChunk
	for rows.Next() {
		var chunk entity.SimilarChunk
		if err := rows.Scan(
			&chunk.ID,
			&chunk.DocumentID,
			&chunk.ChunkIndex,
			&chunk.Content,
			&chunk.Metadata,
			&chunk.CreatedAt,
			&chunk.Similarity,
		); err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

func (r *chunkRepository) FindByDocumentID(ctx context.Context, documentID string) ([]entity.DocumentChunk, error) {
	query := `
		SELECT "id", "documentId", "chunkIndex", "content", "metadata", "createdAt"
		FROM "document_chunks"
		WHERE "documentId" = $1
		ORDER BY "chunkIndex" ASC
	`

	var chunks []entity.DocumentChunk
	if err := r.db.SelectContext(ctx, &chunks, query, documentID); err != nil {
		return nil, err
	}

	return chunks, nil
}

// DeleteByDocumentID deletes all chunks containing the document ID
func (r *chunkRepository) DeleteByDocumentID(ctx context.Context, documentID string) error {
	query := `DELETE FROM "document_chunks" WHERE "documentId" = $1`
	_, err := r.db.ExecContext(ctx, query, documentID)
	return err
}
