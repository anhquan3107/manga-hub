package udp

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"mangahub/pkg/models"
)

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
