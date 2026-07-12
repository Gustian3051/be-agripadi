package symptom

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/domain"
	"github.com/xrash/smetrics"
)

type SymptomRepository interface {
	FindAllSymptoms(ctx context.Context) ([]domain.Symptom, error)
	FindPestByLabelName(ctx context.Context, labelName string) (*domain.Pest, error)
	FindSymptomsByPestID(ctx context.Context, pestID uuid.UUID) ([]domain.Symptom, error)
}

type MatcherService struct {
	normalizer *NormalizerService
	repo       SymptomRepository
}

func NewMatcherService(
	normalizer *NormalizerService,
	repo SymptomRepository,
) *MatcherService {

	return &MatcherService{
		normalizer: normalizer,
		repo:       repo,
	}
}

type MatchResult struct {
	SymptomID          uuid.UUID
	SymptomName        string
	InputText          string
	MatchedText        string
	Confidence         float64
	Similarity         float64
	Source             string
	SymptomType        string
	RuleRole           string
	Severity           string
	GrowthStage        string
	IsCoreSymptom      bool
	RecommendedForRule bool
	DefaultWeight      float64
}

const (
	fuzzyMatchThreshold    = 0.75
	semanticMatchThreshold = 0.65
	contextMatchThreshold  = 0.55
)

func (s *MatcherService) Match(ctx context.Context, input string) (*MatchResult, error) {
	if s == nil || s.normalizer == nil || s.repo == nil {
		return unmatchedResult(input, "matcher_not_configured"), nil
	}

	normalizedInput := s.normalizer.Normalize(input)
	if normalizedInput == "" {
		return unmatchedResult(input, "empty_input"), nil
	}

	symptoms, err := s.repo.FindAllSymptoms(ctx)
	if err != nil {
		return nil, err
	}

	return s.findBestMatch(normalizedInput, input, symptoms, false), nil
}

func (s *MatcherService) MatchForPest(ctx context.Context, input string, pestLabel string) (*MatchResult, error) {
	if s == nil || s.normalizer == nil || s.repo == nil || strings.TrimSpace(pestLabel) == "" {
		return s.Match(ctx, input)
	}

	normalizedInput := s.normalizer.Normalize(input)
	if normalizedInput == "" {
		return unmatchedResult(input, "empty_input"), nil
	}

	pest, err := s.repo.FindPestByLabelName(ctx, pestLabel)
	if err != nil {
		return nil, err
	}

	if pest == nil {
		return s.Match(ctx, input)
	}

	symptoms, err := s.repo.FindSymptomsByPestID(ctx, pest.ID)
	if err != nil {
		return nil, err
	}

	if len(symptoms) == 0 {
		return s.Match(ctx, input)
	}

	return s.findBestMatch(normalizedInput, input, symptoms, true), nil
}

