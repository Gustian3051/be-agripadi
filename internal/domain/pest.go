package domain

import (
	"time"

	"github.com/google/uuid"
)

type Pest struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	Name        string `gorm:"type:text;not null;uniqueIndex" json:"name"`
	LatinName   string `gorm:"type:text;not null" json:"latin_name"`
	Description string `gorm:"type:text;not null" json:"description"`
	LabelName   string `gorm:"type:text;not null;uniqueIndex" json:"label_name"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type PestSymptom struct {
	PestID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"pest_id"`
	SymptomID uuid.UUID `gorm:"type:uuid;primaryKey" json:"symptom_id"`

	Pest    Pest    `gorm:"foreignKey:PestID;constraint:OnDelete:CASCADE" json:"pest"`
	Symptom Symptom `gorm:"foreignKey:SymptomID;constraint:OnDelete:CASCADE" json:"symptom"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
