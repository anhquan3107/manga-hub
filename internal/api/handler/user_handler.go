package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"mangahub/internal/auth"
	chatws "mangahub/internal/websocket"
	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

func (h *Handler) Chat(c *gin.Context) {
	chatws.Handler(h.hub, h.authService)(c)
}

func (h *Handler) AddToLibrary(c *gin.Context) {
	var req models.AddLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	entry, err := h.userService.AddToLibrary(c.Request.Context(), currentUserID(c), req)
	if err != nil {
		status := http.StatusInternalServerError
		if isNotFound(err) {
			status = http.StatusNotFound
		}
		utils.Error(c, status, err.Error())
		return
	}

	utils.OK(c, http.StatusCreated, entry)
}

func (h *Handler) GetLibrary(c *gin.Context) {
	items, err := h.userService.GetLibrary(c.Request.Context(), currentUserID(c))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch library")
		return
	}

	utils.OK(c, http.StatusOK, gin.H{
		"items":         items,
		"reading_lists": buildReadingLists(items),
	})
}

func (h *Handler) UpdateProgress(c *gin.Context) {
	var req models.UpdateProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	entry, err := h.userService.UpdateProgress(c.Request.Context(), currentUserID(c), req)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	if h.broadcaster != nil {
		h.broadcaster.PublishProgress(models.ProgressUpdate{
			UserID:    currentUserID(c),
			MangaID:   entry.MangaID,
			Chapter:   entry.CurrentChapter,
			Timestamp: time.Now().Unix(),
		})
	}

	utils.OK(c, http.StatusOK, entry)
}

func currentUserID(c *gin.Context) string {
	value, _ := c.Get(auth.ContextUserIDKey)
	if userID, ok := value.(string); ok {
		return userID
	}
	return ""
}

func buildReadingLists(items []models.LibraryEntry) gin.H {
	reading := make([]models.LibraryEntry, 0)
	completed := make([]models.LibraryEntry, 0)
	planToRead := make([]models.LibraryEntry, 0)

	for _, item := range items {
		switch strings.ToLower(strings.TrimSpace(item.Status)) {
		case "completed":
			completed = append(completed, item)
		case "plan_to_read", "plantoread", "planned":
			planToRead = append(planToRead, item)
		default:
			reading = append(reading, item)
		}
	}

	return gin.H{
		"reading":      reading,
		"completed":    completed,
		"plan_to_read": planToRead,
	}
}
