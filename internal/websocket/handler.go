package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"

	"mangahub/internal/auth"
	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

var upgrader = gorillaws.Upgrader{
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

type inboundMessage struct {
	Message string `json:"message"`
}

func Handler(hub *Hub, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			utils.Error(c, http.StatusUnauthorized, "missing websocket token")
			return
		}

		claims, err := authService.ParseToken(token)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, "invalid websocket token")
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			utils.Error(c, http.StatusBadRequest, "failed to upgrade connection")
			return
		}

		client := ClientConnection{
			Conn:     conn,
			UserID:   claims.UserID,
			Username: claims.Username,
		}
		hub.Register <- client

		for {
			var incoming inboundMessage
			if err := conn.ReadJSON(&incoming); err != nil {
				log.Printf("websocket read error: %v", err)
				hub.Unregister <- conn
				return
			}

			hub.Broadcast <- models.ChatMessage{
				UserID:    claims.UserID,
				Username:  claims.Username,
				Message:   incoming.Message,
				Timestamp: time.Now().Unix(),
			}
		}
	}
}
