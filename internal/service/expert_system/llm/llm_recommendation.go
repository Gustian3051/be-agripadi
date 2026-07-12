package llm

import (
	"context"
	"log/slog"
	// "strconv"
	"strings"
	"time"

	"github.com/gustian305/backend/logger"
)

type RecommendationService struct {
	client LLMClient
}

func NewRecommendationService(
	client LLMClient,
) *RecommendationService {

	return &RecommendationService{
		client: client,
	}
}

type RecommendationLanguage string

const (
	RecommendationLanguageID RecommendationLanguage = "id"
	RecommendationLanguageEN RecommendationLanguage = "en"
)

type FarmerExperienceLevel string

const (
	FarmerExperienceBeginner     FarmerExperienceLevel = "beginner"
	FarmerExperienceIntermediate FarmerExperienceLevel = "intermediate"
	FarmerExperienceExpert       FarmerExperienceLevel = "expert"
)

type PesticideLLMData struct {
	ProductName       string
	Dose              string
	ActiveIngredient  string
	Formulation       string
	ApplicationTiming string
}

type RecommendationRequest struct {
	PestName            string
	DetectionLabel      string
	DetectionConfidence float64
	Symptoms            []string
	RuleCode            string
	RuleConfidence      float64

	Pesticides []PesticideLLMData

	Severity          string
	Language          RecommendationLanguage
	FarmerName        *string
	FarmerExperience  FarmerExperienceLevel
	GrowthStage       *string
	WaterAvailability *string
}

type RecommendationResponse struct {
	Message         string
	OperationalTips []string
}

func (s *RecommendationService) Generate(ctx context.Context, req RecommendationRequest) (*RecommendationResponse, error) {
	startedAt := time.Now()
	operation := "GenerateRecommendation"

	logger.Request(
		"expert_system.llm",
		operation,
		slog.String("pest_name", req.PestName),
		slog.String("severity", req.Severity),
		slog.Int("symptom_count", len(req.Symptoms)),
		slog.Int("pesticide_count", len(req.Pesticides)),
		slog.String("language", string(req.Language)),
	)

	if s == nil ||
		s.client == nil {

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

		logger.Failure(
			"expert_system.llm",
			operation,
			startedAt,
			err,
		)

		return nil, err
	}

	result = sanitizeLLMOutput(
		result,
	)

	response := &RecommendationResponse{
		Message: result,

		OperationalTips: make(
			[]string,
			0,
		),
	}

	response.OperationalTips =
		s.buildOperationalTips(
			req,
		)

	logger.Response(
		"expert_system.llm",
		operation,
		startedAt,
		slog.Int("message_length", len(response.Message)),
		slog.Int("operational_tip_count", len(response.OperationalTips)),
	)
	logger.DebugPayload(
		"expert_system.llm",
		operation,
		slog.String("message", logger.Truncate(response.Message, 1200)),
	)

	return response, nil
}

