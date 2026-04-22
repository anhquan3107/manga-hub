package udp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"mangahub/pkg/models"
)

type Server struct {
	addr    string
	mu      sync.RWMutex
	clients map[string]*net.UDPAddr
}

func NewServer(addr string) *Server {
	return &Server{
		addr:    addr,
		clients: make(map[string]*net.UDPAddr),
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

		var notification models.Notification
		if err := json.Unmarshal(buffer[:n], &notification); err != nil {
			log.Printf("udp invalid payload from %s: %v", clientAddr.String(), err)
			continue
		}

		switch notification.Type {
		case "register":
			clientID := notification.ClientID
			if clientID == "" {
				clientID = clientAddr.String()
			}
			s.mu.Lock()
			s.clients[clientID] = clientAddr
			s.mu.Unlock()
			log.Printf("udp registered client %s", clientID)
		case "notify":
			if notification.Timestamp == 0 {
				notification.Timestamp = time.Now().Unix()
			}
			s.broadcast(conn, notification)
		default:
			log.Printf("udp unknown message type: %s", notification.Type)
		}
	}
}

func (s *Server) broadcast(conn *net.UDPConn, notification models.Notification) {
	payload, err := json.Marshal(notification)
	if err != nil {
		log.Printf("udp marshal error: %v", err)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for clientID, addr := range s.clients {
		if _, err := conn.WriteToUDP(payload, addr); err != nil {
			log.Printf("udp broadcast error to %s: %v", clientID, err)
		}
	}
}
