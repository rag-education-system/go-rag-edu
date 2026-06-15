package repository

import (
	"context"
	"rag-api/internal/domain/entity"
)

type QueryLogRepository interface {
	Create(ctx context.Context, log *entity.QueryLog) error
}
