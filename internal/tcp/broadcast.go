package tcp

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"mangahub/pkg/models"
)

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
