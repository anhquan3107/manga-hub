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
	grpcservice "mangahub/internal/grpc"
	"mangahub/internal/manga"
	"mangahub/internal/tcp"
	"mangahub/internal/udp"
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
	userService := user.NewService(store)
	tcpServer := tcp.NewServer(cfg.TCPAddr, userService)
	grpcServer := grpcservice.New(cfg.GRPCAddr, mangaService, userService)
	udpServer := udp.NewServer(cfg.UDPAddr)
	hub := websocket.NewHub()

	go hub.Run(ctx)
	go func() {
		if err := tcpServer.ListenAndServe(ctx); err != nil {
			log.Printf("tcp server error: %v", err)
		}
	}()
	go func() {
		log.Printf("grpc server listening on %s", cfg.GRPCAddr)
		if err := grpcServer.Start(ctx); err != nil {
			log.Printf("grpc server error: %v", err)
		}
	}()
	go func() {
		if err := udpServer.ListenAndServe(ctx); err != nil {
			log.Printf("udp server error: %v", err)
		}
	}()

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: router.NewRouter(cfg, authService, chatService, mangaService, userService, hub, tcpServer),
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
	grpcServer.Stop()
}
