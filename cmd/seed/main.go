package main

import (
	"context"
	"flag"
	"log"
	"os"

	"rag-api/internal/adapter/repository/postgres"
	"rag-api/internal/domain/entity"
	"rag-api/internal/usecase/auth"
	"rag-api/pkg/config"
	"rag-api/pkg/database"
)

func main() {
	email := flag.String("email", os.Getenv("ADMIN_EMAIL"), "admin email")
	password := flag.String("password", os.Getenv("ADMIN_PASSWORD"), "admin password")
	name := flag.String("name", "Administrator", "admin name")
	major := flag.String("major", "admin", "admin major")
	flag.Parse()

	if *email == "" || *password == "" {
		log.Fatal("ADMIN_EMAIL and ADMIN_PASSWORD are required (env or flags)")
	}

	cfg := config.Load()
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	userRepo := postgres.NewUserRepository(db)
	authUsecase := auth.NewAuthUsecase(userRepo, cfg.JWTSecret, cfg.JWTExpiration)

	existing, err := userRepo.FindByEmail(context.Background(), *email)
	if err == nil && existing != nil {
		log.Printf("admin already exists: %s", *email)
		return
	}

	user, err := authUsecase.Register(
		context.Background(),
		*email,
		*password,
		*name,
		*major,
		entity.RoleAdmin,
	)
	if err != nil {
		log.Fatalf("failed to create admin: %v", err)
	}

	log.Printf("admin created: %s (%s)", user.Email, user.ID)
}
