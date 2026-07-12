package diagnose

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/dto"
	"github.com/gustian305/backend/internal/service/expert_system/session"
	"github.com/gustian305/backend/internal/service/expert_system/workflow"
)

type StartService struct {
	sessionUpdater *session.SessionUpdater
	sessionService *session.SessionService
}

func NewStartService(
	sessionUpdater *session.SessionUpdater,
	sessionService *session.SessionService,
) *StartService {

	return &StartService{
		sessionUpdater: sessionUpdater,
		sessionService: sessionService,
	}
}

func (s *StartService) Execute(ctx context.Context, conversationID string, detections []dto.DetectionCandidateRequest) error {

	if len(detections) == 0 {
		return errors.New("detections are required")
	}

	conversationUUID, err := uuid.Parse(conversationID)

	if err != nil {
		return err
	}

	activeSession, err :=
		s.sessionService.GetActiveByConversationID(
			ctx,
			conversationUUID,
		)

	if err != nil {
		return err
	}

	if activeSession != nil {

		if err :=
			s.sessionService.Complete(
				ctx,
				activeSession,
			); err != nil {

			return err
		}
	}

	normalizedDetections :=
		make(
			[]dto.DetectionCandidateRequest,
			0,
			len(detections),
		)

	for _, detection := range detections {

		detection.Label =
			sanitizeDetectionLabel(
				detection.Label,
			)

		detection.Model =
			strings.TrimSpace(
				detection.Model,
			)

		if detection.Label == "" {
			continue
		}

		normalizedDetections =
			append(
				normalizedDetections,
				detection,
			)
	}

	if len(normalizedDetections) == 0 {
		return errors.New("detected label is required")
	}

	primary := normalizedDetections[0]

	for _, detection := range normalizedDetections[1:] {

		if detection.Confidence >
			primary.Confidence {

			primary = detection
		}
	}

	session, err :=
		s.sessionService.Create(
			ctx,
			conversationID,
		)

	if err != nil {
		return err
	}

	if err :=
		s.sessionUpdater.UpdateDetection(
			ctx,
			session,
			primary.Label,
			primary.Confidence,
			primary.Model,
		); err != nil {

		return err
	}

	// =====================================================
	// NO PEST SHORT-CIRCUIT
	// =====================================================

	if isNoPestLabel(
		primary.Label,
	) {

		return s.sessionUpdater.UpdateState(
			ctx,
			session,
			workflow.StateImageAnalyzed,
		)
	}

	// =====================================================
	// NORMAL FLOW
	// =====================================================

	if err := s.sessionUpdater.UpdateState(
		ctx,
		session,
		workflow.StateImageAnalyzed,
	); err != nil {

		return err
	}

	return s.sessionUpdater.UpdateState(
		ctx,
		session,
		workflow.StateCollectSymptoms,
	)
}

func sanitizeDetectionLabel(label string) string {
	label = strings.TrimSpace(label)
	label = strings.ToLower(label)
	label = strings.ReplaceAll(label, " ", "_")
	label = strings.ReplaceAll(label, "-", "_")

	for strings.Contains(label, "__") {
		label = strings.ReplaceAll(label, "__", "_")
	}

	return strings.Trim(label, "_")
}

func isNoPestLabel(label string) bool {
	switch sanitizeDetectionLabel(label) {
	case "tidak_ada_hama",
		"tanpa_hama",
		"no_pest",
		"no_pests",
		"none":
		return true
	default:
		return false
	}
}
