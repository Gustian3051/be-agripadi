package expertSystem

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/domain"
)

func TestAdjustDosageBySeverityUsesOfficialRange(t *testing.T) {
	minDose := 0.75
	maxDose := 1.5
	dosage := domain.PesticideDosage{
		PestID:   uuid.New(),
		DoseRaw:  "0.75-1.5",
		MinDose:  &minDose,
		MaxDose:  &maxDose,
		DoseUnit: "ml/l",
	}

	low := adjustDosageBySeverity(dosage, "low")
	medium := adjustDosageBySeverity(dosage, "medium")
	high := adjustDosageBySeverity(dosage, "high")

	if !strings.HasPrefix(low.DoseRaw, "0.75 ") {
		t.Fatalf("expected low severity to use minimum dose, got %q", low.DoseRaw)
	}

	if !strings.HasPrefix(medium.DoseRaw, "1.125 ") {
		t.Fatalf("expected medium severity to use midpoint dose, got %q", medium.DoseRaw)
	}

	if !strings.HasPrefix(high.DoseRaw, "1.5 ") {
		t.Fatalf("expected high severity to use maximum dose, got %q", high.DoseRaw)
	}

	for _, value := range []string{low.DoseRaw, medium.DoseRaw, high.DoseRaw} {
		if !strings.Contains(value, "rentang label 0.75-1.5") {
			t.Fatalf("expected dose to preserve official range, got %q", value)
		}
	}
}

func TestFormatRecommendationMessageUsesRequestedMobileSections(t *testing.T) {
	message := FormatRecommendationMessage(
		&DiagnosisRecommendationResult{
			Pest: &domain.Pest{
				Name: "wereng batang cokelat",
			},
			DetectionConfidence: 0.95,
			RuleConfidence:      0.86,
			Severity:            "berat",
			GrowthStage:         "vegetatif",
			LLMSummary:          "Tanaman kemungkinan kuat terserang wereng batang cokelat.",
		},
	)

	for _, expected := range []string{
		"Diagnosis:",
		"Keyakinan:",
		"Keparahan:",
		"Fase tanaman:",
		"Kesimpulan:",
		"Rekomendasi bahan aktif:",
		"Waktu pengaplikasian:",
		"Cara aplikasi:",
		"Tindakan lanjut:",
		"Catatan keamanan:",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected message to contain %q, got:\n%s", expected, message)
		}
	}
}
