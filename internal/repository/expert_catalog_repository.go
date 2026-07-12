package repository

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gustian305/backend/internal/domain"
)

type ExpertCatalogRepository interface {
	FindPestByLabelName(ctx context.Context, labelName string) (*domain.Pest, error)
	FindByPestID(ctx context.Context, pestID uuid.UUID) ([]domain.Pesticide, error)
	FindAllSymptoms(ctx context.Context) ([]domain.Symptom, error)
	FindSymptomsByPestID(ctx context.Context, pestID uuid.UUID) ([]domain.Symptom, error)
	FindSeverityByName(ctx context.Context, name string) (*domain.Severity, error)
	FindGrowthStageByName(ctx context.Context, name string) (*domain.GrowthStage, error)
	FindActiveRulesByPestID(ctx context.Context, pestID uuid.UUID) ([]domain.ExpertRule, error)
	FindPesticideApplicationTimings(ctx context.Context, pestID uuid.UUID, pestName string, activeIngredients []string, formulation string, growthStage string, severity string) ([]domain.PesticideApplicationTiming, error)
}

type expertCatalogRepository struct {
	db *gorm.DB
}

func NewExpertCatalogRepository(db *gorm.DB) ExpertCatalogRepository {
	return &expertCatalogRepository{
		db: db,
	}
}

func (r *expertCatalogRepository) FindPestByLabelName(ctx context.Context, labelName string) (*domain.Pest, error) {
	labelName = normalizeCatalogLabel(labelName)

	if labelName == "" {
		return nil, nil
	}

	var pest domain.Pest

	err := r.db.WithContext(ctx).
		Where(
			"LOWER(TRIM(label_name)) = ? OR LOWER(REPLACE(TRIM(name), ' ', '_')) = ?",
			labelName,
			labelName,
		).
		First(&pest).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	return &pest, nil
}

func (r *expertCatalogRepository) FindByPestID(ctx context.Context, pestID uuid.UUID) ([]domain.Pesticide, error) {
	if pestID == uuid.Nil {
		return []domain.Pesticide{}, nil
	}

	var pesticides []domain.Pesticide

	err := r.db.WithContext(ctx).
		Distinct("pesticides.*").
		Joins("JOIN pesticide_targets ON pesticide_targets.pesticide_id = pesticides.id").
		Where("pesticide_targets.pest_id = ?", pestID).
		Preload("Ingredients").
		Preload("Dosages", "pest_id = ?", pestID).
		Preload("Targets", "pest_id = ?", pestID).
		Find(&pesticides).
		Error

	if err != nil {
		return nil, err
	}

	return pesticides, nil
}

func (r *expertCatalogRepository) FindAllSymptoms(ctx context.Context) ([]domain.Symptom, error) {
	var symptoms []domain.Symptom

	err := r.db.WithContext(ctx).
		Order("symptoms.name ASC").
		Find(&symptoms).
		Error

	if err != nil {
		return nil, err
	}

	return symptoms, nil
}

func (r *expertCatalogRepository) FindSymptomsByPestID(ctx context.Context, pestID uuid.UUID) ([]domain.Symptom, error) {
	if pestID == uuid.Nil {
		return []domain.Symptom{}, nil
	}

	var symptoms []domain.Symptom

	err := r.db.WithContext(ctx).
		Joins("JOIN pest_symptoms ON pest_symptoms.symptom_id = symptoms.id").
		Where("pest_symptoms.pest_id = ?", pestID).
		Order("symptoms.name ASC").
		Find(&symptoms).
		Error

	if err != nil {
		return nil, err
	}

	return symptoms, nil
}

func (r *expertCatalogRepository) FindSeverityByName(ctx context.Context, name string) (*domain.Severity, error) {
	candidates := severityNameCandidates(name)

	if len(candidates) == 0 {
		return nil, nil
	}

	var severity domain.Severity

	err := r.db.WithContext(ctx).
		Where("LOWER(TRIM(name)) IN ?", candidates).
		First(&severity).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	return &severity, nil
}

