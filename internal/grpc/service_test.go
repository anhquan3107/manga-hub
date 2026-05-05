package grpcservice

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"mangahub/internal/manga"
	"mangahub/internal/user"
	"mangahub/pkg/database"
	"mangahub/pkg/models"
	pb "mangahub/proto"
)

func TestGRPCMangaService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, mangaSvc, userSvc := setupTestServices(t)
	defer store.Close()

	// Seed some manga data
	_ = store.InsertManga(ctx, models.Manga{
		ID:            "manga-grpc-1",
		Title:         "gRPC Manga",
		Author:        "gRPC Author",
		Genres:        []string{"Action", "Fantasy"},
		Status:        "ongoing",
		TotalChapters: 100,
	})

	server, clientConn := startTestGRPCServer(t, ctx, mangaSvc, userSvc)
	defer server.Stop()
	defer clientConn.Close()

	mangaClient := pb.NewMangaServiceClient(clientConn)

	t.Run("GetManga - Success", func(t *testing.T) {
		resp, err := mangaClient.GetManga(ctx, &pb.GetMangaRequest{Id: "manga-grpc-1"})
		if err != nil {
			t.Fatalf("GetManga failed: %v", err)
		}
		if resp.Manga == nil || resp.Manga.Title != "gRPC Manga" {
			t.Errorf("Expected title 'gRPC Manga', got %v", resp.Manga)
		}
	})

	t.Run("GetManga - Not Found", func(t *testing.T) {
		_, err := mangaClient.GetManga(ctx, &pb.GetMangaRequest{Id: "missing-manga"})
		if err == nil {
			t.Fatal("Expected error for missing manga, got nil")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.NotFound {
			t.Errorf("Expected NotFound error code, got %v", err)
		}
	})

	t.Run("GetManga - Empty ID", func(t *testing.T) {
		_, err := mangaClient.GetManga(ctx, &pb.GetMangaRequest{Id: ""})
		if err == nil {
			t.Fatal("Expected error for empty ID, got nil")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument error code, got %v", err)
		}
	})

	t.Run("SearchManga - Success", func(t *testing.T) {
		resp, err := mangaClient.SearchManga(ctx, &pb.SearchRequest{Query: "gRPC", Limit: 10})
		if err != nil {
			t.Fatalf("SearchManga failed: %v", err)
		}
		if len(resp.Items) == 0 || resp.Items[0].Id != "manga-grpc-1" {
			t.Errorf("Expected manga-grpc-1 in search results, got %v", resp.Items)
		}
	})
}

func TestGRPCUserService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, mangaSvc, userSvc := setupTestServices(t)
	defer store.Close()

	// Seed user data
	_, err := store.CreateUser(ctx, "user-grpc-1", "grpcuser", "grpc@example.com", "hash")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	server, clientConn := startTestGRPCServer(t, ctx, mangaSvc, userSvc)
	defer server.Stop()
	defer clientConn.Close()

	userClient := pb.NewUserServiceClient(clientConn)

	t.Run("GetUser - By ID", func(t *testing.T) {
		resp, err := userClient.GetUser(ctx, &pb.GetUserRequest{UserId: "user-grpc-1"})
		if err != nil {
			t.Fatalf("GetUser failed: %v", err)
		}
		if resp.User == nil || resp.User.Username != "grpcuser" {
			t.Errorf("Expected username 'grpcuser', got %v", resp.User)
		}
	})

	t.Run("GetUser - Not Found", func(t *testing.T) {
		_, err := userClient.GetUser(ctx, &pb.GetUserRequest{UserId: "non-existent"})
		if err == nil {
			t.Fatal("Expected error for non-existent user, got nil")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.NotFound {
			t.Errorf("Expected NotFound error code, got %v", err)
		}
	})
}

func TestGRPCProgressUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, mangaSvc, userSvc := setupTestServices(t)
	defer store.Close()

	// Seed user and manga
	_, _ = store.CreateUser(ctx, "user-grpc-1", "grpcuser", "grpc@example.com", "hash")
	_ = store.InsertManga(ctx, models.Manga{
		ID:            "manga-grpc-1",
		Title:         "gRPC Manga",
		TotalChapters: 100,
	})
	_, _ = store.UpsertLibraryEntry(ctx, "user-grpc-1", models.LibraryEntry{
		MangaID: "manga-grpc-1",
		Status:  "reading",
	})

	server, clientConn := startTestGRPCServer(t, ctx, mangaSvc, userSvc)
	defer server.Stop()
	defer clientConn.Close()

	mangaClient := pb.NewMangaServiceClient(clientConn)

	t.Run("UpdateProgress - Success", func(t *testing.T) {
		resp, err := mangaClient.UpdateProgress(ctx, &pb.ProgressRequest{
			UserId:  "user-grpc-1",
			MangaId: "manga-grpc-1",
			Chapter: 5,
		})
		if err != nil {
			t.Fatalf("UpdateProgress failed: %v", err)
		}
		if resp.Result == nil || resp.Result.Entry.CurrentChapter != 5 {
			t.Errorf("Expected current chapter 5, got %v", resp.Result)
		}
	})

	t.Run("UpdateProgress - Missing Args", func(t *testing.T) {
		_, err := mangaClient.UpdateProgress(ctx, &pb.ProgressRequest{
			UserId: "user-grpc-1",
		})
		if err == nil {
			t.Fatal("Expected error for missing manga ID")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument error code, got %v", err)
		}
	})
}

// Helper to set up SQLite and services
func setupTestServices(t *testing.T) (*database.Store, *manga.Service, *user.Service) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "grpc-test.db")
	store, err := database.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}
	return store, manga.NewService(store), user.NewService(store)
}

// Helper to start server and get a client connection
func startTestGRPCServer(t *testing.T, ctx context.Context, mangaSvc *manga.Service, userSvc *user.Service) (*Server, *grpc.ClientConn) {
	t.Helper()

	// Find free port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	addr := l.Addr().String()
	l.Close() // Server struct will re-listen

	server := New(addr, mangaSvc, userSvc)
	go func() {
		_ = server.Start(ctx)
	}()

	// Wait briefly for server to start
	time.Sleep(50 * time.Millisecond)

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial gRPC server: %v", err)
	}

	return server, conn
}
