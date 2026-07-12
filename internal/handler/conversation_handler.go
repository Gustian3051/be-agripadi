package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	conversation "github.com/gustian305/backend/internal/dto/conversation"
	"github.com/gustian305/backend/internal/middleware"
	"github.com/gustian305/backend/internal/service/app"
)

type ConversationHandler struct {
	conversationApp *app.ConversationAppService
}

func NewConversationHandler(
	conversationApp *app.ConversationAppService,
) *ConversationHandler {

	return &ConversationHandler{
		conversationApp: conversationApp,
	}
}

// GetOrCreateConversation godoc
// @Summary Ambil atau buat percakapan
// @Description Mengambil percakapan berdasarkan ID. Jika ID kosong pada endpoint POST /api/conversations, sistem membuat percakapan baru.
// @Tags Conversations
// @Produce json
// @Security BearerAuth
// @Param conversation_id path string false "ID percakapan"
// @Success 200 {object} conversation.ItemResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/conversations/{conversation_id} [get]
// @Router /api/conversations [post]
func (h *ConversationHandler) GetOrCreateConversation(c *gin.Context) {

	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{"error": "unauthorized"},
		)
		return
	}

	conversationIDParam := c.Param(
		"conversation_id",
	)

	if conversationIDParam == "" {
		conversationIDParam = c.Query(
			"conversation_id",
		)
	}

	var conversationID uuid.UUID

	if conversationIDParam != "" {

		parsedID, err :=
			uuid.Parse(
				conversationIDParam,
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

		conversationID = parsedID
	}

	result, err :=
		h.conversationApp.GetOrCreateConversation(
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
		result,
	)
}

// ListConversations godoc
// @Summary Daftar percakapan
// @Description Mengambil daftar percakapan pengguna dengan pagination.
// @Tags Conversations
// @Produce json
// @Security BearerAuth
// @Param page query int false "Halaman"
// @Param limit query int false "Jumlah data per halaman"
// @Success 200 {object} conversation.ListResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/conversations [get]
func (h *ConversationHandler) ListConversations(c *gin.Context) {

	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{"error": "unauthorized"},
		)
		return
	}

	var query conversation.ListQuery

	if err := c.ShouldBindQuery(
		&query,
	); err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": err.Error(),
			},
		)

		return
	}

	result, err :=
		h.conversationApp.ListConversations(
			c.Request.Context(),
			userID,
			query,
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
		result,
	)
}

// DeleteConversation godoc
// @Summary Hapus satu percakapan
// @Description Menghapus satu percakapan beserta lampiran lokal milik pengguna.
// @Tags Conversations
// @Produce json
// @Security BearerAuth
// @Param conversation_id path string true "ID percakapan"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/conversations/{conversation_id} [delete]
func (h *ConversationHandler) DeleteConversation(c *gin.Context) {

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

	err = h.conversationApp.DeleteConversation(
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
			"message": "conversation deleted",
		},
	)
}

// DeleteManyConversations godoc
// @Summary Hapus beberapa percakapan
// @Description Menghapus beberapa percakapan berdasarkan daftar ID.
// @Tags Conversations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body conversation.DeleteManyRequest true "Daftar ID percakapan"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/conversations/delete-many [post]
// @Router /api/conversations [delete]
func (h *ConversationHandler) DeleteManyConversations(c *gin.Context) {

	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{"error": "unauthorized"},
		)
		return
	}

	var req conversation.DeleteManyRequest

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

	err := h.conversationApp.DeleteManyConversations(
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
		http.StatusOK,
		gin.H{
			"message": "conversations deleted",
		},
	)
}

// DeleteAllConversations godoc
// @Summary Hapus seluruh percakapan
// @Description Menghapus seluruh percakapan milik pengguna yang sedang login.
// @Tags Conversations
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/conversations/delete-all [post]
func (h *ConversationHandler) DeleteAllConversations(c *gin.Context) {

	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{"error": "unauthorized"},
		)
		return
	}

	deletedCount, err := h.conversationApp.DeleteAllConversations(
		c.Request.Context(),
		userID,
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
			"message":       "all conversations deleted",
			"deleted_count": deletedCount,
		},
	)
}
