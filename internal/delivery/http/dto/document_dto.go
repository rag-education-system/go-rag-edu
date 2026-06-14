package dto

import "time"

type UploadDocumentResponse struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type DocumentInfo struct {
	ID           string    `json:"id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"originalName"`
	FileSize     int64     `json:"fileSize"`
	MimeType     string    `json:"mimeType"`
	Status       string    `json:"status"`
	TotalChunks  int       `json:"totalChunks"`
	Visibility   string    `json:"visibility"`
	CreatedAt    time.Time `json:"createdAt"`
}

type ListDocumentsResponse struct {
	Data []DocumentInfo `json:"data"`
	Meta PaginationMeta `json:"meta"`
}


type PaginationMeta struct {
	Total    int `json:"total"`
	Page     int `json:"page"`
	Limit    int `json:"limit"`
	TotalPages int `json:"totalPages"`
}	


type QueryDocumentRequest struct {
	Query   string               `json:"query" binding:"required"`
	History []ChatHistoryMessage `json:"history"`
}

type ChatHistoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type QueryDocumentResponse struct {
	Query   string        `json:"query"`
	Answer  string        `json:"answer"`
	Sources []ChunkSource `json:"sources"`
}

type ChunkSource struct {
	DocumentID string  `json:"documentId"`
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"`
	ChunkIndex int     `json:"chunkIndex"`
}

type DocumentChunkInfo struct {
	ChunkIndex int    `json:"chunkIndex"`
	Content    string `json:"content"`
}

type DocumentPreviewResponse struct {
	Document DocumentInfo        `json:"document"`
	Chunks   []DocumentChunkInfo `json:"chunks"`
}