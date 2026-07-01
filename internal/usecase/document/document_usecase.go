package document

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"rag-api/internal/domain/docaccess"
	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"

	"github.com/pgvector/pgvector-go"
)

type ChatService interface {
	GenerateAnswer(ctx context.Context, query, docContext string, history []ChatMessage) (string, error)
	GenerateAnswerStream(ctx context.Context, query, docContext string, history []ChatMessage) (<-chan string, <-chan error)
}

type QueryReformulator interface {
	ReformulateQuery(ctx context.Context, query string, history []ChatMessage) (string, error)
	Enabled() bool
}

type EmbeddingService interface {
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([]pgvector.Vector, error)
}

type DocumentUsecase struct {
	docRepo         repository.DocumentRepository
	chunkRepo       repository.ChunkRepository
	fileStorage     repository.FileStorage
	embedder        EmbeddingService
	chatService     ChatService
	reformulator    QueryReformulator
	extractor       *ContentExtractor
	chunker         *Chunker
	topK            int
	threshold       float64
	useHybridSearch bool
	fullDocMaxChars int
}

func NewDocumentUsecase(
	docRepo repository.DocumentRepository,
	chunkRepo repository.ChunkRepository,
	fileStorage repository.FileStorage,
	embedder EmbeddingService,
	chatService ChatService,
	reformulator QueryReformulator,
	chunkSize, chunkOverlap int,
	topK int,
	threshold float64,
	useHybridSearch bool,
	ocrEnabled bool,
	ocrLang string,
	ocrMinTextLength int,
	fullDocMaxChars int,
) *DocumentUsecase {
	return &DocumentUsecase{
		docRepo:         docRepo,
		chunkRepo:       chunkRepo,
		fileStorage:     fileStorage,
		embedder:        embedder,
		chatService:     chatService,
		reformulator:    reformulator,
		extractor:       NewContentExtractor(ocrEnabled, ocrLang, ocrMinTextLength),
		chunker:         NewChunker(chunkSize, chunkOverlap),
		topK:            topK,
		threshold:       threshold,
		useHybridSearch: useHybridSearch,
		fullDocMaxChars: fullDocMaxChars,
	}
}

// upload document
func (uc *DocumentUsecase) UploadDocument(
	ctx context.Context,
	userID string,
	filename string,
	fileData []byte,
	mimeType string,
	visibility entity.DocumentVisibility,
) (*entity.Document, error) {

	// create document record
	doc := &entity.Document{
		UserID:       userID,
		Filename:     fmt.Sprintf("%s_%d_%s", userID, time.Now().Unix(), filename),
		OriginalName: filename,
		FileSize:     int64(len(fileData)),
		MimeType:     mimeType,
		Status:       entity.StatusProcessing,
		Visibility:   visibility,
		TotalChunks:  0,
	}

	if err := uc.docRepo.Create(ctx, doc); err != nil {
		return nil, err
	}

	storagePath := fmt.Sprintf("%s/%s/%s", userID, doc.ID, doc.Filename)
	if err := uc.fileStorage.Upload(ctx, storagePath, fileData, mimeType); err != nil {
		_ = uc.docRepo.Delete(ctx, doc.ID)
		return nil, fmt.Errorf("failed to store file: %w", err)
	}

	if err := uc.docRepo.UpdateStoragePath(ctx, doc.ID, storagePath); err != nil {
		_ = uc.fileStorage.Delete(ctx, storagePath)
		_ = uc.docRepo.Delete(ctx, doc.ID)
		return nil, fmt.Errorf("failed to update storage path: %w", err)
	}
	doc.StoragePath = storagePath

	// process document in background
	go func() {
		// recovery for panic in background process
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in document processing for doc %s: %v", doc.ID, r)
				uc.docRepo.UpdateStatus(context.Background(), doc.ID, entity.StatusFailed)
			}
		}()

		if err := uc.ProcessDocument(context.Background(), doc.ID, fileData, mimeType); err != nil {
			log.Printf("Error processing document %s: %v", doc.ID, err)
			uc.docRepo.UpdateStatus(context.Background(), doc.ID, entity.StatusFailed)
		}
	}()

	return doc, nil

}

