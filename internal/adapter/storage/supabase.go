package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"rag-api/internal/domain/repository"
)

type SupabaseStorage struct {
	baseURL    string
	apiKey     string
	bucket     string
	httpClient *http.Client
}

func NewSupabaseStorage(baseURL, apiKey, bucket string) repository.FileStorage {
	return &SupabaseStorage{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		bucket:  bucket,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (s *SupabaseStorage) objectURL(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	encodedPath := strings.Join(segments, "/")
	return fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, s.bucket, encodedPath)
}

func (s *SupabaseStorage) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("apikey", s.apiKey)
}

func (s *SupabaseStorage) Upload(ctx context.Context, path string, data []byte, contentType string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.objectURL(path), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}

	s.setAuthHeaders(req)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload to supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func (s *SupabaseStorage) Download(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.objectURL(path), nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}

	s.setAuthHeaders(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download from supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read download body: %w", err)
	}

	return data, nil
}

func (s *SupabaseStorage) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.objectURL(path), nil)
	if err != nil {
		return fmt.Errorf("create delete request: %w", err)
	}

	s.setAuthHeaders(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete from supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}
