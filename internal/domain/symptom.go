package domain

import (
	"time"

	"github.com/google/uuid"
)

type Symptom struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	Name         string `gorm:"type:text;not null;unique" json:"name"`
	OriginalName string `gorm:"type:text;index" json:"original_name"`
	Description  string `gorm:"type:text" json:"description"`

	GrowthStage           string  `gorm:"type:text;index" json:"growth_stage"`
	Severity              string  `gorm:"type:text;index" json:"severity"`
	SymptomType           string  `gorm:"type:varchar(50);index" json:"symptom_type"`
	RuleRole              string  `gorm:"type:varchar(50);index" json:"rule_role"`
	IsCoreSymptom         bool    `gorm:"default:false;index" json:"is_core_symptom"`
	FieldReliability      string  `gorm:"type:varchar(30)" json:"field_reliability"`
	DiagnosticSpecificity string  `gorm:"type:varchar(30)" json:"diagnostic_specificity"`
	UserObservable        bool    `gorm:"default:true" json:"user_observable"`
	RecommendedForRule    bool    `gorm:"default:false;index" json:"recommended_for_rule"`
	DefaultWeight         float64 `gorm:"type:decimal(5,2);default:0.7" json:"default_weight"`
	ExpertNote            string  `gorm:"type:text" json:"expert_note"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
