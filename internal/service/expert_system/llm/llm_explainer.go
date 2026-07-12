package llm

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/gustian305/backend/logger"
)

type LLMClient interface {
	Generate(
		ctx context.Context,
		prompt string,
	) (string, error)
}

type Language string

const (
	LanguageIndonesian Language = "id"
	LanguageSundanese  Language = "su"
	LanguageJavanese   Language = "jv"
)

type ExplanationTone string

const (
	ExplanationToneFormal   ExplanationTone = "formal"
	ExplanationToneFriendly ExplanationTone = "friendly"
	ExplanationToneFarmer   ExplanationTone = "farmer"
)

type ExplainerService struct {
	client LLMClient
}

func NewExplainerService(client LLMClient) *ExplainerService {
	return &ExplainerService{
		client: client,
	}
}

type ExplainRequest struct {
	PestName          string
	Severity          string
	GrowthStage       string
	Symptoms          []string
	Pesticides        []string
	Reasoning         []string
	Language          Language
	LocalDialect      *string
	Tone              ExplanationTone
	IncludeEducation  bool
	IncludePrevention bool
	IncludeSafety     bool
}

type ExplainResponse struct {
	Language        string
	Dialect         *string
	Summary         string
	Reasoning       []string
	FarmerEducation []string
	Prevention      []string
	Warnings        []string
	FollowUpActions []string
}

func (s *ExplainerService) Generate(ctx context.Context, req ExplainRequest) (*ExplainResponse, error) {
	startedAt := time.Now()
	operation := "GenerateExplanation"

	logger.Request(
		"expert_system.llm",
		operation,
		slog.String("pest_name", req.PestName),
		slog.String("severity", req.Severity),
		slog.String("growth_stage", req.GrowthStage),
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
		logger.Failure("expert_system.llm", operation, startedAt, err)
		return nil, err
	}

	response := &ExplainResponse{
		Language:  string(req.Language),
		Dialect:   req.LocalDialect,
		Summary:   result,
		Reasoning: req.Reasoning,
		FarmerEducation: make(
			[]string,
			0,
		),

		Prevention: make(
			[]string,
			0,
		),

		Warnings: make(
			[]string,
			0,
		),

		FollowUpActions: make(
			[]string,
			0,
		),
	}

	if req.IncludeEducation {

		response.FarmerEducation =
			s.buildEducation(
				req,
			)
	}

	if req.IncludePrevention {

		response.Prevention =
			s.buildPrevention(
				req,
			)
	}

	if req.IncludeSafety {

		response.Warnings =
			s.buildWarnings(
				req,
			)
	}

	response.FollowUpActions =
		s.buildFollowUpActions(
			req,
		)

	logger.Response(
		"expert_system.llm",
		operation,
		startedAt,
		slog.Int("summary_length", len(response.Summary)),
		slog.Int("reasoning_count", len(response.Reasoning)),
		slog.Int("warning_count", len(response.Warnings)),
		slog.Int("follow_up_count", len(response.FollowUpActions)),
	)
	logger.DebugPayload(
		"expert_system.llm",
		operation,
		slog.String("summary", logger.Truncate(response.Summary, 1200)),
	)

	return response, nil
}

