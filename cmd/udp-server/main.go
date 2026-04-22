package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"mangahub/internal/config"
	"mangahub/internal/udp"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := udp.NewServer(cfg.UDPAddr)
	if err := server.ListenAndServe(ctx); err != nil {
		log.Fatalf("udp server error: %v", err)
	}
}
