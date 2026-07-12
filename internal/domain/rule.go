package domain

import (
	"time"

	"github.com/google/uuid"
)

type ExpertRule struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PestID        uuid.UUID `gorm:"type:uuid;not null;index" json:"pest_id"`
	SeverityID    uuid.UUID `gorm:"type:uuid;not null;index" json:"severity_id"`
	GrowthStageID uuid.UUID `gorm:"type:uuid;not null;index" json:"growth_stage_id"`

	Code              string      `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Name              string      `gorm:"type:text;not null" json:"name"`
	Description       string      `gorm:"type:text" json:"description"`
	Pest              Pest        `gorm:"foreignKey:PestID" json:"pest"`
	Severity          Severity    `gorm:"foreignKey:SeverityID" json:"severity"`
	GrowthStage       GrowthStage `gorm:"foreignKey:GrowthStageID" json:"growth_stage"`
	MinimumMatchScore float64     `gorm:"type:decimal(5,2);default:0.6" json:"minimum_match_score"`
	ConfidenceScore   float64     `gorm:"type:decimal(5,2);default:0" json:"confidence_score"`
	Priority          int         `gorm:"default:1" json:"priority"`
	IsActive          bool        `gorm:"default:true" json:"is_active"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Symptoms []ExpertRuleSymptom `gorm:"foreignKey:RuleID" json:"symptoms"`
}

type ExpertRuleSymptom struct {
	RuleID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"rule_id"`
	SymptomID uuid.UUID `gorm:"type:uuid;primaryKey" json:"symptom_id"`

	Rule    ExpertRule `gorm:"foreignKey:RuleID;constraint:OnDelete:CASCADE" json:"rule"`
	Symptom Symptom    `gorm:"foreignKey:SymptomID;constraint:OnDelete:CASCADE" json:"symptom"`
	Weight  float64    `gorm:"type:decimal(5,2);default:1" json:"weight"`

	CreatedAt time.Time `json:"created_at"`
}
