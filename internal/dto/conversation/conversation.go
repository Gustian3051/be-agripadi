package conversation

import (
	"time"

	"github.com/google/uuid"
)

type ItemResponse struct {
	ID              uuid.UUID  `json:"id"`
	Title           string     `json:"title"`
	Mode            string     `json:"mode"`
	State           string     `json:"state"`
	LastMessage     *string    `json:"last_message,omitempty"`
	LastMessageRole *string    `json:"last_message_role,omitempty"`
	LastMessageType *string    `json:"last_message_type,omitempty"`
	LastMessageAt   *time.Time `json:"last_message_at,omitempty"`
	UnreadCount     int        `json:"unread_count"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type ListResponse struct {
	Items      []ItemResponse `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

type PaginationMeta struct {
	Page      int  `json:"page"`
	Limit     int  `json:"limit"`
	Total     int  `json:"total"`
	TotalPage int  `json:"total_page"`
	HasNext   bool `json:"has_next"`
	HasPrev   bool `json:"has_prev"`
}

type ListQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

type DeleteManyRequest struct {
	ConversationIDs []uuid.UUID `json:"conversation_ids" binding:"required,min=1,max=100,dive,required"`
}
