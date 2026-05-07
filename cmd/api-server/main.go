package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "mangahub/docs/swagger"
	"mangahub/internal/api/router"
	"mangahub/internal/auth"
	"mangahub/internal/chat"
	"mangahub/internal/config"
	"mangahub/internal/manga"
	"mangahub/internal/review"
	"mangahub/internal/tcp"
	"mangahub/internal/user"
	"mangahub/internal/websocket"
	"mangahub/pkg/database"
)

// @title MangaHub API
// @version 1.0
// @description MangaHub REST API documentation.
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

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

	authService := auth.NewService(store, cfg.JWTSecret)
	chatService := chat.NewService(store)
	mangaService := manga.NewService(store)
	reviewService := review.NewService(store)
	userService := user.NewService(store)
	broadcaster := tcp.NewRemoteBroadcaster(cfg.TCPServerAddr)
	hub := websocket.NewHub()

	go hub.Run(ctx)

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: router.NewRouter(cfg, authService, chatService, mangaService, reviewService, userService, hub, broadcaster),
	}

	go func() {
		log.Printf("http api listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}
}
