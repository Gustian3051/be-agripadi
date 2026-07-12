package app

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/gustian305/backend/internal/domain"
	"github.com/gustian305/backend/internal/repository"
)

type AttachmentAppService struct {
	repository repository.AttachmentRepository
}

func NewAttachmentAppService(repository repository.AttachmentRepository) *AttachmentAppService {
	return &AttachmentAppService{
		repository: repository,
	}
}

const (
	MaxImageSize = 10 * 1024 * 1024 // 10 MB
)

var AllowedImageMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
}

func (s *AttachmentAppService) CreateImageAttachment(ctx context.Context, messageID uuid.UUID, imageURL string) (*domain.Attachment, error) {

	if messageID == uuid.Nil {

		return nil, errors.New(
			"invalid message id",
		)
	}

	imageURL = strings.TrimSpace(
		imageURL,
	)

	if imageURL == "" {

		return nil, errors.New(
			"image url is required",
		)
	}

	now := time.Now()

	attachment := &domain.Attachment{
		ID:        uuid.New(),
		MessageID: messageID,
		Type:      domain.AttachmentTypeImage,
		URL:       imageURL,
		MimeType:  DetectMimeType(imageURL),
		Size:      0,
		Metadata:  datatypes.JSON([]byte(`{}`)),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.ValidateAttachment(attachment); err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, attachment); err != nil {

		return nil, err
	}

	return attachment, nil
}

func (s *AttachmentAppService) ValidateAttachment(attachment *domain.Attachment) error {

	if attachment == nil {
		return errors.New(
			"attachment is nil",
		)
	}

	if attachment.MessageID == uuid.Nil {
		return errors.New(
			"invalid message id",
		)
	}

	if strings.TrimSpace(attachment.URL) == "" {
		return errors.New(
			"attachment url is required",
		)
	}

	if !AllowedImageMimeTypes[attachment.MimeType] {
		return errors.New(
			"unsupported image mime type",
		)
	}

	if attachment.Size > MaxImageSize {
		return errors.New(
			"image size exceeds limit",
		)
	}

	return nil
}


func (s *AttachmentAppService) GetAttachment(ctx context.Context, attachmentID uuid.UUID) (*domain.Attachment, error) {
	if attachmentID == uuid.Nil {
		return nil, errors.New(
			"invalid attachment id",
		)
	}

	return s.repository.GetByID(ctx, attachmentID)
}

func (s *AttachmentAppService) ListAttachments(ctx context.Context, messageID uuid.UUID) ([]domain.Attachment, error) {
	if messageID == uuid.Nil {

		return nil, errors.New(
			"invalid message id",
		)
	}

	return s.repository.ListByMessageID(
		ctx,
		messageID,
	)
}

func (s *AttachmentAppService) DeleteAttachment(ctx context.Context, attachmentID uuid.UUID) error {

	if attachmentID == uuid.Nil {

		return errors.New(
			"invalid attachment id",
		)
	}

	return s.repository.Delete(
		ctx,
		attachmentID,
	)
}

func (s *AttachmentAppService) UpdateMetadata(ctx context.Context, attachmentID uuid.UUID, metadata datatypes.JSON) error {

	if attachmentID == uuid.Nil {

		return errors.New(
			"invalid attachment id",
		)
	}

	attachment, err := s.repository.GetByID(
		ctx,
		attachmentID,
	)

	if err != nil {
		return err
	}

	attachment.Metadata = metadata

	attachment.UpdatedAt = time.Now()

	return s.repository.Update(
		ctx,
		attachment,
	)
}

func DetectMimeType(url string) string {

	ext := strings.ToLower(filepath.Ext(url))

	switch ext {

	case ".jpg":
		return "image/jpg"

	case ".jpeg":
		return "image/jpeg"

	case ".png":
		return "image/png"

	case ".webp":
		return "image/webp"

	default:
		return "application/octet-stream"
	}
}
