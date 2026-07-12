package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/domain"
	"github.com/gustian305/backend/internal/dto"
	"github.com/gustian305/backend/internal/service/consultation"
	"github.com/gustian305/backend/internal/service/expert_system"
	"github.com/gustian305/backend/logger"
)

type ImageClassifier interface {
	ClassifyFile(ctx context.Context, imagePath string) ([]dto.DetectionCandidateRequest, error)
}

type ChatAppService struct {
	expertService       *expertSystem.ExpertSystemService
	consultationService *consultation.ConsultationService
	messageApp          *MessageAppService
	conversationApp     *ConversationAppService
	imageClassifier     ImageClassifier
	uploadDir           string
}

func NewChatAppService(
	expertService *expertSystem.ExpertSystemService,
	consultationService *consultation.ConsultationService,
	messageApp *MessageAppService,
	conversationApp *ConversationAppService,
	imageClassifier ImageClassifier,
	uploadDir string,
) *ChatAppService {

	return &ChatAppService{
		expertService:       expertService,
		consultationService: consultationService,
		messageApp:          messageApp,
		conversationApp:     conversationApp,
		imageClassifier:     imageClassifier,
		uploadDir:           uploadDir,
	}
}

func (s *ChatAppService) HandleUserMessage(ctx context.Context, userID uuid.UUID, req dto.CreateMessageRequest) (*dto.OrchestratorResponse, error) {
	startedAt := time.Now()
	operation := "HandleUserMessage"

	logger.Request(
		"app.chat",
		operation,
		slog.String("conversation_id", req.ConversationID.String()),
		slog.String("message_type", string(req.Type)),
		slog.Int("content_length", len(req.Content)),
		slog.Int("attachment_count", len(req.Attachments)),
		slog.Int("detection_count", len(req.Detections)),
	)
	logger.DebugPayload(
		"app.chat",
		operation,
		slog.String("content", logger.Truncate(req.Content, logger.DefaultTextLimit)),
		slog.Any("detections", req.Detections),
	)

	logCNNDetections(
		"ReceiveClientDetections",
		req.ConversationID,
		req.Detections,
		slog.String("message_type", string(req.Type)),
	)

	if s == nil || s.messageApp == nil {

		err := errors.New(
			"chat service is not configured",
		)
		logger.Failure("app.chat", operation, startedAt, err)
		return nil, err
	}

	if userID == uuid.Nil {
		err := errors.New("user id is required")
		logger.Failure("app.chat", operation, startedAt, err)
		return nil, err
	}

	if len(req.Detections) == 0 && requestHasImageAttachment(req) {
		if err := s.enrichDetectionsFromImageAttachment(ctx, &req); err != nil {
			logger.Failure("app.chat", operation, startedAt, err)
			return nil, err
		}
	}

	_, err :=
		s.messageApp.CreateUserMessage(
			ctx,
			userID,
			req,
		)

	if err != nil {
		logger.Failure("app.chat", operation, startedAt, err)
		return nil, err
	}

	if len(req.Detections) > 0 {

		response, err := s.handleDiagnoseFlow(
			ctx,
			userID,
			req,
		)
		logChatResponse(operation, startedAt, "diagnose_start", response, err)
		return response, err
	}

	diagnoseResponse, err := s.handleActiveDiagnoseFlow(
		ctx,
		userID,
		req,
	)

	if err != nil {
		logger.Failure("app.chat", operation, startedAt, err)
		return nil, err
	}

	if diagnoseResponse != nil {
		logChatResponse(operation, startedAt, "diagnose_continue", diagnoseResponse, nil)
		return diagnoseResponse, nil
	}

	response, err := s.handleConsultationFlow(
		ctx,
		userID,
		req,
	)
	logChatResponse(operation, startedAt, "consultation", response, err)
	return response, err
}

func (s *ChatAppService) enrichDetectionsFromImageAttachment(ctx context.Context, req *dto.CreateMessageRequest) error {
	if s.imageClassifier == nil {
		return errors.New("image classifier is not configured")
	}

	imagePath, err := s.resolveUploadAttachmentPath(req.Attachments)
	if err != nil {
		return err
	}

	detections, err := s.imageClassifier.ClassifyFile(ctx, imagePath)
	if err != nil {
		return fmt.Errorf("image classification failed: %w", err)
	}

	if len(detections) == 0 {
		return errors.New("image classification returned no detection candidates")
	}

	req.Detections = detections
	logCNNDetections(
		"ClassifyImageAttachment",
		req.ConversationID,
		detections,
		slog.String("image_path", imagePath),
		slog.String("cnn_source", "server_api"),
	)
	logger.DebugPayload(
		"app.chat",
		"ClassifyImageAttachment",
		slog.String("image_path", imagePath),
		slog.Int("detection_count", len(detections)),
		slog.Any("detections", detections),
	)

	return nil
}

func (s *ChatAppService) resolveUploadAttachmentPath(attachments []dto.CreateAttachmentRequest) (string, error) {
	for _, attachment := range attachments {
		if attachment.Type != string(domain.AttachmentTypeImage) {
			continue
		}

		uploadPath, err := attachmentUploadPath(attachment.URL)
		if err != nil {
			return "", err
		}

		filename := filepath.Base(uploadPath)
		if filename == "." || filename == string(filepath.Separator) || strings.TrimSpace(filename) == "" {
			return "", errors.New("invalid upload image filename")
		}

		uploadDir := strings.TrimSpace(s.uploadDir)
		if uploadDir == "" {
			uploadDir = "uploads"
		}

		return filepath.Join(uploadDir, filename), nil
	}

	return "", errors.New("image attachment is required")
}

