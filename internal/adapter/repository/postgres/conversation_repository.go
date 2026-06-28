package postgres

import (
	"context"
	"database/sql"
	"time"

	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type conversationRepository struct {
	db *sqlx.DB
}

func NewConversationRepository(db *sqlx.DB) repository.ConversationRepository {
	return &conversationRepository{db: db}
}

func (r *conversationRepository) Create(ctx context.Context, conv *entity.Conversation) error {
	conv.ID = uuid.New().String()
	conv.CreatedAt = time.Now()
	conv.UpdatedAt = time.Now()

	query := `
		INSERT INTO "conversations" ("id", "userId", "title", "documentId", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		conv.ID, conv.UserID, conv.Title, conv.DocumentID, conv.CreatedAt, conv.UpdatedAt,
	)
	return err
}

func (r *conversationRepository) FindByID(ctx context.Context, id string) (*entity.Conversation, error) {
	var conv entity.Conversation
	query := `SELECT * FROM "conversations" WHERE "id" = $1`
	err := r.db.GetContext(ctx, &conv, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *conversationRepository) FindByIDAndUserID(ctx context.Context, id, userID string) (*entity.Conversation, error) {
	var conv entity.Conversation
	query := `SELECT * FROM "conversations" WHERE "id" = $1 AND "userId" = $2`
	err := r.db.GetContext(ctx, &conv, query, id, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *conversationRepository) List(ctx context.Context, userID string, page, limit int) ([]entity.Conversation, int, error) {
	offset := (page - 1) * limit

	var convs []entity.Conversation
	query := `
		SELECT * FROM "conversations"
		WHERE "userId" = $1
		ORDER BY "pinned" DESC, "pinnedAt" DESC NULLS LAST, "updatedAt" DESC
		LIMIT $2 OFFSET $3
	`
	if err := r.db.SelectContext(ctx, &convs, query, userID, limit, offset); err != nil {
		return nil, 0, err
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM "conversations" WHERE "userId" = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, userID); err != nil {
		return nil, 0, err
	}

	return convs, total, nil
}

func (r *conversationRepository) Update(ctx context.Context, conv *entity.Conversation) error {
	conv.UpdatedAt = time.Now()
	query := `UPDATE "conversations" SET "title" = $1, "updatedAt" = $2 WHERE "id" = $3`
	_, err := r.db.ExecContext(ctx, query, conv.Title, conv.UpdatedAt, conv.ID)
	return err
}

func (r *conversationRepository) SetPinned(ctx context.Context, id string, pinned bool) error {
	now := time.Now()
	var pinnedAt *time.Time
	if pinned {
		pinnedAt = &now
	}

	query := `UPDATE "conversations" SET "pinned" = $1, "pinnedAt" = $2, "updatedAt" = $3 WHERE "id" = $4`
	_, err := r.db.ExecContext(ctx, query, pinned, pinnedAt, now, id)
	return err
}

func (r *conversationRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM "conversations" WHERE "id" = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
