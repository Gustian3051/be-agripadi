package pesticide

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gustian305/backend/internal/domain"
)

type PesticideRepository interface {
	FindByPestID(ctx context.Context, pestID uuid.UUID) ([]domain.Pesticide, error)
}

type ResolverService struct {
	repo          PesticideRepository
	safetyService *SafetyService
}

const (
	targetFitWeight     = 0.25
	ingredientFitWeight = 0.20
	doseQualityWeight   = 0.15
	severityFitWeight   = 0.15
	safetyWeight        = 0.15
	growthStageWeight   = 0.07
	dataQualityWeight   = 0.03
)

func NewResolverService(
	repo PesticideRepository,
	safetyServices ...*SafetyService,
) *ResolverService {

	safetyService := NewSafetyService()

	if len(safetyServices) > 0 &&
		safetyServices[0] != nil {

		safetyService = safetyServices[0]
	}

	return &ResolverService{
		repo:          repo,
		safetyService: safetyService,
	}
}

type ResolveRequest struct {
	PestID      uuid.UUID
	PestName    string
	Severity    string
	GrowthStage string
	TopK        int
}

type ScoreBreakdown struct {
	TargetFit     float64
	IngredientFit float64
	DoseQuality   float64
	SeverityFit   float64
	Safety        float64
	GrowthStage   float64
	DataQuality   float64
	WeightedScore float64
}

type Recommendation struct {
	Pesticide         domain.Pesticide
	MatchScore        float64
	Confidence        float64
	RegulationAllowed bool
	ScoreBreakdown    ScoreBreakdown
	Reasoning         []string
}

func (s *ResolverService) Resolve(ctx context.Context, req ResolveRequest) ([]Recommendation, error) {

	items, err :=
		s.repo.FindByPestID(
			ctx,
			req.PestID,
		)

	if err != nil {
		return nil, err
	}

	results := make(
		[]Recommendation,
		0,
	)

	for _, item := range items {

		regulationAllowed :=
			s.checkRegistrationStatus(
				item,
			)

		if !regulationAllowed {
			continue
		}

		// Hard filter: pestisida hanya layak masuk ranking utama jika memiliki
		// dosis yang spesifik untuk OPT/hama target. Ini menjaga prinsip
		// tepat sasaran dan tepat dosis.
		if !hasTargetDosage(item, req.PestID) {
			continue
		}

		// Diagnosis lapang tidak boleh menampilkan perlakuan benih/pratanam
		// sebagai rekomendasi utama. Produk FS tetap boleh tersimpan di database
		// untuk kebutuhan pencegahan, tetapi bukan untuk tanaman yang sudah
		// bergejala di lapangan.
		if !fieldDiagnosisAllowedForProduct(item, req) {
			continue
		}

		breakdown :=
			s.calculateScoreBreakdown(
				item,
				req,
			)

		score := breakdown.WeightedScore

		confidence :=
			s.calculateConfidence(
				score,
			)

		results = append(
			results,
			Recommendation{
				Pesticide:         item,
				MatchScore:        score,
				Confidence:        confidence,
				RegulationAllowed: regulationAllowed,
				ScoreBreakdown:    breakdown,
				Reasoning:         s.buildReasoning(item, req, breakdown),
			},
		)
	}

	sort.Slice(
		results,
		func(i, j int) bool {
			if results[i].MatchScore != results[j].MatchScore {
				return results[i].MatchScore >
					results[j].MatchScore
			}

			return strings.ToLower(results[i].Pesticide.ProductName) <
				strings.ToLower(results[j].Pesticide.ProductName)
		},
	)

	topK := req.TopK

	if topK <= 0 {
		topK = 5
	}

	if len(results) > topK {

		results = results[:topK]
	}

	return results, nil
}

