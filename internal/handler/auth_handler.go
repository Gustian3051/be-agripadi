package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	auth "github.com/gustian305/backend/internal/dto/auth"
	authservice "github.com/gustian305/backend/internal/service/auth"
)

type AuthHandler struct {
	authService *authservice.Service
}

func NewAuthHandler(authService *authservice.Service) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register godoc
// @Summary Registrasi pengguna
// @Description Membuat akun pengguna baru untuk aplikasi AgriPadi.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body auth.RegisterRequest true "Data registrasi"
// @Success 201 {object} auth.AuthResponse
// @Failure 400 {object} map[string]string
// @Router /api/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req auth.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// Login godoc
// @Summary Login pengguna
// @Description Login menggunakan nomor telepon dan password untuk mendapatkan JWT.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body auth.LoginRequest true "Data login"
// @Success 200 {object} auth.AuthResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Logout godoc
// @Summary Logout pengguna
// @Description Mengakhiri sesi pengguna di sisi aplikasi.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} auth.LogoutResponse
// @Failure 401 {object} map[string]string
// @Router /api/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(
		http.StatusOK,
		auth.LogoutResponse{
			Message: "logout success",
		},
	)
}
