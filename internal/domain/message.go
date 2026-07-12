package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type MessageType string

const (
	MessageTypeText  MessageType = "text"
	MessageTypeImage MessageType = "image"
)

type AIResponseSource string

const (
	AIResponseSourceDiagnose     AIResponseSource = "diagnose"
	AIResponseSourceConsultation AIResponseSource = "consultation"
)

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
)

type MessageStatus string

const (
	MessageStatusPending MessageStatus = "pending"
	MessageStatusSent    MessageStatus = "sent"
	MessageStatusRead    MessageStatus = "read"
	MessageStatusFailed  MessageStatus = "failed"
)

type AttachmentType string

const (
	AttachmentTypeImage AttachmentType = "image"
)

type Conversation struct {
	ID     uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`
	User   User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	Title         string     `gorm:"type:text;not null"`
	Mode          string     `gorm:"type:varchar(32);not null;default:'consultation'"`
	State         string     `gorm:"type:varchar(32);not null;default:'idle'"`
	LastMessageAt *time.Time `gorm:"type:timestamp"`
	LastMessage   *string    `gorm:"-"`
	CreatedAt     time.Time  `gorm:"type:timestamp;not null"`
	UpdatedAt     time.Time  `gorm:"type:timestamp;not null"`
	Messages      []Message  `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE"`
}

type Message struct {
	ID             uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ConversationID uuid.UUID        `gorm:"type:uuid;not null;index"`
	Conversation   Conversation     `gorm:"foreignKey:ConversationID"`
	Role           MessageRole      `gorm:"type:varchar(32);not null"`
	Type           MessageType      `gorm:"type:varchar(32);not null"`
	Source         AIResponseSource `gorm:"type:varchar(32)"`
	Status         MessageStatus    `gorm:"type:varchar(32);not null"`
	Content        string           `gorm:"type:text;not null"`
	Metadata       datatypes.JSON   `gorm:"type:jsonb;default:'{}'"`
	TokenCount     int              `gorm:"type:integer;not null;default:0"`
	IsStreaming    bool             `gorm:"type:boolean;not null;default:false"`
	CreatedAt      time.Time        `gorm:"type:timestamp;not null"`
	UpdatedAt      time.Time        `gorm:"type:timestamp;not null"`
	Attachments    []Attachment     `gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE"`
}

type Attachment struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	MessageID uuid.UUID `gorm:"type:uuid;not null;index"`

	Message Message `gorm:"foreignKey:MessageID"`

	Type     AttachmentType `gorm:"type:varchar(32);not null"`
	URL      string         `gorm:"type:text;not null"`
	MimeType string         `gorm:"type:varchar(64);not null"`
	Size     int64          `gorm:"type:bigint;not null"`
	Metadata datatypes.JSON `gorm:"type:jsonb"`

	CreatedAt time.Time `gorm:"type:timestamp;not null"`
	UpdatedAt time.Time `gorm:"type:timestamp;not null"`
}