func (r *expertCatalogRepository) FindGrowthStageByName(ctx context.Context, name string) (*domain.GrowthStage, error) {
	candidates := growthStageNameCandidates(name)

	if len(candidates) == 0 {
		return nil, nil
	}

	var growthStage domain.GrowthStage

	err := r.db.WithContext(ctx).
		Where("LOWER(TRIM(name)) IN ?", candidates).
		First(&growthStage).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	return &growthStage, nil
}

func (r *expertCatalogRepository) FindActiveRulesByPestID(ctx context.Context, pestID uuid.UUID) ([]domain.ExpertRule, error) {
	if pestID == uuid.Nil {
		return []domain.ExpertRule{}, nil
	}

	var rules []domain.ExpertRule

	err := r.db.WithContext(ctx).
		Where("pest_id = ? AND is_active = ?", pestID, true).
		Preload("Pest").
		Preload("Severity").
		Preload("GrowthStage").
		Preload("Symptoms").
		Preload("Symptoms.Symptom").
		Find(&rules).
		Error

	if err != nil {
		return nil, err
	}

	return rules, nil
}

func (r *expertCatalogRepository) FindPesticideApplicationTimings(ctx context.Context, pestID uuid.UUID, pestName string, activeIngredients []string, formulation string, growthStage string, severity string) ([]domain.PesticideApplicationTiming, error) {
	var timings []domain.PesticideApplicationTiming

	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("priority ASC").
		Order("created_at ASC").
		Find(&timings).
		Error

	if err != nil {
		return nil, err
	}

	matched := make([]domain.PesticideApplicationTiming, 0, len(timings))
	scores := make(map[uuid.UUID]int)

	for _, timing := range timings {
		// Data timing bisa berisi konteks pratanam/pencegahan, misalnya FS.
		// Untuk diagnosis lapang, konteks tersebut tidak boleh muncul sebagai
		// rekomendasi utama agar sistem tidak menyarankan perlakuan benih
		// pada tanaman yang sudah bergejala.
		if !timing.CanBeShownForFieldDiagnosis() {
			continue
		}

		if !timingMatchesPest(timing, pestID, pestName) {
			continue
		}

		if !timingMatchesFormulation(timing, formulation) {
			continue
		}

		if !timingMatchesGrowthStage(timing, growthStage) {
			continue
		}

		if !timingMatchesSeverity(timing, severity) {
			continue
		}

		if !timingMatchesIngredient(timing, activeIngredients) {
			continue
		}

		scores[timing.ID] = timingSpecificityScore(timing, pestID, pestName, activeIngredients, formulation, growthStage, severity)
		matched = append(matched, timing)
	}

	sort.SliceStable(matched, func(i, j int) bool {
		left := scores[matched[i].ID]
		right := scores[matched[j].ID]
		if left == right {
			return matched[i].Priority < matched[j].Priority
		}
		return left > right
	})

	return matched, nil
}

func timingMatchesPest(timing domain.PesticideApplicationTiming, pestID uuid.UUID, pestName string) bool {
	rowPest := normalizeCatalogPhrase(timing.PestName)
	queryPest := normalizeCatalogPhrase(pestName)

	if timing.PestID != nil && pestID != uuid.Nil && *timing.PestID == pestID {
		return true
	}

	if rowPest == "" || isGeneralTimingValue(rowPest) {
		return true
	}

	if queryPest == "" {
		return false
	}

	return rowPest == queryPest || strings.Contains(rowPest, queryPest) || strings.Contains(queryPest, rowPest)
}

