package handler

import (
	"rag-api/internal/domain/docaccess"
	"rag-api/internal/domain/entity"

	"github.com/gofiber/fiber/v2"
)

func documentAccessFromCtx(c *fiber.Ctx) docaccess.Context {
	userID, _ := c.Locals("userID").(string)
	role, _ := c.Locals("role").(string)
	major, _ := c.Locals("major").(string)

	return docaccess.Context{
		UserID: userID,
		Role:   docaccess.ParseRole(role),
		Major:  major,
	}
}

func parseUploadVisibility(c *fiber.Ctx, role entity.UserRole) entity.DocumentVisibility {
	requested := entity.VisibilityPrivate
	if c.FormValue("visibility") == "PUBLIC" {
		requested = entity.VisibilityPublic
	}
	return parseRequestedVisibility(role, requested)
}

func parseRequestedVisibility(role entity.UserRole, requested entity.DocumentVisibility) entity.DocumentVisibility {
	return docaccess.ResolveUploadVisibility(role, requested)
}
