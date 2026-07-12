package pesticide

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/domain"
)

func TestResolverSeverityChangesRanking(t *testing.T) {
	pestID := uuid.New()
	repo := &fakePesticideRepo{
		items: []domain.Pesticide{
			{
				ID:            uuid.New(),
				ProductName:   "AREA WG",
				PesticideType: "INSECTICIDE",
				Formulation:   "WG",
				RegisteredAt:  time.Now(),
				Dosages: []domain.PesticideDosage{
					{
						PestID:   pestID,
						DoseRaw:  "200",
						DoseUnit: "g/ha",
					},
				},
			},
			{
				ID:            uuid.New(),
				ProductName:   "SPOT SL",
				PesticideType: "INSECTICIDE",
				Formulation:   "SL",
				RegisteredAt:  time.Now(),
				Dosages: []domain.PesticideDosage{
					{
						PestID:   pestID,
						DoseRaw:  "2",
						DoseUnit: "ml/l",
					},
				},
			},
		},
	}

	resolver := NewResolverService(repo)

	high, err := resolver.Resolve(
		context.Background(),
		ResolveRequest{
			PestID:   pestID,
			Severity: "high",
			TopK:     2,
		},
	)
	if err != nil {
		t.Fatalf("resolve high severity: %v", err)
	}

	low, err := resolver.Resolve(
		context.Background(),
		ResolveRequest{
			PestID:   pestID,
			Severity: "low",
			TopK:     2,
		},
	)
	if err != nil {
		t.Fatalf("resolve low severity: %v", err)
	}

	if high[0].Pesticide.ProductName != "AREA WG" {
		t.Fatalf("expected area product for high severity, got %s", high[0].Pesticide.ProductName)
	}

	if low[0].Pesticide.ProductName != "SPOT SL" {
		t.Fatalf("expected spot product for low severity, got %s", low[0].Pesticide.ProductName)
	}
}

func TestResolverSkipsProductWithoutTargetDosage(t *testing.T) {
	pestID := uuid.New()
	otherPestID := uuid.New()
	repo := &fakePesticideRepo{
		items: []domain.Pesticide{
			{
				ID:           uuid.New(),
				ProductName:  "NO TARGET DOSE",
				Formulation:  "WG",
				RegisteredAt: time.Now(),
				Dosages: []domain.PesticideDosage{
					{
						PestID:   otherPestID,
						DoseRaw:  "200",
						DoseUnit: "g/ha",
					},
				},
			},
			{
				ID:           uuid.New(),
				ProductName:  "VALID TARGET DOSE",
				Formulation:  "WG",
				RegisteredAt: time.Now(),
				Dosages: []domain.PesticideDosage{
					{
						PestID:   pestID,
						DoseRaw:  "200",
						DoseUnit: "g/ha",
					},
				},
			},
		},
	}

	resolver := NewResolverService(repo)
	results, err := resolver.Resolve(
		context.Background(),
		ResolveRequest{
			PestID:   pestID,
			Severity: "high",
			TopK:     5,
		},
	)
	if err != nil {
		t.Fatalf("resolve recommendations: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected one recommendation after hard filter, got %d", len(results))
	}

	if results[0].Pesticide.ProductName != "VALID TARGET DOSE" {
		t.Fatalf("expected target-dose product, got %s", results[0].Pesticide.ProductName)
	}
}

