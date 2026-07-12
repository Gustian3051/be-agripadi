package symptom

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/domain"
)

//
// ============================================================
// REPOSITORY
// ============================================================
//

type PestSymptomRepository interface {
	FindSymptomsByPestID(
		ctx context.Context,
		pestID uuid.UUID,
	) ([]domain.Symptom, error)
}

//
// ============================================================
// LLM CLIENT
// ============================================================
//

type FollowUpQuestionGenerator interface {
	GenerateFollowUpQuestion(
		ctx context.Context,
		prompt string,
	) (string, error)
}

//
// ============================================================
// SELECTOR SERVICE
// ============================================================
//

type SelectorService struct {
	repo PestSymptomRepository
	llm  FollowUpQuestionGenerator
}

func NewSelectorService(
	repo PestSymptomRepository,
	llm FollowUpQuestionGenerator,
) *SelectorService {

	return &SelectorService{
		repo: repo,
		llm:  llm,
	}
}

//
// ============================================================
// REQUEST
// ============================================================
//

type SelectSymptomsRequest struct {
	PestID     uuid.UUID
	Severity   string
	UserInputs []string
	Limit      int
}

//
// ============================================================
// RESPONSE
// ============================================================
//

type SelectSymptomsResponse struct {
	Items            []SuggestedSymptom
	FollowUpQuestion string
}

//
// ============================================================
// SUGGESTED SYMPTOM
// ============================================================
//

type SuggestedSymptom struct {
	ID          uuid.UUID
	Name        string
	Description string
	Score       float64
	Priority    int
	Reason      string
}

//
// ============================================================
// SELECT BY PEST
// ============================================================
//

func (s *SelectorService) SelectByPestID(ctx context.Context, req SelectSymptomsRequest) (*SelectSymptomsResponse, error) {

	symptoms, err :=
		s.repo.FindSymptomsByPestID(
			ctx,
			req.PestID,
		)

	if err != nil {
		return nil, err
	}

	ranked :=
		s.rankSymptoms(
			symptoms,
			req,
		)

	limit := req.Limit

	if limit > 0 && len(ranked) > limit {

		ranked = ranked[:limit]
	}

	followUp :=
		s.generateFollowUpQuestion(
			ctx,
			ranked,
			req,
		)

	return &SelectSymptomsResponse{
		Items: ranked,

		FollowUpQuestion: followUp,
	}, nil
}

//
// ============================================================
// RANK SYMPTOMS
// ============================================================
//

func (s *SelectorService) rankSymptoms(symptoms []domain.Symptom, req SelectSymptomsRequest) []SuggestedSymptom {

	results := make(
		[]SuggestedSymptom,
		0,
		len(symptoms),
	)

	for _, symptom := range symptoms {

		score :=
			s.calculateScore(
				symptom,
				req,
			)

		if !symptom.UserObservable {
			continue
		}

		results = append(
			results,
			SuggestedSymptom{
				ID: symptom.ID,

				Name: symptom.Name,

				Description: symptom.Description,

				Score: score,

				Priority: s.calculatePriority(
					score,
				),

				Reason: s.buildReason(
					score,
					req.Severity,
				),
			},
		)
	}

	sort.Slice(
		results,
		func(i, j int) bool {

			return results[i].Score >
				results[j].Score
		},
	)

	return results
}

//
// ============================================================
// CALCULATE SCORE
// ============================================================
//

