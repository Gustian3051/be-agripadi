package consultation

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gustian305/backend/internal/dto"
	"github.com/gustian305/backend/logger"
)

type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type ConsultationService struct {
	llmClient LLMClient
}

func NewConsultationService(
	llmClient LLMClient,
) *ConsultationService {
	return &ConsultationService{
		llmClient: llmClient,
	}
}

const (
	MaxConsultationMessageLength = 1000
	DefaultLLMTimeout            = 10 * time.Second
)

func ValidateConsultationMessage(message string) error {

	message = strings.TrimSpace(message)

	if message == "" {
		return errors.New(
			"message cannot be empty",
		)
	}

	if len(message) > MaxConsultationMessageLength {

		return errors.New(
			"message too long",
		)
	}

	return nil
}

func (s *ConsultationService) ProcessMessage(ctx context.Context, message string) (*dto.OrchestratorResponse, error) {
	startedAt := time.Now()
	operation := "ProcessMessage"

	message = strings.TrimSpace(message)
	logger.Request(
		"app.consultation",
		operation,
		slog.Int("message_length", len(message)),
		slog.Bool("suggest_image", ShouldSuggestImage(message)),
	)
	logger.DebugPayload(
		"app.consultation",
		operation,
		slog.String("message", logger.Truncate(message, logger.DefaultTextLimit)),
	)

	if err := ValidateConsultationMessage(
		message,
	); err != nil {

		logger.Failure("app.consultation", operation, startedAt, err)
		return nil, err
	}
	// ========================================================
	// BUILD PROMPT
	// ========================================================

	prompt := BuildConsultationPrompt(
		message,
	)

	// ========================================================
	// CALL LLM
	// ========================================================

	if s.llmClient == nil {

		response := BuildLocalConsultationResponse(
			message,
		)

		result := s.BuildChatResponse(
			response,
			ShouldSuggestImage(message),
		)
		logger.Response(
			"app.consultation",
			operation,
			startedAt,
			slog.String("source", "local_fallback"),
			slog.Int("response_length", len(result.Message)),
			slog.Int("action_count", len(result.Actions)),
		)
		return result, nil
	}

	llmCtx, cancel := withDefaultTimeout(
		ctx,
		DefaultLLMTimeout,
	)
	defer cancel()

	response, err := s.llmClient.Generate(
		llmCtx,
		prompt,
	)

	if err != nil {
		logger.Failure(
			"app.consultation",
			"llm.Generate",
			startedAt,
			err,
			slog.String("fallback", "local_response"),
		)
		response = BuildLocalConsultationResponse(
			message,
		)
	} else {
		response = SanitizeConsultationResponse(
			response,
		)
	}

	if response == "" {
		response = BuildLocalConsultationResponse(
			message,
		)
	}

	if ShouldSuggestImage(message) {

		response += "\n\n" +
			BuildImageSuggestion()
	}

	result := s.BuildChatResponse(
		response,
		ShouldSuggestImage(message),
	)

	logger.Response(
		"app.consultation",
		operation,
		startedAt,
		slog.Int("response_length", len(result.Message)),
		slog.Int("action_count", len(result.Actions)),
	)
	logger.DebugPayload(
		"app.consultation",
		operation,
		slog.String("response", logger.Truncate(result.Message, logger.DefaultTextLimit)),
	)

	return result, nil
}

func (s *ConsultationService) BuildChatResponse(message string, suggestImage bool) *dto.OrchestratorResponse {

	actions := make(
		[]dto.ChatAction,
		0,
	)

	if suggestImage {

		actions = append(
			actions,
			dto.ChatAction{
				Type: dto.ActionUploadImage,

				Label: "Upload Gambar Hama",

				Value: "upload_image",
			},
		)
	}

	return &dto.OrchestratorResponse{
		SessionID: uuid.New(),

		Mode: dto.ConversationModeConsultation,

		State: dto.DiagnoseFlowStateIdle,

		Message: message,

		Actions: actions,

		CreatedAt: time.Now(),
	}
}

func BuildConsultationPrompt(message string) string {

	return `
Anda adalah pendamping percakapan untuk petani padi.

Gaya bicara yang diinginkan:
- hangat, natural, dan ramah
- seolah sedang ngobrol dengan petani
- tidak kaku seperti laporan
- singkat, tapi tetap membantu
- hindari kalimat yang terlalu formal atau berulang

Tugas:
- menjawab pertanyaan pertanian padi dengan penjelasan yang mudah dipahami
- memberi saran praktis yang aman
- membantu petani memahami kondisi tanaman, perawatan, dan gejala umum
- menyarankan upload gambar hama jika ada indikasi serangan hama

Aturan penting:
- gunakan bahasa Indonesia sederhana
- tulis sebagai teks biasa; jangan gunakan markdown seperti **bold**
- maksimal 5 paragraf pendek, atau bullet singkat jika lebih enak dibaca
- jangan membuat diagnosis atau klasifikasi hama pasti tanpa gambar hama atau data yang jelas
- jangan membuat dosis pestisida baru
- jangan membuat data baru atau mengarang istilah teknis
- kalau pertanyaan masih umum, jawab dengan penjelasan umum yang relevan
- kalau mengarah ke serangan hama, arahkan pengguna untuk upload gambar hama secara sopan
- jangan meminta upload gambar tanaman padi untuk diagnosis penyakit; fitur diagnosis aplikasi berfokus pada deteksi/klasifikasi hama dari gambar hama
- bila perlu, beri 1-2 langkah praktis yang bisa langsung dilakukan petani

PENTING:
- Jangan tampilkan proses berpikir.
- Jangan gunakan tag <think>.
- Jangan tampilkan reasoning internal.
- Keluarkan hanya jawaban akhir untuk pengguna.

PERTANYAAN PENGGUNA:
` + message
}

