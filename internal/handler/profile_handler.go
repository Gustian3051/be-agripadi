package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	profile "github.com/gustian305/backend/internal/dto/profile"
	"github.com/gustian305/backend/internal/middleware"
	profileservice "github.com/gustian305/backend/internal/service/profile"
)

type ProfileHandler struct {
	profileService *profileservice.Service
}

func NewProfileHandler(profileService *profileservice.Service) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

// GetProfile godoc
// @Summary Ambil profil pengguna
// @Description Mengambil detail profil pengguna yang sedang login.
// @Tags Profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} profile.Response
// @Failure 401 {object} map[string]string
// @Router /api/profile [get]
func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	result, err := h.profileService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateProfile godoc
// @Summary Perbarui profil pengguna
// @Description Memperbarui nama, email, atau nomor telepon pengguna.
// @Tags Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body profile.UpdateRequest true "Data profil"
// @Success 200 {object} profile.Response
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/profile [patch]
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req profile.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.profileService.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ChangePassword godoc
// @Summary Ubah password
// @Description Mengubah password akun pengguna yang sedang login.
// @Tags Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body profile.ChangePasswordRequest true "Data perubahan password"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/profile/password [patch]
func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req profile.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.profileService.ChangePassword(c.Request.Context(), userID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}
