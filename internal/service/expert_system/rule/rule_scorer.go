package rule

import "math"

type ScorerService struct {
}

func NewScorerService() *ScorerService {

	return &ScorerService{}
}

type ScoreRequest struct {
	Candidate     MatchCandidate
	CNNConfidence float64
	SeverityLevel string
}

type ScoreResult struct {
	BaseScore          float64
	WeightedScore      float64
	BayesianScore      float64
	SemanticScore      float64
	CNNScore           float64
	SeverityMultiplier float64
	PriorityBoost      float64
	AdaptiveConfidence float64
	FinalScore         float64
	Reasoning          []string
}

func (s *ScorerService) ScoreCandidate(req ScoreRequest) ScoreResult {

	candidate := req.Candidate

	reasoning := make(
		[]string,
		0,
	)

	base :=
		candidate.MatchRatio

	if candidate.InputCoverageRatio > base {
		base =
			(base * 0.60) +
				(candidate.InputCoverageRatio * 0.40)
	}

	reasoning = append(
		reasoning,
		"base symptom and input coverage score calculated",
	)

	// ========================================================
	// DYNAMIC WEIGHTING
	// ========================================================

	dynamicWeight :=
		s.calculateDynamicWeight(
			candidate,
		)

	weighted :=
		base * dynamicWeight

	reasoning = append(
		reasoning,
		"dynamic weighting applied",
	)

	// ========================================================
	// CNN INTEGRATION
	// ========================================================

	cnnContribution :=
		s.calculateCNNContribution(
			req.CNNConfidence,
		)

	reasoning = append(
		reasoning,
		"CNN confidence integrated",
	)

	// ========================================================
	// SEVERITY MULTIPLIER
	// ========================================================

	severityMultiplier :=
		s.calculateSeverityMultiplier(
			req.SeverityLevel,
		)

	reasoning = append(
		reasoning,
		"severity multiplier applied",
	)

	// ========================================================
	// PRIORITY BOOST
	// ========================================================

	priorityBoost :=
		s.calculatePriorityBoost(
			candidate.Rule.Priority,
		)

	reasoning = append(
		reasoning,
		"priority boost applied",
	)

	// ========================================================
	// HYBRID FUSION
	// ========================================================

	fusion :=
		(weighted * 0.30) +
			(candidate.InputCoverageRatio * 0.20) +
			(candidate.BayesianScore * 0.25) +
			(candidate.SemanticScore * 0.10) +
			(cnnContribution * 0.25)

	// ========================================================
	// FINAL SCORE
	// ========================================================

	final :=
		(fusion * severityMultiplier) +
			priorityBoost

	final = clamp(
		final,
	)

	// ========================================================
	// ADAPTIVE CONFIDENCE
	// ========================================================

	adaptiveConfidence :=
		s.calculateAdaptiveConfidence(
			final,
			candidate,
			req.CNNConfidence,
		)

	reasoning = append(
		reasoning,
		"adaptive confidence calculated",
	)

	return ScoreResult{
		BaseScore: base,

		WeightedScore: weighted,

		BayesianScore: candidate.BayesianScore,

		SemanticScore: candidate.SemanticScore,

		CNNScore: cnnContribution,

		SeverityMultiplier: severityMultiplier,

		PriorityBoost: priorityBoost,

		AdaptiveConfidence: adaptiveConfidence,

		FinalScore: final,

		Reasoning: reasoning,
	}
}

func (s *ScorerService) calculateDynamicWeight(candidate MatchCandidate) float64 {

	base :=
		candidate.Rule.ConfidenceScore

	if base <= 0 {
		base = 0.70
	}

	// ========================================================
	// MATCH QUALITY BOOST
	// ========================================================

	if candidate.MatchRatio >= 0.9 {

		base += 0.15
	}

	// ========================================================
	// SEMANTIC BOOST
	// ========================================================

	if candidate.SemanticScore >= 0.8 {

		base += 0.10
	}

	// ========================================================
	// BAYES BOOST
	// ========================================================

	if candidate.BayesianScore >= 0.8 {

		base += 0.10
	}

	return clamp(
		base,
	)
}

func (s *ScorerService) calculateCNNContribution(confidence float64) float64 {

	switch {

	case confidence >= 0.95:
		return confidence * 1.0

	case confidence >= 0.80:
		return confidence * 0.9

	case confidence >= 0.60:
		return confidence * 0.75

	default:
		return confidence * 0.5
	}
}

func (s *ScorerService) calculateSeverityMultiplier(severity string) float64 {

	switch severity {

	case "high", "parah", "berat", "tinggi":
		return 1.20

	case "medium", "sedang":
		return 1.10

	case "low", "ringan":
		return 1.00
	}

	return 1.0
}

func (s *ScorerService) calculatePriorityBoost(priority int) float64 {

	switch priority {

	case 1:
		return 0.15

	case 2:
		return 0.10

	case 3:
		return 0.05
	}

	return 0
}

func (s *ScorerService) calculateAdaptiveConfidence(final float64, candidate MatchCandidate, cnn float64) float64 {

	confidence := final

	// ========================================================
	// STRONG RULE MATCH
	// ========================================================

	if candidate.MatchRatio >= 0.9 {
		confidence += 0.05
	}

	// ========================================================
	// STRONG CNN SIGNAL
	// ========================================================

	if cnn >= 0.9 {

		confidence += 0.05
	}

	// ========================================================
	// STRONG BAYESIAN
	// ========================================================

	if candidate.BayesianScore >= 0.85 {

		confidence += 0.05
	}

	// ========================================================
	// SEMANTIC CONSISTENCY
	// ========================================================

	if candidate.SemanticScore >= 0.8 {
		confidence += 0.05
	}

	return clamp(
		confidence,
	)
}

func clamp(value float64) float64 {

	return math.Max(
		0,
		math.Min(
			1,
			value,
		),
	)
}
