package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"mangahub/internal/config"
	"mangahub/internal/grpcserver"
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

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("listen grpc: %v", err)
	}

	server := grpc.NewServer(grpcserver.ServerOptions()...)
	grpcserver.Register(server, grpcserver.NewService(store))

	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	log.Printf("grpc server listening on %s", cfg.GRPCAddr)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("grpc serve failed: %v", err)
	}
}