func fieldDiagnosisAllowedForProduct(item domain.Pesticide, req ResolveRequest) bool {
	formulation := strings.ToUpper(strings.TrimSpace(item.Formulation))
	if formulation == "" {
		return true
	}

	if isPrePlantFormulation(formulation) && !isPrePlantGrowthStage(req.GrowthStage) {
		return false
	}

	for _, dosage := range item.Dosages {
		doseText := strings.ToLower(strings.TrimSpace(dosage.DoseRaw + " " + dosage.DoseUnit))
		if strings.Contains(doseText, "kg benih") || strings.Contains(doseText, "benih") {
			if !isPrePlantGrowthStage(req.GrowthStage) {
				return false
			}
		}
	}

	return true
}

func isPrePlantFormulation(formulation string) bool {
	switch strings.ToUpper(strings.TrimSpace(formulation)) {
	case "FS", "WS", "DS":
		return true
	default:
		return false
	}
}

func isPrePlantGrowthStage(stage string) bool {
	normalized := strings.ToLower(strings.TrimSpace(stage))
	normalized = strings.ReplaceAll(normalized, "_", " ")
	normalized = strings.ReplaceAll(normalized, "-", " ")
	normalized = strings.Join(strings.Fields(normalized), " ")

	switch normalized {
	case "pratanam", "pra tanam", "sebelum tanam", "sebelum semai", "persemaian", "semai", "seedling":
		return true
	default:
		return false
	}
}

func (s *ResolverService) calculateScoreBreakdown(item domain.Pesticide, req ResolveRequest) ScoreBreakdown {

	breakdown := ScoreBreakdown{
		TargetFit:     targetFitScore(item, req.PestID),
		IngredientFit: ingredientFitScore(item, req.PestName),
		DoseQuality:   doseQualityScore(item, req.PestID),
		SeverityFit: severityFitScore(
			item,
			req.PestID,
			req.Severity,
		),
		Safety:      s.safetyFitScore(item),
		GrowthStage: growthStageFitScore(item, req.GrowthStage),
		DataQuality: dataQualityScore(item, req.PestID),
	}

	weightedScore := breakdown.TargetFit*targetFitWeight +
		breakdown.IngredientFit*ingredientFitWeight +
		breakdown.DoseQuality*doseQualityWeight +
		breakdown.SeverityFit*severityFitWeight +
		breakdown.Safety*safetyWeight +
		breakdown.GrowthStage*growthStageWeight +
		breakdown.DataQuality*dataQualityWeight

	// Safety cap: produk dengan risiko tinggi tidak boleh tampak sebagai
	// rekomendasi sangat unggul meskipun cocok secara target/dosis. Ini menjaga
	// prinsip PHT dan mencegah ranking terlalu agresif pada bahan aktif berisiko.
	if breakdown.Safety <= 0.20 {
		weightedScore = math.Min(weightedScore, 0.50)
	} else if breakdown.Safety <= 0.30 {
		weightedScore = math.Min(weightedScore, 0.70)
	} else if breakdown.Safety <= 0.60 {
		weightedScore = math.Min(weightedScore, 0.85)
	}

	breakdown.WeightedScore = roundScore(weightedScore)

	breakdown.TargetFit = roundScore(breakdown.TargetFit)
	breakdown.IngredientFit = roundScore(breakdown.IngredientFit)
	breakdown.DoseQuality = roundScore(breakdown.DoseQuality)
	breakdown.SeverityFit = roundScore(breakdown.SeverityFit)
	breakdown.Safety = roundScore(breakdown.Safety)
	breakdown.GrowthStage = roundScore(breakdown.GrowthStage)
	breakdown.DataQuality = roundScore(breakdown.DataQuality)

	return breakdown
}

func (s *ResolverService) safetyFitScore(item domain.Pesticide) float64 {
	if s == nil || s.safetyService == nil {
		return 0.80
	}

	return s.safetyService.ScoreProduct(item)
}

func (s *ResolverService) checkRegistrationStatus(item domain.Pesticide) bool {

	now := time.Now()

	if item.ExpiredAt != nil &&
		item.ExpiredAt.Before(now) {

		return false
	}

	return true
}

