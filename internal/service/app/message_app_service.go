package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/domain"
	"github.com/gustian305/backend/internal/dto"
	"github.com/gustian305/backend/internal/repository"
	"github.com/gustian305/backend/logger"
)

type MessageAppService struct {
	messageRepo      repository.MessageRepository
	conversationRepo repository.ConversationRepository
}

func NewMessageAppService(
	messageRepo repository.MessageRepository,
	conversationRepo repository.ConversationRepository,
) *MessageAppService {

	return &MessageAppService{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
	}
}

func (s *MessageAppService) CreateUserMessage(ctx context.Context, userID uuid.UUID, req dto.CreateMessageRequest) (*dto.MessageResponse, error) {
	startedAt := time.Now()
	operation := "CreateUserMessage"

	logger.Request(
		"app.message",
		operation,
		slog.String("conversation_id", req.ConversationID.String()),
		slog.String("message_type", string(req.Type)),
		slog.Int("content_length", len(req.Content)),
		slog.Int("attachment_count", len(req.Attachments)),
		slog.Int("detection_count", len(req.Detections)),
	)

	// ========================================================
	// VALIDATION
	// ========================================================

	if userID == uuid.Nil {
		err := errors.New("user id is required")
		logger.Failure("app.message", operation, startedAt, err)
		return nil, err
	}

	if req.ConversationID == uuid.Nil {

		err := errors.New(
			"conversation id is required",
		)
		logger.Failure("app.message", operation, startedAt, err)
		return nil, err
	}

	// ========================================================
	// LOAD CONVERSATION
	// ========================================================

	conversation, err :=
		s.conversationRepo.GetByID(
			ctx,
			userID,
			req.ConversationID,
		)

	if err != nil {
		logger.Failure("app.message", operation, startedAt, err)
		return nil, err
	}

	if conversation == nil {

		err := errors.New(
			"conversation not found",
		)
		logger.Failure("app.message", operation, startedAt, err)
		return nil, err
	}

	now := time.Now()

	// ========================================================
	// CREATE MESSAGE
	// ========================================================

	message := &domain.Message{
		ID:             uuid.New(),
		ConversationID: req.ConversationID,

		Role: domain.MessageRoleUser,

		Type: req.Type,

		Status: domain.MessageStatusSent,

		Content: req.Content,

		IsStreaming: false,

		CreatedAt: now,
		UpdatedAt: now,
	}

	// ========================================================
	// ATTACHMENTS
	// ========================================================

	if len(req.Attachments) > 0 {

		message.Attachments =
			make(
				[]domain.Attachment,
				0,
				len(req.Attachments),
			)

		for _, item := range req.Attachments {
			if err := validateAttachmentRequest(item); err != nil {
				logger.Failure("app.message", operation, startedAt, err)
				return nil, err
			}

			attachment := domain.Attachment{
				ID:        uuid.New(),
				MessageID: message.ID,

				Type: domain.AttachmentType(
					item.Type,
				),

				URL: item.URL,

				MimeType: item.MimeType,

				Size: item.Size,

				CreatedAt: now,
				UpdatedAt: now,
			}

			message.Attachments =
				append(
					message.Attachments,
					attachment,
				)
		}
	}

	// ========================================================
	// SAVE MESSAGE
	// ========================================================

	if err := s.messageRepo.Create(
		ctx,
		message,
	); err != nil {

		logger.Failure("app.message", operation, startedAt, err)
		return nil, err
	}

	// ========================================================
	// UPDATE CONVERSATION
	// ========================================================

	conversation.LastMessage = &message.Content
	conversation.LastMessageAt = &now
	conversation.UpdatedAt = now

	if conversation.Title == "" ||
		conversation.Title == "Percakapan Baru" {

		generatedTitle :=
			GenerateConversationTitle(req)

		conversation.Title = generatedTitle
	}

	if err := s.conversationRepo.Update(
		ctx,
		userID,
		conversation,
	); err != nil {

		logger.Failure("app.message", operation, startedAt, err)
		return nil, err
	}

	logger.Response(
		"app.message",
		operation,
		startedAt,
		slog.String("message_id", message.ID.String()),
		slog.String("conversation_id", message.ConversationID.String()),
	)

	return ToMessageResponse(
		message,
	), nil
}

func validateAttachmentRequest(item dto.CreateAttachmentRequest) error {
	if strings.TrimSpace(item.URL) == "" {
		return errors.New("attachment url is required")
	}

	if strings.HasPrefix(item.URL, "/uploads/") {
		return nil
	}

	parsedURL, err := url.ParseRequestURI(item.URL)
	if err != nil {
		return errors.New("attachment url must be an http(s) url or /uploads path")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("attachment url scheme must be http or https")
	}

	return nil
}

func (s *MessageAppService) CreateAssistantMessage(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, content string, source domain.AIResponseSource) (*dto.MessageResponse, error) {
	return s.createAssistantMessage(
		ctx,
		userID,
		conversationID,
		content,
		source,
		nil,
	)
}

func (s *MessageAppService) CreateAssistantMessageWithOrchestrator(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, response *dto.OrchestratorResponse, source domain.AIResponseSource) (*dto.MessageResponse, error) {
	if response == nil {
		return nil, errors.New("orchestrator response is required")
	}

	metadata, err := buildAssistantMessageMetadata(response, source)
	if err != nil {
		return nil, err
	}

	return s.createAssistantMessage(
		ctx,
		userID,
		conversationID,
		response.Message,
		source,
		metadata,
	)
}

