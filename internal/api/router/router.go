package router

import (
	"github.com/gin-gonic/gin"

	handler "mangahub/internal/api/handler"
	"mangahub/internal/api/middleware"
	"mangahub/internal/auth"
	"mangahub/internal/config"
	"mangahub/internal/manga"
	"mangahub/internal/user"
	chatws "mangahub/internal/websocket"
)

func NewRouter(
	cfg config.Config,
	authService *auth.Service,
	mangaService *manga.Service,
	userService *user.Service,
	hub *chatws.Hub,
	progressBroadcaster handler.ProgressBroadcaster,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), middleware.CORS(cfg.AllowedOrigin))
	h := handler.New(handler.Dependencies{
		AuthService:  authService,
		MangaService: mangaService,
		UserService:  userService,
		Hub:          hub,
		Broadcaster:  progressBroadcaster,
	})

	router.GET("/health", h.Health)
	router.POST("/auth/register", h.Register)
	router.POST("/auth/login", h.Login)
	router.POST("/auth/logout", auth.Middleware(authService), h.Logout)
	router.POST("/auth/change-password", auth.Middleware(authService), h.ChangePassword)
	router.GET("/manga", h.ListManga)
	router.GET("/manga/:id", h.GetManga)
	router.GET("/ws/chat", h.Chat)

	protected := router.Group("/users")
	protected.Use(auth.Middleware(authService))
	{
		protected.GET("/me", h.GetMe)
		protected.POST("/library", h.AddToLibrary)
		protected.GET("/library", h.GetLibrary)
		protected.PUT("/progress", h.UpdateProgress)
		protected.DELETE("/library/:id", h.RemoveFromLibrary)
	}

	mangaProtected := router.Group("/manga")
	mangaProtected.Use(auth.Middleware(authService))
	{
		mangaProtected.POST("", h.CreateManga)
		mangaProtected.PUT("/:id", h.UpdateManga)
		mangaProtected.DELETE("/:id", h.DeleteManga)
	}

	return router
}
