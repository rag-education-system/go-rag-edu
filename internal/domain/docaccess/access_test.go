package docaccess

import (
	"strings"
	"testing"

	"rag-api/internal/domain/entity"
)

func TestResolveUploadVisibility(t *testing.T) {
	tests := []struct {
		role      entity.UserRole
		requested entity.DocumentVisibility
		want      entity.DocumentVisibility
	}{
		{entity.RoleStudent, entity.VisibilityPublic, entity.VisibilityPrivate},
		{entity.RoleStudent, entity.VisibilityPrivate, entity.VisibilityPrivate},
		{entity.RoleTeacher, entity.VisibilityPublic, entity.VisibilityPublic},
		{entity.RoleTeacher, entity.VisibilityPrivate, entity.VisibilityPrivate},
		{entity.RoleAdmin, entity.VisibilityPublic, entity.VisibilityPublic},
		{entity.RoleAdmin, entity.VisibilityPrivate, entity.VisibilityPrivate},
	}

	for _, tt := range tests {
		got := ResolveUploadVisibility(tt.role, tt.requested)
		if got != tt.want {
			t.Errorf("ResolveUploadVisibility(%s, %s) = %s, want %s", tt.role, tt.requested, got, tt.want)
		}
	}
}

func TestSQLConditionAdmin(t *testing.T) {
	cond, args := SQLCondition("d", Context{Role: entity.RoleAdmin}, 1)
	if cond != "TRUE" || len(args) != 0 {
		t.Fatalf("admin condition = %q args=%v, want TRUE with no args", cond, args)
	}
}

func TestSQLConditionNonAdmin(t *testing.T) {
	cond, args := SQLCondition("d", Context{UserID: "u1", Role: entity.RoleStudent, Major: "PTIK"}, 3)
	if cond == "TRUE" {
		t.Fatal("non-admin should not get TRUE condition")
	}
	if len(args) != 2 || args[0] != "u1" || args[1] != "PTIK" {
		t.Fatalf("unexpected args: %v", args)
	}
	for _, part := range []string{`d."userId"`, `'PUBLIC'`, `u.role = 'ADMIN'`, `u.role = 'TEACHER'`, `u.major = $4`} {
		if !strings.Contains(cond, part) {
			t.Fatalf("SQL missing %q: %s", part, cond)
		}
	}
}
