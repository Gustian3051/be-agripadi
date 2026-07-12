package session

import (
	"context"
	"errors"

	"github.com/gustian305/backend/internal/domain"
	"github.com/gustian305/backend/internal/repository"
	"github.com/google/uuid"
)

type SessionLoader struct {
	repo repository.ExpertSessionRepository
}

func NewSessionLoader(
	repo repository.ExpertSessionRepository,
) *SessionLoader {

	return &SessionLoader{
		repo: repo,
	}
}

func (s *SessionLoader) GetByID(ctx context.Context, sessionID uuid.UUID) (*domain.ExpertSession, error) {

	if sessionID == uuid.Nil {

		return nil,
			errors.New(
				"session id is required",
			)
	}

	session, err := s.repo.GetByID(
		ctx,
		sessionID,
	)

	if err != nil {
		return nil, err
	}

	if session == nil {

		return nil,
			errors.New(
				"session not found",
			)
	}

	return session, nil
}

func (s *SessionLoader) GetActiveByConversationID(ctx context.Context,conversationID string) (*domain.ExpertSession, error) {

	if conversationID == "" {

		return nil,
			errors.New(
				"conversation id is required",
			)
	}

	session, err := s.repo.GetActiveByConversationID(
		ctx,
		conversationID,
	)

	if err != nil {
		return nil, err
	}

	if session == nil {

		return nil,
			errors.New(
				"active session not found",
			)
	}

	return session, nil
}

func (s *SessionLoader) RequireActiveSession(ctx context.Context, conversationID string) (*domain.ExpertSession, error) {

	session, err := s.GetActiveByConversationID(ctx, conversationID)

	if err != nil {
		return nil, err
	}

	if session.IsCompleted {

		return nil,
			errors.New(
				"diagnosis session already completed",
			)
	}

	return session, nil
}
