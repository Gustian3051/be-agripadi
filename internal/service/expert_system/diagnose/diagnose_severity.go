package diagnose

import (
	"context"

	"github.com/gustian305/backend/internal/service/expert_system/session"
	"github.com/gustian305/backend/internal/service/expert_system/workflow"
)

type SeverityService struct {
	sessionLoader  *session.SessionLoader
	sessionUpdater *session.SessionUpdater
}

func NewSeverityService(
	sessionLoader *session.SessionLoader,
	sessionUpdater *session.SessionUpdater,
) *SeverityService {

	return &SeverityService{
		sessionLoader:  sessionLoader,
		sessionUpdater: sessionUpdater,
	}
}

func (s *SeverityService) Submit(ctx context.Context, conversationID string, severity string) error {

	session, err :=
		s.sessionLoader.RequireActiveSession(
			ctx,
			conversationID,
		)

	if err != nil {
		return err
	}

	if err := s.sessionUpdater.UpdateSeverity(
		ctx,
		session,
		severity,
	); err != nil {

		return err
	}

	return s.sessionUpdater.UpdateState(
		ctx,
		session,
		workflow.StateCollectGrowthStage,
	)
}
