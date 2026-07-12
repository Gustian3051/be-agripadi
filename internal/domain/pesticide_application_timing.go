package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type PesticideApplicationTiming struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	PestID *uuid.UUID `gorm:"type:uuid;index" json:"pest_id"`
	Pest   *Pest      `gorm:"foreignKey:PestID;constraint:OnDelete:SET NULL" json:"pest,omitempty"`

	PestName                string `gorm:"type:varchar(120);not null;index" json:"pest_name"`
	ActiveIngredientPattern string `gorm:"type:text" json:"active_ingredient_pattern"`
	FormulationCategory     string `gorm:"type:varchar(50);not null;index" json:"formulation_category"`
	FormulationCodes        string `gorm:"type:text" json:"formulation_codes"`
	GrowthStage             string `gorm:"type:varchar(80);index" json:"growth_stage"`
	Severity                string `gorm:"type:varchar(80);index" json:"severity"`

	// ApplicationContext membedakan aturan timing yang aman untuk diagnosis lapang,
	// pencegahan/pratanam, evaluasi lapang, atau fallback umum.
	ApplicationContext    string `gorm:"type:varchar(50);not null;default:'diagnosis_lapang';index" json:"application_context"`
	FieldDiagnosisAllowed bool   `gorm:"not null;default:true;index" json:"field_diagnosis_allowed"`

	ApplicationMethod  string `gorm:"type:varchar(120);not null" json:"application_method"`
	ApplicationTarget  string `gorm:"type:text" json:"application_target"`
	ApplicationTrigger string `gorm:"type:text" json:"application_trigger"`
	TimingWindow       string `gorm:"type:text;not null" json:"timing_window"`
	TimingInstruction  string `gorm:"type:text;not null" json:"timing_instruction"`
	WaterManagement    string `gorm:"type:text" json:"water_management"`
	WeatherCondition   string `gorm:"type:text" json:"weather_condition"`

	PreharvestIntervalNote string `gorm:"type:text" json:"preharvest_interval_note"`
	DisplayWarning         string `gorm:"type:text" json:"display_warning"`

	ReferenceTitle       string `gorm:"type:text;not null" json:"reference_title"`
	ReferenceInstitution string `gorm:"type:text" json:"reference_institution"`
	ReferenceYear        string `gorm:"type:varchar(30)" json:"reference_year"`
	ReferenceURL         string `gorm:"type:text" json:"reference_url"`
	ReferenceNote        string `gorm:"type:text" json:"reference_note"`

	Priority int  `gorm:"not null;default:100;index" json:"priority"`
	IsActive bool `gorm:"not null;default:true;index" json:"is_active"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// Matches membantu service rekomendasi memilih timing yang sesuai dengan formulasi,
// fase tanaman, dan tingkat serangan. Nilai kosong pada field rule berarti fallback/umum.
func (t PesticideApplicationTiming) Matches(formulationCode string, growthStage string, severity string) bool {
	return t.IsActive &&
		matchesTimingCode(t.FormulationCodes, formulationCode) &&
		matchesTimingPipeValue(t.GrowthStage, growthStage) &&
		matchesTimingPipeValue(t.Severity, severity)
}

// CanBeShownForFieldDiagnosis bernilai false untuk konteks pratanam seperti FS,
// sehingga tidak muncul sebagai rekomendasi utama pada tanaman yang sudah bergejala.
func (t PesticideApplicationTiming) CanBeShownForFieldDiagnosis() bool {
	if !t.IsActive || !t.FieldDiagnosisAllowed {
		return false
	}
	context := normalizeTimingDomainText(t.ApplicationContext)
	return context != "pencegahan_pratanam" && context != "pratanam"
}

func matchesTimingPipeValue(ruleValue string, expected string) bool {
	ruleValue = normalizeTimingDomainText(ruleValue)
	expected = normalizeTimingDomainText(expected)
	if ruleValue == "" || expected == "" {
		return true
	}
	if ruleValue == "semua" || ruleValue == "all" {
		return true
	}
	for _, item := range strings.Split(ruleValue, "|") {
		item = normalizeTimingDomainText(item)
		if item == expected || item == "semua" || item == "all" {
			return true
		}
	}
	return false
}

func matchesTimingCode(ruleCodes string, expected string) bool {
	ruleCodes = strings.ToUpper(strings.TrimSpace(ruleCodes))
	expected = strings.ToUpper(strings.TrimSpace(expected))
	if ruleCodes == "" || expected == "" {
		return true
	}
	for _, item := range strings.FieldsFunc(ruleCodes, func(r rune) bool {
		return r == ';' || r == ',' || r == '|'
	}) {
		if strings.TrimSpace(item) == expected {
			return true
		}
	}
	return false
}

func normalizeTimingDomainText(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}
