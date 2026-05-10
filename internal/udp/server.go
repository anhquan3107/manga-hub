package udp

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)

type Server struct {
	addr    string
	mu      sync.RWMutex
	clients map[string]*registeredClient
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

		s.handlePacket(ctx, conn, clientAddr, buffer[:n])
	}
}
