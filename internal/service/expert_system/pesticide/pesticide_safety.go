package pesticide

import (
	"strings"

	"github.com/gustian305/backend/internal/domain"
)

type SafetyService struct {
	toxicityDB map[string]ToxicityProfile
}

func NewSafetyService() *SafetyService {

	return &SafetyService{
		toxicityDB: buildToxicityDatabase(),
	}
}

type ToxicityProfile struct {
	ToxicityClass       string
	EnvironmentalImpact string
	PollinatorRisk      string
	WorkerRisk          string
	MaxApplication      string
	Warnings            []string
}

type SafetyResult struct {
	ToxicityClass       string
	EnvironmentalImpact string
	PollinatorRisk      string
	WorkerRisk          string
	Warnings            []string
	Recommendations     []string
	PreHarvestInterval  string
	ReentryInterval     string
	MaxApplication      string
	IsSafe              bool
	IsRestricted        bool

	matchedProfileCount int
}

func (s *SafetyService) Analyze(recommendation Recommendation) SafetyResult {
	return s.AnalyzeProduct(recommendation.Pesticide)
}

func (s *SafetyService) AnalyzeProduct(product domain.Pesticide) SafetyResult {

	result := SafetyResult{
		ToxicityClass:       "moderate",
		EnvironmentalImpact: "medium",
		PollinatorRisk:      "medium",
		WorkerRisk:          "medium",
		Warnings: make(
			[]string,
			0,
		),
		Recommendations: make(
			[]string,
			0,
		),
		PreHarvestInterval: "14 hari",
		ReentryInterval:    "24 jam",
		MaxApplication:     "2 kali aplikasi per musim",
		IsSafe:             true,
		IsRestricted:       false,
	}

	if s == nil {
		return result
	}

	for _, ingredient := range product.Ingredients {

		profile, ok :=
			s.profileForIngredient(
				ingredient.Name,
			)

		if !ok {
			continue
		}

		result =
			s.mergeProfile(
				result,
				profile,
			)
	}

	result.Recommendations =
		append(
			result.Recommendations,
			"Gunakan alat pelindung diri lengkap saat penyemprotan.",
		)

	result.Recommendations =
		append(
			result.Recommendations,
			"Hindari penyemprotan saat angin kencang.",
		)

	if result.PollinatorRisk ==
		"high" {

		result.Warnings =
			append(
				result.Warnings,
				"Hindari aplikasi saat serangga penyerbuk sedang aktif.",
			)

		result.Recommendations =
			append(
				result.Recommendations,
				"Jika perlu aplikasi, lakukan setelah matahari terbenam untuk mengurangi paparan pada serangga penyerbuk.",
			)
	}

	if result.WorkerRisk ==
		"high" {

		result.IsSafe = false

		result.Warnings =
			append(
				result.Warnings,
				"Risiko paparan pada pekerja tergolong tinggi.",
			)
	}

	if result.EnvironmentalImpact ==
		"high" {

		result.Warnings =
			append(
				result.Warnings,
				"Ada potensi pencemaran lingkungan bila penggunaan tidak tepat.",
			)
	}

	return result
}

func (s *SafetyService) ScoreProduct(product domain.Pesticide) float64 {
	result := s.AnalyzeProduct(product)

	score := 0.80

	if result.ToxicityClass == "low" &&
		(result.EnvironmentalImpact == "low" || result.EnvironmentalImpact == "medium") &&
		(result.PollinatorRisk == "low" || result.PollinatorRisk == "medium") &&
		(result.WorkerRisk == "low" || result.WorkerRisk == "medium") {

		score = 1.00
	}

	if result.ToxicityClass == "moderate" &&
		result.EnvironmentalImpact != "high" &&
		result.PollinatorRisk != "high" &&
		result.WorkerRisk != "high" {

		score = 0.80
	}

	if result.EnvironmentalImpact == "high" ||
		result.PollinatorRisk == "high" {

		score = minFloat(score, 0.60)
	}

	if result.WorkerRisk == "high" ||
		result.ToxicityClass == "high" ||
		!result.IsSafe {

		score = minFloat(score, 0.30)
	}

	if result.IsRestricted {
		score = minFloat(score, 0.20)
	}

	return score
}

