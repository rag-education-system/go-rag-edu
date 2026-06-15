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

type documentRepository struct {
	db *sqlx.DB
}

func NewDocumentRepository(db *sqlx.DB) repository.DocumentRepository {
	return &documentRepository{db: db}
}

// create document
func (r *documentRepository) Create(ctx context.Context, doc *entity.Document) error {
	doc.ID = uuid.New().String()
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()

	query := `
			INSERT INTO documents (id, "userId", filename, "originalName", "fileSize", "mimeType", "storagePath", status, "totalChunks", visibility, "createdAt", "updatedAt")
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`
	_, err := r.db.ExecContext(ctx, query, doc.ID, doc.UserID, doc.Filename, doc.OriginalName, doc.FileSize, doc.MimeType, doc.StoragePath, doc.Status, doc.TotalChunks, doc.Visibility, doc.CreatedAt, doc.UpdatedAt)
	return err

}

// find document by id
func (r *documentRepository) FindByID(ctx context.Context, id string) (*entity.Document, error) {
	var doc entity.Document
	query := `SELECT * FROM documents WHERE id = $1`
	err := r.db.GetContext(ctx, &doc, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// find document by id and user id
func (r *documentRepository) FindByIDAndUserID(ctx context.Context, id, userID string) (*entity.Document, error) {
	var doc entity.Document
	query := `SELECT * FROM documents WHERE id = $1 AND "userId" = $2`
	err := r.db.GetContext(ctx, &doc, query, id, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// list document
func (r *documentRepository) List(
	ctx context.Context,
	userID string,
	page, limit int,
	status *entity.DocumentStatus,
) ([]entity.Document, int, error) {
	offset := (page - 1) * limit

	var docs []entity.Document
	var total int

	if status != nil {
		query := `SELECT * FROM documents WHERE "userId" = $1 AND status = $2 ORDER BY "createdAt" DESC LIMIT $3 OFFSET $4`
		if err := r.db.SelectContext(ctx, &docs, query, userID, *status, limit, offset); err != nil {
			return nil, 0, err
		}

		countQuery := `SELECT COUNT(*) FROM documents WHERE "userId" = $1 AND status = $2`
		if err := r.db.GetContext(ctx, &total, countQuery, userID, *status); err != nil {
			return nil, 0, err
		}

		return docs, total, nil
	}

	query := `SELECT * FROM documents WHERE "userId" = $1 ORDER BY "createdAt" DESC LIMIT $2 OFFSET $3`
	if err := r.db.SelectContext(ctx, &docs, query, userID, limit, offset); err != nil {
		return nil, 0, err
	}

	countQuery := `SELECT COUNT(*) FROM documents WHERE "userId" = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, userID); err != nil {
		return nil, 0, err
	}

	return docs, total, nil
}

// update  status
func (r *documentRepository) UpdateStatus(ctx context.Context, id string, status entity.DocumentStatus) error {
	query := `UPDATE documents SET status = $1, "updatedAt" = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// update total chunks
func (r *documentRepository) UpdateTotalChunks(ctx context.Context, id string, totalChunks int) error {
	query := `UPDATE documents SET "totalChunks" = $1, "updatedAt" = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, totalChunks, id)
	return err
}

func (r *documentRepository) UpdateStoragePath(ctx context.Context, id string, storagePath string) error {
	query := `UPDATE documents SET "storagePath" = $1, "updatedAt" = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, storagePath, id)
	return err
}

// delete document
func (r *documentRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM documents WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