// process document
func (uc DocumentUsecase) ProcessDocument(
	ctx context.Context,
	documentID string,
	fileData []byte,
	mimeType string,
) error {
	log.Printf("Starting processing for document %s", documentID)

	// 1 extract text (plain PDF text layer and/or Tesseract OCR)
	extraction, err := uc.extractor.Extract(fileData, mimeType)
	if err != nil {
		return fmt.Errorf("failed to extract content: %w", err)
	}

	extraction.Text = SanitizeUTF8(extraction.Text)
	text := strings.TrimSpace(CleanReadableContent(extraction.Text))
	if len(text) == 0 {
		return fmt.Errorf("no text extracted from document")
	}
	log.Printf(
		"Extracted %d characters from document %s (source=%s, pages=%d)",
		len(text),
		documentID,
		extraction.Source,
		len(extraction.Pages),
	)

	var textChunks []string
	var chunkPageNumbers []int
	if len(extraction.Pages) > 0 {
		textChunks, chunkPageNumbers = uc.chunker.ChunkPages(extraction.Pages)
	} else {
		textChunks = uc.chunker.ChunkText(text)
		chunkPageNumbers = make([]int, len(textChunks))
		for i := range chunkPageNumbers {
			chunkPageNumbers[i] = 1
		}
	}

	if len(textChunks) == 0 {
		return fmt.Errorf("no chunks generated")
	}
	log.Printf("Generated %d raw chunks from document %s", len(textChunks), documentID)

	filteredChunks, filteredIndices := FilterChunksForEmbedding(textChunks)
	if len(filteredChunks) == 0 {
		return fmt.Errorf("no quality chunks after filtering")
	}

	filteredPageNumbers := make([]int, len(filteredIndices))
	for i, idx := range filteredIndices {
		if idx < len(chunkPageNumbers) {
			filteredPageNumbers[i] = chunkPageNumbers[idx]
		} else {
			filteredPageNumbers[i] = 1
		}
	}

	textChunks = filteredChunks
	chunkPageNumbers = filteredPageNumbers
	log.Printf("Filtered to %d quality chunks from document %s", len(textChunks), documentID)

	// 3 generate embeddings
	embeddings, err := uc.embedder.GenerateBatchEmbeddings(ctx, textChunks)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}
	log.Printf("Generated %d embeddings from document %s", len(embeddings), documentID)

	// 4 create chunks with embeddings
	var chunks []entity.DocumentChunk
	for i, content := range textChunks {
		pageNumber := 1
		if i < len(chunkPageNumbers) {
			pageNumber = chunkPageNumbers[i]
		}

		content = SanitizeUTF8(content)

		metadata, _ := json.Marshal(entity.ChunkMetadata{
			Source:     extraction.Source,
			PageNumber: pageNumber,
		})
		chunks = append(chunks, entity.DocumentChunk{
			DocumentID: documentID,
			ChunkIndex: i,
			Content:    content,
			Embedding:  embeddings[i],
			Metadata:   metadata,
		})
	}

	// 5 save chunks
	if err := uc.chunkRepo.CreateBatch(ctx, chunks); err != nil {
		return fmt.Errorf("failed to save chunks: %w", err)
	}
	log.Printf("Saved %d chunks to database for document %s", len(chunks), documentID)

	// 6 update document status
	if err := uc.docRepo.UpdateTotalChunks(ctx, documentID, len(chunks)); err != nil {
		return err
	}

	if err := uc.docRepo.UpdateStatus(ctx, documentID, entity.StatusCompleted); err != nil {
		return err
	}

	log.Printf("Document %s processed successfully with %d chunks", documentID, len(chunks))
	return nil

}

// list document
func (uc *DocumentUsecase) ListDocuments(
	ctx context.Context,
	access docaccess.Context,
	page, limit int,
	status *entity.DocumentStatus,
) ([]entity.DocumentWithUploader, int, error) {
	return uc.docRepo.List(ctx, access, page, limit, status)
}

func (uc *DocumentUsecase) GetDocumentOriginalName(ctx context.Context, documentID string) string {
	doc, err := uc.docRepo.FindByID(ctx, documentID)
	if err != nil || doc == nil {
		return ""
	}
	return doc.OriginalName
}

// get document by id
func (uc *DocumentUsecase) GetDocumentByID(
	ctx context.Context,
	documentID string,
	access docaccess.Context,
) (*entity.Document, error) {
	doc, err := uc.docRepo.FindByIDWithAccess(ctx, documentID, access)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, nil
	}

	return doc, nil

}

func (uc *DocumentUsecase) GetDocumentPreview(
	ctx context.Context,
	documentID string,
	access docaccess.Context,
) (*entity.Document, []entity.DocumentChunk, error) {
	doc, err := uc.docRepo.FindByIDWithAccess(ctx, documentID, access)
	if err != nil {
		return nil, nil, err
	}
	if doc == nil {
		return nil, nil, fmt.Errorf("document not found")
	}

	chunks, err := uc.chunkRepo.FindByDocumentID(ctx, documentID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get document chunks: %w", err)
	}

	return doc, chunks, nil
}

