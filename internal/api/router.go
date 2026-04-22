package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mangahub/internal/auth"
	"mangahub/internal/config"
	"mangahub/internal/manga"
	"mangahub/internal/user"
	chatws "mangahub/internal/websocket"
	"mangahub/pkg/models"
	"mangahub/pkg/utils"
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

	router.GET("/health", func(c *gin.Context) {
		utils.OK(c, http.StatusOK, gin.H{"status": "ok"})
	})

	router.POST("/auth/register", func(c *gin.Context) {
		var req models.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.Error(c, http.StatusBadRequest, err.Error())
			return
		}

		resp, err := authService.Register(c.Request.Context(), req)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "exists") {
				utils.Error(c, http.StatusConflict, err.Error())
				return
			}
			utils.Error(c, http.StatusBadRequest, err.Error())
			return
		}

		utils.OK(c, http.StatusCreated, resp)
	})

	router.POST("/auth/login", func(c *gin.Context) {
		var req models.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.Error(c, http.StatusBadRequest, err.Error())
			return
		}

		resp, err := authService.Login(c.Request.Context(), req)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, err.Error())
			return
		}

		utils.OK(c, http.StatusOK, resp)
	})

	router.GET("/manga", func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		items, err := mangaService.List(c.Request.Context(), models.MangaQuery{
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
	})

	router.GET("/manga/:id", func(c *gin.Context) {
		item, err := mangaService.GetByID(c.Request.Context(), c.Param("id"))
		if err != nil {
			if isNotFound(err) {
				utils.Error(c, http.StatusNotFound, "manga not found")
				return
			}
			utils.Error(c, http.StatusInternalServerError, "failed to fetch manga")
			return
		}

		utils.OK(c, http.StatusOK, item)
	})

	router.GET("/ws/chat", chatws.Handler(hub, authService))

	protected := router.Group("/users")
	protected.Use(auth.Middleware(authService))
	{
		protected.POST("/library", func(c *gin.Context) {
			var req models.AddLibraryRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.Error(c, http.StatusBadRequest, err.Error())
				return
			}

			entry, err := userService.AddToLibrary(c.Request.Context(), currentUserID(c), req)
			if err != nil {
				status := http.StatusInternalServerError
				if isNotFound(err) {
					status = http.StatusNotFound
				}
				utils.Error(c, status, err.Error())
				return
			}

			utils.OK(c, http.StatusCreated, entry)
		})

		protected.GET("/library", func(c *gin.Context) {
			items, err := userService.GetLibrary(c.Request.Context(), currentUserID(c))
			if err != nil {
				utils.Error(c, http.StatusInternalServerError, "failed to fetch library")
				return
			}

			utils.OK(c, http.StatusOK, gin.H{"items": items})
		})

		protected.PUT("/progress", func(c *gin.Context) {
			var req models.UpdateProgressRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				utils.Error(c, http.StatusBadRequest, err.Error())
				return
			}

			entry, err := userService.UpdateProgress(c.Request.Context(), currentUserID(c), req)
			if err != nil {
				utils.Error(c, http.StatusInternalServerError, err.Error())
				return
			}

			utils.OK(c, http.StatusOK, entry)
		})
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

func currentUserID(c *gin.Context) string {
	value, _ := c.Get(auth.ContextUserIDKey)
	userID, _ := value.(string)
	return userID
}

func isNotFound(err error) bool {
	return errors.Is(err, http.ErrMissingFile) || strings.Contains(strings.ToLower(err.Error()), "no rows")
}
