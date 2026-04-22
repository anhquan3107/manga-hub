package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"mangahub/internal/config"
	"mangahub/internal/tcp"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := tcp.NewServer(cfg.TCPAddr)
	if err := server.ListenAndServe(ctx); err != nil {
		log.Fatalf("tcp server error: %v", err)
	}
}
