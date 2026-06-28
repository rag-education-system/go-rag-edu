package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"rag-api/internal/domain/docaccess"
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

// SearchSimilar searches for similar chunks using vector similarity (no access control — use SearchSimilarWithAccess for RAG)
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

	return r.scanSimilarChunks(ctx, query, embedding, threshold, topK)
}

// SearchSimilarWithAccess performs vector similarity search filtered by document visibility and ownership.
// Pattern adapted from AI-Hukum-BE SimilaritySearchWithAccess.
func (r *chunkRepository) SearchSimilarWithAccess(
	ctx context.Context,
	embedding pgvector.Vector,
	access docaccess.Context,
	topK int,
	threshold float64,
	documentID string,
) ([]entity.SimilarChunk, error) {
	distanceThreshold := 1 - threshold
	args := []any{embedding, distanceThreshold}
	accessCond, accessArgs := docaccess.SQLCondition("d", access, 3)
	args = append(args, accessArgs...)

	docFilter := ""
	if documentID = strings.TrimSpace(documentID); documentID != "" {
		docFilter = fmt.Sprintf(` AND dc."documentId" = $%d`, len(args)+1)
		args = append(args, documentID)
	}

	limitParam := len(args) + 1
	args = append(args, topK)

	query := fmt.Sprintf(`
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
		AND (dc."embedding" <=> $1) < $2
		AND %s%s
		ORDER BY dc."embedding" <=> $1
		LIMIT $%d
	`, accessCond, docFilter, limitParam)

	return r.scanSimilarChunks(ctx, query, args...)
}

// HybridSearchWithAccess combines vector + full-text search with Reciprocal Rank Fusion (pattern from AI-Hukum-BE).
func (r *chunkRepository) HybridSearchWithAccess(
	ctx context.Context,
	query string,
	embedding pgvector.Vector,
	access docaccess.Context,
	topK int,
	threshold float64,
	documentID string,
) ([]entity.SimilarChunk, error) {
	distanceThreshold := 1 - threshold
	args := []any{embedding, distanceThreshold}
	accessCond, accessArgs := docaccess.SQLCondition("d", access, len(args)+1)
	args = append(args, accessArgs...)

	docFilter := ""
	if documentID = strings.TrimSpace(documentID); documentID != "" {
		docFilter = fmt.Sprintf(` AND dc."documentId" = $%d`, len(args)+1)
		args = append(args, documentID)
	}

	textQueryParam := len(args) + 1
	args = append(args, query)
	limitParam := len(args) + 1
	args = append(args, topK)

	sql := fmt.Sprintf(`
		WITH vector_search AS (
			SELECT
				dc."id" AS chunk_id,
				dc."documentId" AS document_id,
				dc."content",
				dc."chunkIndex" AS chunk_index,
				dc."metadata",
				dc."createdAt",
				(dc."embedding" <=> $1) AS vector_score,
				ROW_NUMBER() OVER (ORDER BY dc."embedding" <=> $1 ASC) AS vector_rank
			FROM "document_chunks" dc
			INNER JOIN "documents" d ON dc."documentId" = d."id"
			WHERE (dc."embedding" <=> $1) < $2
				AND d."status" = 'COMPLETED'
				AND %s%s
		),
		text_search AS (
			SELECT
				dc."id" AS chunk_id,
				ts_rank(to_tsvector('simple', dc."content"), plainto_tsquery('simple', $%d)) AS text_score,
				ROW_NUMBER() OVER (
					ORDER BY ts_rank(to_tsvector('simple', dc."content"), plainto_tsquery('simple', $%d)) DESC
				) AS text_rank
			FROM "document_chunks" dc
			INNER JOIN "documents" d ON dc."documentId" = d."id"
			WHERE to_tsvector('simple', dc."content") @@ plainto_tsquery('simple', $%d)
				AND d."status" = 'COMPLETED'
				AND %s%s
		),
		combined AS (
			SELECT
				v.chunk_id,
				v.document_id,
				v."content",
				v.chunk_index,
				v."metadata",
				v."createdAt",
				(1.0 / (60 + v.vector_rank)) + COALESCE((1.0 / (60 + t.text_rank)), 0) AS hybrid_score
			FROM vector_search v
			LEFT JOIN text_search t ON v.chunk_id = t.chunk_id
		)
		SELECT
			chunk_id,
			document_id,
			"content",
			chunk_index,
			"metadata",
			"createdAt",
			hybrid_score AS similarity
		FROM combined
		ORDER BY hybrid_score DESC
		LIMIT $%d
	`, accessCond, docFilter, textQueryParam, textQueryParam, textQueryParam, accessCond, docFilter, limitParam)

	rows, err := r.db.QueryContext(ctx, sql, args...)
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

func (r *chunkRepository) scanSimilarChunks(ctx context.Context, query string, args ...any) ([]entity.SimilarChunk, error) {
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

func (r *chunkRepository) SearchByKeywords(ctx context.Context, terms []string, access docaccess.Context, topK int, documentID string) ([]entity.SimilarChunk, error) {
	if len(terms) == 0 {
		return nil, nil
	}

	conditions := make([]string, 0, len(terms))
	args := make([]any, 0, len(terms)+3)
	for i, term := range terms {
		conditions = append(conditions, `dc."content" ILIKE $`+strconv.Itoa(i+1))
		args = append(args, "%"+term+"%")
	}

	accessStart := len(args) + 1
	accessCond, accessArgs := docaccess.SQLCondition("d", access, accessStart)
	args = append(args, accessArgs...)

	docFilter := ""
	if documentID = strings.TrimSpace(documentID); documentID != "" {
		docFilter = fmt.Sprintf(` AND dc."documentId" = $%d`, len(args)+1)
		args = append(args, documentID)
	}

	limitParam := len(args) + 1
	args = append(args, topK)

	query := fmt.Sprintf(`
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
		AND %s%s
		AND (`+strings.Join(conditions, " OR ")+`)
		ORDER BY dc."chunkIndex" ASC
		LIMIT $%d`, accessCond, docFilter, limitParam)

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
