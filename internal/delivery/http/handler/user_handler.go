package handler

import (
	"strings"

	"rag-api/internal/delivery/http/dto"
	"rag-api/internal/domain/entity"
	"rag-api/internal/usecase/auth"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	authUsecase *auth.AuthUsecase
}

func NewUserHandler(authUsecase *auth.AuthUsecase) *UserHandler {
	return &UserHandler{authUsecase: authUsecase}
}

// CreateUser godoc
// @Summary      Create a new user (admin only)
// @Description  Admin creates a STUDENT or TEACHER account
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateUserRequest  true  "Create User Request"
// @Success      201      {object}  dto.CreateUserResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      403      {object}  dto.ErrorResponse
// @Failure      409      {object}  dto.ErrorResponse
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /api/users [post]
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	var req dto.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	user, err := h.authUsecase.CreateUserByAdmin(
		c.Context(),
		req.Email,
		req.Password,
		req.Name,
		req.Major,
		entity.UserRole(strings.ToUpper(req.Role)),
	)
	if err != nil {
		switch {
		case err.Error() == "email already registered":
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		case strings.Contains(err.Error(), "required"), strings.Contains(err.Error(), "can only create"):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(dto.CreateUserResponse{
		Message: "User created successfully",
		User: dto.UserInfo{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
			Major: user.Major,
			Role:  string(user.Role),
		},
	})
}

// ListUsers godoc
// @Summary      List all users (admin only)
// @Description  Get all registered users
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.ListUsersResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/users [get]
func (h *UserHandler) ListUsers(c *fiber.Ctx) error {
	users, err := h.authUsecase.ListUsers(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	data := make([]dto.UserInfo, 0, len(users))
	for _, user := range users {
		data = append(data, dto.UserInfo{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
			Major: user.Major,
			Role:  string(user.Role),
		})
	}

	return c.Status(fiber.StatusOK).JSON(dto.ListUsersResponse{Data: data})
}
