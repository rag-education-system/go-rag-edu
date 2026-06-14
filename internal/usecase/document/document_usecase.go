package document

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"

	"github.com/pgvector/pgvector-go"
)

type ChatService interface {
	GenerateAnswer(ctx context.Context, query, context string) (string, error)
}

type EmbeddingService interface {
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([]pgvector.Vector, error)
}

type DocumentUsecase struct {
	docRepo     repository.DocumentRepository
	chunkRepo   repository.ChunkRepository
	embedder    EmbeddingService
	chatService ChatService
	extractor   *ContentExtractor
	chunker     *Chunker
	topK        int
	threshold   float64
}

func NewDocumentUsecase(
	docRepo repository.DocumentRepository,
	chunkRepo repository.ChunkRepository,
	embedder EmbeddingService,
	chatService ChatService,
	chunkSize, chunkOverlap int,
	topK int,
	threshold float64,
	ocrEnabled bool,
	ocrLang string,
	ocrMinTextLength int,
) *DocumentUsecase {
	return &DocumentUsecase{
		docRepo:     docRepo,
		chunkRepo:   chunkRepo,
		embedder:    embedder,
		chatService: chatService,
		extractor:   NewContentExtractor(ocrEnabled, ocrLang, ocrMinTextLength),
		chunker:     NewChunker(chunkSize, chunkOverlap),
		topK:        topK,
		threshold:   threshold,
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

	text := strings.TrimSpace(extraction.Text)
	if len(text) == 0 {
		return fmt.Errorf("no text extracted from document")
	}
	log.Printf(
		"Extracted %d characters from document %s (source=%s)",
		len(text),
		documentID,
		extraction.Source,
	)

	// 2 chunk text
	textChunks := uc.chunker.ChunkText(text)
	if len(textChunks) == 0 {
		return fmt.Errorf("no chunks generated")
	}
	log.Printf("Generated %d chunks from document %s", len(textChunks), documentID)

	// 3 generate embeddings
	embeddings, err := uc.embedder.GenerateBatchEmbeddings(ctx, textChunks)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}
	log.Printf("Generated %d embeddings from document %s", len(embeddings), documentID)

	// 4 create chunks with embeddings
	var chunks []entity.DocumentChunk
	for i, content := range textChunks {
		metadata, _ := json.Marshal(entity.ChunkMetadata{
			Source:     extraction.Source,
			PageNumber: i/10 + 1,
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
	userID string,
	page, limit int,
) ([]entity.Document, int, error) {
	return uc.docRepo.List(ctx, userID, page, limit)
}

// get document by id
func (uc *DocumentUsecase) GetDocumentByID(
	ctx context.Context,
	documentID string,
	userID string,
) (*entity.Document, error) {
	doc, err := uc.docRepo.FindByIDAndUserID(ctx, documentID, userID)
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
	userID string,
) (*entity.Document, []entity.DocumentChunk, error) {
	doc, err := uc.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return nil, nil, err
	}
	if doc == nil {
		return nil, nil, fmt.Errorf("document not found")
	}

	if doc.UserID != userID && doc.Visibility != entity.VisibilityPublic {
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

	return uc.docRepo.Delete(ctx, documentID)

}

// query document
func (uc *DocumentUsecase) QueryDocuments(
	ctx context.Context,
	query string,
) (string, []entity.SimilarChunk, error) {
	if isGreeting(query) {
		return "Halo! Saya siap membantu Anda. Silakan tanyakan apa saja tentang dokumen yang telah Anda upload.", nil, nil
	}

	// 1. generate embedding untuk query
	queryEmbedding, err := uc.embedder.GenerateBatchEmbeddings(ctx, []string{query})
	if err != nil {
		return "", nil, fmt.Errorf("failed to  generate query embedding: %w", err)
	}

	// 2. search similar chunks
	if len(queryEmbedding) == 0 {
		return "", nil, fmt.Errorf("no embedding generated for query")
	}
	chunks, err := uc.chunkRepo.SearchSimilar(ctx, queryEmbedding[0], uc.topK, uc.threshold)
	if err != nil {
		return "", nil, fmt.Errorf("failed to search similar chunks: %w", err)
	}

	if len(chunks) == 0 {
		return "Maaf, saya tidak menemukan informasi yang relevan dalam dokumen", nil, nil
	}

	// 3. build context from chunks
	var contextBuilder strings.Builder
	for i, chunk := range chunks {
		contextBuilder.WriteString(fmt.Sprintf("[Dokumen %d - Similarity: %.2f]\n%s\n\n", i+1, chunk.Similarity, chunk.Content))
	}

	// 4, generate answer using LLM
	answer, err := uc.chatService.GenerateAnswer(ctx, query, contextBuilder.String())
	if err != nil {
		return "", chunks, fmt.Errorf("failed to generate answer: %w", err)
	}

	return answer, chunks, nil
}
