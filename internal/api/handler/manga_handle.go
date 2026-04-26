package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

func (h *Handler) ListManga(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, err := h.mangaService.List(c.Request.Context(), models.MangaQuery{
		Query:  c.Query("q"),
		Genre:  c.Query("genre"),
		Status: c.Query("status"),
		Limit:  limit,
	})
	if err != nil {
		log.Printf("list manga error: %v", err)
		utils.Error(c, http.StatusInternalServerError, "failed to fetch manga")
		return
	}

	utils.OK(c, http.StatusOK, gin.H{"items": items})
}

func (h *Handler) GetManga(c *gin.Context) {
	item, err := h.mangaService.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if isNotFound(err) {
			utils.Error(c, http.StatusNotFound, "manga not found")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to fetch manga")
		return
	}

	utils.OK(c, http.StatusOK, item)
}

func (h *Handler) CreateManga(c *gin.Context) {
	var req models.CreateMangaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	item, err := h.mangaService.Create(c.Request.Context(), req)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			utils.Error(c, http.StatusConflict, "manga id already exists")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to create manga")
		return
	}

	utils.OK(c, http.StatusCreated, item)
}

func (h *Handler) UpdateManga(c *gin.Context) {
	var req models.UpdateMangaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	item, err := h.mangaService.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		if isNotFound(err) {
			utils.Error(c, http.StatusNotFound, "manga not found")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to update manga")
		return
	}

	utils.OK(c, http.StatusOK, item)
}

func (h *Handler) DeleteManga(c *gin.Context) {
	err := h.mangaService.Delete(c.Request.Context(), c.Param("id"))
	if err != nil {
		if isNotFound(err) {
			utils.Error(c, http.StatusNotFound, "manga not found")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to delete manga")
		return
	}

	utils.OK(c, http.StatusOK, gin.H{"message": "manga deleted"})
}
