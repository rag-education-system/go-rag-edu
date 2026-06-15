package postgres

import (
	"context"
	"time"

	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type queryLogRepository struct {
	db *sqlx.DB
}

func NewQueryLogRepository(db *sqlx.DB) repository.QueryLogRepository {
	return &queryLogRepository{db: db}
}

func (r *queryLogRepository) Create(ctx context.Context, log *entity.QueryLog) error {
	log.ID = uuid.New().String()
	log.CreatedAt = time.Now()

	query := `
		INSERT INTO "query_logs" (
			"id", "conversationId", "userId", "query", "reformulatedQuery",
			"searchType", "chunksRetrieved", "responseTimeMs", "metadata", "createdAt"
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		log.ID, log.ConversationID, log.UserID, log.Query, log.ReformulatedQuery,
		log.SearchType, log.ChunksRetrieved, log.ResponseTimeMs, log.Metadata, log.CreatedAt,
	)
	return err
}