func (s *ExplainerService) buildPrompt(
	req ExplainRequest,
) string {

	var builder strings.Builder

	builder.WriteString(`
Anda adalah AI Agronomist profesional untuk budidaya padi.

PERAN ANDA:

Anda BUKAN sistem diagnosis.

Diagnosis telah ditentukan oleh sistem pakar dan merupakan hasil final.

Tugas Anda HANYA menjelaskan hasil diagnosis kepada petani dengan bahasa yang mudah dipahami.

==================================================
ATURAN WAJIB
==================================================

1. Jangan melakukan diagnosis ulang.
2. Jangan menebak hama lain.
3. Jangan menyebutkan kemungkinan hama lain.
4. Jangan mengubah nama hama.
5. Jangan membuat kesimpulan baru.
6. Jangan memberikan identifikasi baru.
7. Jangan menyebutkan penyakit atau hama lain sebagai pembanding.
8. Gunakan hanya informasi yang diberikan sistem.
9. Jika informasi tidak lengkap, tetap jelaskan berdasarkan hasil diagnosis sistem.
10. Fokus pada edukasi petani.

==================================================
HASIL DIAGNOSIS SISTEM
==================================================
`)

	builder.WriteString(
		"Nama Hama: " +
			req.PestName +
			"\n",
	)

	builder.WriteString(
		"Tingkat Keparahan: " +
			req.Severity +
			"\n",
	)

	if strings.TrimSpace(
		req.GrowthStage,
	) != "" {

		builder.WriteString(
			"Fase Pertumbuhan: " +
				req.GrowthStage +
				"\n",
		)
	}

	//--------------------------------------------------
	// GEJALA
	//--------------------------------------------------

	if len(req.Symptoms) > 0 {

		builder.WriteString(
			"\nGejala yang diamati (untuk konteks penjelasan, bukan untuk diagnosis):\n",
		)

		for _, symptom :=
			range req.Symptoms {

			if strings.TrimSpace(
				symptom,
			) == "" {

				continue
			}

			builder.WriteString(
				"- " +
					symptom +
					"\n",
			)
		}
	}

	//--------------------------------------------------
	// CRITICAL GUARDRAIL
	//--------------------------------------------------

	builder.WriteString(`
==================================================
ATURAN KRITIS
==================================================

HANYA gunakan nama hama berikut:

`)

	builder.WriteString(
		req.PestName,
	)

	builder.WriteString(`

Jangan menyebutkan nama hama lain.
Jangan menyebutkan kemungkinan hama lain.
Jangan melakukan identifikasi ulang.
Jangan memberikan diagnosis alternatif.

==================================================
TUGAS
==================================================

Jelaskan kepada petani:

1. Apa itu hama tersebut.
2. Bagaimana hama tersebut mempengaruhi tanaman padi.
3. Hubungan gejala yang diamati dengan serangan hama tersebut.
4. Dampak tingkat keparahan terhadap tanaman.
5. Hal yang perlu dipantau petani di lapangan.

==================================================
FORMAT JAWABAN
==================================================

Paragraf 1:
Penjelasan singkat mengenai hama yang telah didiagnosis sistem.

Paragraf 2:
Hubungan gejala yang diamati dengan serangan hama tersebut.

Paragraf 3:
Dampak tingkat keparahan terhadap tanaman.

Paragraf 4:
Hal yang perlu dipantau petani.

==================================================
BATASAN OUTPUT
==================================================

- Maksimal 250 kata.
- Gunakan bahasa Indonesia yang sederhana.
- Gunakan istilah yang mudah dipahami petani.
- Jangan membuat daftar kemungkinan hama.
- Jangan membuat diagnosis baru.
- Jangan mengubah hasil diagnosis sistem.
- Jangan menyebutkan hama selain yang diberikan sistem.
`)

	return builder.String()
}


func (s *ExplainerService) buildEducation(req ExplainRequest) []string {

	items := make(
		[]string,
		0,
	)

	items = append(
		items,
		"Pemantauan rutin membantu mendeteksi serangan sejak awal sehingga tindakan bisa lebih cepat.",
	)

	items = append(
		items,
		"Rotasi bahan aktif penting agar hama tidak cepat kebal terhadap satu jenis pestisida.",
	)

	items = append(
		items,
		"Penggunaan dosis berlebih bisa merusak tanaman, menambah biaya, dan mencemari lingkungan.",
	)

	if strings.ToLower(
		req.Severity,
	) == "high" {

		items = append(
			items,
			"Serangan berat perlu dipantau lebih intensif agar penyebarannya tidak meluas.",
		)
	}

	return items
}

func (s *ExplainerService) buildPrevention(req ExplainRequest) []string {

	items := make(
		[]string,
		0,
	)

	items = append(
		items,
		"Gunakan varietas yang lebih tahan terhadap hama bila tersedia di daerah Anda.",
	)

	items = append(
		items,
		"Jaga sanitasi sawah dengan membersihkan sisa tanaman, gulma, dan sumber persembunyian hama.",
	)

	items = append(
		items,
		"Hindari penggunaan pestisida terus-menerus dengan bahan aktif yang sama agar tidak memicu resistensi.",
	)

	items = append(
		items,
		"Lakukan pengamatan minimal 1-2 kali per minggu, terutama pada tanaman yang baru menunjukkan gejala.",
	)

	return items
}

func (s *ExplainerService) buildWarnings(req ExplainRequest) []string {

	items := make(
		[]string,
		0,
	)

	items = append(
		items,
		"Gunakan alat pelindung diri lengkap saat mencampur dan menyemprot, termasuk sarung tangan, masker, dan pelindung mata.",
	)

	items = append(
		items,
		"Jangan menyemprot saat angin kencang atau cuaca tidak stabil agar semprotan tidak terbawa ke area lain.",
	)

	items = append(
		items,
		"Ikuti dosis sesuai label pestisida dan jangan menambah takaran hanya karena ingin hasil lebih cepat.",
	)

	if strings.ToLower(
		req.Severity,
	) == "high" {

		items = append(
			items,
			"Segera lakukan pengendalian lanjutan jika serangan masih berat supaya penyebaran tidak semakin luas.",
		)
	}

	return items
}

func (s *ExplainerService) buildFollowUpActions(req ExplainRequest) []string {

	items := make(
		[]string,
		0,
	)

	items = append(
		items,
		"Amati perkembangan gejala selama 3-5 hari setelah tindakan agar terlihat apakah tanaman mulai membaik.",
	)

	items = append(
		items,
		"Catat perubahan kondisi tanaman setelah aplikasi, misalnya warna daun, jumlah hama, dan bagian yang mulai pulih.",
	)

	items = append(
		items,
		"Ulangi pemeriksaan bila gejala tidak berkurang atau muncul gejala baru di titik lain.",
	)

	return items
}