func timingMatchesFormulation(timing domain.PesticideApplicationTiming, formulation string) bool {
	formulation = strings.ToUpper(strings.TrimSpace(formulation))
	codes := timingFormulationCodes(timing.FormulationCodes)

	if len(codes) > 0 {
		for _, code := range codes {
			if formulation == strings.ToUpper(strings.TrimSpace(code)) {
				return true
			}
		}
		return false
	}

	category := normalizeCatalogPhrase(timing.FormulationCategory)
	if category == "" || isGeneralTimingValue(category) {
		return true
	}

	if formulation == "" {
		return false
	}

	if category == "granul" || category == "granule" || category == "butiran" {
		return formulation == "GR" || strings.Contains(formulation, "GRANUL")
	}

	if category == "semprot" || category == "cair" || category == "larutan" || category == "spray" {
		return formulation != "GR"
	}

	return strings.Contains(category, strings.ToLower(formulation))
}

func timingMatchesGrowthStage(timing domain.PesticideApplicationTiming, growthStage string) bool {
	return timingMatchesPipeValue(timing.GrowthStage, growthStage, normalizeStageTimingValue)
}

func timingMatchesSeverity(timing domain.PesticideApplicationTiming, severity string) bool {
	return timingMatchesPipeValue(timing.Severity, severity, normalizeSeverityTimingValue)
}

func timingMatchesPipeValue(raw string, expected string, normalizer func(string) string) bool {
	query := normalizer(expected)
	if query == "" {
		return true
	}

	values := timingPipeValues(raw, normalizer)
	if len(values) == 0 {
		return true
	}

	for _, value := range values {
		if value == query || isGeneralTimingValue(value) {
			return true
		}
	}

	return false
}

func timingMatchesIngredient(timing domain.PesticideApplicationTiming, activeIngredients []string) bool {
	pattern := normalizeCatalogPhrase(timing.ActiveIngredientPattern)
	if pattern == "" || isGeneralTimingValue(pattern) {
		return true
	}

	for _, ingredient := range activeIngredients {
		candidate := normalizeCatalogPhrase(ingredient)
		if candidate == "" {
			continue
		}

		if strings.Contains(candidate, pattern) || strings.Contains(pattern, candidate) {
			return true
		}
	}

	return false
}

func timingSpecificityScore(timing domain.PesticideApplicationTiming, pestID uuid.UUID, pestName string, activeIngredients []string, formulation string, growthStage string, severity string) int {
	score := 0

	rowPest := normalizeCatalogPhrase(timing.PestName)
	queryPest := normalizeCatalogPhrase(pestName)
	if timing.PestID != nil && pestID != uuid.Nil && *timing.PestID == pestID {
		score += 120
	} else if rowPest != "" && rowPest == queryPest {
		score += 100
	} else if rowPest != "" && !isGeneralTimingValue(rowPest) {
		score += 70
	} else {
		score += 10
	}

	formulation = strings.ToUpper(strings.TrimSpace(formulation))
	for _, code := range timingFormulationCodes(timing.FormulationCodes) {
		if formulation != "" && strings.EqualFold(code, formulation) {
			score += 50
			break
		}
	}

	category := normalizeCatalogPhrase(timing.FormulationCategory)
	if category != "" && !isGeneralTimingValue(category) {
		score += 20
	}

	if timingPipeContains(timing.GrowthStage, growthStage, normalizeStageTimingValue) {
		score += 20
	}

	if timingPipeContains(timing.Severity, severity, normalizeSeverityTimingValue) {
		score += 10
	}

	context := normalizeCatalogPhrase(timing.ApplicationContext)
	switch context {
	case "diagnosis lapang":
		score += 30
	case "evaluasi lapang":
		score += 20
	case "fallback umum":
		score += 0
	default:
		if context != "" && !isGeneralTimingValue(context) {
			score += 5
		}
	}

	if strings.TrimSpace(timing.ActiveIngredientPattern) != "" && timingMatchesIngredient(timing, activeIngredients) {
		score += 15
	}

	return score
}

