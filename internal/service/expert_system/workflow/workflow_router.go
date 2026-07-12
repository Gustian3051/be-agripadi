package workflow

import (
	"context"
	"errors"

	"github.com/gustian305/backend/internal/dto"
)

type Event string

const (
	EventDiagnosisStarted Event = "diagnosis_started"
	EventUserResponseReceived Event = "user_response_received"
	EventGenerateDiagnosis Event = "generate_diagnosis"
)

type RouteRequest struct {
	Session any
	UserInput string
	Event Event
}


type RouteResult struct {
	State dto.DiagnoseFlowState
	Message string
	Actions []dto.ChatAction
}

type RouterService struct {

}

func NewRouterService() *RouterService {
	return &RouterService{}
}

func (s *RouterService) Handle(ctx context.Context, req RouteRequest) (*dto.OrchestratorResponse, error) {

	_ = ctx

	switch req.Event {

	// ========================================================
	// START DIAGNOSIS
	// ========================================================

	case EventDiagnosisStarted:

		return s.handleDiagnosisStarted()

	// ========================================================
	// USER RESPONSE
	// ========================================================

	case EventUserResponseReceived:

		return s.handleUserResponse(
			req,
		)

	// ========================================================
	// GENERATE
	// ========================================================

	case EventGenerateDiagnosis:

		return s.handleGenerateDiagnosis()
	}

	return nil,
		errors.New(
			"workflow event not supported",
		)
}

func (s *RouterService) handleDiagnosisStarted() (*dto.OrchestratorResponse, error) {

	return &dto.OrchestratorResponse{
		State: dto.DiagnoseFlowStateCollectSymptoms,

		Message: "Silakan pilih gejala yang sesuai dengan kondisi tanaman.",

		Actions: []dto.ChatAction{
			{
				Type: dto.ActionSelectSymptom,

				Label: "Pilih Gejala",

				Value: "select_symptom",
			},
		},
	}, nil
}


func (s *RouterService) handleUserResponse(req RouteRequest) (*dto.OrchestratorResponse, error) {

	_ = req

	return &dto.OrchestratorResponse{
		State: dto.DiagnoseFlowStateCollectSeverity,

		Message: "Silakan pilih tingkat keparahan serangan.",

		Actions: []dto.ChatAction{
			{
				Type: dto.ActionSelectSeverity,
				Label: "Pilih Tingkat Keparahan",
				Value: "select_severity",
			},
		},
	}, nil
}

func (s *RouterService) handleGenerateDiagnosis() (*dto.OrchestratorResponse, error) {

	return &dto.OrchestratorResponse{
		State: dto.DiagnoseFlowStateGenerateResult,

		Message: "Sistem sedang menghasilkan rekomendasi terbaik.",

		Actions: []dto.ChatAction{
			{
				Type: dto.ActionGenerateResult,

				Label: "Generate Diagnosis",

				Value: "generate_result",
			},
		},
	}, nil
}