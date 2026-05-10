package tcp

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"mangahub/internal/user"
)

type Server struct {
	addr        string
	userService *user.Service

	mu      sync.RWMutex
	clients map[string]*client
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