func timingPipeContains(raw string, expected string, normalizer func(string) string) bool {
	query := normalizer(expected)
	if query == "" {
		return false
	}

	for _, value := range timingPipeValues(raw, normalizer) {
		if value == query || isGeneralTimingValue(value) {
			return true
		}
	}

	return false
}

func timingPipeValues(raw string, normalizer func(string) string) []string {
	replacer := strings.NewReplacer("|", ",", ";", ",", "/", ",")
	raw = replacer.Replace(raw)
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	seen := make(map[string]struct{})

	for _, part := range parts {
		value := normalizer(part)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}

	return values
}

func timingFormulationCodes(raw string) []string {
	replacer := strings.NewReplacer("|", ",", ";", ",", "/", ",")
	raw = replacer.Replace(raw)
	parts := strings.Split(raw, ",")
	results := make([]string, 0, len(parts))
	seen := make(map[string]struct{})

	for _, part := range parts {
		part = strings.ToUpper(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		if _, exists := seen[part]; exists {
			continue
		}
		seen[part] = struct{}{}
		results = append(results, part)
	}

	return results
}

func normalizeCatalogPhrase(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", " ", "-", " ", "/", " ", ",", " ", ".", " ")
	value = replacer.Replace(value)
	for strings.Contains(value, "  ") {
		value = strings.ReplaceAll(value, "  ", " ")
	}
	return strings.TrimSpace(value)
}

func normalizeStageTimingValue(value string) string {
	value = normalizeCatalogPhrase(value)
	switch value {
	case "vegetatif", "vegetative", "anakan":
		return "vegetatif"
	case "generatif", "reproduktif", "reproductive", "berbunga", "bunting", "pengisian bulir", "masak susu":
		return "generatif"
	}
	return value
}

func normalizeSeverityTimingValue(value string) string {
	value = normalizeCatalogPhrase(value)
	switch value {
	case "ringan", "rendah", "low":
		return "ringan"
	case "sedang", "medium", "moderate":
		return "sedang"
	case "berat", "tinggi", "parah", "high":
		return "berat"
	}
	return value
}

func isGeneralTimingValue(value string) bool {
	value = normalizeCatalogPhrase(value)
	switch value {
	case "", "umum", "general", "semua", "all", "padi", "tanaman padi", "semua hama":
		return true
	default:
		return false
	}
}

func normalizeCatalogLabel(label string) string {
	label = strings.TrimSpace(label)
	label = strings.ToLower(label)
	label = strings.ReplaceAll(label, " ", "_")
	label = strings.ReplaceAll(label, "-", "_")

	for strings.Contains(label, "__") {
		label = strings.ReplaceAll(label, "__", "_")
	}

	return strings.Trim(label, "_")
}

func severityNameCandidates(name string) []string {
	normalized := strings.ToLower(strings.TrimSpace(name))

	switch normalized {
	case "rendah", "ringan", "low":
		return []string{"rendah", "ringan", "low"}

	case "sedang", "medium", "moderate":
		return []string{"sedang", "medium", "moderate"}

	case "tinggi", "berat", "parah", "high":
		return []string{"tinggi", "berat", "parah", "high"}
	}

	if normalized == "" {
		return []string{}
	}

	return []string{normalized}
}

func growthStageNameCandidates(name string) []string {
	normalized := strings.ToLower(strings.TrimSpace(name))

	switch normalized {
	case "persemaian", "semai", "seedling":
		return []string{"persemaian", "semai", "seedling"}

	case "vegetatif", "vegetative", "anakan":
		return []string{"vegetatif", "vegetative", "anakan"}

	case "generatif", "reproduktif", "reproductive", "berbunga", "bunting":
		return []string{"generatif", "reproduktif", "reproductive", "berbunga", "bunting"}

	case "pemasakan", "masak", "ripening", "panen", "pengisian bulir":
		return []string{"pemasakan", "masak", "ripening", "panen", "pengisian bulir"}
	}

	if normalized == "" {
		return []string{}
	}

	return []string{normalized}
}
