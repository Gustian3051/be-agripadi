package llm

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gustian305/backend/logger"
)

//
// ============================================================
// FALLBACK SERVICE
// ============================================================
//

type FallbackService struct {
	client LLMClient
}

func NewFallbackService(client LLMClient) *FallbackService {
	return &FallbackService{
		client: client,
	}
}

type FallbackLevel string

const (
	FallbackLevelLow    FallbackLevel = "low"
	FallbackLevelMedium FallbackLevel = "medium"
	FallbackLevelHigh   FallbackLevel = "high"
)

type EscalationType string

const (
	EscalationTypeNone            EscalationType = "none"
	EscalationTypeHumanExpert     EscalationType = "human_expert"
	EscalationTypeFieldInspection EscalationType = "field_inspection"
	EscalationTypeLaboratory      EscalationType = "laboratory"
)

type ActiveQuestion struct {
	Code     string `json:"code"`
	Question string `json:"question"`
	Reason   string `json:"reason"`
}

type FallbackRequest struct {
	UserInput       string
	Symptoms        []string
	DetectedPest    string
	TopCandidates   []string
	Confidence      float64
	UnknownSymptoms []string
	Language        string
	GrowthStage     *string
	Weather         *string
}

type FallbackResponse struct {
	Message             string
	Explanation         string
	UncertaintyReason   []string
	NeedHumanValidation bool
	FallbackLevel       FallbackLevel
	Escalation          EscalationType
	Questions           []ActiveQuestion
	RecommendedActions  []string
	Observations        []string
}

func (s *FallbackService) Generate(ctx context.Context, req FallbackRequest) (*FallbackResponse, error) {
	startedAt := time.Now()
	operation := "GenerateFallback"

	logger.Request(
		"expert_system.llm",
		operation,
		slog.String("detected_pest", req.DetectedPest),
		slog.Float64("confidence", req.Confidence),
		slog.Int("symptom_count", len(req.Symptoms)),
		slog.Int("candidate_count", len(req.TopCandidates)),
		slog.Int("unknown_symptom_count", len(req.UnknownSymptoms)),
		slog.String("language", req.Language),
	)

	if s == nil || s.client == nil {

		logger.Failure("expert_system.llm", operation, startedAt, ErrLLMClientNotConfigured)
		return nil, ErrLLMClientNotConfigured
	}

	prompt :=
		s.buildPrompt(
			req,
		)
	logger.DebugPayload(
		"expert_system.llm",
		operation,
		slog.String("prompt", logger.Truncate(prompt, 1200)),
	)

	llmCtx, cancel := contextWithDefaultTimeout(
		ctx,
	)
	defer cancel()

	result, err :=
		s.client.Generate(
			llmCtx,
			prompt,
		)

	if err != nil {
		logger.Failure("expert_system.llm", operation, startedAt, err)
		return nil, err
	}

	level :=
		s.calculateFallbackLevel(
			req.Confidence,
		)

	escalation :=
		s.determineEscalation(
			level,
			req,
		)

	response := &FallbackResponse{
		Message: result,

		Explanation: s.buildExplanation(
			req,
		),

		UncertaintyReason: s.buildUncertaintyReason(
			req,
		),

		NeedHumanValidation: escalation !=
			EscalationTypeNone,

		FallbackLevel: level,

		Escalation: escalation,

		Questions: s.buildQuestions(
			req,
		),

		RecommendedActions: s.buildActions(
			req,
		),

		Observations: s.buildObservations(
			req,
		),
	}

	logger.Response(
		"expert_system.llm",
		operation,
		startedAt,
		slog.String("fallback_level", string(response.FallbackLevel)),
		slog.String("escalation", string(response.Escalation)),
		slog.Bool("need_human_validation", response.NeedHumanValidation),
		slog.Int("message_length", len(response.Message)),
		slog.Int("question_count", len(response.Questions)),
	)
	logger.DebugPayload(
		"expert_system.llm",
		operation,
		slog.String("message", logger.Truncate(response.Message, 1200)),
	)

	return response, nil
}

func (s *FallbackService) buildPrompt(req FallbackRequest) string {

	builder := strings.Builder{}

	builder.WriteString(`
Anda adalah AI pertanian padi.

Sistem pakar belum cukup yakin.

ATURAN:
- jangan memberikan kepastian palsu
- jangan membuat diagnosis final
- jelaskan ketidakpastian dengan bahasa yang mudah dipahami
- gunakan bahasa sederhana
- bantu petani melakukan observasi tambahan
- gunakan gaya ramah petani
- fokus pada pemulihan dari ketidakpastian
`)

	builder.WriteString(`

Berikan:
- penjelasan singkat mengapa sistem belum yakin
- observasi tambahan yang perlu dilakukan petani
- pertanyaan lanjutan yang mudah dijawab
- langkah validasi di lapangan
- kapan perlu bantuan ahli
`)

	return builder.String()
}