func (s *ResolverService) calculateConfidence(score float64) float64 {

	switch {

	case score >= 0.90:
		return 1.0

	case score >= 0.80:
		return 0.90

	case score >= 0.70:
		return 0.80

	case score >= 0.60:
		return 0.70

	default:
		return 0.50
	}
}

func (s *ResolverService) buildReasoning(item domain.Pesticide, req ResolveRequest, breakdown ScoreBreakdown) []string {

	reasons := make(
		[]string,
		0,
	)

	reasons = append(
		reasons,
		"Sesuai dengan target hama pada database dan memiliki dosis spesifik untuk OPT target.",
	)

	reasons = append(
		reasons,
		fmt.Sprintf(
			"Skor dihitung dengan bobot target OPT %.0f%%, kesesuaian bahan aktif %.0f%%, kualitas dosis %.0f%%, kesesuaian severity %.0f%%, keamanan %.0f%%, fase pertumbuhan %.0f%%, dan kelengkapan data %.0f%%.",
			targetFitWeight*100,
			ingredientFitWeight*100,
			doseQualityWeight*100,
			severityFitWeight*100,
			safetyWeight*100,
			growthStageWeight*100,
			dataQualityWeight*100,
		),
	)

	if strings.TrimSpace(item.Formulation) != "" {
		reasons = append(
			reasons,
			"Formulasi "+item.Formulation+" dipertimbangkan untuk pola aplikasi produk.",
		)
	}

	if strings.TrimSpace(req.Severity) != "" {
		reasons = append(
			reasons,
			"Ranking disesuaikan dengan tingkat serangan "+severityLabel(req.Severity)+".",
		)
	}

	if strings.TrimSpace(req.GrowthStage) != "" {
		reasons = append(
			reasons,
			"Fase "+req.GrowthStage+" digunakan sebagai konteks pemilihan.",
		)
	}

	if breakdown.Safety <= 0.30 {
		reasons = append(
			reasons,
			"Skor akhir dibatasi karena profil keamanan bahan aktif tergolong berisiko tinggi.",
		)
	} else if breakdown.Safety <= 0.60 {
		reasons = append(
			reasons,
			"Skor keamanan menurunkan ranking karena terdapat risiko lingkungan, penyerbuk, atau pekerja yang perlu diperhatikan.",
		)
	}

	reasons = append(
		reasons,
		fmt.Sprintf(
			"Rincian skor: target %.0f%%, bahan aktif %.0f%%, dosis %.0f%%, severity %.0f%%, keamanan %.0f%%, fase %.0f%%, data %.0f%%; skor akhir %.0f%%.",
			breakdown.TargetFit*100,
			breakdown.IngredientFit*100,
			breakdown.DoseQuality*100,
			breakdown.SeverityFit*100,
			breakdown.Safety*100,
			breakdown.GrowthStage*100,
			breakdown.DataQuality*100,
			breakdown.WeightedScore*100,
		),
	)

	return reasons
}

func ingredientFitScore(item domain.Pesticide, pestName string) float64 {
	pestName = strings.ToLower(strings.TrimSpace(pestName))

	if len(item.Ingredients) == 0 {
		return 0.70
	}

	best := 0.55

	for _, ingredient := range item.Ingredients {
		name := normalizeIngredientForRelevance(ingredient.Name)
		if name == "" {
			continue
		}

		score := ingredientScoreForPest(name, pestName)
		if score > best {
			best = score
		}
	}

	return best
}

func ingredientScoreForPest(ingredientName string, pestName string) float64 {
	switch {
	case strings.Contains(pestName, "wereng"):
		return wbcIngredientScore(ingredientName)
	case strings.Contains(pestName, "penggerek"):
		return stemBorerIngredientScore(ingredientName)
	case strings.Contains(pestName, "walang"):
		return riceBugIngredientScore(ingredientName)
	default:
		return 0.70
	}
}

