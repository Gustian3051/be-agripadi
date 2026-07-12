package session

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/gustian305/backend/internal/domain"
	"github.com/gustian305/backend/internal/repository"
	"github.com/gustian305/backend/internal/service/expert_system/workflow"
)

type SessionService struct {
	repo repository.ExpertSessionRepository
}

func NewSessionService(
	repo repository.ExpertSessionRepository,
) *SessionService {

	return &SessionService{
		repo: repo,
	}
}

func (s *SessionService) Create(ctx context.Context, conversationID string) (*domain.ExpertSession, error) {

	if conversationID == "" {

		return nil,
			errors.New(
				"conversation id is required",
			)
	}

	now := time.Now()

	conversationUUID, err := uuid.Parse(conversationID)

	if err != nil {
		return nil,
			err
	}

	session := &domain.ExpertSession{
		ID: uuid.New(),

		ConversationID: conversationUUID,

		State: string(
			workflow.StateIdle,
		),

		IsCompleted: false,

		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(
		ctx,
		session,
	); err != nil {

		return nil, err
	}

	return session, nil
}


func (s *SessionService) GetActiveByConversationID(ctx context.Context, conversationID uuid.UUID) (*domain.ExpertSession, error) {

	if conversationID == uuid.Nil {

		return nil,
			errors.New(
				"conversation id is required",
			)
	}

	return s.repo.GetActiveByConversationID(
		ctx,
		conversationID.String(),
	)
}

func (s *SessionService) UpdateState(ctx context.Context, session *domain.ExpertSession, next workflow.State) error {

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

func (s *SessionService) Complete(ctx context.Context, session *domain.ExpertSession) error {

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
