package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"rag-api/internal/domain/docaccess"
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

func (r *documentRepository) FindByIDWithAccess(ctx context.Context, id string, access docaccess.Context) (*entity.Document, error) {
	accessCond, accessArgs := docaccess.SQLCondition("d", access, 2)
	query := fmt.Sprintf(`SELECT d.* FROM documents d WHERE d.id = $1 AND %s`, accessCond)

	args := append([]any{id}, accessArgs...)
	var doc entity.Document
	err := r.db.GetContext(ctx, &doc, query, args...)
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
	access docaccess.Context,
	page, limit int,
	status *entity.DocumentStatus,
) ([]entity.DocumentWithUploader, int, error) {
	offset := (page - 1) * limit

	accessCond, accessArgs := docaccess.SQLCondition("d", access, 1)
	whereParts := []string{accessCond}
	args := append([]any{}, accessArgs...)
	argIdx := len(args) + 1

	if status != nil {
		whereParts = append(whereParts, fmt.Sprintf(`d.status = $%d`, argIdx))
		args = append(args, *status)
		argIdx++
	}

	whereClause := ""
	for i, part := range whereParts {
		if i > 0 {
			whereClause += " AND "
		}
		whereClause += part
	}

	listQuery := fmt.Sprintf(
		`SELECT d.*, u.name AS uploader_name, u.role AS uploader_role
		 FROM documents d
		 LEFT JOIN users u ON u.id = d."userId"
		 WHERE %s
		 ORDER BY d."createdAt" DESC
		 LIMIT $%d OFFSET $%d`,
		whereClause,
		argIdx,
		argIdx+1,
	)
	listArgs := append(append([]any{}, args...), limit, offset)

	var docs []entity.DocumentWithUploader
	if err := r.db.SelectContext(ctx, &docs, listQuery, listArgs...); err != nil {
		return nil, 0, err
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM documents d WHERE %s`, whereClause)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
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

func (r *documentRepository) UpdateVisibility(ctx context.Context, id string, visibility entity.DocumentVisibility) error {
	query := `UPDATE documents SET visibility = $1, "updatedAt" = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, visibility, id)
	return err
}

// delete document
func (r *documentRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM documents WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
