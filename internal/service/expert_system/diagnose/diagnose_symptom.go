package diagnose

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/service/expert_system/session"
	"github.com/gustian305/backend/internal/service/expert_system/symptom"
	"github.com/gustian305/backend/internal/service/expert_system/workflow"
)

type SymptomService struct {
	sessionLoader  *session.SessionLoader
	sessionUpdater *session.SessionUpdater
	normalizer     *symptom.NormalizerService
	matcher        *symptom.MatcherService
}

func NewSymptomService(
	sessionLoader *session.SessionLoader,
	sessionUpdater *session.SessionUpdater,
	normalizer *symptom.NormalizerService,
	matcher *symptom.MatcherService,
) *SymptomService {

	return &SymptomService{
		sessionLoader:  sessionLoader,
		sessionUpdater: sessionUpdater,
		normalizer:     normalizer,
		matcher:        matcher,
	}
}

type SymptomSessionData struct {
	UserInputs []string                 `json:"user_inputs"`
	Normalized []NormalizedSymptomData  `json:"normalized"`
	Unknown    []string                 `json:"unknown,omitempty"`
	Confidence SymptomSessionConfidence `json:"confidence"`
}

type NormalizedSymptomData struct {
	InputText          string     `json:"input_text"`
	NormalizedText     string     `json:"normalized_text"`
	MatchedSymptomID   *uuid.UUID `json:"matched_symptom_id,omitempty"`
	MatchedSymptomName string     `json:"matched_symptom_name,omitempty"`
	MatchedText        string     `json:"matched_text,omitempty"`
	Confidence         float64    `json:"confidence"`
	Source             string     `json:"source"`
	SymptomType        string     `json:"symptom_type,omitempty"`
	RuleRole           string     `json:"rule_role,omitempty"`
	Severity           string     `json:"severity,omitempty"`
	GrowthStage        string     `json:"growth_stage,omitempty"`
	IsCoreSymptom      bool       `json:"is_core_symptom,omitempty"`
	RecommendedForRule bool       `json:"recommended_for_rule,omitempty"`
	DefaultWeight      float64    `json:"default_weight,omitempty"`
}

type SymptomSessionConfidence struct {
	Normalization float64 `json:"normalization"`
	RuleMatching  float64 `json:"rule_matching"`
}

func (s *SymptomService) Submit(ctx context.Context, conversationID string, symptoms []string) error {

	session, err := s.sessionLoader.RequireActiveSession(ctx, conversationID)

	if err != nil {
		return err
	}

	symptoms = mergeExistingSymptomInputs([]byte(session.Symptoms), symptoms)

	analysis, err := s.analyzeSymptoms(ctx, symptoms, session.DetectedLabel)

	if err != nil {
		return err
	}

	raw, err := json.Marshal(analysis)

	if err != nil {
		return err
	}

	if err := s.sessionUpdater.UpdateSymptoms(ctx, session, raw); err != nil {

		return err
	}

	if needsMoreFieldSymptomsBeforeSeverity(analysis) {
		// Identitas hama saja belum cukup untuk menentukan severity.
		// Session tetap berada pada collect_symptoms agar user menambahkan
		// gejala kerusakan/anchor severity yang terlihat di lapangan.
		return nil
	}

	if err := s.sessionUpdater.UpdateState(ctx, session, workflow.StateNormalizeSymptoms); err != nil {
		return err
	}

	return s.sessionUpdater.UpdateState(ctx, session, workflow.StateCollectSeverity)
}

