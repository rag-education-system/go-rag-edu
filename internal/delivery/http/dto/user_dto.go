package dto

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
	Name     string `json:"name" binding:"required" example:"John Doe"`
	Major    string `json:"major" example:"Computer Science"`
	Role     string `json:"role" example:"STUDENT" enums:"STUDENT,TEACHER"`
}

type UpdateUserRequest struct {
	Email    string `json:"email" binding:"required" example:"user@example.com"`
	Password string `json:"password" example:"password123"`
	Name     string `json:"name" binding:"required" example:"John Doe"`
	Major    string `json:"major" example:"Computer Science"`
	Role     string `json:"role" example:"STUDENT" enums:"STUDENT,TEACHER"`
	IsActive *bool  `json:"isActive" example:"true"`
}

type AdminUserInfo struct {
	ID              string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email           string `json:"email" example:"user@example.com"`
	Name            string `json:"name" example:"John Doe"`
	Major           string `json:"major" example:"Computer Science"`
	Role            string `json:"role" example:"STUDENT"`
	IsActive        bool   `json:"isActive" example:"true"`
	InitialPassword string `json:"initialPassword,omitempty" example:"password123"`
}

type ListUsersResponse struct {
	Data []AdminUserInfo `json:"data"`
}

type BulkImportUsersResponse struct {
	Message string              `json:"message"`
	Total   int                 `json:"total"`
	Success int                 `json:"success"`
	Failed  int                 `json:"failed"`
	Results []BulkImportResult  `json:"results"`
}

type BulkImportResult struct {
	Row      int    `json:"row"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
	Password string `json:"password,omitempty"`
}

type CreateUserResponse struct {
	Message string        `json:"message" example:"User created successfully"`
	User    AdminUserInfo `json:"user"`
}

type UpdateUserResponse struct {
	Message string        `json:"message" example:"User updated successfully"`
	User    AdminUserInfo `json:"user"`
}
