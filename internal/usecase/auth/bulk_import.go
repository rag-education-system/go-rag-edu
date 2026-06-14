package auth

import (
	"context"
	"strconv"
	"strings"

	"rag-api/internal/domain/entity"
	"rag-api/pkg/password"
	"rag-api/pkg/userimport"
)

type BulkImportResult struct {
	Row      int    `json:"row"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
	Password string `json:"password,omitempty"`
}

type BulkImportSummary struct {
	Total   int                `json:"total"`
	Success int                `json:"success"`
	Failed  int                `json:"failed"`
	Results []BulkImportResult `json:"results"`
}

func (uc *AuthUsecase) BulkCreateUsersByAdmin(
	ctx context.Context,
	rows []userimport.Row,
) BulkImportSummary {
	summary := BulkImportSummary{
		Total:   len(rows),
		Results: make([]BulkImportResult, 0, len(rows)),
	}

	seenEmails := make(map[string]int)

	for _, row := range rows {
		result := BulkImportResult{
			Row:   row.RowNum,
			Email: row.Email,
			Name:  row.Name,
		}

		pass := strings.TrimSpace(row.Password)
		if pass == "" {
			generated, err := password.GenerateRandom(8)
			if err != nil {
				result.Error = "gagal membuat password otomatis"
				summary.Failed++
				summary.Results = append(summary.Results, result)
				continue
			}
			pass = generated
		}

		email := strings.TrimSpace(strings.ToLower(row.Email))
		if email == "" {
			result.Error = "email wajib diisi"
			summary.Failed++
			summary.Results = append(summary.Results, result)
			continue
		}
		if prevRow, exists := seenEmails[email]; exists {
			result.Error = "email duplikat di file (baris " + strconv.Itoa(prevRow) + ")"
			summary.Failed++
			summary.Results = append(summary.Results, result)
			continue
		}
		seenEmails[email] = row.RowNum

		roleInput := strings.ToUpper(strings.TrimSpace(row.Role))
		var role entity.UserRole
		switch roleInput {
		case "STUDENT", "MAHASISWA":
			role = entity.RoleStudent
		case "TEACHER", "DOSEN":
			role = entity.RoleTeacher
		default:
			role = entity.UserRole(roleInput)
		}

		user, err := uc.CreateUserByAdmin(
			ctx,
			email,
			pass,
			strings.TrimSpace(row.Name),
			strings.TrimSpace(row.Major),
			role,
		)
		if err != nil {
			result.Error = err.Error()
			summary.Failed++
			summary.Results = append(summary.Results, result)
			continue
		}

		result.Success = true
		result.Password = user.InitialPassword
		result.Email = user.Email
		result.Name = user.Name
		summary.Success++
		summary.Results = append(summary.Results, result)
	}

	return summary
}
