package main

import (
	"context"
	"flag"
	"log"
	"os"

	"rag-api/internal/adapter/ollama"
	"rag-api/internal/adapter/openai"
	"rag-api/internal/adapter/repository/postgres"
	"rag-api/internal/adapter/storage"
	"rag-api/internal/domain/entity"
	"rag-api/internal/usecase/document"
	"rag-api/pkg/config"
	"rag-api/pkg/database"
)

func main() {
	documentID := flag.String("id", "", "document ID to reprocess")
	flag.Parse()
	if *documentID == "" {
		log.Fatal("usage: go run ./cmd/reprocess -id <document-id>")
	}

	cfg := config.Load()
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	var embedder document.EmbeddingService
	if cfg.IsEmbeddingLocal {
		embedder = ollama.NewEmbeddingClient(cfg.OllamaBaseURL, cfg.OllamaEmbeddingModel, cfg.OllamaEmbeddingDimension)
	} else {
		embedder = openai.NewEmbeddingClient(cfg.OpenAIKey, cfg.OpenAIEmbeddingModel)
	}

	docRepo := postgres.NewDocumentRepository(db)
	chunkRepo := postgres.NewChunkRepository(db)
	fileStorage := storage.NewSupabaseStorage(cfg.SupabaseURL, cfg.SupabaseServiceKey, cfg.SupabaseStorageBucket)
	chatClient := openai.NewChatClient(cfg.OpenAIKey, cfg.OpenAIChatModel)
	chatService := openai.NewDocumentChatService(chatClient)
	reformulator := openai.NewReformulatorAdapter(openai.NewQueryReformulator(
		cfg.OpenAIKey,
		cfg.QueryReformulationModel,
		cfg.QueryReformulationTimeout,
		cfg.QueryReformulationEnabled,
	))

	docUC := document.NewDocumentUsecase(
		docRepo,
		chunkRepo,
		fileStorage,
		embedder,
		chatService,
		reformulator,
		cfg.ChunkSize,
		cfg.ChunkOverlap,
		cfg.TopKResults,
		cfg.SimilarityThreshold,
		cfg.UseHybridSearch,
		cfg.OCREnabled,
		cfg.OCRLang,
		cfg.OCRMinTextLength,
		cfg.FullDocMaxChars,
	)

	ctx := context.Background()
	doc, err := docRepo.FindByID(ctx, *documentID)
	if err != nil {
		log.Fatalf("find document: %v", err)
	}
	if doc == nil {
		log.Fatal("document not found")
	}
	if doc.StoragePath == "" {
		log.Fatal("document has no storage path")
	}

	log.Printf("reprocessing %s (%s)", doc.OriginalName, doc.ID)

	fileData, err := fileStorage.Download(ctx, doc.StoragePath)
	if err != nil {
		log.Fatalf("download file: %v", err)
	}

	if err := chunkRepo.DeleteByDocumentID(ctx, doc.ID); err != nil {
		log.Fatalf("delete chunks: %v", err)
	}
	if err := docRepo.UpdateTotalChunks(ctx, doc.ID, 0); err != nil {
		log.Fatalf("reset chunk count: %v", err)
	}
	if err := docRepo.UpdateStatus(ctx, doc.ID, entity.StatusProcessing); err != nil {
		log.Fatalf("set processing status: %v", err)
	}

	if err := docUC.ProcessDocument(ctx, doc.ID, fileData, doc.MimeType); err != nil {
		log.Fatalf("process document: %v", err)
	}

	updated, err := docRepo.FindByID(ctx, doc.ID)
	if err != nil {
		log.Fatalf("reload document: %v", err)
	}
	log.Printf("done: %s status=%s totalChunks=%d", updated.OriginalName, updated.Status, updated.TotalChunks)
	os.Exit(0)
}