func wbcIngredientScore(name string) float64 {
	switch {
	case containsAnyIngredient(name,
		"buprofezin",
		"pimetrozin", "pymetrozine",
		"triflumezopyrim",
		"dinotefuran",
		"nitenpiram", "nitenpyram",
		"imidakloprid", "imidacloprid",
		"tiametoksam", "thiamethoxam",
		"klotianidin", "clothianidin",
		"fipronil",
		"dimehipo", "dimehypo",
		"bpmc",
		"etofenproks", "etofenprox",
	):
		return 1.00

	case containsAnyIngredient(name,
		"beauveria",
		"metarhizium",
		"karbofuran", "carbofuran",
		"karbosulfan", "carbosulfan",
		"bisultap",
		"monosultap",
		"acetamiprid", "asetamiprid",
	):
		return 0.85

	case containsAnyIngredient(name,
		"abamektin", "abamectin",
	):
		return 0.78

	case containsAnyIngredient(name,
		"emamectin", "emamektin",
		"indoxacarb", "indoksakarb",
		"chlorantraniliprole", "klorantraniliprol",
		"spinetoram",
	):
		return 0.35
	}

	return 0.65
}

func stemBorerIngredientScore(name string) float64 {
	switch {
	case containsAnyIngredient(name,
		"chlorantraniliprole", "klorantraniliprol",
		"emamectin", "emamektin",
		"indoxacarb", "indoksakarb",
		"fipronil",
		"cartap", "kartap",
		"bisultap",
		"monosultap",
		"spinetoram",
	):
		return 1.00
	case containsAnyIngredient(name,
		"abamectin", "abamektin",
		"klorfenapir", "chlorfenapyr",
	):
		return 0.80
	}

	return 0.65
}

func riceBugIngredientScore(name string) float64 {
	switch {
	case containsAnyIngredient(name,
		"fipronil",
		"dimehipo", "dimehypo",
		"bpmc",
		"dinotefuran",
		"nitenpiram", "nitenpyram",
		"imidakloprid", "imidacloprid",
		"tiametoksam", "thiamethoxam",
		"klotianidin", "clothianidin",
	):
		return 1.00
	case containsAnyIngredient(name,
		"beauveria",
		"metarhizium",
		"abamectin", "abamektin",
	):
		return 0.80
	case containsAnyIngredient(name,
		"emamectin", "emamektin",
		"chlorantraniliprole", "klorantraniliprol",
	):
		return 0.45
	}

	return 0.65
}

func normalizeIngredientForRelevance(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		"-", " ",
		"_", " ",
		"/", " ",
		",", " ",
		".", " ",
	)
	value = replacer.Replace(value)

	for strings.Contains(value, "  ") {
		value = strings.ReplaceAll(value, "  ", " ")
	}

	return strings.TrimSpace(value)
}

func containsAnyIngredient(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		candidate = normalizeIngredientForRelevance(candidate)
		if candidate == "" {
			continue
		}

		if strings.Contains(value, candidate) {
			return true
		}
	}

	return false
}

func hasTargetDosage(item domain.Pesticide, pestID uuid.UUID) bool {
	_, ok := targetDosage(item, pestID)
	return ok
}

func targetDosage(item domain.Pesticide, pestID uuid.UUID) (domain.PesticideDosage, bool) {
	for _, dosage := range item.Dosages {
		if dosage.PestID == pestID {
			return dosage, true
		}
	}

	return domain.PesticideDosage{}, false
}

func targetFitScore(item domain.Pesticide, pestID uuid.UUID) float64 {
	if hasTargetDosage(item, pestID) {
		return 1.00
	}

	return 0.00
}

func doseQualityScore(item domain.Pesticide, pestID uuid.UUID) float64 {
	dosage, ok := targetDosage(item, pestID)
	if !ok {
		return 0.00
	}

	hasRaw := strings.TrimSpace(dosage.DoseRaw) != ""
	hasUnit := strings.TrimSpace(dosage.DoseUnit) != ""
	hasRange := dosage.MinDose != nil &&
		dosage.MaxDose != nil &&
		*dosage.MinDose > 0 &&
		*dosage.MaxDose > 0

	if hasRaw && hasUnit && hasRange {
		return 1.00
	}

	if (hasRaw && hasUnit) || hasRange {
		return 0.80
	}

	if hasRaw || hasUnit {
		return 0.50
	}

	return 0.00
}

