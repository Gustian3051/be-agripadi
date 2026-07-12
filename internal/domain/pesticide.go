package domain

import (
	"time"

	"github.com/google/uuid"
)

type Pesticide struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	ProductName   string     `gorm:"type:text;not null;uniqueIndex:idx_pesticide_product_commodity,priority:1" json:"product_name"`
	PesticideType string     `gorm:"type:varchar(50);not null;index" json:"pesticide_type"`
	Formulation   string     `gorm:"type:varchar(20);not null;index" json:"formulation"`
	Commodity     string     `gorm:"type:varchar(100);not null;default:'padi';uniqueIndex:idx_pesticide_product_commodity,priority:2" json:"commodity"`
	RegisteredAt  time.Time  `gorm:"not null;index" json:"registered_at"`
	ExpiredAt     *time.Time `gorm:"index" json:"expired_at"`

	Ingredients []PesticideIngredient `gorm:"foreignKey:PesticideID;constraint:OnDelete:CASCADE" json:"ingredients"`
	Targets     []PesticideTarget     `gorm:"foreignKey:PesticideID;constraint:OnDelete:CASCADE" json:"targets"`
	Dosages     []PesticideDosage     `gorm:"foreignKey:PesticideID;constraint:OnDelete:CASCADE" json:"dosages"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type PesticideIngredient struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PesticideID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_pesticide_ingredient,priority:1" json:"pesticide_id"`

	Name               string   `gorm:"type:text;not null;index;uniqueIndex:idx_pesticide_ingredient,priority:2" json:"name"`
	ConcentrationRaw   string   `gorm:"type:text;not null" json:"concentration_raw"`
	ConcentrationValue *float64 `gorm:"type:decimal(20,4)" json:"concentration_value"`
	ConcentrationType  string   `gorm:"type:varchar(30);not null;index" json:"concentration_type"`
	IngredientUnit     string   `gorm:"type:varchar(30);not null" json:"ingredient_unit"`

	Pesticide Pesticide `gorm:"foreignKey:PesticideID;constraint:OnDelete:CASCADE" json:"pesticide"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type PesticideTarget struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PesticideID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_pesticide_target,priority:1" json:"pesticide_id"`
	PestID      uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_pesticide_target,priority:2" json:"pest_id"`

	Pesticide Pesticide `gorm:"foreignKey:PesticideID;constraint:OnDelete:CASCADE" json:"pesticide"`
	Pest      Pest      `gorm:"foreignKey:PestID;constraint:OnDelete:CASCADE" json:"pest"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type PesticideDosage struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PesticideID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_pesticide_dosage,priority:1" json:"pesticide_id"`
	PestID      uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_pesticide_dosage,priority:2" json:"pest_id"`

	DoseRaw  string   `gorm:"type:text;not null;uniqueIndex:idx_pesticide_dosage,priority:3" json:"dose_raw"`
	MinDose  *float64 `gorm:"type:decimal(10,4)" json:"min_dose"`
	MaxDose  *float64 `gorm:"type:decimal(10,4)" json:"max_dose"`
	DoseUnit string   `gorm:"type:varchar(30);not null;uniqueIndex:idx_pesticide_dosage,priority:4" json:"dose_unit"`

	Pesticide Pesticide `gorm:"foreignKey:PesticideID;constraint:OnDelete:CASCADE" json:"pesticide"`
	Pest      Pest      `gorm:"foreignKey:PestID;constraint:OnDelete:CASCADE" json:"pest"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
