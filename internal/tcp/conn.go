package tcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"mangahub/pkg/models"
)

func (s *Server) handleConn(ctx context.Context, c *client) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, c.id)
		s.mu.Unlock()
		_ = c.conn.Close()
		log.Printf("tcp client disconnected: id=%s user_id=%s", c.id, c.getUserID())
	}()

	log.Printf("tcp client connected: id=%s", c.id)

	scanner := bufio.NewScanner(c.conn)
	scanner.Buffer(make([]byte, 0, 4096), 64*1024)
	for scanner.Scan() {
		var msg clientMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			log.Printf("tcp invalid payload from %s: %v", c.id, err)
			_ = s.send(c, serverMessage{Type: "error", Error: "invalid json payload", Timestamp: time.Now().Unix()})
			continue
		}

		err := s.handleMessage(ctx, c, msg)
		if err != nil {
			log.Printf("tcp message handling error from %s: %v", c.id, err)
			_ = s.send(c, serverMessage{Type: "error", RequestID: msg.RequestID, Error: err.Error(), Timestamp: time.Now().Unix()})
		}
	}

	if err := scanner.Err(); err != nil {
		if !errors.Is(err, net.ErrClosed) {
			log.Printf("tcp scanner error from %s: %v", c.id, err)
		}
	}
}

func (s *Server) handleMessage(ctx context.Context, c *client, msg clientMessage) error {
	switch strings.ToLower(strings.TrimSpace(msg.Type)) {
	case "hello":
		if strings.TrimSpace(msg.UserID) == "" {
			return errors.New("user_id is required for hello")
		}

		c.setUserID(strings.TrimSpace(msg.UserID))
		userID := c.getUserID()

		sessionID := fmt.Sprintf("sess-%d", time.Now().UnixNano())
		username := userID
		if s.userService != nil {
			user, err := s.userService.GetUserByID(ctx, userID)
			if err == nil {
				username = user.Username
			}
		}
		// count devices for this user
		deviceCount := 0
		s.mu.RLock()
		clients := make([]*client, 0, len(s.clients))
		for _, client := range s.clients {
			clients = append(clients, client)
		}
		s.mu.RUnlock()
		for _, client := range clients {
			if client.getUserID() == userID {
				deviceCount++
			}
		}
		return s.send(c, serverMessage{
			Type:        "hello_ack",
			SessionID:   sessionID,
			UserID:      userID,
			Username:    username,
			Devices:     deviceCount,
			ConnectedAt: time.Now().Unix(),
			Timestamp:   time.Now().Unix(),
		})
	case "ping":
		return s.send(c, serverMessage{Type: "pong", RequestID: msg.RequestID, Timestamp: time.Now().Unix()})
	case "health":
		return s.send(c, serverMessage{Type: "health_ok", Message: "tcp server is healthy", Timestamp: time.Now().Unix()})
	case "progress":
		if s.userService == nil {
			return errors.New("progress service is not configured")
		}

		userID := strings.TrimSpace(msg.UserID)
		if userID == "" {
			userID = c.getUserID()
		}
		if userID == "" {
			return errors.New("user_id is required")
		}
		if strings.TrimSpace(msg.MangaID) == "" {
			return errors.New("manga_id is required")
		}
		if msg.Chapter < 0 {
			return errors.New("chapter must be >= 0")
		}

		incomingTimestamp := msg.Timestamp
		if incomingTimestamp == 0 {
			incomingTimestamp = time.Now().Unix()
		}

		strategy := strings.ToLower(strings.TrimSpace(msg.Strategy))
		if strategy == "" {
			strategy = "last_write_wins"
		}

		if s.userService != nil {
			if entry, err := s.userService.GetLibraryEntry(ctx, userID, strings.TrimSpace(msg.MangaID)); err == nil {
				if msg.Chapter < entry.CurrentChapter {
					resolution := "server_newer"
					if strategy == "user_choice" {
						resolution = "needs_user_choice"
					}

					return s.send(c, serverMessage{
						Type:      "conflict",
						RequestID: msg.RequestID,
						Message:   "conflict detected",
						Conflict: &models.ConflictResolution{
							Strategy:   strategy,
							Timestamp:  incomingTimestamp,
							DeviceID:   strings.TrimSpace(msg.DeviceID),
							Resolution: resolution,
						},
						Timestamp: time.Now().Unix(),
					})
				}

				currentTS := entry.UpdatedAt.Unix()
				if currentTS > incomingTimestamp {
					resolution := "server_newer"
					if strategy == "merge" && msg.Chapter > entry.CurrentChapter {
						resolution = "merged_forward"
					} else if strategy == "merge" {
						resolution = "server_newer"
					} else if strategy == "user_choice" {
						resolution = "needs_user_choice"
					}

					if strategy != "merge" || resolution == "server_newer" || resolution == "needs_user_choice" {
						return s.send(c, serverMessage{
							Type:      "conflict",
							RequestID: msg.RequestID,
							Message:   "conflict detected",
							Conflict: &models.ConflictResolution{
								Strategy:   strategy,
								Timestamp:  incomingTimestamp,
								DeviceID:   strings.TrimSpace(msg.DeviceID),
								Resolution: resolution,
							},
							Timestamp: time.Now().Unix(),
						})
					}
				}
			}
		}

		status := strings.TrimSpace(msg.Status)
		if status == "" {
			status = "reading"
		}

		result, err := s.userService.UpdateProgress(ctx, userID, models.UpdateProgressRequest{
			MangaID:        strings.TrimSpace(msg.MangaID),
			CurrentChapter: msg.Chapter,
			Status:         status,
		})
		if err != nil {
			return fmt.Errorf("update progress: %w", err)
		}

		update := models.ProgressUpdate{
			UserID:    userID,
			MangaID:   result.Entry.MangaID,
			Chapter:   result.Entry.CurrentChapter,
			Timestamp: time.Now().Unix(),
		}

		s.PublishProgress(update)
		return s.send(c, serverMessage{Type: "ack", RequestID: msg.RequestID, Message: "progress updated", Progress: &update, Timestamp: time.Now().Unix()})
	case "progress_broadcast":
		// Relay progress broadcast from remote broadcaster to all connected clients
		if msg.Progress == nil {
			return errors.New("progress data is required for broadcast")
		}
		log.Printf("tcp relaying progress broadcast: manga=%s user=%s chapter=%d", msg.Progress.MangaID, msg.Progress.UserID, msg.Progress.Chapter)
		s.broadcast(serverMessage{
			Type:      "progress_broadcast",
			Progress:  msg.Progress,
			Timestamp: msg.Progress.Timestamp,
		})
		return s.send(c, serverMessage{
			Type:      "ack",
			Message:   "broadcast relayed",
			Timestamp: time.Now().Unix(),
		})
	case "disconnect":
		log.Printf("user %s requested disconnect", c.getUserID())
		return s.send(c, serverMessage{
			Type:      "ack",
			Message:   "disconnected",
			Timestamp: time.Now().Unix(),
		})
	default:
		return fmt.Errorf("unsupported message type %q", msg.Type)
	}

}