func attachmentUploadPath(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", errors.New("attachment url is required")
	}

	if strings.HasPrefix(rawURL, "/uploads/") {
		return rawURL, nil
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.New("invalid attachment url")
	}

	if parsedURL.Path == "" || !strings.HasPrefix(parsedURL.Path, "/uploads/") {
		return "", errors.New("image attachment must reference an uploaded file")
	}

	return parsedURL.Path, nil
}

func logCNNDetections(operation string, conversationID uuid.UUID, detections []dto.DetectionCandidateRequest, attrs ...slog.Attr) {
	if len(detections) == 0 {
		return
	}

	baseAttrs := []slog.Attr{
		slog.String("conversation_id", conversationID.String()),
		slog.Int("detection_count", len(detections)),
		slog.Any("detections", detections),
		slog.String("detections_json", logger.PrettyJSON(detections)),
	}

	logger.InfoPayload(
		"cnn",
		operation,
		"cnn detection result json",
		append(baseAttrs, attrs...)...,
	)
}

func requestHasImageAttachment(req dto.CreateMessageRequest) bool {
	for _, attachment := range req.Attachments {
		if attachment.Type == string(domain.AttachmentTypeImage) {
			return true
		}
	}

	return false
}

func (s *ChatAppService) handleConsultationFlow(ctx context.Context, userID uuid.UUID, req dto.CreateMessageRequest) (*dto.OrchestratorResponse, error) {

	if s.consultationService == nil {

		return nil, errors.New(
			"consultation service is not configured",
		)
	}

	response, err :=
		s.consultationService.ProcessMessage(
			ctx,
			req.Content,
		)

	if err != nil {
		return nil, err
	}

	if err := s.updateConversationFlowState(
		ctx,
		userID,
		req.ConversationID,
		response.Mode,
		response.State,
	); err != nil {
		return nil, err
	}

	_, err = s.messageApp.CreateAssistantMessageWithOrchestrator(
		ctx,
		userID,
		req.ConversationID,
		response,
		domain.AIResponseSourceConsultation,
	)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (s *ChatAppService) handleDiagnoseFlow(ctx context.Context, userID uuid.UUID, req dto.CreateMessageRequest) (*dto.OrchestratorResponse, error) {

	if s.expertService == nil {

		return nil, errors.New(
			"expert service is not configured",
		)
	}

	if len(req.Detections) == 0 {

		return nil, errors.New(
			"detection candidates are required",
		)
	}

	response, err :=
		s.expertService.StartDiagnosis(
			ctx,
			req.ConversationID,
			req.Detections,
		)

	if err != nil {
		return nil, err
	}

	if err := s.updateConversationFlowState(
		ctx,
		userID,
		req.ConversationID,
		response.Mode,
		response.State,
	); err != nil {
		return nil, err
	}

	_, err = s.messageApp.CreateAssistantMessageWithOrchestrator(
		ctx,
		userID,
		req.ConversationID,
		response,
		domain.AIResponseSourceDiagnose,
	)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (s *ChatAppService) handleActiveDiagnoseFlow(ctx context.Context, userID uuid.UUID, req dto.CreateMessageRequest) (*dto.OrchestratorResponse, error) {

	if s.expertService == nil {
		return nil, nil
	}

	response, err :=
		s.expertService.ContinueActiveDiagnosis(
			ctx,
			req.ConversationID,
			req.Content,
		)

	if err != nil {
		return nil, err
	}

	if response == nil {
		return nil, nil
	}

	if err := s.updateConversationFlowState(
		ctx,
		userID,
		req.ConversationID,
		response.Mode,
		response.State,
	); err != nil {
		return nil, err
	}

	_, err = s.messageApp.CreateAssistantMessageWithOrchestrator(
		ctx,
		userID,
		req.ConversationID,
		response,
		domain.AIResponseSourceDiagnose,
	)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (s *ChatAppService) updateConversationFlowState(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, mode dto.ConversationMode, state dto.DiagnoseFlowState) error {

	if s.conversationApp == nil {
		return nil
	}

	if mode == dto.ConversationModeConsultation &&
		state == dto.DiagnoseFlowStateCompleted {

		state = dto.DiagnoseFlowStateIdle
	}

	return s.conversationApp.UpdateState(
		ctx,
		userID,
		conversationID,
		string(mode),
		string(state),
	)
}

func (s *ChatAppService) ListConversationMessages(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) ([]dto.MessageResponse, error) {

	if s == nil || s.messageApp == nil {
		return nil, errors.New("chat service is not configured")
	}

	return s.messageApp.ListConversationMessages(
		ctx,
		userID,
		conversationID,
	)
}

func logChatResponse(operation string, startedAt time.Time, route string, response *dto.OrchestratorResponse, err error) {
	if err != nil {
		logger.Failure(
			"app.chat",
			operation,
			startedAt,
			err,
			slog.String("route", route),
		)
		return
	}

	if response == nil {
		logger.Response(
			"app.chat",
			operation,
			startedAt,
			slog.String("route", route),
			slog.Bool("empty_response", true),
		)
		return
	}

	logger.Response(
		"app.chat",
		operation,
		startedAt,
		slog.String("route", route),
		slog.String("session_id", response.SessionID.String()),
		slog.String("mode", string(response.Mode)),
		slog.String("state", string(response.State)),
		slog.Int("message_length", len(response.Message)),
		slog.Int("action_count", len(response.Actions)),
	)
	logger.DebugPayload(
		"app.chat",
		operation,
		slog.String("response_message", logger.Truncate(response.Message, logger.DefaultTextLimit)),
	)
}
