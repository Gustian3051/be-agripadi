package rule

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/gustian305/backend/internal/domain"
)

type RuleRepository interface {
	FindActiveRulesByPestID(ctx context.Context, pestID uuid.UUID) ([]domain.ExpertRule, error)
}

type SemanticMatcher interface {
	CalculateSimilarity(ctx context.Context, inputSymptoms []uuid.UUID, ruleSymptoms []uuid.UUID) (float64, error)
}

type MatcherService struct {
	repo            RuleRepository
	semanticMatcher SemanticMatcher
}

func NewMatcherService(repo RuleRepository, semanticMatcher SemanticMatcher) *MatcherService {

	return &MatcherService{
		repo:            repo,
		semanticMatcher: semanticMatcher,
	}
}

type MatchRequest struct {
	PestID            uuid.UUID
	SeverityID        uuid.UUID
	GrowthStageID     uuid.UUID
	SymptomIDs        []uuid.UUID
	CNNConfidence     float64
	MinimumBayesScore float64
}

type MatchCandidate struct {
	Rule               domain.ExpertRule
	MatchedSymptoms    []uuid.UUID
	UnmatchedSymptoms  []uuid.UUID
	MatchedWeight      float64
	MatchRatio         float64
	InputCoverageRatio float64
	InputSymptomCount  int
	BayesianScore      float64
	SemanticScore      float64
	FinalProbability   float64
	Confidence         float64
	Reasoning          []string
}

func (s *MatcherService) FindMatches(ctx context.Context, req MatchRequest) ([]MatchCandidate, error) {

	rules, err :=
		s.repo.FindActiveRulesByPestID(
			ctx,
			req.PestID,
		)

	if err != nil {
		return nil, err
	}

	results := make(
		[]MatchCandidate,
		0,
	)

	for _, rule := range rules {

		// ====================================================
		// SEVERITY FILTER
		// ====================================================

		if req.SeverityID != uuid.Nil &&
			rule.SeverityID != req.SeverityID {

			continue
		}

		// ====================================================
		// GROWTH STAGE FILTER
		// ====================================================

		if req.GrowthStageID != uuid.Nil &&
			rule.GrowthStageID != req.GrowthStageID {

			continue
		}

		candidate, err :=
			s.buildCandidate(
				ctx,
				rule,
				req,
			)

		if err != nil {
			return nil, err
		}

		if len(candidate.MatchedSymptoms) == 0 {
			continue
		}

		// ====================================================
		// FIELD-EVIDENCE FILTER
		// ====================================================

		if !s.hasSufficientFieldEvidence(rule, candidate) {
			continue
		}

		// ====================================================
		// PROBABILISTIC FILTER
		// ====================================================

		minimumMatchScore :=
			s.effectiveMinimumMatchScore(
				rule.MinimumMatchScore,
				req,
				candidate,
			)

		if candidate.FinalProbability < minimumMatchScore {

			continue
		}

		// ====================================================
		// BAYES FILTER
		// ====================================================

		if candidate.BayesianScore <
			req.MinimumBayesScore {

			continue
		}

		results = append(
			results,
			candidate,
		)
	}

	sort.Slice(
		results,
		func(i, j int) bool {

			return results[i].FinalProbability >
				results[j].FinalProbability
		},
	)

	return results, nil
}

