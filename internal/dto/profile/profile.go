package profile

import (
	"time"

	"github.com/google/uuid"
)

type Response struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UpdateRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=32"`
	Email       *string `json:"email" binding:"omitempty,email"`
	PhoneNumber *string `json:"phone_number" binding:"omitempty,max=32"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}
