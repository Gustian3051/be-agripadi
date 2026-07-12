package session

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/gustian305/backend/internal/domain"
	"github.com/gustian305/backend/internal/repository"
	"github.com/gustian305/backend/internal/service/expert_system/workflow"
)

type SessionUpdater struct {
	repo repository.ExpertSessionRepository
}

func NewSessionUpdater(
	repo repository.ExpertSessionRepository,
) *SessionUpdater {

	return &SessionUpdater{
		repo: repo,
	}
}


func (s *SessionUpdater) UpdateState(ctx context.Context,session *domain.ExpertSession,next workflow.State) error {

	if session == nil {

		return errors.New(
			"session is required",
		)
	}

	current :=
		workflow.State(
			session.State,
		)

	if err := workflow.ValidateTransition(
		current,
		next,
	); err != nil {

		return err
	}

	session.State = string(next)

	session.UpdatedAt = time.Now()

	if next == workflow.StateCompleted {

		session.IsCompleted = true
	}

	return s.repo.Update(
		ctx,
		session,
	)
}

func (s *SessionUpdater) UpdateDetection(ctx context.Context,session *domain.ExpertSession,label string,confidence float64,model string) error {

	if session == nil {

		return errors.New(
			"session is required",
		)
	}

	if label == "" {

		return errors.New(
			"detected label is required",
		)
	}

	session.DetectedLabel = label

	session.DetectedConfidence = confidence

	session.DetectedModel = model

	session.UpdatedAt = time.Now()

	return s.repo.Update(
		ctx,
		session,
	)
}

func (s *SessionUpdater) UpdateSymptoms(ctx context.Context,session *domain.ExpertSession,symptomsJSON datatypes.JSON) error {

	if session == nil {

		return errors.New(
			"session is required",
		)
	}

	session.Symptoms = symptomsJSON

	session.UpdatedAt = time.Now()

	return s.repo.Update(
		ctx,
		session,
	)
}

func (s *SessionUpdater) UpdateSeverity(ctx context.Context,session *domain.ExpertSession,severity string) error {

	if session == nil {

		return errors.New(
			"session is required",
		)
	}

	if severity == "" {

		return errors.New(
			"severity is required",
		)
	}

	session.Severity = severity

	session.UpdatedAt = time.Now()

	return s.repo.Update(
		ctx,
		session,
	)
}

func (s *SessionUpdater) UpdateGrowthStage(ctx context.Context,session *domain.ExpertSession,stage string) error {

	if session == nil {

		return errors.New(
			"growth stage is required",
		)
	}

	session.GrowthStage = stage

	session.UpdatedAt = time.Now()

	return s.repo.Update(
		ctx,
		session,
	)
}

func (s *SessionUpdater) Complete(ctx context.Context,session *domain.ExpertSession) error {

	if session == nil {

		return errors.New(
			"session is required",
		)
	}

	session.State = string(
		workflow.StateCompleted,
	)

	session.IsCompleted = true

	session.UpdatedAt = time.Now()

	return s.repo.Update(
		ctx,
		session,
	)
}

func (s *SessionUpdater) Reset(ctx context.Context,session *domain.ExpertSession) error {

	if session == nil {

		return errors.New(
			"session is required",
		)
	}

	session.State = string(
		workflow.StateIdle,
	)

	session.DetectedLabel = ""

	session.DetectedConfidence = 0

	session.DetectedModel = ""

	session.Severity = ""

	session.GrowthStage = ""

	session.Symptoms = nil

	session.IsCompleted = false

	session.UpdatedAt = time.Now()

	return s.repo.Update(
		ctx,
		session,
	)
}

func (s *SessionUpdater) Touch(ctx context.Context,sessionID uuid.UUID) error {

	if sessionID == uuid.Nil {

		return errors.New(
			"session id is required",
		)
	}

	session, err := s.repo.GetByID(
		ctx,
		sessionID,
	)

	if err != nil {
		return err
	}

	if session == nil {

		return errors.New(
			"session not found",
		)
	}

	session.UpdatedAt = time.Now()

	return s.repo.Update(
		ctx,
		session,
	)
}