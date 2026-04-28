package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"mangahub/internal/config"
	"mangahub/internal/tcp"
	"mangahub/internal/user"
	"mangahub/pkg/database"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := database.NewSQLiteStore(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("database setup failed: %v", err)
	}
	defer store.Close()

	if err := store.InitSchema(ctx); err != nil {
		log.Fatalf("database schema init failed: %v", err)
	}
	if err := store.SeedMangaFromJSON(ctx, cfg.SeedFile); err != nil {
		log.Fatalf("manga seed failed: %v", err)
	}

	userService := user.NewService(store)
	server := tcp.NewServer(cfg.TCPAddr, userService)
	if err := server.ListenAndServe(ctx); err != nil {
		log.Fatalf("tcp server error: %v", err)
	}
}
