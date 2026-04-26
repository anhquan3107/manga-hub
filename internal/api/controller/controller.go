package controller

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mangahub/internal/auth"
	"mangahub/internal/manga"
	"mangahub/internal/user"
	chatws "mangahub/internal/websocket"
	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

type Dependencies struct {
	AuthService  *auth.Service
	MangaService *manga.Service
	UserService  *user.Service
	Hub          *chatws.Hub
}

type Handler struct {
	authService  *auth.Service
	mangaService *manga.Service
	userService  *user.Service
	hub          *chatws.Hub
}

func New(deps Dependencies) *Handler {
	return &Handler{
		authService:  deps.AuthService,
		mangaService: deps.MangaService,
		userService:  deps.UserService,
		hub:          deps.Hub,
	}
}

func (h *Handler) Health(c *gin.Context) {
	utils.OK(c, http.StatusOK, gin.H{"status": "ok"})
}

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

	utils.OK(c, http.StatusOK, entry)
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

func currentUserID(c *gin.Context) string {
	value, _ := c.Get(auth.ContextUserIDKey)
	userID, _ := value.(string)
	return userID
}

func isNotFound(err error) bool {
	return errors.Is(err, http.ErrMissingFile) || strings.Contains(strings.ToLower(err.Error()), "no rows")
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
