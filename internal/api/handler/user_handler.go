package handler

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	chatws "mangahub/internal/websocket"
	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

// currentUserID extracts the user ID from the request context
func currentUserID(c *gin.Context) string {
	userID, exists := c.Get("userID")
	if !exists {
		return ""
	}
	id, ok := userID.(string)
	if !ok {
		return ""
	}
	return id
}

func (h *Handler) Chat(c *gin.Context) {
	chatws.Handler(h.hub, h.authService, h.chatService)(c)
}

// RoomsUsers returns online users grouped by room.
func (h *Handler) RoomsUsers(c *gin.Context) {
	allRooms := h.hub.GetAllRoomUsers()
	roomIDs := make([]string, 0, len(allRooms))
	for roomID := range allRooms {
		roomIDs = append(roomIDs, roomID)
	}
	sort.Strings(roomIDs)

	total := 0
	rooms := make([]gin.H, 0, len(roomIDs))
	for _, roomID := range roomIDs {
		users := allRooms[roomID]
		list := make([]gin.H, 0, len(users))
		for _, u := range users {
			list = append(list, gin.H{
				"user_id":  u.UserID,
				"username": u.Username,
			})
		}
		total += len(list)
		rooms = append(rooms, gin.H{
			"room":  roomID,
			"count": len(list),
			"users": list,
		})
	}

	utils.OK(c, http.StatusOK, gin.H{"rooms": rooms, "total_users": total})
}

// RoomUsers returns the list of users currently connected to a room
func (h *Handler) RoomUsers(c *gin.Context) {
	roomID := c.Param("room")
	if strings.TrimSpace(roomID) == "" {
		utils.Error(c, http.StatusBadRequest, "room id required")
		return
	}

	users := h.hub.GetRoomUsers(roomID)
	out := make([]gin.H, 0, len(users))
	for _, u := range users {
		out = append(out, gin.H{
			"user_id":  u.UserID,
			"username": u.Username,
			"room":     u.RoomID,
		})
	}

	utils.OK(c, http.StatusOK, gin.H{"users": out, "count": len(out)})
}

// RoomHistory returns recent messages for a room.
func (h *Handler) RoomHistory(c *gin.Context) {
	roomID := c.Param("room")
	if strings.TrimSpace(roomID) == "" {
		utils.Error(c, http.StatusBadRequest, "room id required")
		return
	}

	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			utils.Error(c, http.StatusBadRequest, "invalid limit")
			return
		}
		if parsed > 200 {
			parsed = 200
		}
		limit = parsed
	}

	if h.chatService == nil {
		utils.Error(c, http.StatusServiceUnavailable, "chat history unavailable")
		return
	}

	messages, err := h.chatService.GetRoomHistory(c.Request.Context(), roomID, limit)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to load chat history")
		return
	}

	utils.OK(c, http.StatusOK, gin.H{"room": roomID, "limit": limit, "messages": messages})
}

func (h *Handler) GetMe(c *gin.Context) {
	userID := currentUserID(c)
	if userID == "" {
		utils.Error(c, http.StatusUnauthorized, "missing user id")
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "user not found")
		return
	}

	utils.OK(c, http.StatusOK, user)
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

	result, err := h.userService.UpdateProgress(c.Request.Context(), currentUserID(c), req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if h.broadcaster != nil {
		h.broadcaster.PublishProgress(models.ProgressUpdate{
			UserID:    currentUserID(c),
			MangaID:   result.Entry.MangaID,
			Chapter:   result.Entry.CurrentChapter,
			Timestamp: time.Now().Unix(),
		})
	}

	utils.OK(c, http.StatusOK, gin.H{
		"manga_id":         result.Entry.MangaID,
		"title":            result.MangaTitle,
		"previous_chapter": result.PreviousChapter,
		"current_chapter":  result.Entry.CurrentChapter,
		"previous_volume":  result.PreviousVolume,
		"current_volume":   result.Entry.CurrentVolume,
		"updated_at":       result.Entry.UpdatedAt,
		"total_chapters":   result.TotalChapters,
		"notes":            result.Entry.Notes,
		"status":           result.Entry.Status,
	})
}

func (h *Handler) GetProgressHistory(c *gin.Context) {
	mangaID := strings.TrimSpace(c.Query("manga_id"))
	items, err := h.userService.GetProgressHistory(c.Request.Context(), currentUserID(c), mangaID)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.OK(c, http.StatusOK, gin.H{"items": items})
}

func (h *Handler) RemoveFromLibrary(c *gin.Context) {
	mangaID := c.Param("id")
	if strings.TrimSpace(mangaID) == "" {
		utils.Error(c, http.StatusBadRequest, "manga id required")
		return
	}

	if err := h.userService.RemoveFromLibrary(c.Request.Context(), currentUserID(c), mangaID); err != nil {
		status := http.StatusInternalServerError
		if isNotFound(err) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}
		utils.Error(c, status, err.Error())
		return
	}

	utils.OK(c, http.StatusOK, gin.H{"message": "removed"})
}

func (h *Handler) UpdateLibrary(c *gin.Context) {
	mangaID := c.Param("id")
	if strings.TrimSpace(mangaID) == "" {
		utils.Error(c, http.StatusBadRequest, "manga id required")
		return
	}

	var req models.UpdateLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	entry, err := h.userService.UpdateLibrary(c.Request.Context(), currentUserID(c), mangaID, req)
	if err != nil {
		status := http.StatusInternalServerError
		if isNotFound(err) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}
		utils.Error(c, status, err.Error())
		return
	}

	utils.OK(c, http.StatusOK, entry)
}

func buildReadingLists(items []models.LibraryEntry) gin.H {
	reading := make([]models.LibraryEntry, 0)
	completed := make([]models.LibraryEntry, 0)
	planToRead := make([]models.LibraryEntry, 0)
	onHold := make([]models.LibraryEntry, 0)
	dropped := make([]models.LibraryEntry, 0)

	for _, item := range items {
		switch strings.ToLower(strings.TrimSpace(item.Status)) {
		case "completed":
			completed = append(completed, item)
		case "on-hold", "on_hold", "on hold":
			onHold = append(onHold, item)
		case "dropped":
			dropped = append(dropped, item)
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
		"on_hold":      onHold,
		"dropped":      dropped,
	}
}

// SendPM sends a private message to another user.
func (h *Handler) SendPM(c *gin.Context) {
	if h.chatService == nil {
		utils.Error(c, http.StatusServiceUnavailable, "chat service unavailable")
		return
	}

	var req models.SendPMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	senderID := currentUserID(c)
	if senderID == "" {
		utils.Error(c, http.StatusUnauthorized, "missing user id")
		return
	}

	sender, err := h.userService.GetUserByID(c.Request.Context(), senderID)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "sender not found")
		return
	}

	recipient, err := h.userService.GetUserByUsername(c.Request.Context(), req.RecipientUsername)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "recipient not found")
		return
	}

	if sender.ID == recipient.ID {
		utils.Error(c, http.StatusBadRequest, "cannot send message to yourself")
		return
	}

	pm := models.PrivateMessage{
		SenderID:          sender.ID,
		SenderUsername:    sender.Username,
		RecipientID:       recipient.ID,
		RecipientUsername: recipient.Username,
		Message:           strings.TrimSpace(req.Message),
		Timestamp:         time.Now().Unix(),
	}

	if err := h.chatService.SendPrivateMessage(c.Request.Context(), pm); err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to send message")
		return
	}

	utils.OK(c, http.StatusCreated, gin.H{"message": "PM sent", "recipient": recipient.Username})
}
