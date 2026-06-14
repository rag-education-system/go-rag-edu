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
}

type AdminUserInfo struct {
	ID              string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email           string `json:"email" example:"user@example.com"`
	Name            string `json:"name" example:"John Doe"`
	Major           string `json:"major" example:"Computer Science"`
	Role            string `json:"role" example:"STUDENT"`
	InitialPassword string `json:"initialPassword,omitempty" example:"password123"`
}

type ListUsersResponse struct {
	Data []AdminUserInfo `json:"data"`
}

type CreateUserResponse struct {
	Message string        `json:"message" example:"User created successfully"`
	User    AdminUserInfo `json:"user"`
}

type UpdateUserResponse struct {
	Message string        `json:"message" example:"User updated successfully"`
	User    AdminUserInfo `json:"user"`
}