func severityFitScore(item domain.Pesticide, pestID uuid.UUID, severity string) float64 {
	dosage, _ := targetDosage(item, pestID)
	unit := strings.ToLower(strings.TrimSpace(dosage.DoseUnit))
	formulation := strings.ToUpper(strings.TrimSpace(item.Formulation))

	hasPerHa := strings.Contains(unit, "/ha")
	hasPerLiter := strings.Contains(unit, "/l") || strings.Contains(unit, "/liter") || strings.Contains(unit, "/ltr")
	formulationAvailable := formulation != ""

	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "high":
		formulationFit := inStringSet(formulation, "WG", "WP", "SC", "EC", "SL", "GR")

		switch {
		case hasPerHa && formulationFit:
			return 1.00
		case hasPerHa || formulationFit:
			return 0.80
		default:
			return 0.60
		}

	case "medium":
		if inStringSet(formulation, "SC", "WG", "SL", "SG", "EC") {
			return 1.00
		}

		if formulationAvailable {
			return 0.80
		}

		return 0.60

	case "low":
		if formulation == "GR" {
			return 0.40
		}

		formulationFit := inStringSet(formulation, "SL", "SG", "SC")

		switch {
		case hasPerLiter && formulationFit:
			return 1.00
		case hasPerLiter || formulationFit:
			return 0.70
		default:
			return 0.60
		}
	}

	return 0.70
}

func growthStageFitScore(item domain.Pesticide, growthStage string) float64 {
	stage := strings.ToLower(strings.TrimSpace(growthStage))
	formulation := strings.ToUpper(strings.TrimSpace(item.Formulation))

	if stage == "" {
		return 0.70
	}

	switch {
	case strings.Contains(stage, "vegetatif"):
		if inStringSet(formulation, "GR", "SG", "SL", "SC") {
			return 1.00
		}
		if formulation != "" {
			return 0.70
		}
		return 0.50

	case strings.Contains(stage, "generatif"):
		if inStringSet(formulation, "EC", "WG", "WP", "SC") {
			return 1.00
		}
		if formulation != "" {
			return 0.70
		}
		return 0.50
	}

	return 0.70
}

func dataQualityScore(item domain.Pesticide, pestID uuid.UUID) float64 {
	missing := 0

	if strings.TrimSpace(item.ProductName) == "" {
		missing++
	}

	if strings.TrimSpace(item.Formulation) == "" {
		missing++
	}

	if !hasNamedIngredient(item.Ingredients) {
		missing++
	}

	dosage, ok := targetDosage(item, pestID)
	if !ok || strings.TrimSpace(dosage.DoseRaw) == "" || strings.TrimSpace(dosage.DoseUnit) == "" {
		missing++
	}

	if item.RegisteredAt.IsZero() || item.ExpiredAt == nil {
		missing++
	}

	switch missing {
	case 0:
		return 1.00
	case 1:
		return 0.70
	default:
		return 0.40
	}
}

func hasNamedIngredient(ingredients []domain.PesticideIngredient) bool {
	for _, ingredient := range ingredients {
		if strings.TrimSpace(ingredient.Name) != "" {
			return true
		}
	}

	return false
}

func inStringSet(value string, options ...string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}

	return false
}

func roundScore(value float64) float64 {
	if value > 0 {
		value = math.Round(value*100) / 100
	}

	if value > 1 {
		return 1
	}

	if value < 0 {
		return 0
	}

	return value
}

func severityLabel(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "high", "berat", "tinggi":
		return "berat"
	case "medium", "sedang":
		return "sedang"
	case "low", "ringan", "rendah":
		return "ringan"
	default:
		return severity
	}
}
