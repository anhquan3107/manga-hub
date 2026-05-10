package udp

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"strings"
	"time"

	"mangahub/pkg/models"
)

func (s *Server) handlePacket(ctx context.Context, conn *net.UDPConn, clientAddr *net.UDPAddr, payload []byte) {
	var msg clientMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		log.Printf("udp invalid payload from %s: %v", clientAddr.String(), err)
		return
	}

	s.handleMessage(ctx, conn, clientAddr, msg)
}

func (s *Server) handleMessage(ctx context.Context, conn *net.UDPConn, clientAddr *net.UDPAddr, msg clientMessage) {
	switch strings.ToLower(strings.TrimSpace(msg.Type)) {
	case "register", "subscribe":
		// CLI sends "client" while some tests send "client_id" — prefer either
		clientID := strings.TrimSpace(msg.ClientID)
		if clientID == "" {
			clientID = strings.TrimSpace(msg.Client)
		}
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
			clientID = strings.TrimSpace(msg.Client)
		}
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
			return
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
	case "health":
		_ = s.send(conn, clientAddr, serverMessage{
			Type:      "health_ok",
			Message:   "udp server is healthy",
			Timestamp: time.Now().Unix(),
		})
	case "unsubscribe", "unregister":
		clientID := strings.TrimSpace(msg.ClientID)
		if clientID == "" {
			clientID = strings.TrimSpace(msg.Client)
		}
		if clientID == "" {
			clientID = clientAddr.String()
		}
		// remove from registered clients
		s.mu.Lock()
		if _, ok := s.clients[clientID]; ok {
			delete(s.clients, clientID)
			s.mu.Unlock()
			log.Printf("udp unregistered client %s", clientID)
			_ = s.send(conn, clientAddr, serverMessage{
				Type:      "unregister_ack",
				ClientID:  clientID,
				Message:   "client unregistered",
				Timestamp: time.Now().Unix(),
			})
		} else {
			s.mu.Unlock()
			_ = s.send(conn, clientAddr, serverMessage{
				Type:      "error",
				ClientID:  clientID,
				Error:     "client not registered",
				Timestamp: time.Now().Unix(),
			})
		}
	case "test":
		clientID := strings.TrimSpace(msg.ClientID)
		if clientID == "" {
			clientID = strings.TrimSpace(msg.Client)
		}
		if clientID == "" {
			clientID = clientAddr.String()
		}

		if !s.isRegistered(clientID) {
			_ = s.send(conn, clientAddr, serverMessage{
				Type:      "error",
				ClientID:  clientID,
				Error:     "client must be registered to test",
				Timestamp: time.Now().Unix(),
			})
			log.Printf("udp test rejected for unregistered client %s", clientID)
			return
		}

		// respond with an OK so CLI can report success
		_ = s.send(conn, clientAddr, serverMessage{
			Type:      "ok",
			ClientID:  clientID,
			Message:   "test received",
			Timestamp: time.Now().Unix(),
		})
		log.Printf("udp test passed for client %s", clientID)
	case "test_broadcast":
		notification := models.Notification{
			Type:      "notification",
			ClientID:  "server",
			MangaID:   strings.TrimSpace(msg.MangaID),
			Message:   "test broadcast received",
			Timestamp: time.Now().Unix(),
		}

		if err := s.broadcast(conn, notification); err != nil {
			log.Printf("udp test broadcast error: %v", err)
			_ = s.send(conn, clientAddr, serverMessage{
				Type:      "error",
				Error:     err.Error(),
				Timestamp: time.Now().Unix(),
			})
			return
		}

		_ = s.send(conn, clientAddr, serverMessage{
			Type:      "ok",
			Message:   "broadcast sent to all registered clients",
			Timestamp: time.Now().Unix(),
		})
		log.Printf("udp test broadcast sent for manga %s", strings.TrimSpace(msg.MangaID))
	default:
		log.Printf("udp unknown message type: %s", msg.Type)
	}
}