func (s *RecommendationService) buildPrompt(
	req RecommendationRequest,
) string {

	var builder strings.Builder

	pestName := strings.TrimSpace(
		req.PestName,
	)

	if pestName == "" {
		pestName =
			strings.TrimSpace(
				req.DetectionLabel,
			)
	}

	builder.WriteString(`
Anda adalah AI Agronomist untuk budidaya padi.

Gunakan mode /no_think. Jangan menampilkan proses berpikir.

Diagnosis dan rekomendasi pestisida sudah ditentukan oleh sistem.

Tugas Anda hanya membuat narasi pendamping untuk hasil sistem:

2. Ringkasan Diagnosis
4. Alasan Pemilihan Pestisida
5. Tindakan Pengendalian Berdasarkan Severity

Jangan:

- membuat diagnosis baru
- mengubah nama hama
- mengubah tingkat keparahan
- membuat pestisida baru
- mempromosikan pestisida; nama produk boleh disebut sebagai identitas rekomendasi jika berasal dari database sistem
- mengubah dosis
- membuat waktu aplikasi sendiri di luar data backend
- membuat cara penggunaan atau instruksi aplikasi teknis pestisida
- membuat aturan keamanan penggunaan pestisida
- menggunakan kata "semprot" seolah semua formulasi diaplikasikan dengan cara semprot; gunakan istilah netral "aplikasi sesuai formulasi dan label"
- mengklaim bahan aktif pasti efektif atau paling ampuh
- menggunakan istilah "tinggi" untuk severity; gunakan istilah sistem: ringan, sedang, atau berat
- membuat kalimat promosi; gunakan bahasa netral berbasis database sistem
- menjelaskan pestisida alternatif
- menampilkan proses berpikir
- menampilkan tag <think>

`)

	builder.WriteString(
		"Nama Hama: " +
			pestName +
			"\n",
	)

	builder.WriteString(
		"Tingkat Keparahan: " +
			req.Severity +
			"\n",
	)

	if req.GrowthStage != nil {

		builder.WriteString(
			"Fase Pertumbuhan: " +
				*req.GrowthStage +
				"\n",
		)
	}

	if len(req.Symptoms) > 0 {

		builder.WriteString(
			"Gejala: ",
		)

		builder.WriteString(
			strings.Join(
				req.Symptoms,
				", ",
			),
		)

		builder.WriteString("\n")
	}

	builder.WriteString("\n")

	if len(req.Pesticides) > 0 {

		builder.WriteString(
			"REKOMENDASI PENGENDALIAN KIMIA TOP 1 DARI DATABASE SISTEM\n",
		)

		builder.WriteString(
			formatPesticideForPrompt(
				req.Pesticides[0],
			),
		)

		builder.WriteString("\n")
	}

	builder.WriteString(`

FORMAT JAWABAN

2. Ringkasan Diagnosis
- 1 paragraf pendek.
- Wajib menyebut hama, severity, fase pertumbuhan, dan konteks gejala.
- Jangan mengubah hasil diagnosis sistem.

4. Alasan Pemilihan Pestisida
- 1 paragraf pendek.
- Fokus pada rekomendasi Top 1 dari sistem.
- Sebut nama produk Top 1 sebagai identitas rekomendasi utama jika tersedia pada database sistem.
- Sebut bahan aktif dan formulasi sebagai informasi pendukung.
- Gunakan frasa netral seperti "diprioritaskan berdasarkan database sistem" atau "tercatat sesuai target hama".
- Jangan menulis "efektif mengendalikan", "paling efektif", "ampuh", atau klaim kepastian lain.
- Nama produk Top 1 boleh menjadi subjek utama kalimat, tetapi jelaskan secara netral dan bukan promosi.
- Tetap sebut bahan aktif Top 1 agar pengguna memahami kandungan produk.
- Jangan membuat atau mengubah dosis.
- Jangan membuat waktu aplikasi sendiri; jika waktu aplikasi tersedia pada data sistem, boleh diringkas sebagai informasi pendukung.
- Jangan menampilkan acuan waktu/referensi mentah seperti nama instansi, tahun, URL, atau judul artikel.
- Jangan menggabungkan waktu aplikasi dengan catatan PHT; waktu, cara, dan catatan penggunaan akan diformat oleh backend secara terpisah.

5. Tindakan Pengendalian Berdasarkan Severity
- Maksimal 4 poin.
- Wajib menyesuaikan tindakan dengan severity dan fase pertumbuhan.
- Jangan membuat instruksi pencampuran, interval aplikasi, atau dosis baru.
- Wajib memakai istilah netral "aplikasi sesuai formulasi/label" karena rekomendasi dapat berisi EC, SC, SL, atau GR.
- Wajib menyebut PHT, pemantauan populasi hama, dan rotasi bahan aktif untuk menekan risiko resistensi.

Maksimal 220 kata.
Jangan membuat bagian selain nomor 2, 4, dan 5.
`)

	return builder.String()
}

