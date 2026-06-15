package main

import (
	"fmt"
	"log"

	_ "rag-api/docs"
	"rag-api/internal/adapter/ollama"
	"rag-api/internal/adapter/openai"
	"rag-api/internal/adapter/repository/postgres"
	"rag-api/internal/adapter/storage"
	"rag-api/internal/delivery/http/handler"
	"rag-api/internal/delivery/http/middleware"
	"rag-api/internal/usecase/auth"
	"rag-api/internal/usecase/chat"
	"rag-api/internal/usecase/document"
	"rag-api/pkg/config"
	"rag-api/pkg/database"

	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/swaggo/fiber-swagger"

	"github.com/gofiber/fiber/v2/middleware/logger"
)

// @title           RAG API
// @version         1.0
// @description     API documentation for the RAG (Retrieval-Augmented Generation) service
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	cfg := config.Load()

	if cfg.SupabaseURL == "" || cfg.SupabaseServiceKey == "" {
		log.Fatal("SUPABASE_URL and SUPABASE_SERVICE_KEY are required for file storage")
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("connected to database")

	var embedder document.EmbeddingService
	if cfg.IsEmbeddingLocal {
		embedder = ollama.NewEmbeddingClient(cfg.OllamaBaseURL, cfg.OllamaEmbeddingModel, cfg.OllamaEmbeddingDimension)
		log.Printf("🔢 Embedding: Ollama local model=%s dim=%d", cfg.OllamaEmbeddingModel, cfg.OllamaEmbeddingDimension)
	} else {
		embedder = openai.NewEmbeddingClient(cfg.OpenAIKey, cfg.OpenAIEmbeddingModel)
		log.Printf("🔢 Embedding: OpenAI model=%s", cfg.OpenAIEmbeddingModel)
	}

	chatClient := openai.NewChatClient(cfg.OpenAIKey, cfg.OpenAIChatModel)
	chatService := openai.NewDocumentChatService(chatClient)

	reformulator := openai.NewReformulatorAdapter(openai.NewQueryReformulator(
		cfg.OpenAIKey,
		cfg.QueryReformulationModel,
		cfg.QueryReformulationTimeout,
		cfg.QueryReformulationEnabled,
	))

	userRepo := postgres.NewUserRepository(db)
	docRepo := postgres.NewDocumentRepository(db)
	chunkRepo := postgres.NewChunkRepository(db)
	convRepo := postgres.NewConversationRepository(db)
	msgRepo := postgres.NewMessageRepository(db)
	queryLogRepo := postgres.NewQueryLogRepository(db)
	fileStorage := storage.NewSupabaseStorage(cfg.SupabaseURL, cfg.SupabaseServiceKey, cfg.SupabaseStorageBucket)

	authUsecase := auth.NewAuthUsecase(userRepo, cfg.JWTSecret, cfg.JWTExpiration)
	docUsecase := document.NewDocumentUsecase(
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
	)
	chatUsecase := chat.NewChatUsecase(convRepo, msgRepo, queryLogRepo, docUsecase)

	authHandler := handler.NewAuthHandler(authUsecase)
	userHandler := handler.NewUserHandler(authUsecase)
	docHandler := handler.NewDocumentHandler(docUsecase)
	chatHandler := handler.NewChatHandler(chatUsecase, docUsecase)

	app := fiber.New(middleware.FiberConfig(cfg))
	middleware.ApplySecurity(app, cfg)
	app.Use(logger.New())

	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	api := app.Group("/api")
	api.Post("/auth/login", middleware.AuthRateLimit(cfg), authHandler.Login)

	protected := api.Group("", middleware.JWTAuth(cfg.JWTSecret))
	protected.Get("/auth/me", authHandler.Me)

	admin := protected.Group("", middleware.RequireAdmin(), middleware.AdminRateLimit(cfg))
	admin.Post("/users", userHandler.CreateUser)
	admin.Get("/users", userHandler.ListUsers)
	admin.Put("/users/:id", userHandler.UpdateUser)
	admin.Post("/users/import", userHandler.BulkImportUsers)
	admin.Get("/users/import/template", userHandler.DownloadImportTemplate)

	protected.Post("/documents/upload", middleware.UploadRateLimit(cfg), docHandler.Upload)
	protected.Get("/documents", docHandler.List)
	protected.Get("/documents/:id/chunks", docHandler.GetPreview)
	protected.Get("/documents/:id/download", docHandler.Download)
	protected.Get("/documents/:id", docHandler.GetByID)
	protected.Post("/documents/:id/reprocess", docHandler.Reprocess)
	protected.Delete("/documents/:id", docHandler.Delete)
	protected.Post("/documents/query", middleware.QueryRateLimit(cfg), docHandler.Query)

	protected.Post("/chat/conversations", middleware.QueryRateLimit(cfg), chatHandler.CreateConversation)
	protected.Get("/chat/conversations", chatHandler.ListConversations)
	protected.Get("/chat/conversations/:id", chatHandler.GetConversation)
	protected.Post("/chat/conversations/:id/messages", middleware.QueryRateLimit(cfg), chatHandler.SendMessage)
	protected.Post("/chat/stream", middleware.QueryRateLimit(cfg), chatHandler.StreamChat)
	protected.Delete("/chat/conversations/:id", chatHandler.DeleteConversation)

	log.Printf("🚀 Server starting on port %d", cfg.Port)
	log.Printf("🧠 RAG: hybrid=%t reformulation=%t threshold=%.2f topK=%d",
		cfg.UseHybridSearch, cfg.QueryReformulationEnabled, cfg.SimilarityThreshold, cfg.TopKResults)
	log.Printf("🛡️  Anti-abuse: global=%d/%s auth=%d/%s query=%d/%s upload=%d/%s",
		cfg.RateLimitGlobalMax, cfg.RateLimitGlobalWindow,
		cfg.RateLimitAuthMax, cfg.RateLimitAuthWindow,
		cfg.RateLimitQueryMax, cfg.RateLimitQueryWindow,
		cfg.RateLimitUploadMax, cfg.RateLimitUploadWindow,
	)
	log.Printf("📚 Swagger UI: http://localhost:%d/swagger/index.html", cfg.Port)
	log.Printf("🔎 OCR: enabled=%t lang=%s min_text=%d", cfg.OCREnabled, cfg.OCRLang, cfg.OCRMinTextLength)
	log.Printf("📦 Supabase Storage: bucket=%s", cfg.SupabaseStorageBucket)
	if err := app.Listen(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
