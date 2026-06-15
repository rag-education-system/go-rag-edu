package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pgvector/pgvector-go"
)

type EmbeddingClient struct {
	baseURL   string
	model     string
	dimension int
	client    *http.Client
}

type embedRequest struct {
	Model      string `json:"model"`
	Input      string `json:"input"`
	Dimensions int    `json:"dimensions,omitempty"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func NewEmbeddingClient(baseURL, model string, dimension int) *EmbeddingClient {
	return &EmbeddingClient{
		baseURL:   baseURL,
		model:     model,
		dimension: dimension,
		client:    &http.Client{Timeout: 5 * time.Minute},
	}
}

func (c *EmbeddingClient) GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error) {
	vectors, err := c.GenerateBatchEmbeddings(ctx, []string{text})
	if err != nil {
		return pgvector.Vector{}, err
	}
	if len(vectors) == 0 {
		return pgvector.Vector{}, fmt.Errorf("no embedding returned")
	}
	return vectors[0], nil
}

func (c *EmbeddingClient) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([]pgvector.Vector, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	vectors := make([]pgvector.Vector, 0, len(texts))
	for i, text := range texts {
		embedding, err := c.generateOne(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		vectors = append(vectors, pgvector.NewVector(embedding))
		if i < len(texts)-1 {
			time.Sleep(50 * time.Millisecond)
		}
	}
	return vectors, nil
}

func (c *EmbeddingClient) generateOne(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	body, err := json.Marshal(embedRequest{
		Model:      c.model,
		Input:      text,
		Dimensions: c.dimension,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API status %d: %s", resp.StatusCode, string(respBody))
	}

	var result embedResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("no embedding in ollama response")
	}
	return result.Embeddings[0], nil
}
