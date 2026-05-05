package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mangahub/internal/auth"
	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

// Register godoc
// @Summary Register account
// @Description Creates a new user account and returns JWT token.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body models.RegisterRequest true "Register payload"
// @Success 201 {object} authResponse
// @Failure 400 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Router /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "exists") {
			utils.Error(c, http.StatusConflict, err.Error())
			return
		}
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.OK(c, http.StatusCreated, resp)
}

// Login godoc
// @Summary Login
// @Description Authenticates a user and returns JWT token.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body models.LoginRequest true "Login payload"
// @Success 200 {object} authResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	utils.OK(c, http.StatusOK, resp)
}

// Logout godoc
// @Summary Logout
// @Description Revokes current JWT token.
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} messageResponse
// @Failure 401 {object} errorResponse
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	rawToken, ok := c.Get("token")
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "invalid token")
		return
	}

	token, ok := rawToken.(string)
	if !ok || strings.TrimSpace(token) == "" {
		utils.Error(c, http.StatusUnauthorized, "invalid token")
		return
	}

	if err := h.authService.Logout(token); err != nil {
		utils.Error(c, http.StatusUnauthorized, "invalid token")
		return
	}

	utils.OK(c, http.StatusOK, gin.H{"message": "logged out"})
}

// ChangePassword godoc
// @Summary Change password
// @Description Changes current user password and revokes current token.
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.ChangePasswordRequest true "Change password payload"
// @Success 200 {object} messageResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Router /auth/change-password [post]
func (h *Handler) ChangePassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, ok := c.Get(auth.ContextUserIDKey)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "invalid token")
		return
	}

	userIDStr, ok := userID.(string)
	if !ok || strings.TrimSpace(userIDStr) == "" {
		utils.Error(c, http.StatusUnauthorized, "invalid token")
		return
	}

	rawToken, ok := c.Get(auth.ContextTokenKey)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "invalid token")
		return
	}

	token, ok := rawToken.(string)
	if !ok || strings.TrimSpace(token) == "" {
		utils.Error(c, http.StatusUnauthorized, "invalid token")
		return
	}

	if err := h.authService.ChangePassword(c.Request.Context(), userIDStr, req.CurrentPassword, req.NewPassword); err != nil {
		switch {
		case strings.Contains(strings.ToLower(err.Error()), "invalid current password"):
			utils.Error(c, http.StatusUnauthorized, err.Error())
		default:
			utils.Error(c, http.StatusBadRequest, err.Error())
		}
		return
	}

	_ = h.authService.Logout(token)

	utils.OK(c, http.StatusOK, gin.H{"message": "password changed successfully"})
}
