package handler

import (
	"io"
	"rag-api/internal/delivery/http/dto"
	"rag-api/internal/domain/entity"
	"rag-api/internal/usecase/document"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

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
// @Param        page   query  int  false  "Page number" default(1)
// @Param        limit  query  int  false  "Items per page" default(10)
// @Success      200  {object}  dto.ListDocumentsResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/documents [get]
func (h *DocumentHandler) List(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	docs, total, err := h.docUsecase.ListDocuments(c.Context(), userID, page, limit)
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
	var req dto.QueryDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	answer, chunks, err := h.docUsecase.QueryDocuments(c.Context(), req.Query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Convert chunks to sources
	var sources []dto.ChunkSource
	for _, chunk := range chunks {
		sources = append(sources, dto.ChunkSource{
			DocumentID: chunk.DocumentID,
			Content:    chunk.Content,
			Similarity: chunk.Similarity,
			ChunkIndex: chunk.ChunkIndex,
		})
	}

	return c.Status(fiber.StatusOK).JSON(dto.QueryDocumentResponse{
		Query:   req.Query,
		Answer:  answer,
		Sources: sources,
	})
}
