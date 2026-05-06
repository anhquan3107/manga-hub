package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mangahub/internal/auth"
	"mangahub/internal/chat"
	"mangahub/internal/manga"
	"mangahub/internal/review"
	"mangahub/internal/user"
	chatws "mangahub/internal/websocket"
	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

type ProgressBroadcaster interface {
	PublishProgress(update models.ProgressUpdate)
}

type Dependencies struct {
	AuthService  *auth.Service
	ChatService  *chat.Service
	MangaService *manga.Service
	ReviewService *review.Service
	UserService  *user.Service
	Hub          *chatws.Hub
	Broadcaster  ProgressBroadcaster
}

type Handler struct {
	authService  *auth.Service
	chatService  *chat.Service
	mangaService *manga.Service
	reviewService *review.Service
	userService  *user.Service
	hub          *chatws.Hub
	broadcaster  ProgressBroadcaster
}

func New(deps Dependencies) *Handler {
	return &Handler{
		authService:  deps.AuthService,
		chatService:  deps.ChatService,
		mangaService: deps.MangaService,
		reviewService: deps.ReviewService,
		userService:  deps.UserService,
		hub:          deps.Hub,
		broadcaster:  deps.Broadcaster,
	}
}

// Health godoc
// @Summary Health check
// @Description Returns API server health status.
// @Tags system
// @Produce json
// @Success 200 {object} healthResponse
// @Router /health [get]
func (h *Handler) Health(c *gin.Context) {
	utils.OK(c, http.StatusOK, gin.H{"status": "ok"})
}

func isNotFound(err error) bool {
	return errors.Is(err, http.ErrMissingFile) || strings.Contains(strings.ToLower(err.Error()), "no rows")
}
