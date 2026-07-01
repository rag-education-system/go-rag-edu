package handler

import (
	"rag-api/internal/delivery/http/dto"
	"rag-api/internal/domain/entity"
	"rag-api/internal/usecase/auth"
	"rag-api/pkg/loginlimit"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authUsecase  *auth.AuthUsecase
	loginLimiter *loginlimit.Limiter
}

func NewAuthHandler(authUsecase *auth.AuthUsecase, loginLimiter *loginlimit.Limiter) *AuthHandler {
	return &AuthHandler{
		authUsecase:  authUsecase,
		loginLimiter: loginLimiter,
	}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account with email, password, name, major, and role
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.RegisterRequest        true  "Register Request"
// @Success      200      {object}  dto.RegisterSuccessResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /api/auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	user, err := h.authUsecase.Register(
		c.Context(),
		req.Email,
		req.Password,
		req.Name,
		req.Major,
		entity.UserRole(req.Role),
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "User registered successfully", "user": dto.UserInfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Major: user.Major,
		Role:  string(user.Role),
	}})
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate a user with email and password, returns a JWT token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.LoginRequest         true  "Login Request"
// @Success      200      {object}  dto.LoginSuccessResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      401      {object}  dto.ErrorResponse
// @Router       /api/auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if h.loginLimiter != nil && !h.loginLimiter.Allow(c.IP(), req.Email) {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Terlalu banyak percobaan login. Coba lagi dalam beberapa menit.",
		})
	}

	token, user, err := h.authUsecase.Login(
		c.Context(),
		req.Email,
		req.Password,
	)

	if err != nil {
		switch err.Error() {
		case "account is disabled":
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "User logged in successfully", "token": token, "user": dto.UserInfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Major: user.Major,
		Role:  string(user.Role),
	}})
}

// Get User
// @Summary      Get user info
// @Description  Get user info
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200      {object}  dto.MeResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /api/auth/me [get]
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user session"})
	}

	user, err := h.authUsecase.GetUserInfo(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"user": dto.UserInfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Major: user.Major,
		Role:  string(user.Role),
	}})
}
