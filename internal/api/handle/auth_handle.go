package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

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
