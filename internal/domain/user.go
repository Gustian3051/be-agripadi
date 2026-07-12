package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Email       string    `gorm:"type:text;not null;unique" json:"email"`
	Password    string    `gorm:"type:text;not null" json:"-"`
	Name        string    `gorm:"type:varchar(32);not null" json:"name"`
	PhoneNumber string    `gorm:"type:varchar(32)" json:"phone_number"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	Conversations []Conversation `gorm:"foreignKey:UserID"`
}
