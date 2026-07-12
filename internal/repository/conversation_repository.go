package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gustian305/backend/internal/domain"
)

type ConversationRepository interface {
	Create(ctx context.Context, userID uuid.UUID, conversation *domain.Conversation) error
	GetByID(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) (*domain.Conversation, error)
	GetAttachmentsByConversationID(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) ([]domain.Attachment, error)
	GetAttachmentsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Attachment, error)
	Update(ctx context.Context, userID uuid.UUID, conversation *domain.Conversation) error
	UpdateTitle(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, title string) error
	UpdateState(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, mode string, state string) error
	UpdateLastMessageAt(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, lastMessageAt time.Time) error
	ListByPaginated(ctx context.Context, userID uuid.UUID, page int, limit int) ([]domain.Conversation, int, error)
	Delete(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error
	DeleteAllByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

type conversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) *conversationRepository {
	return &conversationRepository{
		db: db,
	}
}

func (r *conversationRepository) Create(ctx context.Context, userID uuid.UUID, conversation *domain.Conversation) error {

	conversation.UserID = userID

	return r.db.WithContext(ctx).
		Create(conversation).
		Error
}

func (r *conversationRepository) GetByID(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) (*domain.Conversation, error) {

	var conversation domain.Conversation

	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", conversationID, userID).
		First(&conversation).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &conversation, nil
}

func (r *conversationRepository) GetAttachmentsByConversationID(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) ([]domain.Attachment, error) {

	var attachments []domain.Attachment

	err := r.db.
		WithContext(ctx).
		Model(&domain.Attachment{}).
		Joins("JOIN messages ON messages.id = attachments.message_id").
		Joins("JOIN conversations ON conversations.id = messages.conversation_id").
		Where(
			"messages.conversation_id = ? AND conversations.user_id = ?",
			conversationID,
			userID,
		).
		Find(&attachments).
		Error

	return attachments, err
}

func (r *conversationRepository) GetAttachmentsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Attachment, error) {

	var attachments []domain.Attachment

	err := r.db.
		WithContext(ctx).
		Model(&domain.Attachment{}).
		Joins("JOIN messages ON messages.id = attachments.message_id").
		Joins("JOIN conversations ON conversations.id = messages.conversation_id").
		Where("conversations.user_id = ?", userID).
		Find(&attachments).
		Error

	return attachments, err
}

func (r *conversationRepository) Update(ctx context.Context, userID uuid.UUID, conversation *domain.Conversation) error {

	result := r.db.WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("id = ? AND user_id = ?", conversation.ID, userID).
		Updates(map[string]any{
			"title":           conversation.Title,
			"mode":            conversation.Mode,
			"state":           conversation.State,
			"last_message":    conversation.LastMessage,
			"last_message_at": conversation.LastMessageAt,
			"updated_at":      conversation.UpdatedAt,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *conversationRepository) UpdateTitle(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, title string) error {

	result := r.db.WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]any{
			"title":      title,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *conversationRepository) UpdateState(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, mode string, state string) error {

	result := r.db.WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]any{
			"mode":       mode,
			"state":      state,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *conversationRepository) UpdateLastMessageAt(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, lastMessageAt time.Time) error {

	result := r.db.WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]any{
			"last_message_at": lastMessageAt,
			"updated_at":      time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *conversationRepository) ListByPaginated(ctx context.Context, userID uuid.UUID, page int, limit int) ([]domain.Conversation, int, error) {

	var conversations []domain.Conversation
	var total int64

	if page <= 0 {
		page = 1
	}

	if limit <= 0 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	if err := r.db.WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&conversations).Error; err != nil {
		return nil, 0, err
	}

	return conversations, int(total), nil
}

func (r *conversationRepository) Delete(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error {

	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", conversationID, userID).
		Delete(&domain.Conversation{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *conversationRepository) DeleteAllByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {

	var deletedCount int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		conversationSubQuery := tx.
			Model(&domain.Conversation{}).
			Select("id").
			Where("user_id = ?", userID)

		messageSubQuery := tx.
			Model(&domain.Message{}).
			Select("id").
			Where("conversation_id IN (?)", conversationSubQuery)

		if err := tx.
			Where("conversation_id IN (?)", conversationSubQuery).
			Delete(&domain.ExpertSession{}).
			Error; err != nil {
			return err
		}

		if err := tx.
			Where("message_id IN (?)", messageSubQuery).
			Delete(&domain.Attachment{}).
			Error; err != nil {
			return err
		}

		if err := tx.
			Where("conversation_id IN (?)", conversationSubQuery).
			Delete(&domain.Message{}).
			Error; err != nil {
			return err
		}

		result := tx.
			Where("user_id = ?", userID).
			Delete(&domain.Conversation{})

		if result.Error != nil {
			return result.Error
		}

		deletedCount = result.RowsAffected

		return nil
	})

	return deletedCount, err
}
