package postgres

import (
	"context"
	"time"

	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type messageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) repository.MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(ctx context.Context, msg *entity.Message) error {
	msg.ID = uuid.New().String()
	msg.CreatedAt = time.Now()

	query := `
		INSERT INTO "messages" ("id", "conversationId", "role", "content", "sources", "metadata", "createdAt")
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		msg.ID, msg.ConversationID, msg.Role, msg.Content,
		msg.Sources, msg.Metadata, msg.CreatedAt,
	)
	return err
}

func (r *messageRepository) ListByConversation(ctx context.Context, conversationID string, limit int) ([]entity.Message, error) {
	var messages []entity.Message
	query := `
		SELECT * FROM "messages"
		WHERE "conversationId" = $1
		ORDER BY "createdAt" ASC
		LIMIT $2
	`
	if err := r.db.SelectContext(ctx, &messages, query, conversationID, limit); err != nil {
		return nil, err
	}
	return messages, nil
}