func (s *SafetyService) profileForIngredient(name string) (ToxicityProfile, bool) {
	if s == nil {
		return ToxicityProfile{}, false
	}

	normalized := normalizeIngredientName(name)
	if normalized == "" {
		return ToxicityProfile{}, false
	}

	if profile, ok := s.toxicityDB[normalized]; ok {
		return profile, true
	}

	for key, profile := range s.toxicityDB {
		if strings.Contains(normalized, key) {
			return profile, true
		}
	}

	return ToxicityProfile{}, false
}

func normalizeIngredientName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.Join(strings.Fields(name), " ")

	return name
}

func (s *SafetyService) mergeProfile(current SafetyResult, profile ToxicityProfile) SafetyResult {

	firstProfile := current.matchedProfileCount == 0

	current.ToxicityClass = higherRiskLabel(
		current.ToxicityClass,
		profile.ToxicityClass,
		firstProfile,
	)

	current.EnvironmentalImpact = higherRiskLabel(
		current.EnvironmentalImpact,
		profile.EnvironmentalImpact,
		firstProfile,
	)

	current.PollinatorRisk = higherRiskLabel(
		current.PollinatorRisk,
		profile.PollinatorRisk,
		firstProfile,
	)

	current.WorkerRisk = higherRiskLabel(
		current.WorkerRisk,
		profile.WorkerRisk,
		firstProfile,
	)

	if current.ToxicityClass == "high" ||
		current.WorkerRisk == "high" {

		current.IsSafe = false
	}

	if profile.MaxApplication != "" {

		current.MaxApplication =
			profile.MaxApplication
	}

	current.Warnings =
		append(
			current.Warnings,
			profile.Warnings...,
		)

	current.matchedProfileCount++

	return current
}

func higherRiskLabel(current string, candidate string, firstCandidate bool) string {
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	current = strings.ToLower(strings.TrimSpace(current))

	if candidate == "" {
		return current
	}

	if firstCandidate {
		return candidate
	}

	if riskRank(candidate) > riskRank(current) {
		return candidate
	}

	return current
}

func riskRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high":
		return 3
	case "moderate", "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func minFloat(a float64, b float64) float64 {
	if a < b {
		return a
	}

	return b
}

func buildToxicityDatabase() map[string]ToxicityProfile {

	return map[string]ToxicityProfile{

		"chlorpyrifos": {
			ToxicityClass:       "high",
			EnvironmentalImpact: "high",
			PollinatorRisk:      "high",
			WorkerRisk:          "high",
			MaxApplication:      "1 kali aplikasi per musim",
			Warnings: []string{
				"Pestisida organofosfat dengan toksisitas tinggi.",
				"Berbahaya bagi organisme air.",
				"Berpotensi menimbulkan efek neurotoksik.",
			},
		},

		// ====================================================
		// FIPRONIL
		// ====================================================

		"fipronil": {
			ToxicityClass:       "moderate",
			EnvironmentalImpact: "high",
			PollinatorRisk:      "high",
			WorkerRisk:          "medium",
			MaxApplication:      "2 kali aplikasi per musim",
			Warnings: []string{
				"Toksik bagi lebah dan serangga penyerbuk.",
				"Dapat mencemari ekosistem air.",
			},
		},

		// ====================================================
		// CYPERMETHRIN
		// ====================================================

		"cypermethrin": {
			ToxicityClass:       "moderate",
			EnvironmentalImpact: "medium",
			PollinatorRisk:      "medium",
			WorkerRisk:          "medium",
			MaxApplication:      "3 kali aplikasi per musim",
			Warnings: []string{
				"Hindari kontak langsung saat penyemprotan.",
			},
		},

		// ====================================================
		// IMIDACLOPRID
		// ====================================================

		"imidacloprid": {
			ToxicityClass:       "moderate",
			EnvironmentalImpact: "medium",
			PollinatorRisk:      "high",
			WorkerRisk:          "medium",
			MaxApplication:      "2 kali aplikasi per musim",
			Warnings: []string{
				"Senyawa neonikotinoid yang dapat memengaruhi serangga penyerbuk.",
			},
		},

		// ====================================================
		// ABAMECTIN
		// ====================================================

		"abamectin": {
			ToxicityClass:       "low",
			EnvironmentalImpact: "medium",
			PollinatorRisk:      "medium",
			WorkerRisk:          "low",
			MaxApplication:      "3 kali aplikasi per musim",
			Warnings: []string{
				"Hindari aplikasi berulang secara berlebihan.",
			},
		},
	}
}
