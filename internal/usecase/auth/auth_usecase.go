package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"
	"rag-api/pkg/jwt"
	"rag-api/pkg/password"
)

type AuthUsecase struct {
	userRepo  repository.UserRepository
	jwtSecret string
	jwtExpiry time.Duration
}

func NewAuthUsecase(
	userRepo repository.UserRepository,
	jwtSecret string,
	jwtExpiry time.Duration,
) *AuthUsecase {
	return &AuthUsecase{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		jwtExpiry: jwtExpiry,
	}
}

// register user
func (uc *AuthUsecase) Register(
	ctx context.Context,
	email, pass, name, major string,
	role entity.UserRole,
) (*entity.User, error) {
	// Validate input
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || pass == "" || name == "" || major == "" {
		return nil, errors.New("all fields are required")
	}

	// Check if email already exists
	existing, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if existing != nil && err == nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := password.HashPassword(pass)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &entity.User{
		Email:           email,
		Password:        hashedPassword,
		InitialPassword: pass,
		Name:            name,
		Major:           major,
		Role:            role,
		IsActive:        true,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (uc *AuthUsecase) CreateUserByAdmin(
	ctx context.Context,
	email, pass, name, major string,
	role entity.UserRole,
) (*entity.User, error) {
	role = entity.UserRole(strings.ToUpper(string(role)))
	if role != entity.RoleStudent && role != entity.RoleTeacher {
		return nil, errors.New("admin can only create STUDENT or TEACHER accounts")
	}

	return uc.Register(ctx, email, pass, name, major, role)
}

func (uc *AuthUsecase) ListUsers(ctx context.Context) ([]entity.User, error) {
	return uc.userRepo.ListAll(ctx)
}

func (uc *AuthUsecase) UpdateUserByAdmin(
	ctx context.Context,
	userID, email, pass, name, major string,
	role entity.UserRole,
	isActive bool,
) (*entity.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	major = strings.TrimSpace(major)
	role = entity.UserRole(strings.ToUpper(string(role)))

	if email == "" || name == "" || major == "" {
		return nil, errors.New("email, name, and major are required")
	}
	if role != entity.RoleStudent && role != entity.RoleTeacher {
		return nil, errors.New("admin can only update STUDENT or TEACHER accounts")
	}

	user, err := uc.userRepo.FindById(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	if user.Role == entity.RoleAdmin {
		return nil, errors.New("admin accounts cannot be edited")
	}

	existing, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil && existing.ID != userID {
		return nil, errors.New("email already registered")
	}

	user.Email = email
	user.Name = name
	user.Major = major
	user.Role = role
	user.IsActive = isActive

	if pass != "" {
		if len(pass) < 5 {
			return nil, errors.New("password must be at least 5 characters")
		}
		hashedPassword, err := password.HashPassword(pass)
		if err != nil {
			return nil, err
		}
		user.Password = hashedPassword
		user.InitialPassword = pass
	}

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// login user
func (uc *AuthUsecase) Login(
	ctx context.Context,
	email, pass string,
) (string, *entity.User, error) {
	// Validate input
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || pass == "" {
		return "", nil, errors.New("email and password are required")
	}

	// Find user
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil, errors.New("invalid credentials")
		}
		return "", nil, err
	}
	if user == nil {
		return "", nil, errors.New("invalid credentials")
	}

	if !user.IsActive {
		return "", nil, errors.New("account is disabled")
	}

	// Verify password
	if err := password.ComparePassword(user.Password, pass); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	// Generate JWT token
	token, err := jwt.GenerateToken(
		user.ID,
		user.Email,
		string(user.Role),
		user.Major,
		uc.jwtSecret,
		uc.jwtExpiry,
	)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil

}

// get user
func (uc *AuthUsecase) GetUserInfo(
	ctx context.Context,
	userID string,
) (*entity.User, error) {
	return uc.userRepo.FindById(ctx, userID)
}