func (s *FallbackService) calculateFallbackLevel(confidence float64) FallbackLevel {

	switch {

	case confidence >= 0.75:
		return FallbackLevelLow

	case confidence >= 0.45:
		return FallbackLevelMedium

	default:
		return FallbackLevelHigh
	}
}

func (s *FallbackService) determineEscalation(level FallbackLevel, req FallbackRequest) EscalationType {

	switch level {

	case FallbackLevelLow:
		return EscalationTypeNone

	case FallbackLevelMedium:
		return EscalationTypeFieldInspection

	case FallbackLevelHigh:

		if len(req.UnknownSymptoms) > 3 {

			return EscalationTypeLaboratory
		}

		return EscalationTypeHumanExpert
	}

	return EscalationTypeNone
}

func (s *FallbackService) buildExplanation(req FallbackRequest) string {

	if req.Confidence >= 0.75 {

		return "Sistem masih memiliki keyakinan cukup baik, namun beberapa gejala memerlukan validasi tambahan."
	}

	if req.Confidence >= 0.45 {

		return "Terdapat beberapa kemungkinan hama dengan gejala yang mirip sehingga diperlukan observasi tambahan."
	}

	return "Gejala yang ditemukan belum cukup spesifik untuk menghasilkan diagnosis yang aman dan akurat."
}

func (s *FallbackService) buildUncertaintyReason(req FallbackRequest) []string {

	items := make(
		[]string,
		0,
	)

	if len(req.TopCandidates) > 1 {

		items = append(
			items,
			"Terdapat beberapa kandidat hama dengan gejala serupa.",
		)
	}

	if len(req.UnknownSymptoms) > 0 {

		items = append(
			items,
			"Ditemukan gejala yang belum dikenali sistem.",
		)
	}

	if req.Confidence < 0.50 {

		items = append(
			items,
			"Tingkat keyakinan model masih rendah.",
		)
	}

	items = append(
		items,
		"Diperlukan observasi tambahan untuk meningkatkan akurasi.",
	)

	return items
}

func (s *FallbackService) buildQuestions(req FallbackRequest) []ActiveQuestion {

	questions := make(
		[]ActiveQuestion,
		0,
	)

	questions = append(
		questions,
		ActiveQuestion{
			Code: "leaf_pattern",

			Question: "Apakah terdapat bercak atau perubahan warna pada daun?",

			Reason: "Membantu membedakan jenis serangan hama.",
		},
	)

	questions = append(
		questions,
		ActiveQuestion{
			Code: "spread_pattern",

			Question: "Apakah gejala menyebar merata atau hanya di area tertentu?",

			Reason: "Membantu identifikasi pola penyebaran.",
		},
	)

	questions = append(
		questions,
		ActiveQuestion{
			Code: "insect_presence",

			Question: "Apakah terlihat serangga langsung pada tanaman?",

			Reason: "Validasi keberadaan hama utama.",
		},
	)

	if req.GrowthStage != nil {

		questions = append(
			questions,
			ActiveQuestion{
				Code: "growth_impact",

				Question: "Apakah fase pertumbuhan saat ini mengalami hambatan?",

				Reason: "Menilai dampak serangan terhadap pertumbuhan.",
			},
		)
	}

	return questions
}

func (s *FallbackService) buildActions(req FallbackRequest) []string {

	items := make(
		[]string,
		0,
	)

	items = append(
		items,
		"Lakukan observasi ulang pada daun dan batang tanaman.",
	)

	items = append(
		items,
		"Ambil foto tambahan dengan pencahayaan lebih baik.",
	)

	items = append(
		items,
		"Pantau perkembangan gejala selama 2-3 hari.",
	)

	if req.Confidence < 0.45 {

		items = append(
			items,
			"Konsultasikan dengan penyuluh atau ahli pertanian setempat.",
		)
	}

	return items
}

func (s *FallbackService) buildObservations(req FallbackRequest) []string {

	items := make(
		[]string,
		0,
	)

	items = append(
		items,
		"Perhatikan pola kerusakan daun.",
	)

	items = append(
		items,
		"Amati aktivitas serangga pada pagi dan sore hari.",
	)

	items = append(
		items,
		"Periksa apakah terdapat tanaman lain dengan gejala serupa.",
	)

	return items
}

func formatFloat(value float64) string {

	return strings.TrimRight(
		strings.TrimRight(
			strconv.FormatFloat(
				value,
				'f',
				2,
				64,
			),
			"0",
		),
		".",
	)
}
