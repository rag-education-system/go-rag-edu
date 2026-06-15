package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"rag-api/internal/delivery/http/dto"
	"rag-api/internal/domain/entity"
	"rag-api/internal/usecase/document"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func parseChunkPageNumber(metadata []byte) int {
	if len(metadata) == 0 {
		return 0
	}

	var meta entity.ChunkMetadata
	if err := json.Unmarshal(metadata, &meta); err != nil {
		return 0
	}

	return meta.PageNumber
}

type DocumentHandler struct {
	docUsecase *document.DocumentUsecase
}

func NewDocumentHandler(docUsecase *document.DocumentUsecase) *DocumentHandler {
	return &DocumentHandler{docUsecase: docUsecase}
}

// Upload godoc
// @Summary      Upload a document
// @Description  Upload a PDF or image file for processing
// @Tags         Documents
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file        formData  file    true  "File to upload"
// @Param        visibility  formData  string  false "Visibility (PUBLIC or PRIVATE)" default(PRIVATE)
// @Success      201  {object}  dto.UploadDocumentResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/documents/upload [post]
func (h *DocumentHandler) Upload(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)

	// get file from form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to get file"})
	}

	// get visibility from form
	visibility := entity.VisibilityPrivate
	if c.FormValue("visibility") == "PUBLIC" {
		visibility = entity.VisibilityPublic
	}

	// read file data
	fileData, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to open file"})
	}
	defer fileData.Close()

	buf, err := io.ReadAll(fileData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read file"})
	}

	// upload document
	doc, err := h.docUsecase.UploadDocument(
		c.Context(),
		userID,
		file.Filename,
		buf,
		file.Header.Get("Content-Type"),
		visibility,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(dto.UploadDocumentResponse{
		ID:       doc.ID,
		Filename: doc.Filename,
		Status:   string(doc.Status),
		Message:  "Document uploaded successfully. Processing in background.",
	})
}

