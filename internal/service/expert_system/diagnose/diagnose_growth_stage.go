package diagnose

import (
	"context"

	"github.com/gustian305/backend/internal/service/expert_system/session"
	"github.com/gustian305/backend/internal/service/expert_system/workflow"
)

type GrowthStageService struct {
	sessionLoader  *session.SessionLoader
	sessionUpdater *session.SessionUpdater
}

func NewGrowthStageService(
	sessionLoader *session.SessionLoader,
	sessionUpdater *session.SessionUpdater,
) *GrowthStageService {

	return &GrowthStageService{
		sessionLoader:  sessionLoader,
		sessionUpdater: sessionUpdater,
	}
}

func (s *GrowthStageService) Submit(ctx context.Context, conversationID string, stage string) error {

	session, err :=
		s.sessionLoader.RequireActiveSession(
			ctx,
			conversationID,
		)

	if err != nil {
		return err
	}

	if err := s.sessionUpdater.UpdateGrowthStage(
		ctx,
		session,
		stage,
	); err != nil {

		return err
	}

	return s.sessionUpdater.UpdateState(
		ctx,
		session,
		workflow.StateGenerateResult,
	)
}