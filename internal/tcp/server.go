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
	"sync"
	"time"

	"mangahub/internal/user"
	"mangahub/pkg/models"
)

type Server struct {
	addr        string
	userService *user.Service

	mu      sync.RWMutex
	clients map[string]*client
}

type client struct {
	id     string
	conn   net.Conn
	userID string
	mu     sync.Mutex
}

type clientMessage struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	MangaID   string `json:"manga_id,omitempty"`
	Chapter   int    `json:"chapter,omitempty"`
	Status    string `json:"status,omitempty"`
}

type serverMessage struct {
	Type        string                 `json:"type"`
	RequestID   string                 `json:"request_id,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Progress    *models.ProgressUpdate `json:"progress,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	Username    string                 `json:"username,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	Devices     int                    `json:"devices,omitempty"`
	ConnectedAt int64                  `json:"connected_at,omitempty"`
	Timestamp   int64                  `json:"timestamp"`
}

func NewServer(addr string, userService *user.Service) *Server {
	return &Server{
		addr:        addr,
		userService: userService,
		clients:     make(map[string]*client),
	}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen tcp: %w", err)
	}
	defer listener.Close()

	log.Printf("tcp server listening on %s", s.addr)

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("accept tcp connection: %w", err)
			}
		}

		clientID := conn.RemoteAddr().String()
		c := &client{id: clientID, conn: conn}

		s.mu.Lock()
		s.clients[clientID] = c
		s.mu.Unlock()

		go s.handleConn(ctx, c)
	}
}

func (s *Server) handleConn(ctx context.Context, c *client) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, c.id)
		s.mu.Unlock()
		_ = c.conn.Close()
		log.Printf("tcp client disconnected: id=%s user_id=%s", c.id, c.userID)
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

		c.userID = strings.TrimSpace(msg.UserID)

		sessionID := fmt.Sprintf("sess-%d", time.Now().UnixNano())
		username := c.userID
		if s.userService != nil {
			user, err := s.userService.GetUserByID(ctx, c.userID)
			if err == nil {
				username = user.Username
			}
		}
		// count devices for this user
		deviceCount := 0
		for _, client := range s.clients {
			if client.userID == c.userID {
				deviceCount++
			}
		}
		return s.send(c, serverMessage{
			Type:        "connected",
			Message:     "connected successfully",
			SessionID:   sessionID,
			UserID:      c.userID,
			Username:    username,
			Devices:     deviceCount,
			ConnectedAt: time.Now().Unix(),
			Timestamp:   time.Now().Unix(),
		})
	case "ping":
		return s.send(c, serverMessage{Type: "pong", RequestID: msg.RequestID, Timestamp: time.Now().Unix()})
	case "progress":
		if s.userService == nil {
			return errors.New("progress service is not configured")
		}

		userID := strings.TrimSpace(msg.UserID)
		if userID == "" {
			userID = strings.TrimSpace(c.userID)
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
	case "disconnect":
		log.Printf("user %s requested disconnect", c.userID)
		return s.send(c, serverMessage{
			Type:      "ack",
			Message:   "disconnected",
			Timestamp: time.Now().Unix(),
		})
	default:
		return fmt.Errorf("unsupported message type %q", msg.Type)
	}

}

func (s *Server) PublishProgress(update models.ProgressUpdate) {
	if update.Timestamp == 0 {
		update.Timestamp = time.Now().Unix()
	}

	s.broadcast(serverMessage{
		Type:      "progress_broadcast",
		Progress:  &update,
		Timestamp: update.Timestamp,
	})
}

func (s *Server) broadcast(msg serverMessage) {
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("tcp marshal error: %v", err)
		return
	}
	payload = append(payload, '\n')

	s.mu.RLock()
	clients := make([]*client, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.RUnlock()

	for _, c := range clients {
		if err := s.sendRaw(c, payload); err != nil {
			log.Printf("tcp broadcast error to %s: %v", c.id, err)
		}
	}
}

func (s *Server) send(c *client, msg serverMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	payload = append(payload, '\n')
	return s.sendRaw(c, payload)
}

func (s *Server) sendRaw(c *client, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	_, err := c.conn.Write(payload)
	if err != nil {
		return err
	}
	return c.conn.SetWriteDeadline(time.Time{})
}