// List godoc
// @Summary      List documents
// @Description  Get a list of documents for the authenticated user
// @Tags         Documents
// @Produce      json
// @Security     BearerAuth
// @Param        page    query  int     false  "Page number" default(1)
// @Param        limit   query  int     false  "Items per page" default(10)
// @Param        status  query  string  false  "Filter by status (PROCESSING, COMPLETED, FAILED)"
// @Success      200  {object}  dto.ListDocumentsResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/documents [get]
func (h *DocumentHandler) List(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	statusFilter := parseDocumentStatusFilter(c.Query("status"))

	docs, total, err := h.docUsecase.ListDocuments(c.Context(), userID, page, limit, statusFilter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// convert to dto
	var docInfos []dto.DocumentInfo
	for _, doc := range docs {
		docInfos = append(docInfos, dto.DocumentInfo{
			ID:           doc.ID,
			Filename:     doc.Filename,
			OriginalName: doc.OriginalName,
			FileSize:     doc.FileSize,
			MimeType:     doc.MimeType,
			Status:       string(doc.Status),
			TotalChunks:  doc.TotalChunks,
			Visibility:   string(doc.Visibility),
			CreatedAt:    doc.CreatedAt,
		})
	}

	totalPages := (total + limit - 1) / limit

	return c.Status(fiber.StatusOK).JSON(dto.ListDocumentsResponse{
		Data: docInfos,
		Meta: dto.PaginationMeta{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	})

}

// GetByID godoc
// @Summary      Get document by ID
// @Description  Get a single document's details
// @Tags         Documents
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Document ID"
// @Success      200  {object}  dto.DocumentInfo
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/documents/{id} [get]
func (h *DocumentHandler) GetByID(c *fiber.Ctx) error {
	userId, _ := c.Locals("userID").(string)
	documentID := c.Params("id")

	doc, err := h.docUsecase.GetDocumentByID(c.Context(), documentID, userId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if doc == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Document not found"})
	}

	return c.Status(fiber.StatusOK).JSON(dto.DocumentInfo{
		ID:           doc.ID,
		Filename:     doc.Filename,
		OriginalName: doc.OriginalName,
		FileSize:     doc.FileSize,
		MimeType:     doc.MimeType,
		Status:       string(doc.Status),
		TotalChunks:  doc.TotalChunks,
		Visibility:   string(doc.Visibility),
		CreatedAt:    doc.CreatedAt,
	})
}

// Download godoc
// @Summary      Download document file
// @Description  Download the original uploaded file from storage
// @Tags         Documents
// @Produce      application/octet-stream
// @Security     BearerAuth
// @Param        id  path  string  true  "Document ID"
// @Success      200  {file}  binary
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/documents/{id}/download [get]
func (h *DocumentHandler) Download(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	documentID := c.Params("id")

	doc, data, err := h.docUsecase.DownloadDocument(c.Context(), documentID, userID)
	if err != nil {
		if err.Error() == "document not found" || err.Error() == "file not available" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	c.Set("Content-Type", doc.MimeType)

	disposition := "attachment"
	if c.Query("inline") == "1" || c.Query("inline") == "true" {
		disposition = "inline"
	}
	c.Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, doc.OriginalName))
	return c.Send(data)
}

// GetPreview godoc
// @Summary      Get document preview with chunks
// @Description  Get document metadata and text chunks for preview
// @Tags         Documents
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Document ID"
// @Success      200  {object}  dto.DocumentPreviewResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/documents/{id}/chunks [get]
func (h *DocumentHandler) GetPreview(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	documentID := c.Params("id")

	doc, chunks, err := h.docUsecase.GetDocumentPreview(c.Context(), documentID, userID)
	if err != nil {
		if err.Error() == "document not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var chunkInfos []dto.DocumentChunkInfo
	for _, chunk := range chunks {
		chunkInfos = append(chunkInfos, dto.DocumentChunkInfo{
			ChunkIndex: chunk.ChunkIndex,
			Content:    chunk.Content,
			PageNumber: parseChunkPageNumber(chunk.Metadata),
		})
	}

	return c.Status(fiber.StatusOK).JSON(dto.DocumentPreviewResponse{
		Document: dto.DocumentInfo{
			ID:           doc.ID,
			Filename:     doc.Filename,
			OriginalName: doc.OriginalName,
			FileSize:     doc.FileSize,
			MimeType:     doc.MimeType,
			Status:       string(doc.Status),
			TotalChunks:  doc.TotalChunks,
			Visibility:   string(doc.Visibility),
			CreatedAt:    doc.CreatedAt,
		},
		Chunks: chunkInfos,
	})
}

// Reprocess godoc
// @Summary      Reprocess a document
// @Description  Re-extract text and rebuild chunks/embeddings for an existing document
// @Tags         Documents
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Document ID"
// @Success      200  {object}  dto.UploadDocumentResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/documents/{id}/reprocess [post]
func (h *DocumentHandler) Reprocess(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	documentID := c.Params("id")

	doc, err := h.docUsecase.ReprocessDocument(c.Context(), documentID, userID)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "document not found" || err.Error() == "file not available" {
			status = fiber.StatusNotFound
		}
		if err.Error() == "document is still processing" {
			status = fiber.StatusBadRequest
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(dto.UploadDocumentResponse{
		ID:       doc.ID,
		Filename: doc.Filename,
		Status:   string(doc.Status),
		Message:  "Document reprocessing started",
	})
}

// Delete godoc
// @Summary      Delete a document
// @Description  Delete a document by ID
// @Tags         Documents
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Document ID"
// @Success      200  {object}  dto.MessageResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/documents/{id} [delete]
func (h *DocumentHandler) Delete(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	documentID := c.Params("id")

	if err := h.docUsecase.DeleteDocument(c.Context(), documentID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Document deleted successfully"})
}

// Query godoc
// @Summary      Query documents with RAG
// @Description  Search documents using natural language and get AI-generated answer
// @Tags         Documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.QueryDocumentRequest  true  "Query Request"
// @Success      200      {object}  dto.QueryDocumentResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /api/documents/query [post]
func (h *DocumentHandler) Query(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)

	var req dto.QueryDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	answer, chunks, err := h.docUsecase.QueryDocuments(c.Context(), userID, req.Query, toChatHistory(req.History))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var sources []dto.ChunkSource
	for _, chunk := range chunks {
		documentName := h.docUsecase.GetDocumentOriginalName(c.Context(), chunk.DocumentID)
		sources = append(sources, dto.ChunkSource{
			DocumentID:    chunk.DocumentID,
			DocumentName:  documentName,
			Similarity:    chunk.Similarity,
			Content:       chunk.Content,
			ChunkIndex:    chunk.ChunkIndex,
			PageNumber:    parseChunkPageNumber(chunk.Metadata),
			LowConfidence: document.IsLowConfidenceSource(chunk.Similarity, chunk.Content),
		})
	}

	return c.Status(fiber.StatusOK).JSON(dto.QueryDocumentResponse{
		Query:   req.Query,
		Answer:  answer,
		Sources: sources,
	})
}

func toChatHistory(history []dto.ChatHistoryMessage) []document.ChatMessage {
	messages := make([]document.ChatMessage, 0, len(history))
	for _, item := range history {
		messages = append(messages, document.ChatMessage{
			Role:    item.Role,
			Content: item.Content,
		})
	}
	return messages
}

func parseDocumentStatusFilter(status string) *entity.DocumentStatus {
	switch status {
	case string(entity.StatusProcessing):
		value := entity.StatusProcessing
		return &value
	case string(entity.StatusCompleted):
		value := entity.StatusCompleted
		return &value
	case string(entity.StatusFailed):
		value := entity.StatusFailed
		return &value
	default:
		return nil
	}
}