func (s *SelectorService) calculateScore(symptom domain.Symptom, req SelectSymptomsRequest) float64 {

	score := 0.35

	normalizedName := strings.ToLower(symptom.Name)
	normalizedOriginalName := strings.ToLower(symptom.OriginalName)
	normalizedDescription := strings.ToLower(symptom.Description)

	// ========================================================
	// METADATA BOOST FROM SYMPTOM 10/10
	// ========================================================

	if symptom.RecommendedForRule {
		score += 0.20
	}

	if symptom.IsCoreSymptom {
		score += 0.10
	}

	switch strings.ToLower(strings.TrimSpace(symptom.RuleRole)) {
	case "identity", "severity_anchor":
		score += 0.12
	case "damage":
		score += 0.08
	case "supporting":
		score += 0.03
	}

	switch strings.ToLower(strings.TrimSpace(symptom.FieldReliability)) {
	case "high":
		score += 0.08
	case "medium":
		score += 0.04
	}

	switch strings.ToLower(strings.TrimSpace(symptom.DiagnosticSpecificity)) {
	case "high":
		score += 0.08
	case "medium":
		score += 0.04
	}

	// ========================================================
	// USER INPUT MATCH
	// ========================================================

	for _, input := range req.UserInputs {
		normalizedInput := strings.ToLower(input)

		if strings.Contains(normalizedInput, normalizedName) ||
			(normalizedOriginalName != "" && strings.Contains(normalizedInput, normalizedOriginalName)) ||
			(normalizedDescription != "" && strings.Contains(normalizedDescription, normalizedInput)) {
			score += 0.25
		}
	}

	// ========================================================
	// SEVERITY METADATA MATCH
	// ========================================================

	severity := normalizeSelectorSeverity(req.Severity)
	if severity != "" && pipeValueContains(symptom.Severity, severity) {
		score += 0.12
	}

	if severity == "berat" && containsCriticalKeyword(normalizedName+" "+normalizedOriginalName+" "+normalizedDescription) {
		score += 0.08
	}

	if score > 1 {
		score = 1
	}

	return score
}

func normalizeSelectorSeverity(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "high", "tinggi", "parah", "berat":
		return "berat"
	case "medium", "moderate", "sedang":
		return "sedang"
	case "low", "rendah", "ringan":
		return "ringan"
	default:
		return strings.ToLower(strings.TrimSpace(severity))
	}
}

func pipeValueContains(value string, expected string) bool {
	expected = strings.ToLower(strings.TrimSpace(expected))
	if expected == "" {
		return false
	}

	for _, part := range strings.FieldsFunc(value, func(r rune) bool {
		return r == '|' || r == ',' || r == ';'
	}) {
		if strings.ToLower(strings.TrimSpace(part)) == expected {
			return true
		}
	}

	return false
}

//
// ============================================================
// CALCULATE PRIORITY
// ============================================================
//

func (s *SelectorService) calculatePriority(score float64) int {

	switch {

	case score >= 0.9:
		return 1

	case score >= 0.75:
		return 2

	case score >= 0.6:
		return 3

	default:
		return 4
	}
}

//
// ============================================================
// BUILD REASON
// ============================================================
//

func (s *SelectorService) buildReason(score float64, severity string) string {

	if score >= 0.9 {

		return "Highly relevant based on detected pest and symptom severity."
	}

	if strings.ToLower(severity) == "high" {

		return "Recommended due to high severity indication."
	}

	return "Relevant symptom for further diagnosis."
}

func (s *SelectorService) generateFollowUpQuestion(ctx context.Context, items []SuggestedSymptom, req SelectSymptomsRequest) string {

	if s.llm == nil {

		return s.defaultFollowUpQuestion(
			items,
		)
	}

	names := make(
		[]string,
		0,
		len(items),
	)

	for _, item := range items {

		names = append(
			names,
			item.Name,
		)
	}

	prompt := `
Anda adalah sistem pakar pertanian padi.

Buat SATU pertanyaan follow-up singkat untuk membantu petani memilih gejala tanaman.

Severity:
` + req.Severity + `

Gejala kandidat:
` + strings.Join(names, ", ")

	response, err :=
		s.llm.GenerateFollowUpQuestion(
			ctx,
			prompt,
		)

	if err != nil {

		return s.defaultFollowUpQuestion(
			items,
		)
	}

	return response
}

func (s *SelectorService) defaultFollowUpQuestion(items []SuggestedSymptom) string {

	if len(items) == 0 {

		return "Gejala apa yang terlihat pada tanaman padi?"
	}

	return "Apakah tanaman mengalami " +
		items[0].Name +
		"?"
}

//
// ============================================================
// CRITICAL KEYWORD
// ============================================================
//

func containsCriticalKeyword(text string) bool {

	keywords := []string{
		"kering",
		"mati",
		"layu",
		"busuk",
		"rusak",
		"menguning",
	}

	for _, keyword := range keywords {

		if strings.Contains(
			text,
			keyword,
		) {

			return true
		}
	}

	return false
}
