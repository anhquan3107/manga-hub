package tcp

import (
	"bufio"
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
	addr  string
	mu    sync.RWMutex
	conns map[string]net.Conn
}

func NewServer(addr string) *Server {
	return &Server{
		addr:  addr,
		conns: make(map[string]net.Conn),
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
		s.mu.Lock()
		s.conns[clientID] = conn
		s.mu.Unlock()

		go s.handleConn(clientID, conn)
	}
}

func (s *Server) handleConn(clientID string, conn net.Conn) {
	defer func() {
		s.mu.Lock()
		delete(s.conns, clientID)
		s.mu.Unlock()
		_ = conn.Close()
	}()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var update models.ProgressUpdate
		if err := json.Unmarshal(scanner.Bytes(), &update); err != nil {
			log.Printf("tcp invalid payload from %s: %v", clientID, err)
			continue
		}

		if update.Timestamp == 0 {
			update.Timestamp = time.Now().Unix()
		}
		if update.UserID == "" {
			update.UserID = clientID
		}

		s.broadcast(update)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("tcp scanner error from %s: %v", clientID, err)
	}
}

func (s *Server) broadcast(update models.ProgressUpdate) {
	payload, err := json.Marshal(update)
	if err != nil {
		log.Printf("tcp marshal error: %v", err)
		return
	}
	payload = append(payload, '\n')

	s.mu.RLock()
	defer s.mu.RUnlock()

	for clientID, conn := range s.conns {
		if _, err := conn.Write(payload); err != nil {
			log.Printf("tcp broadcast error to %s: %v", clientID, err)
		}
	}
}
