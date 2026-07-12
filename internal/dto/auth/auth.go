package auth

import (
	"time"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Name            string `json:"name" binding:"required,min=1,max=32" example:"Budi Santoso"`
	Email           string `json:"email" binding:"required,email" example:"[EMAIL_ADDRESS]"`
	PhoneNumber     string `json:"phone_number" binding:"required,max=32" example:"081234567890"`
	Password        string `json:"password" binding:"required,min=8" example:"katasandi123"`
	ConfirmPassword string `json:"confirm_password" binding:"required" example:"katasandi123"`
}

type LoginRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required,max=32" example:"081234567890"`
	Password    string `json:"password" binding:"required" example:"katasandi123"`
}

type UserResponse struct {
	ID          uuid.UUID `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string    `json:"name" example:"Budi Santoso"`
	Email       string    `json:"email" example:"[EMAIL_ADDRESS]"`
	PhoneNumber string    `json:"phone_number" example:"081234567890"`
	CreatedAt   time.Time `json:"created_at" example:"2023-10-27T10:00:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2023-10-27T10:00:00Z"`
}

type AuthResponse struct {
	Token     string       `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"`
	TokenType string       `json:"token_type" example:"Bearer"`
	ExpiresAt time.Time    `json:"expires_at" example:"2023-10-27T10:00:00Z"`
	User      UserResponse `json:"user"`
}

type LogoutResponse struct {
	Message string `json:"message"`
}