func (s *RecommendationService) buildOperationalTips(req RecommendationRequest) []string {

	items := make(
		[]string,
		0,
	)

	switch req.FarmerExperience {

	case FarmerExperienceExpert:

		items = append(
			items,
			"Optimalkan rotasi bahan aktif untuk mencegah resistensi hama dan menjaga efektivitas pengendalian di musim berikutnya.",
		)
	}

	severity :=
		strings.ToLower(
			strings.TrimSpace(
				req.Severity,
			),
		)

	switch severity {

	case "high",
		"berat":

		items = append(
			items,
			"Lakukan monitoring ulang 3-5 hari setelah aplikasi untuk memastikan gejala tidak bertambah parah.",
		)
	}

	return items
}

// func formatConfidence(
// 	value float64,
// ) string {

// 	if value <= 0 {
// 		return "-"
// 	}

// 	return strings.TrimSpace(
// 		strconv.FormatFloat(
// 			value*100,
// 			'f',
// 			2,
// 			64,
// 		),
// 	) + "%"
// }

func sanitizeLLMOutput(text string) string {

	for {

		start :=
			strings.Index(
				text,
				"<think>",
			)

		end :=
			strings.Index(
				text,
				"</think>",
			)

		if start < 0 ||
			end < 0 ||
			end <= start {

			break
		}

		text =
			text[:start] +
				text[end+len("</think>"):]
	}

	text =
		strings.TrimSpace(
			text,
		)

	text = neutralizePesticideClaims(text)

	lines := strings.Split(
		text,
		"\n",
	)

	start := -1

	for i, line := range lines {

		lower :=
			strings.ToLower(
				strings.TrimSpace(
					line,
				),
			)

		if strings.HasPrefix(
			lower,
			"2.",
		) ||
			strings.Contains(
				lower,
				"ringkasan diagnosis",
			) {

			start = i
			break
		}
	}

	if start >= 0 {

		return strings.TrimSpace(
			strings.Join(
				lines[start:],
				"\n",
			),
		)
	}

	return strings.TrimSpace(
		text,
	)
}

func neutralizePesticideClaims(text string) string {
	replacements := map[string]string{
		"dipilih karena efektif mengendalikan":       "direkomendasikan sesuai target pengendalian",
		"dipilih karena efektif untuk mengendalikan": "direkomendasikan sesuai target pengendalian",
		"efektif mengendalikan":                      "sesuai dengan target pengendalian",
		"efektif untuk mengendalikan":                "sesuai dengan target pengendalian",
		"paling efektif":                             "diprioritaskan oleh sistem",
		"ampuh":                                      "sesuai dengan target pengendalian",
		"keparahan yang tinggi":                      "keparahan berat",
		"tingkat keparahan tinggi":                   "tingkat keparahan berat",
		"tingkat serangan tinggi":                    "tingkat serangan berat",
		"keparahan tinggi":                           "keparahan berat",
		"severity tinggi":                            "severity berat",
	}

	for old, replacement := range replacements {
		text = strings.ReplaceAll(text, old, replacement)
		text = strings.ReplaceAll(text, strings.Title(old), strings.Title(replacement))
	}

	return text
}

func formatPesticideForPrompt(p PesticideLLMData) string {

	parts := make([]string, 0, 4)

	if p.ProductName != "" {
		parts = append(parts,
			"Produk rekomendasi: "+p.ProductName,
		)
	}

	if p.Dose != "" {
		parts = append(parts,
			"Dosis: "+p.Dose,
		)
	}

	if p.ActiveIngredient != "" {
		parts = append(parts,
			"Bahan Aktif: "+p.ActiveIngredient,
		)
	}

	if p.Formulation != "" {
		parts = append(parts,
			"Formulasi: "+p.Formulation,
		)
	}

	if p.ApplicationTiming != "" {
		parts = append(parts,
			"Waktu aplikasi dari database: "+p.ApplicationTiming,
		)
	}

	return strings.Join(parts, " | ")
}
