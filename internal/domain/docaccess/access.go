package docaccess

import (
	"fmt"
	"strings"

	"rag-api/internal/domain/entity"
)

// Context identifies the viewer for document access checks.
type Context struct {
	UserID string
	Role   entity.UserRole
	Major  string
}

// ResolveUploadVisibility enforces role-based visibility on upload.
// Students are always private; only admin and teacher may set public.
func ResolveUploadVisibility(role entity.UserRole, requested entity.DocumentVisibility) entity.DocumentVisibility {
	if role == entity.RoleStudent {
		return entity.VisibilityPrivate
	}
	if requested == entity.VisibilityPublic && (role == entity.RoleAdmin || role == entity.RoleTeacher) {
		return entity.VisibilityPublic
	}
	return entity.VisibilityPrivate
}

// CanChoosePublic reports whether the role may select public visibility in the UI.
func CanChoosePublic(role entity.UserRole) bool {
	return role == entity.RoleAdmin || role == entity.RoleTeacher
}

// SQLCondition returns a SQL fragment and args for document access filtering.
// docAlias is the documents table alias (e.g. "d"). startArgIndex is the first $N placeholder.
func SQLCondition(docAlias string, access Context, startArgIndex int) (string, []any) {
	if access.Role == entity.RoleAdmin {
		return "TRUE", nil
	}

	userParam := startArgIndex
	majorParam := startArgIndex + 1

	condition := fmt.Sprintf(`(
		%s."userId" = $%d
		OR (
			%s."visibility" = 'PUBLIC'
			AND EXISTS (
				SELECT 1 FROM users u
				WHERE u.id = %s."userId"
				AND (
					u.role = 'ADMIN'
					OR (u.role = 'TEACHER' AND u.major = $%d)
				)
			)
		)
	)`, docAlias, userParam, docAlias, docAlias, majorParam)

	return condition, []any{access.UserID, access.Major}
}

// ParseRole normalizes a role string from JWT into UserRole.
func ParseRole(role string) entity.UserRole {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case string(entity.RoleAdmin):
		return entity.RoleAdmin
	case string(entity.RoleTeacher):
		return entity.RoleTeacher
	default:
		return entity.RoleStudent
	}
}
