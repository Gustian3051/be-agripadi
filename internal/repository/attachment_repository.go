package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/gustian305/backend/internal/domain"
)

type AttachmentRepository interface {
	Create(ctx context.Context, attachment *domain.Attachment) error
	GetByID(ctx context.Context, attachmentID uuid.UUID) (*domain.Attachment, error)
	ListByMessageID(ctx context.Context, messageID uuid.UUID) ([]domain.Attachment, error)
	Update(ctx context.Context, attachment *domain.Attachment) error
	UpdateMetadata(ctx context.Context, attachmentID uuid.UUID, metadata datatypes.JSON) error
	Delete(ctx context.Context, attachmentID uuid.UUID) error
}

type attachmentRepository struct {
	db *gorm.DB
}

func NewAttachmentRepository(db *gorm.DB) AttachmentRepository {
	return &attachmentRepository{
		db: db,
	}
}

func (r *attachmentRepository) Create(ctx context.Context, attachment *domain.Attachment) error {

	return r.db.WithContext(ctx).
		Create(attachment).
		Error
}

func (r *attachmentRepository) GetByID(ctx context.Context, attachmentID uuid.UUID) (*domain.Attachment, error) {

	var attachment domain.Attachment

	err := r.db.WithContext(ctx).
		Where("id = ?", attachmentID).
		First(&attachment).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &attachment, nil
}

func (r *attachmentRepository) ListByMessageID(ctx context.Context, messageID uuid.UUID) ([]domain.Attachment, error) {

	var attachments []domain.Attachment

	err := r.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		Order("created_at ASC").
		Find(&attachments).
		Error

	if err != nil {
		return nil, err
	}

	return attachments, nil
}

func (r *attachmentRepository) Update(ctx context.Context, attachment *domain.Attachment) error {

	return r.db.WithContext(ctx).
		Save(attachment).
		Error
}

func (r *attachmentRepository) UpdateMetadata(ctx context.Context, attachmentID uuid.UUID, metadata datatypes.JSON) error {

	return r.db.WithContext(ctx).
		Model(&domain.Attachment{}).
		Where("id = ?", attachmentID).
		Update("metadata", metadata).
		Error
}

func (r *attachmentRepository) Delete(ctx context.Context, attachmentID uuid.UUID) error {

	return r.db.WithContext(ctx).
		Delete(&domain.Attachment{}, "id = ?", attachmentID).
		Error
}