func (s *MessageAppService) createAssistantMessage(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, content string, source domain.AIResponseSource, metadata map[string]any) (*dto.MessageResponse, error) {
	startedAt := time.Now()
	operation := "CreateAssistantMessage"

	logger.Request(
		"app.message",
		operation,
		slog.String("conversation_id", conversationID.String()),
		slog.String("source", string(source)),
		slog.Int("content_length", len(content)),
	)
	logger.DebugPayload(
		"app.message",
		operation,
		slog.String("content", logger.Truncate(content, logger.DefaultTextLimit)),
	)

	if userID == uuid.Nil {
		err := errors.New("user id is required")
		logger.Failure("app.message", operation, startedAt, err)
		return nil, err
	}

	if conversationID == uuid.Nil {
		err := errors.New("conversation id is required")
		logger.Failure("app.message", operation, startedAt, err)
		return nil, err
	}

	now := time.Now()
	message := &domain.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           domain.MessageRoleAssistant,
		Type:           domain.MessageTypeText,
		Source:         source,
		Status:         domain.MessageStatusSent,
		Content:        content,
		IsStreaming:    false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if len(metadata) > 0 {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			logger.Failure("app.message", operation, startedAt, err)
			return nil, err
		}
		message.Metadata = metadataBytes
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		logger.Failure("app.message", operation, startedAt, err)
		return nil, err
	}

	conversation, err := s.conversationRepo.GetByID(ctx, userID, conversationID)
	if err == nil && conversation != nil {
		conversation.LastMessage = &content
		conversation.LastMessageAt = &now
		conversation.UpdatedAt = now
		_ = s.conversationRepo.Update(ctx, userID, conversation)
	}

	logger.Response(
		"app.message",
		operation,
		startedAt,
		slog.String("message_id", message.ID.String()),
		slog.String("conversation_id", message.ConversationID.String()),
		slog.String("source", string(source)),
	)

	return ToMessageResponse(message), nil
}

func buildAssistantMessageMetadata(response *dto.OrchestratorResponse, source domain.AIResponseSource) (map[string]any, error) {
	if response == nil {
		return nil, nil
	}

	metadata := map[string]any{
		"source":     string(source),
		"session_id": response.SessionID,
		"mode":       response.Mode,
		"state":      response.State,
		"created_at": response.CreatedAt,
	}

	if len(response.Actions) > 0 {
		metadata["actions"] = response.Actions
	}

	if response.Payload != nil {
		metadata["payload"] = response.Payload
	}

	return metadata, nil
}

func (s *MessageAppService) ListConversationMessages(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) ([]dto.MessageResponse, error) {

	if userID == uuid.Nil {
		return nil, errors.New("user id is required")
	}

	if conversationID == uuid.Nil {
		return nil, errors.New("conversation id is required")
	}

	conversation, err :=
		s.conversationRepo.GetByID(ctx, userID, conversationID)

	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, errors.New(
			"conversation not found",
		)
	}

	messages, err :=
		s.messageRepo.ListByConversationID(
			ctx,
			conversationID,
		)

	if err != nil {
		return nil, err
	}

	results := make(
		[]dto.MessageResponse,
		0,
		len(messages),
	)

	for _, item := range messages {

		response := ToMessageResponse(
			&item,
		)

		if response != nil {

			results = append(
				results,
				*response,
			)
		}
	}

	return results, nil
}

func applyAssistantMessageMetadata(response *dto.MessageResponse, message *domain.Message) {
	if response == nil || message == nil || len(message.Metadata) == 0 {
		return
	}

	var metadata struct {
		Source  string                   `json:"source"`
		Actions []dto.ChatAction         `json:"actions"`
		Payload *dto.OrchestratorPayload `json:"payload"`
	}

	if err := json.Unmarshal(message.Metadata, &metadata); err != nil {
		return
	}

	if strings.TrimSpace(metadata.Source) != "" {
		response.Source = metadata.Source
	}

	if len(metadata.Actions) > 0 {
		response.Actions = metadata.Actions
	}

	if metadata.Payload != nil {
		response.Payload = metadata.Payload
	}
}

func ToMessageResponse(message *domain.Message) *dto.MessageResponse {

	if message == nil {
		return nil
	}

	response := &dto.MessageResponse{
		ID:             message.ID,
		ConversationID: message.ConversationID,
		Role:           string(message.Role),
		Type:           string(message.Type),
		Content:        message.Content,
		Status:         string(message.Status),
		IsStreaming:    message.IsStreaming,
		CreatedAt:      message.CreatedAt,
		Source:         string(message.Source),
	}

	if len(message.Attachments) > 0 {

		response.Attachments =
			make(
				[]dto.AttachmentResponse,
				0,
				len(message.Attachments),
			)

		for _, item := range message.Attachments {

			response.Attachments =
				append(
					response.Attachments,
					dto.AttachmentResponse{
						ID: item.ID,

						Type: string(
							item.Type,
						),

						URL: item.URL,

						MimeType: item.MimeType,

						Size: item.Size,

						// future-ready fields
						Width:        nil,
						Height:       nil,
						ThumbnailURL: nil,
						Checksum:     nil,
					},
				)
		}
	}

	applyAssistantMessageMetadata(response, message)

	// ========================================================
	// AI METADATA
	// ========================================================

	if message.Role ==
		domain.MessageRoleAssistant {

		provider := "internal"

		response.AI =
			&dto.AIMessageMetadata{
				Provider: &provider,

				// future-ready
				Model:        nil,
				LatencyMs:    nil,
				InputTokens:  nil,
				OutputTokens: nil,
				TotalTokens:  nil,
				FinishReason: nil,
			}
	}

	return response
}
