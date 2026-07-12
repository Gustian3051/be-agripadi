package domain

import (
	"time"

	"github.com/google/uuid"
)

type GrowthStage struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	Name        string `gorm:"type:text;not null;unique" json:"name"`
	Description string `gorm:"type:text;not null" json:"description"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
