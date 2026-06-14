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

	// log
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

	// connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("connected to database")

	// initialize openai client
	embeddingClient := openai.NewEmbeddingClient(cfg.OpenAIKey, cfg.OpenAIEmbeddingModel)
	chatClient := openai.NewChatClient(cfg.OpenAIKey, cfg.OpenAIChatModel)

	// initialize repository
	userRepo := postgres.NewUserRepository(db)
	docRepo := postgres.NewDocumentRepository(db)
	chunkRepo := postgres.NewChunkRepository(db)

	// initialize usecase
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
	)

	// initialize handler
	authHandler := handler.NewAuthHandler(authUsecase)
	userHandler := handler.NewUserHandler(authUsecase)
	docHandler := handler.NewDocumentHandler(docUsecase)

	// initialize fiber app
	app := fiber.New()

	// middleware for log request and response in terminal
	app.Use(logger.New())

	// Swagger route
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Public Routes
	api := app.Group("/api")
	api.Post("/auth/login", authHandler.Login)

	// Protected Routes
	protected := api.Group("", middleware.JWTAuth(cfg.JWTSecret))
	protected.Get("/auth/me", authHandler.Me)

	// Admin-only user management
	admin := protected.Group("", middleware.RequireAdmin())
	admin.Post("/users", userHandler.CreateUser)
	admin.Get("/users", userHandler.ListUsers)

	// document routes
	protected.Post("/documents/upload", docHandler.Upload)
	protected.Get("/documents", docHandler.List)
	protected.Get("/documents/:id", docHandler.GetByID)
	protected.Delete("/documents/:id", docHandler.Delete)
	protected.Post("/documents/query", docHandler.Query)

	//
	//
	//
	// Start server
	log.Printf("🚀 Server starting on port %d", cfg.Port)
	log.Printf("📚 Swagger UI: http://localhost:%d/swagger/index.html", cfg.Port)
	if err := app.Listen(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
