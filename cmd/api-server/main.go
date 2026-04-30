package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mangahub/internal/api/router"
	"mangahub/internal/auth"
	"mangahub/internal/config"
	"mangahub/internal/manga"
	"mangahub/internal/tcp"
	"mangahub/internal/user"
	"mangahub/internal/websocket"
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

	authService := auth.NewService(store, cfg.JWTSecret)
	mangaService := manga.NewService(store)
	userService := user.NewService(store)
	tcpServer := tcp.NewServer(cfg.TCPAddr, userService)
	hub := websocket.NewHub()

	go hub.Run(ctx)
	go func() {
		if err := tcpServer.ListenAndServe(ctx); err != nil {
			log.Printf("tcp server error: %v", err)
		}
	}()

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: router.NewRouter(cfg, authService, mangaService, userService, hub, tcpServer),
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
