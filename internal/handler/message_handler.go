package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gustian305/backend/internal/dto"
	"github.com/gustian305/backend/internal/middleware"
	"github.com/gustian305/backend/internal/service/app"
)

type MessageHandler struct {
	chatApp *app.ChatAppService
}

func NewMessageHandler(
	chatApp *app.ChatAppService,
) *MessageHandler {

	return &MessageHandler{
		chatApp: chatApp,
	}
}

// CreateUserMessage godoc
// @Summary Kirim pesan pengguna
// @Description Mengirim pesan pada percakapan. Payload dapat berisi teks, lampiran gambar, dan kandidat deteksi CNN untuk memulai alur diagnosis.
// @Tags Messages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param conversation_id path string true "ID percakapan"
// @Param request body dto.CreateMessageRequest true "Data pesan"
// @Success 201 {object} dto.OrchestratorResponse
// @Failure 400 {object} dto.APIErrorResponse
// @Failure 401 {object} dto.APIErrorResponse
// @Router /api/conversations/{conversation_id}/messages [post]
func (h *MessageHandler) CreateUserMessage(c *gin.Context) {

	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{"error": "unauthorized"},
		)
		return
	}

	var req dto.CreateMessageRequest

	if err := c.ShouldBindJSON(
		&req,
	); err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": err.Error(),
			},
		)

		return
	}

	conversationID, err := uuid.Parse(
		c.Param("conversation_id"),
	)

	if err != nil || conversationID == uuid.Nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "invalid conversation id",
			},
		)

		return
	}

	if req.ConversationID != uuid.Nil &&
		req.ConversationID != conversationID {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "conversation id mismatch",
			},
		)

		return
	}

	req.ConversationID = conversationID

	result, err :=
		h.chatApp.HandleUserMessage(
			c.Request.Context(),
			userID,
			req,
		)

	if err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": err.Error(),
			},
		)

		return
	}

	c.JSON(
		http.StatusCreated,
		result,
	)
}

// ListConversationMessages godoc
// @Summary Daftar pesan percakapan
// @Description Mengambil seluruh pesan pada satu percakapan milik pengguna.
// @Tags Messages
// @Produce json
// @Security BearerAuth
// @Param conversation_id path string true "ID percakapan"
// @Success 200 {object} dto.MessageListResponse
// @Failure 400 {object} dto.APIErrorResponse
// @Failure 401 {object} dto.APIErrorResponse
// @Router /api/conversations/{conversation_id}/messages [get]
func (h *MessageHandler) ListConversationMessages(c *gin.Context) {

	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{"error": "unauthorized"},
		)
		return
	}

	conversationID, err :=
		uuid.Parse(
			c.Param("conversation_id"),
		)

	if err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "invalid conversation id",
			},
		)

		return
	}

	result, err :=
		h.chatApp.
			ListConversationMessages(
				c.Request.Context(),
				userID,
				conversationID,
			)

	if err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": err.Error(),
			},
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"items": result,
		},
	)
}