func TestResolverSafetyChangesRanking(t *testing.T) {
	pestID := uuid.New()
	minDose := 1.0
	maxDose := 2.0
	repo := &fakePesticideRepo{
		items: []domain.Pesticide{
			{
				ID:            uuid.New(),
				ProductName:   "HIGH RISK PRODUCT",
				PesticideType: "INSECTICIDE",
				Formulation:   "SC",
				RegisteredAt:  time.Now(),
				Ingredients: []domain.PesticideIngredient{
					{Name: "chlorpyrifos", ConcentrationRaw: "400 g/l"},
				},
				Dosages: []domain.PesticideDosage{
					{
						PestID:   pestID,
						DoseRaw:  "1-2",
						MinDose:  &minDose,
						MaxDose:  &maxDose,
						DoseUnit: "ml/l",
					},
				},
			},
			{
				ID:            uuid.New(),
				ProductName:   "LOWER RISK PRODUCT",
				PesticideType: "INSECTICIDE",
				Formulation:   "SC",
				RegisteredAt:  time.Now(),
				Ingredients: []domain.PesticideIngredient{
					{Name: "abamectin", ConcentrationRaw: "18 g/l"},
				},
				Dosages: []domain.PesticideDosage{
					{
						PestID:   pestID,
						DoseRaw:  "1-2",
						MinDose:  &minDose,
						MaxDose:  &maxDose,
						DoseUnit: "ml/l",
					},
				},
			},
		},
	}

	resolver := NewResolverService(repo)
	results, err := resolver.Resolve(
		context.Background(),
		ResolveRequest{
			PestID:   pestID,
			Severity: "low",
			TopK:     2,
		},
	)
	if err != nil {
		t.Fatalf("resolve recommendations: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected two recommendations, got %d", len(results))
	}

	if results[0].Pesticide.ProductName != "LOWER RISK PRODUCT" {
		t.Fatalf("expected lower risk product first, got %s", results[0].Pesticide.ProductName)
	}

	if results[0].ScoreBreakdown.Safety <= results[1].ScoreBreakdown.Safety {
		t.Fatalf(
			"expected first product to have better safety score, got %.2f <= %.2f",
			results[0].ScoreBreakdown.Safety,
			results[1].ScoreBreakdown.Safety,
		)
	}
}

type fakePesticideRepo struct {
	items []domain.Pesticide
}

func (r *fakePesticideRepo) FindByPestID(context.Context, uuid.UUID) ([]domain.Pesticide, error) {
	return r.items, nil
}

func TestResolverPrioritizesWBCSpecificIngredients(t *testing.T) {
	pestID := uuid.New()
	repo := &fakePesticideRepo{
		items: []domain.Pesticide{
			{
				ID:            uuid.New(),
				ProductName:   "LEPIDOPTERA PRODUCT",
				PesticideType: "INSECTICIDE",
				Formulation:   "SC",
				RegisteredAt:  time.Now(),
				Ingredients: []domain.PesticideIngredient{
					{Name: "Emamectin Benzoate", ConcentrationRaw: "40 g/l"},
					{Name: "Indoxacarb", ConcentrationRaw: "160 g/l"},
				},
				Dosages: []domain.PesticideDosage{
					{
						PestID:   pestID,
						DoseRaw:  "1",
						DoseUnit: "ml/l",
					},
				},
			},
			{
				ID:            uuid.New(),
				ProductName:   "WBC SPECIFIC PRODUCT",
				PesticideType: "INSECTICIDE",
				Formulation:   "SC",
				RegisteredAt:  time.Now(),
				Ingredients: []domain.PesticideIngredient{
					{Name: "Pimetrozin", ConcentrationRaw: "50 g/l"},
				},
				Dosages: []domain.PesticideDosage{
					{
						PestID:   pestID,
						DoseRaw:  "1",
						DoseUnit: "ml/l",
					},
				},
			},
		},
	}

	resolver := NewResolverService(repo)
	results, err := resolver.Resolve(
		context.Background(),
		ResolveRequest{
			PestID:   pestID,
			PestName: "wereng batang cokelat",
			Severity: "medium",
			TopK:     2,
		},
	)
	if err != nil {
		t.Fatalf("resolve recommendations: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected two recommendations, got %d", len(results))
	}

	if results[0].Pesticide.ProductName != "WBC SPECIFIC PRODUCT" {
		t.Fatalf("expected WBC-specific product first, got %s", results[0].Pesticide.ProductName)
	}

	if results[0].ScoreBreakdown.IngredientFit <= results[1].ScoreBreakdown.IngredientFit {
		t.Fatalf(
			"expected first product to have better ingredient fit, got %.2f <= %.2f",
			results[0].ScoreBreakdown.IngredientFit,
			results[1].ScoreBreakdown.IngredientFit,
		)
	}
}