func (s *MatcherService) buildCandidate(ctx context.Context, rule domain.ExpertRule, req MatchRequest) (MatchCandidate, error) {

	inputMap := make(
		map[uuid.UUID]struct{},
	)

	for _, item := range req.SymptomIDs {

		inputMap[item] = struct{}{}
	}

	matched := make(
		[]uuid.UUID,
		0,
	)

	unmatched := make(
		[]uuid.UUID,
		0,
	)

	ruleSymptoms := make(
		[]uuid.UUID,
		0,
	)

	totalWeight := 0.0

	matchedWeight := 0.0

	reasoning := make(
		[]string,
		0,
	)

	for _, symptom := range rule.Symptoms {

		totalWeight += symptom.Weight

		ruleSymptoms = append(ruleSymptoms, symptom.SymptomID)

		if _, ok := inputMap[symptom.SymptomID]; ok {
			matched = append(matched, symptom.SymptomID)

			matchedWeight += symptom.Weight

			reasoning = append(reasoning, "matched symptom")

			continue
		}

		unmatched = append(
			unmatched,
			symptom.SymptomID,
		)
	}

	// ========================================================
	// MATCH RATIO
	// ========================================================

	matchRatio := 0.0

	if totalWeight > 0 {

		matchRatio =
			matchedWeight / totalWeight
	}

	inputCoverageRatio := 0.0

	if len(inputMap) > 0 {
		inputCoverageRatio =
			float64(len(matched)) / float64(len(inputMap))
	}

	// ========================================================
	// BAYESIAN SCORE
	// ========================================================

	bayes :=
		s.calculateBayesianProbability(
			matchRatio,
			req.CNNConfidence,
			rule.ConfidenceScore,
		)

	// ========================================================
	// SEMANTIC SCORE
	// ========================================================

	semanticScore := 0.0

	if s.semanticMatcher != nil {

		score, err :=
			s.semanticMatcher.CalculateSimilarity(
				ctx,
				req.SymptomIDs,
				ruleSymptoms,
			)

		if err == nil {

			semanticScore = score
		}
	}

	// ========================================================
	// HYBRID FINAL PROBABILITY
	// ========================================================

	final :=
		s.calculateHybridProbability(
			matchRatio,
			inputCoverageRatio,
			bayes,
			semanticScore,
			req.CNNConfidence,
		)

	confidence :=
		s.calculateConfidence(
			final,
		)

	return MatchCandidate{
		Rule: rule,

		MatchedSymptoms: matched,

		UnmatchedSymptoms: unmatched,

		MatchedWeight: matchedWeight,

		MatchRatio: matchRatio,

		InputCoverageRatio: inputCoverageRatio,

		InputSymptomCount: len(inputMap),

		BayesianScore: bayes,

		SemanticScore: semanticScore,

		FinalProbability: final,

		Confidence: confidence,

		Reasoning: reasoning,
	}, nil
}

func (s *MatcherService) calculateBayesianProbability(matchRatio float64, cnnConfidence float64, ruleConfidence float64) float64 {

	prior := ruleConfidence

	if prior <= 0 {
		prior = 0.70
	}

	likelihood :=
		matchRatio * cnnConfidence

	evidence :=
		(likelihood * prior) +
			((1 - likelihood) * (1 - prior))

	if evidence == 0 {
		return 0
	}

	posterior :=
		(likelihood * prior) / evidence

	return clampProbability(
		posterior,
	)
}

func (s *MatcherService) calculateHybridProbability(matchRatio float64, inputCoverageRatio float64, bayes float64, semantic float64, cnn float64) float64 {

	// ========================================================
	// WEIGHTED HYBRID
	// ========================================================

	final :=
		(matchRatio * 0.25) +
			(inputCoverageRatio * 0.25) +
			(bayes * 0.25) +
			(semantic * 0.10) +
			(cnn * 0.15)

	return clampProbability(
		final,
	)
}

func (s *MatcherService) effectiveMinimumMatchScore(ruleMinimum float64, req MatchRequest, candidate MatchCandidate) float64 {

	minimum := ruleMinimum

	if minimum <= 0 {
		minimum = 0.60
	}

	// Rule 10/10: jangan turunkan threshold hanya karena input user sedikit.
	// Input coverage 100% dari satu gejala tidak sama dengan rule evidence kuat.
	// Threshold boleh dinaikkan untuk rule sedang/berat agar diagnosis tidak overclaim.
	severity := normalizedRuleSeverity(candidate.Rule)

	if severity == "high" {
		minimum = math.Max(minimum, 0.75)

		// Field-expert exception:
		// Beberapa rule berat generatif tidak selalu memuat gejala identitas hama
		// seperti "wereng di pangkal batang", tetapi memuat anchor berat yang
		// sangat kuat seperti hopperburn. Jika user sudah memilih minimal dua
		// gejala terverifikasi dari pest yang sama dan salah satunya cocok dengan
		// anchor berat rule, maka rule boleh lolos dengan threshold khusus.
		// Ini mencegah false-negative pada kasus lapangan nyata tanpa membuka
		// over-diagnosis dari satu gejala tunggal.
		if len(req.SymptomIDs) >= 2 && req.CNNConfidence >= 0.75 && candidateHasStrongSeverityAnchor(candidate) {
			minimum = math.Min(minimum, 0.45)
		}
	}

	if severity == "medium" {
		minimum = math.Max(minimum, 0.70)

		// Field-expert exception for realistic medium WBC cases.
		// Example: user selects an identity symptom such as embun madu/wereng at
		// the stem base and a medium damage symptom such as slowed tiller growth.
		// Some generated rules may not contain every identity synonym, so only one
		// rule symptom can match. The case should still be accepted when the
		// matched symptom is a non-identity medium damage/severity evidence, the
		// user supplied at least two symptoms, severity remains locked to medium,
		// and CNN evidence is reasonably strong.
		if candidate.InputSymptomCount >= 2 &&
			req.CNNConfidence >= 0.75 &&
			candidate.InputCoverageRatio >= 0.45 &&
			candidateHasMediumOrStrongFieldEvidence(candidate) {
			minimum = math.Min(minimum, 0.40)
		}
	}

	return minimum
}