func BuildFallbackResponse() string {

	return strings.TrimSpace(`
Saya bisa bantu konsultasi umum tentang budidaya padi.

Untuk dugaan serangan hama, upload gambar hama yang terlihat agar sistem diagnosis bisa mengklasifikasikan hama dan memberi rekomendasi yang lebih tepat.
`)
}

func BuildLocalConsultationResponse(message string) string {

	normalized := strings.ToLower(
		strings.TrimSpace(message),
	)

	switch {
	case isGreeting(normalized):
		return "Halo, siap bantu ya. Anda bisa cerita soal kondisi padi, pemupukan, air sawah, fase pertumbuhan, atau upload gambar hama kalau ingin dibantu klasifikasi hama."

	case strings.Contains(normalized, "padi"):
		return "Padi memang perlu dipantau rutin, terutama air, warna daun, pertumbuhan anakan, dan kondisi malai. Kalau Anda mau, ceritakan umur tanaman, fase pertumbuhan, kondisi lahan, dan gejala yang terlihat supaya saya bisa bantu lebih tepat. Kalau ada dugaan serangan hama, upload gambar hama yang terlihat juga boleh."

	case ShouldSuggestImage(normalized):
		return "Dari cerita Anda, kemungkinan ada kaitannya dengan serangan hama padi. Supaya lebih aman dan tepat, coba upload gambar hama yang terlihat. Nanti sistem akan bantu lanjutkan dengan pertanyaan gejala dan tingkat keparahan."

	default:
		return "Saya bisa bantu konsultasi umum seputar padi. Coba jelaskan kondisi tanaman, umur atau fase pertumbuhan, kondisi air, warna daun, dan masalah yang sedang Anda lihat."
	}
}

func isGreeting(message string) bool {

	greetings := []string{
		"halo",
		"hai",
		"hi",
		"hello",
		"pagi",
		"siang",
		"sore",
		"malam",
		"assalamualaikum",
	}

	for _, greeting := range greetings {
		if message == greeting ||
			strings.HasPrefix(message, greeting+" ") {

			return true
		}
	}

	return false
}

func SanitizeConsultationResponse(text string) string {

	text = strings.TrimSpace(text)

	
	text = removeThinkingBlocks(text)

	text = strings.TrimSpace(text)

	text = strings.ReplaceAll(
		text,
		"```",
		"",
	)

	text = strings.ReplaceAll(
		text,
		"**",
		"",
	)

	replacements := map[string]string{
		"upload gambar tanaman padi yang menunjukkan gejala serangan hama atau penyakit": "upload gambar hama yang terlihat pada tanaman padi",
		"upload gambar tanaman agar sistem diagnosis bisa memeriksa label hama":          "upload gambar hama agar sistem diagnosis bisa mengklasifikasikan hama",
		"upload gambar tanaman atau hama yang terlihat":                                  "upload gambar hama yang terlihat",
		"gambar tanaman padi yang menunjukkan gejala serangan hama atau penyakit":        "gambar hama yang terlihat pada tanaman padi",
		"gambar tanaman agar sistem diagnosis":                                           "gambar hama agar sistem diagnosis",
	}

	for oldValue, newValue := range replacements {
		text = strings.ReplaceAll(text, oldValue, newValue)
	}

	text = strings.ReplaceAll(
		text,
		"\r\n",
		"\n",
	)

	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(
			text,
			"\n\n\n",
			"\n\n",
		)
	}

	return strings.TrimSpace(text)
}

func IsDiagnoseIntent(message string) bool {

	message = strings.ToLower(
		strings.TrimSpace(message),
	)

	keywords := []string{
		"hama",
		"wereng",
		"ulat",
		"serangga",
		"bercak",
		"daun kuning",
		"menguning",
		"layu",
		"busuk",
		"rusak",
		"mati",
		"kering",
		"jamur",
		"berlubang",
		"berwarna coklat",
		"berwarna hitam",
		"batang patah",
		"tanaman roboh",
	}

	for _, keyword := range keywords {

		if strings.Contains(
			message,
			keyword,
		) {
			return true
		}
	}

	return false
}

func ShouldSuggestImage(message string) bool {

	return IsDiagnoseIntent(message)
}

func BuildImageSuggestion() string {

	return `
Kalau ingin hasil diagnosis yang lebih akurat, silakan upload gambar hama yang terlihat pada tanaman padi agar sistem bisa membantu klasifikasi hama.
`
}

func withDefaultTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {

	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}

	return context.WithTimeout(
		ctx,
		timeout,
	)
}

func removeThinkingBlocks(text string) string {

	for {

		start := strings.Index(text, "<think>")
		end := strings.Index(text, "</think>")

		if start < 0 ||
			end < 0 ||
			end <= start {

			break
		}

		text =
			text[:start] +
				text[end+len("</think>"):]
	}

	// fallback
	text = strings.ReplaceAll(text, "<think>", "")
	text = strings.ReplaceAll(text, "</think>", "")
	text = strings.ReplaceAll(text, "<think/>", "")
	text = strings.ReplaceAll(text, "<think />", "")

	return strings.TrimSpace(text)
}