func (s *SymptomService) analyzeSymptoms(ctx context.Context, inputs []string, pestLabel string) (*SymptomSessionData, error) {

	cleanInputs := cleanSymptomInputs(inputs)

	data := &SymptomSessionData{
		UserInputs: cleanInputs,
		Normalized: make(
			[]NormalizedSymptomData,
			0,
			len(cleanInputs),
		),
		Unknown: make(
			[]string,
			0,
		),
	}

	if s.normalizer == nil {
		s.normalizer = symptom.NewNormalizerService()
	}

	totalNormalization := 0.0
	totalMatching := 0.0

	for _, input := range cleanInputs {
		normalizedText := s.normalizer.Normalize(
			input,
		)

		item := NormalizedSymptomData{
			InputText:      input,
			NormalizedText: normalizedText,
			Confidence:     normalizationConfidence(normalizedText),
			Source:         "normalizer",
		}

		totalNormalization += item.Confidence

		if s.matcher != nil {
			match, err := s.matcher.MatchForPest(
				ctx,
				input,
				pestLabel,
			)

			if err != nil {
				return nil, err
			}

			if match != nil &&
				match.SymptomID != uuid.Nil &&
				match.Confidence >= 0.65 {

				matchedID := match.SymptomID
				item.MatchedSymptomID = &matchedID
				item.MatchedSymptomName = match.SymptomName
				item.MatchedText = match.MatchedText
				item.Confidence = match.Confidence
				item.Source = match.Source
				item.SymptomType = match.SymptomType
				item.RuleRole = match.RuleRole
				item.Severity = match.Severity
				item.GrowthStage = match.GrowthStage
				item.IsCoreSymptom = match.IsCoreSymptom
				item.RecommendedForRule = match.RecommendedForRule
				item.DefaultWeight = match.DefaultWeight
				totalMatching += match.Confidence
			} else {
				data.Unknown = append(
					data.Unknown,
					input,
				)
			}
		}

		data.Normalized = append(
			data.Normalized,
			item,
		)
	}

	if len(cleanInputs) > 0 {
		data.Confidence.Normalization =
			totalNormalization / float64(len(cleanInputs))

		data.Confidence.RuleMatching =
			totalMatching / float64(len(cleanInputs))
	}

	return data, nil
}

func mergeExistingSymptomInputs(raw []byte, newInputs []string) []string {
	merged := make([]string, 0, len(newInputs)+4)

	if len(raw) > 0 {
		var previous SymptomSessionData
		if err := json.Unmarshal(raw, &previous); err == nil {
			merged = append(merged, previous.UserInputs...)
		}
	}

	merged = append(merged, newInputs...)
	return cleanSymptomInputs(merged)
}

func needsMoreFieldSymptomsBeforeSeverity(data *SymptomSessionData) bool {
	if data == nil {
		return true
	}

	matchedCount := 0
	hasIdentity := false
	hasDamageOrAnchor := false

	for _, item := range data.Normalized {
		if item.MatchedSymptomID == nil || item.Confidence < 0.65 {
			continue
		}

		matchedCount++

		role := strings.ToLower(strings.TrimSpace(item.RuleRole))
		symptomType := strings.ToLower(strings.TrimSpace(item.SymptomType))

		switch role {
		case "identity":
			hasIdentity = true
		case "damage", "severity_anchor":
			hasDamageOrAnchor = true
		}

		switch symptomType {
		case "identity":
			hasIdentity = true
		case "damage", "severity_anchor":
			hasDamageOrAnchor = true
		}
	}

	// Minimal dua gejala lapangan diperlukan agar rule engine tidak
	// over-diagnosis dari satu gejala umum atau satu gejala identitas hama.
	if matchedCount < 2 {
		return true
	}

	// Kalau yang terpilih hanya gejala identitas, sistem belum bisa
	// menentukan ringan/sedang/berat secara agronomis.
	if hasIdentity && !hasDamageOrAnchor {
		return true
	}

	return !hasDamageOrAnchor
}

func cleanSymptomInputs(inputs []string) []string {

	seen := make(
		map[string]struct{},
	)

	results := make(
		[]string,
		0,
		len(inputs),
	)

	for _, input := range inputs {
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		key := strings.ToLower(input)

		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}

		results = append(
			results,
			input,
		)
	}

	return results
}

func normalizationConfidence(normalizedText string) float64 {

	if strings.TrimSpace(normalizedText) == "" {
		return 0
	}

	return 1
}
