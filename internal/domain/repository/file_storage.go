package repository

import "context"

type FileStorage interface {
	Upload(ctx context.Context, path string, data []byte, contentType string) error
	Download(ctx context.Context, path string) ([]byte, error)
	Delete(ctx context.Context, path string) error
}
