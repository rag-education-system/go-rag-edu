package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"rag-api/internal/adapter/ollama"
	"rag-api/internal/adapter/openai"
	"rag-api/internal/adapter/repository/postgres"
	"rag-api/internal/adapter/storage"
	"rag-api/internal/domain/docaccess"
	"rag-api/internal/domain/entity"
	"rag-api/internal/usecase/document"
	"rag-api/pkg/config"
	"rag-api/pkg/database"
)

func main() {
	query := flag.String("q", "apa saja keunggulan dari prodi ptik", "query to test")
	flag.Parse()

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

	access := docaccess.Context{Role: entity.RoleAdmin}
	rag, err := docUC.PrepareRAG(context.Background(), access, *query, nil)
	if err != nil {
		log.Fatalf("prepare rag: %v", err)
	}

	fmt.Printf("searchType=%s reformulated=%q\n", rag.SearchType, rag.ReformulatedQuery)
	for _, source := range rag.DisplaySources {
		name := docUC.GetDocumentOriginalName(context.Background(), source.DocumentID)
		fmt.Printf("source: %s (similarity=%.2f)\n", name, source.Similarity)
	}
	if len(rag.DisplaySources) == 0 {
		fmt.Println("no display sources")
		os.Exit(1)
	}
	if name := docUC.GetDocumentOriginalName(context.Background(), rag.DisplaySources[0].DocumentID); name != "Poster-fix-A3.pdf" {
		fmt.Printf("top source is %s, expected Poster-fix-A3.pdf\n", name)
		os.Exit(1)
	}
	fmt.Println("ok: top source is Poster-fix-A3.pdf")
}