// delete
func (uc *DocumentUsecase) DeleteDocument(
	ctx context.Context,
	documentID string,
	userID string,
) error {
	doc, err := uc.docRepo.FindByIDAndUserID(ctx, documentID, userID)
	if err != nil {
		return err
	}
	if doc == nil {
		return fmt.Errorf("document not found")
	}

	// Delete chunks first
	if err := uc.chunkRepo.DeleteByDocumentID(ctx, documentID); err != nil {
		return err
	}

	if doc.StoragePath != "" {
		if err := uc.fileStorage.Delete(ctx, doc.StoragePath); err != nil {
			return fmt.Errorf("failed to delete file from storage: %w", err)
		}
	}

	return uc.docRepo.Delete(ctx, documentID)

}

func (uc *DocumentUsecase) DownloadDocument(
	ctx context.Context,
	documentID string,
	access docaccess.Context,
) (*entity.Document, []byte, error) {
	doc, err := uc.docRepo.FindByIDWithAccess(ctx, documentID, access)
	if err != nil {
		return nil, nil, err
	}
	if doc == nil {
		return nil, nil, fmt.Errorf("document not found")
	}

	if doc.StoragePath == "" {
		return nil, nil, fmt.Errorf("file not available")
	}

	data, err := uc.fileStorage.Download(ctx, doc.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download file: %w", err)
	}

	return doc, data, nil
}

func (uc *DocumentUsecase) UpdateDocumentVisibility(
	ctx context.Context,
	documentID, userID string,
	visibility entity.DocumentVisibility,
) (*entity.Document, error) {
	doc, err := uc.docRepo.FindByIDAndUserID(ctx, documentID, userID)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("document not found")
	}

	if err := uc.docRepo.UpdateVisibility(ctx, documentID, visibility); err != nil {
		return nil, err
	}

	doc.Visibility = visibility
	return doc, nil
}

func (uc *DocumentUsecase) ReprocessDocument(
	ctx context.Context,
	documentID string,
	userID string,
) (*entity.Document, error) {
	doc, err := uc.docRepo.FindByIDAndUserID(ctx, documentID, userID)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("document not found")
	}
	if doc.StoragePath == "" {
		return nil, fmt.Errorf("file not available")
	}
	if doc.Status == entity.StatusProcessing {
		return nil, fmt.Errorf("document is still processing")
	}

	fileData, err := uc.fileStorage.Download(ctx, doc.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	if err := uc.chunkRepo.DeleteByDocumentID(ctx, documentID); err != nil {
		return nil, fmt.Errorf("failed to delete existing chunks: %w", err)
	}

	if err := uc.docRepo.UpdateTotalChunks(ctx, documentID, 0); err != nil {
		return nil, err
	}

	if err := uc.docRepo.UpdateStatus(ctx, documentID, entity.StatusProcessing); err != nil {
		return nil, err
	}

	doc.Status = entity.StatusProcessing
	doc.TotalChunks = 0

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in document reprocess for doc %s: %v", documentID, r)
				uc.docRepo.UpdateStatus(context.Background(), documentID, entity.StatusFailed)
			}
		}()

		if err := uc.ProcessDocument(context.Background(), documentID, fileData, doc.MimeType); err != nil {
			log.Printf("Error reprocessing document %s: %v", documentID, err)
			uc.docRepo.UpdateStatus(context.Background(), documentID, entity.StatusFailed)
		}
	}()

	return doc, nil
}

// query document
func (uc *DocumentUsecase) QueryDocuments(
	ctx context.Context,
	access docaccess.Context,
	query string,
	history []ChatMessage,
) (string, []entity.SimilarChunk, error) {
	return uc.QueryDocumentsForDocument(ctx, access, "", query, history)
}

// QueryDocumentsForDocument answers a query optionally scoped to a single
// document. When documentID is set and the document fits the full-document
// budget, the whole document is used as context (see PrepareRAGForDocument).
func (uc *DocumentUsecase) QueryDocumentsForDocument(
	ctx context.Context,
	access docaccess.Context,
	documentID string,
	query string,
	history []ChatMessage,
) (string, []entity.SimilarChunk, error) {
	rag, err := uc.PrepareRAGForDocument(ctx, access, documentID, query, history)
	if err != nil {
		return "", nil, err
	}

	if rag.Answer != "" {
		return rag.Answer, nil, nil
	}

	plan := PlanRAGResponse(ChatModeHybrid, rag, query)
	if !plan.UseLLM {
		return plan.DirectAnswer, nil, nil
	}

	answer, err := uc.chatService.GenerateAnswer(ctx, query, plan.DocContext, history)
	if err != nil {
		return "", rag.DisplaySources, fmt.Errorf("failed to generate answer: %w", err)
	}

	return answer, rag.DisplaySources, nil
}

func (uc *DocumentUsecase) GenerateAnswerStream(
	ctx context.Context,
	query string,
	docContext string,
	history []ChatMessage,
) (<-chan string, <-chan error) {
	return uc.chatService.GenerateAnswerStream(ctx, query, docContext, history)
}