func (s *MatcherService) hasSufficientFieldEvidence(rule domain.ExpertRule, candidate MatchCandidate) bool {

	matchedCount := len(candidate.MatchedSymptoms)
	severity := normalizedRuleSeverity(rule)

	if matchedCount >= 2 {
		switch severity {
		case "high":
			return candidate.MatchRatio >= 0.20 || candidateHasStrongSeverityAnchor(candidate)
		case "medium":
			return candidate.MatchRatio >= 0.20 || candidateHasMediumOrStrongFieldEvidence(candidate)
		default:
			return true
		}
	}

	// Untuk rule berat, satu gejala yang cocok biasanya belum cukup.
	// Pengecualian hanya diberikan ketika gejala tersebut adalah anchor berat
	// yang sangat khas dan input user berisi minimal dua gejala database.
	// Contoh lapangan: CNN WBC tinggi + "wereng di pangkal batang" +
	// "hopperburn" pada fase generatif. Rule WBC-BER-GEN bisa hanya match
	// hopperburn karena gejala identitas tidak ada di rule generatif, tetapi
	// kombinasi lapangannya tetap kuat.
	if severity == "high" &&
		matchedCount >= 1 &&
		candidate.InputSymptomCount >= 2 &&
		candidate.InputCoverageRatio >= 0.45 &&
		candidateHasStrongSeverityAnchor(candidate) {
		return true
	}

	// Medium field-evidence exception.
	// A medium diagnosis can be agronomically valid with one matched medium
	// damage symptom when another selected symptom is an identity synonym that
	// is not part of the generated rule. Example: "Batang Lengket Oleh Embun
	// Madu" + "Pertumbuhan Anakan Melambat" for WBC sedang vegetatif.
	if severity == "medium" &&
		matchedCount >= 1 &&
		candidate.InputSymptomCount >= 2 &&
		candidate.InputCoverageRatio >= 0.45 &&
		candidateHasMediumOrStrongFieldEvidence(candidate) {
		return true
	}

	return false
}

func candidateHasStrongSeverityAnchor(candidate MatchCandidate) bool {
	if len(candidate.MatchedSymptoms) == 0 {
		return false
	}

	matched := make(map[uuid.UUID]struct{}, len(candidate.MatchedSymptoms))
	for _, id := range candidate.MatchedSymptoms {
		matched[id] = struct{}{}
	}

	for _, item := range candidate.Rule.Symptoms {
		if _, ok := matched[item.SymptomID]; !ok {
			continue
		}

		if isStrongSeverityAnchorSymptom(item.Symptom) {
			return true
		}
	}

	return false
}

func candidateHasMediumOrStrongFieldEvidence(candidate MatchCandidate) bool {
	if candidateHasStrongSeverityAnchor(candidate) {
		return true
	}

	if len(candidate.MatchedSymptoms) == 0 {
		return false
	}

	matched := make(map[uuid.UUID]struct{}, len(candidate.MatchedSymptoms))
	for _, id := range candidate.MatchedSymptoms {
		matched[id] = struct{}{}
	}

	for _, item := range candidate.Rule.Symptoms {
		if _, ok := matched[item.SymptomID]; !ok {
			continue
		}

		if isMediumOrStrongFieldEvidenceSymptom(item.Symptom) {
			return true
		}
	}

	return false
}

