package rule

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/gustian305/backend/internal/domain"
)

type Service struct {
	matcher  *MatcherService
	scorer   *ScorerService
	resolver *ResolverService
}

func NewService(matcher *MatcherService, scorer *ScorerService, resolver *ResolverService) *Service {
	return &Service{
		matcher:  matcher,
		scorer:   scorer,
		resolver: resolver,
	}
}

type ExecuteRequest struct {
	PestID            uuid.UUID
	SeverityID        uuid.UUID
	GrowthStageID     uuid.UUID
	SymptomIDs        []uuid.UUID
	CNNConfidence     float64
	MinimumBayesScore float64
	SeverityLevel     string
}

type ExecuteResponse struct {
	BestMatch        *ResolvedRule
	TopMatches       []ResolvedRule
	IsAmbiguous      bool
	FallbackRequired bool
	AmbiguityScore   float64
	Explainability   ExplainabilityResult
}

func (s *Service) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResponse, error) {

	// ========================================================
	// VALIDATION
	// ========================================================

	if req.PestID == uuid.Nil {

		return nil,
			errors.New(
				"invalid pest id",
			)
	}

	if len(req.SymptomIDs) == 0 {

		return &ExecuteResponse{
			FallbackRequired: true,
			Explainability: ExplainabilityResult{
				Reasoning: []string{
					"Tidak ada symptom yang cocok dengan symptom didalam database",
				},
				ConfidenceSummary: "unknown",
				DecisionSource:    "fallback_rule_engine",
			},
		}, nil
	}

	// ========================================================
	// MATCH
	// ========================================================

	candidates, err :=
		s.findMatchesWithFallback(
			ctx,
			req,
		)

	if err != nil {
		return nil, err
	}

	// ========================================================
	// EMPTY MATCH
	// ========================================================

	if len(candidates) == 0 {

		return &ExecuteResponse{
			FallbackRequired: true,
		}, nil
	}

	// ========================================================
	// RESOLVE
	// ========================================================

	resolved, err :=
		s.resolver.Resolve(
			ResolveRequest{
				Candidates: candidates,

				CNNConfidence: req.CNNConfidence,

				SeverityLevel: req.SeverityLevel,

				TopK: 3,

				EnableFallback: true,

				AmbiguityThreshold: 0.10,
			},
		)

	if err != nil {
		return nil, err
	}

	if resolved == nil ||
		resolved.Primary == nil {

		return &ExecuteResponse{
			FallbackRequired: true,
		}, nil
	}

	return &ExecuteResponse{
		BestMatch: resolved.Primary,

		TopMatches: resolved.TopK,

		IsAmbiguous: resolved.Ambiguous,

		FallbackRequired: resolved.FallbackUsed,

		AmbiguityScore: resolved.AmbiguityScore,

		Explainability: resolved.Explainability,
	}, nil
}

func (s *Service) findMatchesWithFallback(ctx context.Context, req ExecuteRequest) ([]MatchCandidate, error) {

	// Rule engine 10/10: severity tidak boleh dilepas saat fallback.
	// Severity adalah anchor agronomis pada database rule (ringan/sedang/berat).
	// Jika severity dilepas, input rendah dapat salah memilih rule BER.
	strategies := []MatchRequest{
		{
			PestID:            req.PestID,
			SeverityID:        req.SeverityID,
			GrowthStageID:     req.GrowthStageID,
			SymptomIDs:        req.SymptomIDs,
			CNNConfidence:     req.CNNConfidence,
			MinimumBayesScore: req.MinimumBayesScore,
		},
		{
			PestID:            req.PestID,
			SeverityID:        req.SeverityID,
			GrowthStageID:     uuid.Nil,
			SymptomIDs:        req.SymptomIDs,
			CNNConfidence:     req.CNNConfidence,
			MinimumBayesScore: req.MinimumBayesScore,
		},
	}

	for index, strategy := range strategies {
		candidates, err := s.matcher.FindMatches(
			ctx,
			strategy,
		)

		if err != nil {
			return nil, err
		}

		if len(candidates) == 0 {
			continue
		}

		if index > 0 {
			for i := range candidates {
				candidates[i].Reasoning = append(
					candidates[i].Reasoning,
					"matched using relaxed growth-stage fallback; severity remained locked",
				)
			}
		}

		return candidates, nil
	}

	return []MatchCandidate{}, nil
}

func (s *Service) GetBestRule(ctx context.Context, req ExecuteRequest) (*domain.ExpertRule, error) {

	result, err :=
		s.Execute(
			ctx,
			req,
		)

	if err != nil {
		return nil, err
	}

	if result == nil ||
		result.BestMatch == nil {

		return nil, nil
	}

	return &result.BestMatch.Candidate.Rule,
		nil
}
