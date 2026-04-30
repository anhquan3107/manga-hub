package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mangahub/internal/auth"
	"mangahub/internal/manga"
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
	MangaService *manga.Service
	UserService  *user.Service
	Hub          *chatws.Hub
	Broadcaster  ProgressBroadcaster
}

type Handler struct {
	authService  *auth.Service
	mangaService *manga.Service
	userService  *user.Service
	hub          *chatws.Hub
	broadcaster  ProgressBroadcaster
}

func New(deps Dependencies) *Handler {
	return &Handler{
		authService:  deps.AuthService,
		mangaService: deps.MangaService,
		userService:  deps.UserService,
		hub:          deps.Hub,
		broadcaster:  deps.Broadcaster,
	}
}

func (h *Handler) Health(c *gin.Context) {
	utils.OK(c, http.StatusOK, gin.H{"status": "ok"})
}

func isNotFound(err error) bool {
	return errors.Is(err, http.ErrMissingFile) || strings.Contains(strings.ToLower(err.Error()), "no rows")
}
