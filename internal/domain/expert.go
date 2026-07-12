package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ExpertSession struct {
	ID             uuid.UUID    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ConversationID uuid.UUID    `gorm:"type:uuid;not null;index" json:"conversation_id"`
	Conversation   Conversation `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE"`

	State string `gorm:"type:varchar(64);not null" json:"state"`

	DetectedLabel      string  `gorm:"column:detected_label"`
	DetectedConfidence float64 `gorm:"type:float;not null" json:"detected_confidence"`
	DetectedModel      string  `gorm:"column:detected_model"`

	Severity    string         `gorm:"type:varchar(64)" json:"severity"`
	GrowthStage string         `gorm:"type:varchar(64)" json:"growth_stage"`
	Symptoms    datatypes.JSON `gorm:"type:jsonb" json:"symptoms"`
	IsCompleted bool           `gorm:"default:false" json:"is_completed"`

	CreatedAt time.Time `gorm:"type:timestamp;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp;not null" json:"updated_at"`
}
