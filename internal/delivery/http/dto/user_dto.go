package dto

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
	Name     string `json:"name" binding:"required" example:"John Doe"`
	Major    string `json:"major" example:"Computer Science"`
	Role     string `json:"role" example:"STUDENT" enums:"STUDENT,TEACHER"`
}

type ListUsersResponse struct {
	Data []UserInfo `json:"data"`
}

type CreateUserResponse struct {
	Message string   `json:"message" example:"User created successfully"`
	User    UserInfo `json:"user"`
}
