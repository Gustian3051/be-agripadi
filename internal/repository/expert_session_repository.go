package repository

import (
	"context"
	"errors"

	"github.com/gustian305/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ExpertSessionRepository interface {
	Create(ctx context.Context, session *domain.ExpertSession) error
	Update(ctx context.Context, session *domain.ExpertSession) error
	GetByID(ctx context.Context, sessionID uuid.UUID) (*domain.ExpertSession, error)
	GetActiveByConversationID(ctx context.Context, conversationID string) (*domain.ExpertSession, error)
	FindByConversationID(ctx context.Context, conversationID uuid.UUID) ([]domain.ExpertSession, error)
	Delete(ctx context.Context, sessionID uuid.UUID) error
}

type expertSessionRepository struct {
	db *gorm.DB
}

func NewExpertSessionRepository(db *gorm.DB) ExpertSessionRepository {
	return &expertSessionRepository{
		db: db,
	}
}

func (r *expertSessionRepository) Create(ctx context.Context, session *domain.ExpertSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *expertSessionRepository) Update(ctx context.Context, session *domain.ExpertSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

func (r *expertSessionRepository) GetByID(ctx context.Context, sessionID uuid.UUID) (*domain.ExpertSession, error) {
	var session domain.ExpertSession

	err := r.db.WithContext(ctx).
		Where("id = ?", sessionID).
		First(&session).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &session, nil
}

func (r *expertSessionRepository) GetActiveByConversationID(ctx context.Context, conversationID string) (*domain.ExpertSession, error) {
	var session domain.ExpertSession

	err := r.db.WithContext(ctx).
		Where(
			"conversation_id = ? AND is_completed = ?",
			conversationID,
			false,
		).
		Order("created_at DESC").
		First(&session).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &session, nil
}

func (r *expertSessionRepository) FindByConversationID(ctx context.Context, conversationID uuid.UUID) ([]domain.ExpertSession, error) {

	var sessions []domain.ExpertSession

	err := r.db.WithContext(ctx).
		Where(
			"conversation_id = ?",
			conversationID,
		).
		Order("created_at DESC").
		Find(&sessions).
		Error

	if err != nil {
		return nil, err
	}

	return sessions, nil
}

func (r *expertSessionRepository) Delete(ctx context.Context, sessionID uuid.UUID) error {

	return r.db.WithContext(ctx).
		Delete(
			&domain.ExpertSession{},
			"id = ?",
			sessionID,
		).
		Error
}
