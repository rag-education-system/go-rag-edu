package main

import (
	"fmt"
	"log"

	_ "rag-api/docs"
	"rag-api/internal/adapter/openai"
	"rag-api/internal/adapter/repository/postgres"
	"rag-api/internal/delivery/http/handler"
	"rag-api/internal/delivery/http/middleware"
	"rag-api/internal/usecase/auth"
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

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("connected to database")

	embeddingClient := openai.NewEmbeddingClient(cfg.OpenAIKey, cfg.OpenAIEmbeddingModel)
	chatClient := openai.NewChatClient(cfg.OpenAIKey, cfg.OpenAIChatModel)

	userRepo := postgres.NewUserRepository(db)
	docRepo := postgres.NewDocumentRepository(db)
	chunkRepo := postgres.NewChunkRepository(db)

	authUsecase := auth.NewAuthUsecase(userRepo, cfg.JWTSecret, cfg.JWTExpiration)
	docUsecase := document.NewDocumentUsecase(
		docRepo,
		chunkRepo,
		embeddingClient,
		chatClient,
		cfg.ChunkSize,
		cfg.ChunkOverlap,
		cfg.TopKResults,
		cfg.SimilarityThreshold,
		cfg.OCREnabled,
		cfg.OCRLang,
		cfg.OCRMinTextLength,
	)

	authHandler := handler.NewAuthHandler(authUsecase)
	userHandler := handler.NewUserHandler(authUsecase)
	docHandler := handler.NewDocumentHandler(docUsecase)

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

	protected.Post("/documents/upload", middleware.UploadRateLimit(cfg), docHandler.Upload)
	protected.Get("/documents", docHandler.List)
	protected.Get("/documents/:id", docHandler.GetByID)
	protected.Delete("/documents/:id", docHandler.Delete)
	protected.Post("/documents/query", middleware.QueryRateLimit(cfg), docHandler.Query)

	log.Printf("🚀 Server starting on port %d", cfg.Port)
	log.Printf("🛡️  Anti-abuse: global=%d/%s auth=%d/%s query=%d/%s upload=%d/%s",
		cfg.RateLimitGlobalMax, cfg.RateLimitGlobalWindow,
		cfg.RateLimitAuthMax, cfg.RateLimitAuthWindow,
		cfg.RateLimitQueryMax, cfg.RateLimitQueryWindow,
		cfg.RateLimitUploadMax, cfg.RateLimitUploadWindow,
	)
	log.Printf("📚 Swagger UI: http://localhost:%d/swagger/index.html", cfg.Port)
	log.Printf("🔎 OCR: enabled=%t lang=%s min_text=%d", cfg.OCREnabled, cfg.OCRLang, cfg.OCRMinTextLength)
	if err := app.Listen(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