func isMediumOrStrongFieldEvidenceSymptom(symptom domain.Symptom) bool {
	if isStrongSeverityAnchorSymptom(symptom) {
		return true
	}

	role := strings.ToLower(strings.TrimSpace(symptom.RuleRole))
	symptomType := strings.ToLower(strings.TrimSpace(symptom.SymptomType))
	severity := strings.ToLower(strings.TrimSpace(symptom.Severity))

	combined := strings.ToLower(strings.TrimSpace(strings.Join([]string{
		symptom.Name,
		symptom.OriginalName,
		symptom.Description,
		symptom.ExpertNote,
		severity,
		symptomType,
		role,
	}, " ")))

	if combined == "" {
		return false
	}

	// Identity evidence proves pest presence, but does not prove field damage.
	if role == "identity" || symptomType == "identity" {
		return false
	}

	if !(role == "severity_anchor" || role == "damage" || role == "supporting" ||
		symptomType == "severity_anchor" || symptomType == "damage" || symptomType == "supporting") {
		return false
	}

	if strings.Contains(severity, "sedang") ||
		strings.Contains(severity, "medium") ||
		strings.Contains(severity, "moderate") ||
		strings.Contains(severity, "berat") ||
		strings.Contains(severity, "tinggi") ||
		strings.Contains(severity, "high") {
		return true
	}

	mediumKeywords := []string{
		"pertumbuhan anakan melambat",
		"anakan melambat",
		"produksi anakan menurun",
		"anakan menurun",
		"menguning",
		"pangkal batang coklat",
		"pangkal batang cokelat",
		"layu meski air cukup",
		"layu",
		"kerusakan mulai nyata",
		"sebagian rumpun",
		"mengering sebagian",
	}

	for _, keyword := range mediumKeywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}

	return false
}

func isStrongSeverityAnchorSymptom(symptom domain.Symptom) bool {
	role := strings.ToLower(strings.TrimSpace(symptom.RuleRole))
	symptomType := strings.ToLower(strings.TrimSpace(symptom.SymptomType))
	severity := strings.ToLower(strings.TrimSpace(symptom.Severity))

	combined := strings.ToLower(strings.TrimSpace(strings.Join([]string{
		symptom.Name,
		symptom.OriginalName,
		symptom.Description,
		symptom.ExpertNote,
		severity,
		symptomType,
		role,
	}, " ")))

	if combined == "" {
		return false
	}

	if role == "identity" || symptomType == "identity" {
		return false
	}

	if !(role == "severity_anchor" || role == "damage" || symptomType == "severity_anchor" || symptomType == "damage") {
		return false
	}

	if strings.Contains(severity, "berat") || strings.Contains(severity, "tinggi") || strings.Contains(severity, "high") {
		return true
	}

	heavyKeywords := []string{
		"hopperburn",
		"tanaman seperti terbakar",
		"seperti terbakar",
		"terbakar",
		"gosong",
		"mati serempak",
		"rumpun mati",
		"banyak rumpun mati",
		"gagal panen",
		"hampa tinggi",
		"kehilangan hasil",
		"banyak malai putih",
		"titik tumbuh mati",
	}

	for _, keyword := range heavyKeywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}

	return false
}

func normalizedRuleSeverity(rule domain.ExpertRule) string {
	name := strings.ToLower(strings.TrimSpace(rule.Severity.Name))
	combined := name

	switch {
	case strings.Contains(combined, "berat") || strings.Contains(combined, "tinggi") || strings.Contains(combined, "high") || strings.Contains(combined, "parah") || strings.Contains(combined, "ber"):
		return "high"
	case strings.Contains(combined, "sedang") || strings.Contains(combined, "medium") || strings.Contains(combined, "moderate") || strings.Contains(combined, "sed"):
		return "medium"
	default:
		return "low"
	}
}

func (s *MatcherService) calculateConfidence(score float64) float64 {

	switch {

	case score >= 0.90:
		return 1.0

	case score >= 0.75:
		return 0.85

	case score >= 0.60:
		return 0.70

	case score >= 0.45:
		return 0.55

	default:
		return 0.30
	}
}

func clampProbability(value float64) float64 {

	return math.Max(
		0,
		math.Min(
			1,
			value,
		),
	)
}
