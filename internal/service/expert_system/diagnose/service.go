package diagnose

import (
	"context"

	"github.com/gustian305/backend/internal/dto"
)

//
// ============================================================
// SERVICE
// ============================================================
//

type DiagnoseService struct {
	startService       *StartService
	imageService       *ImageService
	symptomService     *SymptomService
	severityService    *SeverityService
	growthStageService *GrowthStageService
	finalizeService    *FinalizeService
}

func NewDiagnoseService(
	startService *StartService,
	imageService *ImageService,
	symptomService *SymptomService,
	severityService *SeverityService,
	growthStageService *GrowthStageService,
	finalizeService *FinalizeService,
) *DiagnoseService {

	return &DiagnoseService{
		startService:       startService,
		imageService:       imageService,
		symptomService:     symptomService,
		severityService:    severityService,
		growthStageService: growthStageService,
		finalizeService:    finalizeService,
	}
}

func (s *DiagnoseService) Start(ctx context.Context, conversationID string, detections []dto.DetectionCandidateRequest) error {
	return s.startService.Execute(ctx, conversationID, detections)
}

func (s *DiagnoseService) SubmitSymptoms(ctx context.Context, conversationID string, symptoms []string) error {
	return s.symptomService.Submit(ctx, conversationID, symptoms)
}

func (s *DiagnoseService) SubmitSeverity(ctx context.Context, conversationID string, severity string) error {
	return s.severityService.Submit(ctx, conversationID, severity)
}

func (s *DiagnoseService) SubmitGrowthStage(ctx context.Context, conversationID string, stage string) error {
	return s.growthStageService.Submit(ctx, conversationID, stage)
}

func (s *DiagnoseService) Finalize(ctx context.Context, conversationID string) error {
	return s.finalizeService.Execute(ctx, conversationID)
}
