package entity

import "time"

type UserRole string

const (
	RoleStudent UserRole = "STUDENT"
	RoleTeacher UserRole = "TEACHER"
	RoleAdmin   UserRole = "ADMIN"
)

type User struct {
	ID        string    `db:"id" json:"id"`
	Email     string    `db:"email" json:"email"`
	Password  string    `db:"password" json:"-"`
	Name      string    `db:"name" json:"name"`
	Major     string    `db:"major" json:"major"`
	Role      UserRole  `db:"role" json:"role"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}
