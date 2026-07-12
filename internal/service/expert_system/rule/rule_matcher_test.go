package rule

import (
	"context"
	"testing"

	"github.com/gustian305/backend/internal/domain"
	"github.com/google/uuid"
)

func TestFindMatchesAcceptsAnchorRuleWhenUserInputsMultipleSymptoms(t *testing.T) {
	pestID := uuid.New()
	severityID := uuid.New()
	growthStageID := uuid.New()
	matchedSymptomID := uuid.New()
	otherSymptomID := uuid.New()

	repo := &fakeRuleRepository{
		rules: []domain.ExpertRule{
			{
				ID:                uuid.New(),
				PestID:            pestID,
				SeverityID:        severityID,
				GrowthStageID:     growthStageID,
				Code:              "RULE-WBC-SED-VEG-01",
				MinimumMatchScore: 0.45,
				ConfidenceScore:   0.85,
				Priority:          2,
				IsActive:          true,
				Symptoms: []domain.ExpertRuleSymptom{
					{
						SymptomID: matchedSymptomID,
						Weight:    0.75,
					},
				},
			},
		},
	}

	matcher := NewMatcherService(repo, nil)

	matches, err := matcher.FindMatches(
		context.Background(),
		MatchRequest{
			PestID:        pestID,
			SeverityID:    severityID,
			GrowthStageID: growthStageID,
			SymptomIDs: []uuid.UUID{
				matchedSymptomID,
				otherSymptomID,
			},
			CNNConfidence: 0.50,
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(matches) != 1 {
		t.Fatalf("expected anchor rule to match, got %d matches", len(matches))
	}
}

type fakeRuleRepository struct {
	rules []domain.ExpertRule
}

func (r *fakeRuleRepository) FindActiveRulesByPestID(
	_ context.Context,
	pestID uuid.UUID,
) ([]domain.ExpertRule, error) {
	result := make([]domain.ExpertRule, 0, len(r.rules))

	for _, rule := range r.rules {
		if rule.PestID == pestID && rule.IsActive {
			result = append(result, rule)
		}
	}

	return result, nil
}
