package entity

import "time"

type DocumentStatus string
type DocumentVisibility string

const (
	StatusProcessing DocumentStatus = "PROCESSING"
	StatusCompleted  DocumentStatus = "COMPLETED"
	StatusFailed     DocumentStatus = "FAILED"

	VisibilityPublic  DocumentVisibility = "PUBLIC"
	VisibilityPrivate DocumentVisibility = "PRIVATE"
)

type Document struct {
	ID           string             `db:"id" json:"id"`
	UserID       string             `db:"userId" json:"userId"`
	Filename     string             `db:"filename" json:"filename"`
	OriginalName string             `db:"originalName" json:"originalName"`
	FileSize     int64              `db:"fileSize" json:"fileSize"`
	MimeType     string             `db:"mimeType" json:"mimeType"`
	StoragePath  string             `db:"storagePath" json:"storagePath,omitempty"`
	Status       DocumentStatus     `db:"status" json:"status"`
	TotalChunks  int                `db:"totalChunks" json:"totalChunks"`
	Visibility   DocumentVisibility `db:"visibility" json:"visibility"`
	CreatedAt    time.Time          `db:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `db:"updatedAt" json:"updatedAt"`
}
