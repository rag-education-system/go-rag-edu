package postgres

import (
	"context"
	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"time"
)

type userRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) repository.UserRepository {
	return &userRepository{db: db}
}

// create user
func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	query := `INSERT INTO users (id, email, password, name, major, role, "createdAt", "updatedAt") 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(ctx, query, user.ID, user.Email, user.Password, user.Name, user.Major, user.Role, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return err
	}
	return nil

}

// find user by email
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	query := `SELECT id, email, password, name, major, role, 
		"createdAt" AS created_at, "updatedAt" AS updated_at 
		FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	return &user, err
}

// find user by id
func (r *userRepository) FindById(ctx context.Context, id string) (*entity.User, error) {
	var user entity.User
	query := `SELECT id, email, password, name, major, role, 
		"createdAt" AS created_at, "updatedAt" AS updated_at 
		FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, id)
	return &user, err
}

func (r *userRepository) ListAll(ctx context.Context) ([]entity.User, error) {
	var users []entity.User
	query := `SELECT id, email, password, name, major, role,
		"createdAt" AS created_at, "updatedAt" AS updated_at
		FROM users ORDER BY "createdAt" DESC`
	err := r.db.SelectContext(ctx, &users, query)
	return users, err
}
