package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"mangahub/internal/config"
	grpcservice "mangahub/internal/grpc"
	"mangahub/internal/manga"
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

	mangaService := manga.NewService(store)
	userService := user.NewService(store)
	server := grpcservice.New(cfg.GRPCAddr, mangaService, userService)

	log.Printf("grpc server listening on %s", cfg.GRPCAddr)
	if err := server.Start(ctx); err != nil {
		log.Fatalf("grpc server error: %v", err)
	}
}
