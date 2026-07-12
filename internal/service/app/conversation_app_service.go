package app

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/domain"
	"github.com/gustian305/backend/internal/dto"
	conversationdto "github.com/gustian305/backend/internal/dto/conversation"
	"github.com/gustian305/backend/internal/repository"
)

type ConversationAppService struct {
	repo repository.ConversationRepository
}

func NewConversationAppService(
	repo repository.ConversationRepository,
) *ConversationAppService {

	return &ConversationAppService{
		repo: repo,
	}
}

func (s *ConversationAppService) GetOrCreateConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) (*conversationdto.ItemResponse, error) {

	if userID == uuid.Nil {
		return nil, errors.New("user id is required")
	}

	if conversationID == uuid.Nil {

		now := time.Now()

		conversation := &domain.Conversation{
			ID:            uuid.New(),
			UserID:        userID,
			Title:         "Percakapan Baru",
			Mode:          "consultation",
			State:         "idle",
			LastMessageAt: nil,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := s.repo.Create(
			ctx,
			userID,
			conversation,
		); err != nil {

			return nil, err
		}

		return ToConversationDTO(
			conversation,
		), nil
	}

	conversation, err := s.repo.GetByID(
		ctx,
		userID,
		conversationID,
	)

	if err != nil {
		return nil, err
	}

	if conversation == nil {

		return nil, errors.New(
			"conversation not found",
		)
	}

	return ToConversationDTO(
		conversation,
	), nil
}

func (s *ConversationAppService) UpdateConversation(ctx context.Context, userID uuid.UUID, conversation *domain.Conversation) error {

	if userID == uuid.Nil {
		return errors.New("user id is required")
	}

	if conversation == nil {

		return errors.New(
			"conversation is required",
		)
	}

	return s.repo.Update(
		ctx,
		userID,
		conversation,
	)
}

func (s *ConversationAppService) UpdateState(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, mode string, state string) error {

	if userID == uuid.Nil {
		return errors.New("user id is required")
	}

	if conversationID == uuid.Nil {

		return errors.New(
			"conversation id is required",
		)
	}

	return s.repo.UpdateState(
		ctx,
		userID,
		conversationID,
		mode,
		state,
	)
}

func (s *ConversationAppService) ListConversations(ctx context.Context, userID uuid.UUID, query conversationdto.ListQuery) (*conversationdto.ListResponse, error) {

	if userID == uuid.Nil {
		return nil, errors.New("user id is required")
	}

	if query.Page <= 0 {
		query.Page = 1
	}

	if query.Limit <= 0 {
		query.Limit = 20
	}

	conversations, total, err :=
		s.repo.ListByPaginated(
			ctx,
			userID,
			query.Page,
			query.Limit,
		)

	if err != nil {
		return nil, err
	}

	items := ToConversationDTOList(
		conversations,
	)

	totalPage := int(
		math.Ceil(
			float64(total) /
				float64(query.Limit),
		),
	)

	return &conversationdto.ListResponse{
		Items: items,

		Pagination: conversationdto.PaginationMeta{
			Page:      query.Page,
			Limit:     query.Limit,
			Total:     total,
			TotalPage: totalPage,
			HasNext:   query.Page < totalPage,
			HasPrev:   query.Page > 1,
		},
	}, nil
}

func (s *ConversationAppService) DeleteConversation(
	ctx context.Context,
	userID uuid.UUID,
	conversationID uuid.UUID,
) error {

	if userID == uuid.Nil {
		return errors.New("user id is required")
	}

	if conversationID == uuid.Nil {
		return errors.New("conversation id is required")
	}

	conversation, err := s.repo.GetByID(
		ctx,
		userID,
		conversationID,
	)

	if err != nil {
		return err
	}

	if conversation == nil {
		return errors.New("conversation not found")
	}

	attachments, err := s.repo.GetAttachmentsByConversationID(
		ctx,
		userID,
		conversationID,
	)

	if err != nil {
		return err
	}

	for _, attachment := range attachments {

		if err := deleteAttachmentFile(
			attachment.URL,
		); err != nil {

			return err
		}
	}

	return s.repo.Delete(
		ctx,
		userID,
		conversationID,
	)
}

func (s *ConversationAppService) DeleteManyConversations(
	ctx context.Context,
	userID uuid.UUID,
	req conversationdto.DeleteManyRequest,
) error {

	if userID == uuid.Nil {
		return errors.New("user id is required")
	}

	if len(req.ConversationIDs) == 0 {
		return errors.New("conversation ids are required")
	}

	for _, conversationID := range req.ConversationIDs {

		if conversationID == uuid.Nil {
			continue
		}

		conversation, err := s.repo.GetByID(
			ctx,
			userID,
			conversationID,
		)

		if err != nil {
			return err
		}

		if conversation == nil {
			return errors.New("conversation not found")
		}

		attachments, err := s.repo.GetAttachmentsByConversationID(
			ctx,
			userID,
			conversationID,
		)

		if err != nil {
			return err
		}

		for _, attachment := range attachments {

			if err := deleteAttachmentFile(
				attachment.URL,
			); err != nil {

				return err
			}
		}

		if err := s.repo.Delete(
			ctx,
			userID,
			conversationID,
		); err != nil {

			return err
		}
	}

	return nil
}

func (s *ConversationAppService) DeleteAllConversations(
	ctx context.Context,
	userID uuid.UUID,
) (int64, error) {

	if userID == uuid.Nil {
		return 0, errors.New("user id is required")
	}

	attachments, err := s.repo.GetAttachmentsByUserID(
		ctx,
		userID,
	)

	if err != nil {
		return 0, err
	}

	for _, attachment := range attachments {

		if err := deleteAttachmentFile(
			attachment.URL,
		); err != nil {

			return 0, err
		}
	}

	return s.repo.DeleteAllByUserID(
		ctx,
		userID,
	)
}

func deleteAttachmentFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	// Hanya hapus file yang memang dikelola folder uploads aplikasi.
	// URL disimpan sebagai /uploads/<nama-file>, sedangkan file fisik berada di ./uploads.
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return nil
	}

	localPath := strings.TrimPrefix(path, "/")
	if !strings.HasPrefix(localPath, "uploads/") {
		return nil
	}

	localPath = filepath.Clean(localPath)
	if localPath == "." || strings.HasPrefix(localPath, "..") || strings.Contains(localPath, string(filepath.Separator)+".."+string(filepath.Separator)) {
		return nil
	}

	err := os.Remove(localPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func ToConversationDTO(conversation *domain.Conversation) *conversationdto.ItemResponse {

	if conversation == nil {
		return nil
	}

	var lastMessage *string
	var lastMessageAt *time.Time

	if conversation.LastMessage != nil {
		lastMessage = conversation.LastMessage
	}

	if conversation.LastMessageAt != nil {
		lastMessageAt = conversation.LastMessageAt
	}

	return &conversationdto.ItemResponse{
		ID: conversation.ID,

		Title: conversation.Title,

		Mode:  conversation.Mode,
		State: conversation.State,

		LastMessage:   lastMessage,
		LastMessageAt: lastMessageAt,

		// future-ready
		LastMessageRole: nil,
		LastMessageType: nil,

		UnreadCount: 0,

		CreatedAt: conversation.CreatedAt,
		UpdatedAt: conversation.UpdatedAt,
	}
}

func ToConversationDTOList(conversations []domain.Conversation) []conversationdto.ItemResponse {

	results := make(
		[]conversationdto.ItemResponse,
		0,
		len(conversations),
	)

	for _, item := range conversations {

		itemDTO := ToConversationDTO(
			&item,
		)

		if itemDTO != nil {

			results = append(
				results,
				*itemDTO,
			)
		}
	}

	return results
}

func GenerateConversationTitle(req dto.CreateMessageRequest) string {

	if len(req.Detections) > 0 {

		best := req.Detections[0]

		for _, item := range req.Detections {

			if item.Confidence >
				best.Confidence {

				best = item
			}
		}

		label := strings.TrimSpace(
			best.Label,
		)

		if label != "" {

			return NormalizeConversationTitle(
				label,
			)
		}
	}

	text := strings.TrimSpace(
		req.Content,
	)

	if text != "" {

		return NormalizeConversationTitle(
			text,
		)
	}

	// =====================================
	// ATTACHMENT FALLBACK
	// =====================================

	if len(req.Attachments) > 0 {

		return "Analisis Hama Padi"
	}

	// =====================================
	// DEFAULT
	// =====================================

	return "Percakapan Baru"
}

func NormalizeConversationTitle(value string) string {

	value = strings.TrimSpace(value)

	if value == "" {
		return "Percakapan Baru"
	}

	value = strings.Join(strings.Fields(value), " ")

	runes := []rune(value)

	if len(runes) > 40 {

		value =
			string(runes[:40]) + "..."
	}

	value = strings.Title(
		strings.ToLower(value),
	)

	return value
}
