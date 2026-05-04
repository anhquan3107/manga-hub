package udp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"mangahub/pkg/models"
)

type Server struct {
	addr    string
	mu      sync.RWMutex
	clients map[string]*registeredClient
}

type registeredClient struct {
	ID        string
	Addr      *net.UDPAddr
	LastSeen  time.Time
	Connected time.Time
}

type clientMessage struct {
	Type      string `json:"type"`
	ClientID  string `json:"client_id,omitempty"`
	MangaID   string `json:"manga_id,omitempty"`
	Message   string `json:"message,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

type serverMessage struct {
	Type      string               `json:"type"`
	ClientID  string               `json:"client_id,omitempty"`
	Message   string               `json:"message,omitempty"`
	Error     string               `json:"error,omitempty"`
	Payload   *models.Notification `json:"payload,omitempty"`
	Timestamp int64                `json:"timestamp"`
}

func NewServer(addr string) *Server {
	return &Server{
		addr:    addr,
		clients: make(map[string]*registeredClient),
	}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	udpAddr, err := net.ResolveUDPAddr("udp", s.addr)
	if err != nil {
		return fmt.Errorf("resolve udp addr: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("listen udp: %w", err)
	}
	defer conn.Close()

	log.Printf("udp server listening on %s", s.addr)

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	buffer := make([]byte, 2048)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("read udp packet: %w", err)
			}
		}

		var msg clientMessage
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			log.Printf("udp invalid payload from %s: %v", clientAddr.String(), err)
			continue
		}

		switch strings.ToLower(strings.TrimSpace(msg.Type)) {
		case "register":
			clientID := strings.TrimSpace(msg.ClientID)
			if clientID == "" {
				clientID = clientAddr.String()
			}

			now := time.Now()
			s.mu.Lock()
			s.clients[clientID] = &registeredClient{ID: clientID, Addr: clientAddr, LastSeen: now, Connected: now}
			s.mu.Unlock()

			log.Printf("udp registered client %s from %s", clientID, clientAddr.String())
			_ = s.send(conn, clientAddr, serverMessage{
				Type:      "register_ack",
				ClientID:  clientID,
				Message:   "client registered",
				Timestamp: now.Unix(),
			})
		case "notify":
			clientID := strings.TrimSpace(msg.ClientID)
			if clientID == "" {
				clientID = clientAddr.String()
			}

			if !s.isRegistered(clientID) {
				_ = s.send(conn, clientAddr, serverMessage{
					Type:      "error",
					ClientID:  clientID,
					Error:     "client must register before sending notifications",
					Timestamp: time.Now().Unix(),
				})
				log.Printf("udp notify rejected for unregistered client %s", clientID)
				continue
			}

			notification := models.Notification{
				Type:      "notification",
				ClientID:  clientID,
				MangaID:   strings.TrimSpace(msg.MangaID),
				Message:   strings.TrimSpace(msg.Message),
				Timestamp: msg.Timestamp,
			}
			if notification.Timestamp == 0 {
				notification.Timestamp = time.Now().Unix()
			}

			if err := s.broadcast(conn, notification); err != nil {
				log.Printf("udp broadcast error: %v", err)
			}
		default:
			log.Printf("udp unknown message type: %s", msg.Type)
		}
	}
}

func (s *Server) broadcast(conn *net.UDPConn, notification models.Notification) error {
	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	s.mu.RLock()
	clients := make([]*registeredClient, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	s.mu.RUnlock()

	for _, client := range clients {
		if _, err := conn.WriteToUDP(payload, client.Addr); err != nil {
			log.Printf("udp broadcast error to %s: %v", client.ID, err)
		}
	}

	return nil
}

func (s *Server) send(conn *net.UDPConn, addr *net.UDPAddr, msg serverMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	_, err = conn.WriteToUDP(payload, addr)
	return err
}

func (s *Server) isRegistered(clientID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.clients[clientID]
	return ok
}
