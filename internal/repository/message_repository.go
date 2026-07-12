package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/domain"
	"gorm.io/gorm"
)

// ============================================================
// MESSAGE REPOSITORY
// ============================================================

type MessageRepository interface {
	Create(ctx context.Context, message *domain.Message) error
	GetByID(ctx context.Context, messageID uuid.UUID) (*domain.Message, error)
	ListByConversationID(ctx context.Context, conversationID uuid.UUID) ([]domain.Message, error)
	Update(ctx context.Context, message *domain.Message) error
	UpdateStatus(ctx context.Context, messageID uuid.UUID, status domain.MessageStatus) error
	UpdateStreaming(ctx context.Context, messageID uuid.UUID, content string, isStreaming bool, tokenCount int) error
	Delete(ctx context.Context, messageID uuid.UUID) error
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{
		db: db,
	}
}

func (r *messageRepository) Create(ctx context.Context, message *domain.Message) error {

	return r.db.WithContext(ctx).
		Create(message).
		Error
}

func (r *messageRepository) GetByID(ctx context.Context, messageID uuid.UUID) (*domain.Message, error) {

	var message domain.Message

	err := r.db.WithContext(ctx).
		Preload("Attachments").
		Where("id = ?", messageID).
		First(&message).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &message, nil
}

func (r *messageRepository) ListByConversationID(ctx context.Context, conversationID uuid.UUID) ([]domain.Message, error) {

	var messages []domain.Message

	err := r.db.WithContext(ctx).
		Preload("Attachments").
		Where(
			"conversation_id = ?",
			conversationID,
		).
		Order("created_at ASC").
		Find(&messages).
		Error

	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *messageRepository) Update(ctx context.Context, message *domain.Message) error {

	return r.db.WithContext(ctx).
		Save(message).
		Error
}

func (r *messageRepository) UpdateStatus(ctx context.Context, messageID uuid.UUID, status domain.MessageStatus) error {

	return r.db.WithContext(ctx).
		Model(&domain.Message{}).
		Where("id = ?", messageID).
		Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now(),
		}).
		Error
}

func (r *messageRepository) UpdateStreaming(ctx context.Context, messageID uuid.UUID, content string, isStreaming bool, tokenCount int) error {

	return r.db.WithContext(ctx).
		Model(&domain.Message{}).
		Where("id = ?", messageID).
		Updates(map[string]any{
			"content":      content,
			"is_streaming": isStreaming,
			"token_count":  tokenCount,
			"updated_at":   time.Now(),
		}).
		Error
}

func (r *messageRepository) Delete(ctx context.Context, messageID uuid.UUID) error {

	return r.db.WithContext(ctx).
		Delete(
			&domain.Message{},
			"id = ?",
			messageID,
		).
		Error
}
