package entity

import (
	"time"

	"github.com/pgvector/pgvector-go"
)

type ChunkMetadata struct {
	Source     string  `json:"source"` // "text" or "ocr"
	PageNumber int     `json:"pageNumber,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

type DocumentChunk struct {
	ID         string          `db:"id" json:"id"`
	DocumentID string          `db:"documentId" json:"documentId"`
	ChunkIndex int             `db:"chunkIndex" json:"chunkIndex"`
	Content    string          `db:"content" json:"content"`
	Embedding  pgvector.Vector `db:"embedding" json:"-"`
	Metadata   []byte          `db:"metadata" json:"metadata"`
	CreatedAt  time.Time       `db:"createdAt" json:"createdAt"`
}

type SimilarChunk struct {
	DocumentChunk
	Similarity   float64 `db:"similarity" json:"similarity"`
	DocumentName string  `json:"documentName,omitempty"`
}
