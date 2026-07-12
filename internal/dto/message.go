package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/domain"
)

//
// ============================================================
// DETECTION
// ============================================================
//

type DetectionCandidateRequest struct {
	Label      string  `json:"label" binding:"required"`
	Confidence float64 `json:"confidence" binding:"required,min=0,max=1"`
	Model      string  `json:"model" binding:"required"`
	Source     string  `json:"source,omitempty"`
	Rank       *int    `json:"rank,omitempty"`
	LatencyMs  *int64  `json:"latency_ms,omitempty"`
}

//
// ============================================================
// CREATE MESSAGE
// ============================================================
//

type CreateMessageRequest struct {
	ConversationID uuid.UUID                   `json:"conversation_id" binding:"omitempty"`
	Type           domain.MessageType          `json:"type" binding:"required"`
	Content        string                      `json:"content" binding:"required"`
	Attachments    []CreateAttachmentRequest   `json:"attachments,omitempty"`
	Detections     []DetectionCandidateRequest `json:"detections,omitempty"` // top-k detections from cnn model
}

//
// ============================================================
// ATTACHMENT REQUEST
// ============================================================
//

type CreateAttachmentRequest struct {
	Type         string  `json:"type" binding:"required"`
	URL          string  `json:"url" binding:"required"`
	MimeType     string  `json:"mime_type" binding:"required"`
	Size         int64   `json:"size" binding:"required,min=1"`
	Width        *int    `json:"width,omitempty"`
	Height       *int    `json:"height,omitempty"`
	ThumbnailURL *string `json:"thumbnail_url,omitempty"`
	Checksum     *string `json:"checksum,omitempty"`
}

//
// ============================================================
// MESSAGE RESPONSE
// ============================================================
//

type MessageResponse struct {
	ID             uuid.UUID            `json:"id"`
	ConversationID uuid.UUID            `json:"conversation_id"`
	Role           string               `json:"role"`
	Type           string               `json:"type"`
	Content        string               `json:"content"`
	Status         string               `json:"status"`
	IsStreaming    bool                 `json:"is_streaming"`
	CreatedAt      time.Time            `json:"created_at"`
	Attachments    []AttachmentResponse `json:"attachments,omitempty"`
	AI             *AIMessageMetadata   `json:"ai,omitempty"`
	Source         string               `json:"source,omitempty"`
	Actions        []ChatAction         `json:"actions,omitempty"`
	Payload        *OrchestratorPayload `json:"payload,omitempty"`
}

//
// ============================================================
// AI METADATA
// ============================================================
//

type AIMessageMetadata struct {
	Provider     *string `json:"provider,omitempty"`
	Model        *string `json:"model,omitempty"`
	LatencyMs    *int64  `json:"latency_ms,omitempty"`
	InputTokens  *int    `json:"input_tokens,omitempty"`
	OutputTokens *int    `json:"output_tokens,omitempty"`
	TotalTokens  *int    `json:"total_tokens,omitempty"`
	FinishReason *string `json:"finish_reason,omitempty"`
}

//
// ============================================================
// ATTACHMENT RESPONSE
// ============================================================
//

type AttachmentResponse struct {
	ID           uuid.UUID `json:"id"`
	Type         string    `json:"type"`
	URL          string    `json:"url"`
	MimeType     string    `json:"mime_type"`
	Size         int64     `json:"size"`
	Width        *int      `json:"width,omitempty"`
	Height       *int      `json:"height,omitempty"`
	ThumbnailURL *string   `json:"thumbnail_url,omitempty"`
	Checksum     *string   `json:"checksum,omitempty"`
}
