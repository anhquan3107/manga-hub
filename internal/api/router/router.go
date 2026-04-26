package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mangahub/internal/api/controller"
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
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), corsMiddleware(cfg.AllowedOrigin))
	h := controller.New(controller.Dependencies{
		AuthService:  authService,
		MangaService: mangaService,
		UserService:  userService,
		Hub:          hub,
	})

	router.GET("/health", h.Health)
	router.POST("/auth/register", h.Register)
	router.POST("/auth/login", h.Login)
	router.GET("/manga", h.ListManga)
	router.GET("/manga/:id", h.GetManga)
	router.GET("/ws/chat", h.Chat)

	protected := router.Group("/users")
	protected.Use(auth.Middleware(authService))
	{
		protected.POST("/library", h.AddToLibrary)
		protected.GET("/library", h.GetLibrary)
		protected.PUT("/progress", h.UpdateProgress)
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

func corsMiddleware(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
