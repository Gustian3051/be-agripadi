package dto

import (
	"time"

	"github.com/google/uuid"
)

//
// ============================================================
// SHARED ENUM
// ============================================================
//

type ConversationMode string

const (
	ConversationModeConsultation ConversationMode = "consultation"
	ConversationModeDiagnose     ConversationMode = "diagnose"
)

type DiagnoseFlowState string

const (
	DiagnoseFlowStateIdle               DiagnoseFlowState = "idle"
	DiagnoseFlowStateImageAnalyzed      DiagnoseFlowState = "image_analyzed"
	DiagnoseFlowStateCollectSymptoms    DiagnoseFlowState = "collect_symptoms"
	DiagnoseFlowStateNormalizeSymptoms  DiagnoseFlowState = "normalize_symptoms"
	DiagnoseFlowStateCollectSeverity    DiagnoseFlowState = "collect_severity"
	DiagnoseFlowStateCollectGrowthStage DiagnoseFlowState = "collect_growth_stage"
	DiagnoseFlowStateGenerateResult     DiagnoseFlowState = "generate_result"
	DiagnoseFlowStateCompleted          DiagnoseFlowState = "completed"
)

//
// ============================================================
// ORCHESTRATOR RESPONSE
// ============================================================
//

type OrchestratorResponse struct {
	SessionID uuid.UUID            `json:"session_id"`
	Mode      ConversationMode     `json:"mode"`
	State     DiagnoseFlowState    `json:"state"`
	Message   string               `json:"message"`
	Actions   []ChatAction         `json:"actions,omitempty"`
	Payload   *OrchestratorPayload `json:"payload,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
}

type OrchestratorPayload struct {
	Diagnose *DiagnosePayload `json:"diagnose,omitempty"`
}

//
// ============================================================
// CHAT ACTION
// ============================================================
//

type ActionType string

const (
	ActionUploadImage     ActionType = "upload_image"
	ActionSelectSymptom   ActionType = "select_symptom"
	ActionSelectSeverity  ActionType = "select_severity"
	ActionSelectGrowth    ActionType = "select_growth_stage"
	ActionGenerateResult  ActionType = "generate_result"
	ActionContinueConsult ActionType = "continue_consultation"
)

type ChatAction struct {
	Type  ActionType `json:"type"`
	Label string     `json:"label"`
	Value string     `json:"value"`
}

//
// ============================================================
// DIAGNOSE PAYLOAD
// ============================================================
//

type DiagnosePayload struct {
	Session DiagnoseSessionResponse  `json:"session"`
	Result  *DiagnosisResultResponse `json:"result,omitempty"`
}

//
// ============================================================
// DIAGNOSE SESSION RESPONSE
// ============================================================
//

type DiagnoseSessionResponse struct {
	SessionID      uuid.UUID                `json:"session_id"`
	ConversationID uuid.UUID                `json:"conversation_id"`
	Mode           ConversationMode         `json:"mode"`
	State          DiagnoseFlowState        `json:"state"`
	Image          *DiagnoseImageResponse   `json:"image,omitempty"`
	Symptoms       *SymptomAnalysisResponse `json:"symptoms,omitempty"`
	Severity       *SeverityReference       `json:"severity,omitempty"`
	GrowthStage    *GrowthStageReference    `json:"growth_stage,omitempty"`
	Metadata       DiagnoseSessionMetadata  `json:"metadata"`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
}

type DiagnoseSessionMetadata struct {
	RetryCount    int  `json:"retry_count,omitempty"`
	IsCompleted   bool `json:"is_completed"`
	IsFallbackLLM bool `json:"is_fallback_llm"`
}

//
// ============================================================
// IMAGE ANALYSIS
// ============================================================
//

type DiagnoseImageResponse struct {
	UploadedImageURL string                   `json:"uploaded_image_url,omitempty"`
	PrimaryDetection *PestDetectionCandidate  `json:"primary_detection,omitempty"`
	Candidates       []PestDetectionCandidate `json:"candidates,omitempty"`
	Confidence       DetectionConfidence      `json:"confidence"`
}

type PestDetectionCandidate struct {
	Pest       PestReference `json:"pest"`
	Label      string        `json:"label"`
	Model      string        `json:"model"`
	Confidence float64       `json:"confidence"`
}

type DetectionConfidence struct {
	CNNModel            float64 `json:"cnn_model"`
	FinalClassification float64 `json:"final_classification"`
}

//
// ============================================================
// SYMPTOM ANALYSIS
// ============================================================
//

type SymptomAnalysisResponse struct {
	UserInputs []string                    `json:"user_inputs,omitempty"`
	Selected   []SymptomReference          `json:"selected,omitempty"`
	Normalized []NormalizedSymptomResponse `json:"normalized,omitempty"`
	Unknown    []string                    `json:"unknown,omitempty"`
	Confidence SymptomConfidence           `json:"confidence"`
}

type NormalizedSymptomResponse struct {
	InputText          string    `json:"input_text"`
	MatchedSymptomID   uuid.UUID `json:"matched_symptom_id"`
	MatchedSymptomName string    `json:"matched_symptom_name"`
	Confidence         float64   `json:"confidence"`
	Source             string    `json:"source"`
}

type SymptomConfidence struct {
	Normalization float64 `json:"normalization"`
	RuleMatching  float64 `json:"rule_matching"`
}

//
// ============================================================
// DIAGNOSIS RESULT
// ============================================================
//

type DiagnosisResultResponse struct {
	MatchedRuleID    *uuid.UUID                        `json:"matched_rule_id,omitempty"`
	Pest             PestReference                     `json:"pest"`
	Severity         SeverityReference                 `json:"severity"`
	GrowthStage      GrowthStageReference              `json:"growth_stage"`
	Confidence       DiagnosisConfidence               `json:"confidence"`
	MatchedSymptoms  []SymptomReference                `json:"matched_symptoms"`
	Recommendations  []PesticideRecommendationResponse `json:"recommendations"`
	Deterministic    DeterministicDiagnosisResponse    `json:"deterministic"`
	LLM              *LLMExplanationResponse           `json:"llm,omitempty"`
	ExpertValidation *ExpertValidationResponse         `json:"expert_validation,omitempty"`
}

type DiagnosisConfidence struct {
	CNN                  float64 `json:"cnn"`
	SymptomNormalization float64 `json:"symptom_normalization"`
	RuleEngine           float64 `json:"rule_engine"`
	FinalScore           float64 `json:"final_score"`
}

//
// ============================================================
// DETERMINISTIC RESULT
// ============================================================
//

type DeterministicDiagnosisResponse struct {
	Summary         string   `json:"summary"`
	MatchedRuleCode *string  `json:"matched_rule_code,omitempty"`
	Reasoning       []string `json:"reasoning,omitempty"`
}

//
// ============================================================
// EXPERT VALIDATION
// ============================================================
//

// ExpertValidationResponse disiapkan agar hasil diagnosis dapat dinilai
// ulang oleh pakar pertanian tanpa harus membaca payload teknis mentah.
type ExpertValidationResponse struct {
	ReadinessStatus       string                              `json:"readiness_status"`
	ReadinessLabel        string                              `json:"readiness_label"`
	ValidationTarget      string                              `json:"validation_target"`
	OverallNote           string                              `json:"overall_note"`
	Evidence              []ExpertValidationEvidenceResponse  `json:"evidence,omitempty"`
	Checklist             []ExpertValidationChecklistResponse `json:"checklist,omitempty"`
	Warnings              []string                            `json:"warnings,omitempty"`
	ExpertDecisionOptions []string                            `json:"expert_decision_options,omitempty"`
	RatingScale           []string                            `json:"rating_scale,omitempty"`
	FieldsToReview        []string                            `json:"fields_to_review,omitempty"`
}

type ExpertValidationEvidenceResponse struct {
	Component   string `json:"component"`
	SystemValue string `json:"system_value"`
	Source      string `json:"source"`
	Note        string `json:"note,omitempty"`
}

type ExpertValidationChecklistResponse struct {
	Aspect              string   `json:"aspect"`
	Question            string   `json:"question"`
	SystemAnswer        string   `json:"system_answer"`
	ExpectedExpertInput []string `json:"expected_expert_input,omitempty"`
}

//
// ============================================================
// LLM EXPLANATION
// ============================================================
//

type LLMExplanationResponse struct {
	Model      string   `json:"model"`
	Summary    string   `json:"summary"`
	Reasoning  []string `json:"reasoning,omitempty"`
	Prevention []string `json:"prevention,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

//
// ============================================================
// PESTICIDE RECOMMENDATION
// ============================================================
//

type PesticideRecommendationResponse struct {
	PesticideID              uuid.UUID                           `json:"pesticide_id"`
	ProductName              string                              `json:"product_name"`
	DisplayName              string                              `json:"display_name"`
	RecommendationBasis      string                              `json:"recommendation_basis"`
	TradeNameHidden          bool                                `json:"trade_name_hidden"`
	TradeNamePolicy          string                              `json:"trade_name_policy"`
	PesticideType            string                              `json:"pesticide_type"`
	Formulation              string                              `json:"formulation"`
	Ingredients              []PesticideIngredientResponse       `json:"ingredients,omitempty"`
	Dosage                   PesticideDosageResponse             `json:"dosage"`
	ApplicationTiming        *PesticideApplicationTimingResponse `json:"application_timing,omitempty"`
	Safety                   PesticideSafetyResponse             `json:"safety"`
	ScoreBreakdown           PesticideScoreBreakdownResponse     `json:"score_breakdown"`
	MatchScore               float64                             `json:"match_score"`
	RecommendationConfidence float64                             `json:"recommendation_confidence"`
	Reason                   string                              `json:"reason"`
}

// PesticideApplicationTimingResponse berisi waktu aplikasi yang diambil dari
// tabel pesticide_application_timings. Data ini dipisahkan dari LLM agar waktu
// aplikasi tidak dibuat bebas oleh model bahasa.
type PesticideApplicationTimingResponse struct {
	ApplicationContext     string `json:"application_context,omitempty"`
	FieldDiagnosisAllowed  bool   `json:"field_diagnosis_allowed"`
	ApplicationMethod      string `json:"application_method"`
	ApplicationTarget      string `json:"application_target,omitempty"`
	ApplicationTrigger     string `json:"application_trigger,omitempty"`
	TimingWindow           string `json:"timing_window"`
	TimingInstruction      string `json:"timing_instruction"`
	WaterManagement        string `json:"water_management,omitempty"`
	WeatherCondition       string `json:"weather_condition,omitempty"`
	PreharvestIntervalNote string `json:"preharvest_interval_note,omitempty"`
	DisplayWarning         string `json:"display_warning,omitempty"`
	ReferenceTitle         string `json:"reference_title,omitempty"`
	ReferenceInstitution   string `json:"reference_institution,omitempty"`
	ReferenceYear          string `json:"reference_year,omitempty"`
	ReferenceURL           string `json:"reference_url,omitempty"`
	ReferenceNote          string `json:"reference_note,omitempty"`
}

type PesticideScoreBreakdownResponse struct {
	TargetFit     float64 `json:"target_fit"`
	IngredientFit float64 `json:"ingredient_fit"`
	DoseQuality   float64 `json:"dose_quality"`
	SeverityFit   float64 `json:"severity_fit"`
	Safety        float64 `json:"safety"`
	GrowthStage   float64 `json:"growth_stage"`
	DataQuality   float64 `json:"data_quality"`
	WeightedScore float64 `json:"weighted_score"`
}

type PesticideSafetyResponse struct {
	ToxicityClass      *string  `json:"toxicity_class,omitempty"`
	ReentryInterval    *string  `json:"reentry_interval,omitempty"`
	PreHarvestInterval *string  `json:"pre_harvest_interval,omitempty"`
	MaxApplication     *string  `json:"max_application,omitempty"`
	MixingWarning      []string `json:"mixing_warning,omitempty"`
}

//
// ============================================================
// INGREDIENT
// ============================================================
//

type PesticideIngredientResponse struct {
	Name             string `json:"name"`
	ConcentrationRaw string `json:"concentration_raw"`
}

//
// ============================================================
// DOSAGE
// ============================================================
//

type PesticideDosageResponse struct {
	DoseRaw  string   `json:"dose_raw"`
	MinDose  *float64 `json:"min_dose,omitempty"`
	MaxDose  *float64 `json:"max_dose,omitempty"`
	DoseUnit string   `json:"dose_unit"`
}

//
// ============================================================
// REFERENCE DTO
// ============================================================
//

type PestReference struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name,omitempty"`
	LabelName string    `json:"label_name"`
}

type SeverityReference struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type GrowthStageReference struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type SymptomReference struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