func (s *MatcherService) MatchMany(ctx context.Context, inputs []string) ([]MatchResult, error) {
	results := make([]MatchResult, 0, len(inputs))

	for _, item := range inputs {
		result, err := s.Match(ctx, item)
		if err != nil {
			return nil, err
		}
		if result == nil {
			continue
		}

		results = append(results, *result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	return results, nil
}

func (s *MatcherService) findBestMatch(normalizedInput string, rawInput string, symptoms []domain.Symptom, allowInputCoverage bool) *MatchResult {
	best := MatchResult{
		InputText:  rawInput,
		Confidence: 0,
		Similarity: 0,
		Source:     "unknown",
	}

	for _, symptom := range symptoms {
		result := s.evaluateSymptom(normalizedInput, rawInput, symptom, allowInputCoverage)
		if result == nil {
			continue
		}

		if result.Confidence > best.Confidence {
			best = *result
		}
	}

	return &best
}

func (s *MatcherService) evaluateSymptom(normalizedInput string, rawInput string, symptom domain.Symptom, allowInputCoverage bool) *MatchResult {
	type candidateText struct {
		value  string
		source string
		weight float64
	}

	candidates := []candidateText{
		{value: symptom.Name, source: "name", weight: 1.00},
		{value: symptom.OriginalName, source: "original_name", weight: 0.98},
		{value: symptom.Description, source: "description", weight: 0.75},
	}

	var best *MatchResult

	for _, candidate := range candidates {
		normalizedName := s.normalizer.Normalize(candidate.value)
		if normalizedName == "" {
			continue
		}

		var result *MatchResult

		switch {
		case normalizedInput == normalizedName:
			result = buildMatchResultWithText(symptom, rawInput, candidate.value, 1.0*candidate.weight, candidate.source+"_exact_match")

		case strings.Contains(normalizedInput, normalizedName):
			result = buildMatchResultWithText(symptom, rawInput, candidate.value, 0.92*candidate.weight, candidate.source+"_contains_match")

		case s.hasTokenSubsetMatch(normalizedInput, normalizedName):
			result = buildMatchResultWithText(symptom, rawInput, candidate.value, 0.94*candidate.weight, candidate.source+"_token_subset_match")

		case allowInputCoverage && s.hasInputCoverageMatch(normalizedInput, normalizedName):
			result = buildMatchResultWithText(symptom, rawInput, candidate.value, 0.88*candidate.weight, candidate.source+"_context_input_coverage_match")

		case allowInputCoverage && s.calculateContextTokenSimilarity(normalizedInput, normalizedName) >= contextMatchThreshold:
			score := s.calculateContextTokenSimilarity(normalizedInput, normalizedName) * candidate.weight
			result = buildMatchResultWithText(symptom, rawInput, candidate.value, score, candidate.source+"_context_token_match")

		case s.calculateFuzzySimilarity(normalizedInput, normalizedName) >= fuzzyMatchThreshold:
			score := s.calculateFuzzySimilarity(normalizedInput, normalizedName) * candidate.weight
			result = buildMatchResultWithText(symptom, rawInput, candidate.value, score, candidate.source+"_fuzzy_match")

		case s.calculateTrigramSimilarity(normalizedInput, normalizedName) >= semanticMatchThreshold:
			score := s.calculateTrigramSimilarity(normalizedInput, normalizedName) * candidate.weight
			result = buildMatchResultWithText(symptom, rawInput, candidate.value, score, candidate.source+"_trigram_match")

		case s.calculateSemanticSimilarity(normalizedInput, normalizedName) >= semanticMatchThreshold:
			score := s.calculateSemanticSimilarity(normalizedInput, normalizedName) * candidate.weight
			result = buildMatchResultWithText(symptom, rawInput, candidate.value, score, candidate.source+"_semantic_match")
		}

		if result != nil && (best == nil || result.Confidence > best.Confidence) {
			best = result
		}
	}

	return best
}

func buildMatchResult(symptom domain.Symptom, rawInput string, score float64, source string) *MatchResult {
	return buildMatchResultWithText(symptom, rawInput, symptom.Name, score, source)
}

func buildMatchResultWithText(symptom domain.Symptom, rawInput string, matchedText string, score float64, source string) *MatchResult {
	if score > 1 {
		score = 1
	}

	if score < 0 {
		score = 0
	}

	return &MatchResult{
		SymptomID:          symptom.ID,
		SymptomName:        symptom.Name,
		InputText:          rawInput,
		MatchedText:        matchedText,
		Confidence:         score,
		Similarity:         score,
		Source:             source,
		SymptomType:        symptom.SymptomType,
		RuleRole:           symptom.RuleRole,
		Severity:           symptom.Severity,
		GrowthStage:        symptom.GrowthStage,
		IsCoreSymptom:      symptom.IsCoreSymptom,
		RecommendedForRule: symptom.RecommendedForRule,
		DefaultWeight:      symptom.DefaultWeight,
	}
}

func unmatchedResult(input string, source string) *MatchResult {
	return &MatchResult{
		InputText:  input,
		Confidence: 0,
		Similarity: 0,
		Source:     source,
	}
}

func (s *MatcherService) calculateFuzzySimilarity(a string, b string) float64 {
	distance := smetrics.WagnerFischer(a, b, 1, 1, 2)
	maxLen := math.Max(float64(len(a)), float64(len(b)))

	if maxLen == 0 {
		return 0
	}

	score := 1 - (float64(distance) / maxLen)
	if score < 0 {
		return 0
	}

	return score
}

func (s *MatcherService) hasTokenSubsetMatch(input string, symptomName string) bool {
	inputTokens := tokenSet(input)
	symptomTokens := tokenSet(symptomName)

	if len(inputTokens) == 0 || len(symptomTokens) == 0 {
		return false
	}

	if len(symptomTokens) > len(inputTokens) {
		return false
	}

	for token := range symptomTokens {
		if _, ok := inputTokens[token]; !ok {
			return false
		}
	}

	return true
}

func (s *MatcherService) hasInputCoverageMatch(input string, symptomName string) bool {
	inputTokens := tokenSet(input)
	symptomTokens := tokenSet(symptomName)

	if len(inputTokens) < 2 || len(symptomTokens) == 0 {
		return false
	}

	for token := range inputTokens {
		if _, ok := symptomTokens[token]; ok {
			continue
		}

		matched := false
		for symptomToken := range symptomTokens {
			if s.calculateFuzzySimilarity(token, symptomToken) >= 0.85 {
				matched = true
				break
			}
		}

		if !matched {
			return false
		}
	}

	return true
}

func (s *MatcherService) calculateContextTokenSimilarity(input string, symptomName string) float64 {
	inputTokens := tokenSet(input)
	symptomTokens := tokenSet(symptomName)

	if len(inputTokens) == 0 || len(symptomTokens) == 0 {
		return 0
	}

	matches := 0
	for inputToken := range inputTokens {
		if _, ok := symptomTokens[inputToken]; ok {
			matches++
			continue
		}

		for symptomToken := range symptomTokens {
			if s.calculateFuzzySimilarity(inputToken, symptomToken) >= 0.85 {
				matches++
				break
			}
		}
	}

	if matches == 0 {
		return 0
	}

	inputCoverage := float64(matches) / float64(len(inputTokens))
	symptomCoverage := float64(matches) / float64(len(symptomTokens))
	if matches < 2 && inputCoverage < 1 {
		return 0
	}

	score := (inputCoverage * 0.75) + (symptomCoverage * 0.25)
	return math.Min(0.87, score)
}

func tokenSet(text string) map[string]struct{} {
	result := make(map[string]struct{})

	for _, token := range strings.Fields(text) {
		result[token] = struct{}{}
	}

	return result
}

func (s *MatcherService) calculateTrigramSimilarity(a string, b string) float64 {
	trigramsA := s.buildTrigrams(a)
	trigramsB := s.buildTrigrams(b)

	intersection := 0
	union := make(map[string]struct{})

	for _, item := range trigramsA {
		union[item] = struct{}{}
	}
	for _, item := range trigramsB {
		union[item] = struct{}{}
	}

	setB := make(map[string]struct{})
	for _, item := range trigramsB {
		setB[item] = struct{}{}
	}

	for _, item := range trigramsA {
		if _, ok := setB[item]; ok {
			intersection++
		}
	}

	if len(union) == 0 {
		return 0
	}

	return float64(intersection) / float64(len(union))
}

func (s *MatcherService) buildTrigrams(text string) []string {
	text = "  " + text + "  "
	results := make([]string, 0)

	for i := 0; i < len(text)-2; i++ {
		results = append(results, text[i:i+3])
	}

	return results
}

func (s *MatcherService) calculateSemanticSimilarity(a string, b string) float64 {
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}

	matches := 0
	for _, wa := range wordsA {
		for _, wb := range wordsB {
			if wa == wb {
				matches++
				break
			}

			if s.calculateFuzzySimilarity(wa, wb) >= 0.8 {
				matches++
				break
			}
		}
	}

	maxWords := math.Max(float64(len(wordsA)), float64(len(wordsB)))
	if maxWords == 0 {
		return 0
	}

	return float64(matches) / maxWords
}
