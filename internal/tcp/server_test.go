package tcp

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"

	"mangahub/internal/user"
	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

type testServerMessage struct {
	Type      string                 `json:"type"`
	RequestID string                 `json:"request_id"`
	Error     string                 `json:"error"`
	Message   string                 `json:"message"`
	Progress  *models.ProgressUpdate `json:"progress"`
}

func TestTCPProgressFlowAndBroadcast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, userService := setupStoreAndService(t)
	defer store.Close()

	seedUserAndManga(t, store)

	addr := freeTCPAddr(t)
	server := NewServer(addr, userService)

	go func() {
		_ = server.ListenAndServe(ctx)
	}()

	connA := dialWithRetry(t, addr)
	defer connA.Close()
	readerA := bufio.NewReader(connA)

	connB := dialWithRetry(t, addr)
	defer connB.Close()
	readerB := bufio.NewReader(connB)

	_ = readServerMessage(t, connA, readerA) // connected
	_ = readServerMessage(t, connB, readerB) // connected

	writeJSONLine(t, connA, map[string]any{
		"type":    "hello",
		"user_id": "user-1",
	})
	ack := readServerMessage(t, connA, readerA)
	if ack.Type != "hello_ack" {
		t.Fatalf("expected hello_ack, got %s", ack.Type)
	}

	writeJSONLine(t, connA, map[string]any{
		"type":     "progress",
		"manga_id": "manga-1",
		"chapter":  7,
		"status":   "reading",
	})

	broadcastA := readUntilType(t, connA, readerA, "progress_broadcast")
	if broadcastA.Progress == nil {
		t.Fatalf("expected progress payload in broadcast to sender")
	}
	if broadcastA.Progress.Chapter != 7 {
		t.Fatalf("expected chapter 7 in sender broadcast, got %d", broadcastA.Progress.Chapter)
	}

	broadcastB := readUntilType(t, connB, readerB, "progress_broadcast")
	if broadcastB.Progress == nil {
		t.Fatalf("expected progress payload in broadcast to listener")
	}
	if broadcastB.Progress.UserID != "user-1" {
		t.Fatalf("expected user-1 in listener broadcast, got %s", broadcastB.Progress.UserID)
	}

	library, err := userService.GetLibrary(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get library: %v", err)
	}
	if len(library) != 1 {
		t.Fatalf("expected 1 library item, got %d", len(library))
	}
	if library[0].CurrentChapter != 7 {
		t.Fatalf("expected persisted chapter 7, got %d", library[0].CurrentChapter)
	}
}

func TestTCPReturnsErrorForInvalidProgress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, userService := setupStoreAndService(t)
	defer store.Close()

	seedUserAndManga(t, store)

	addr := freeTCPAddr(t)
	server := NewServer(addr, userService)

	go func() {
		_ = server.ListenAndServe(ctx)
	}()

	conn := dialWithRetry(t, addr)
	defer conn.Close()
	reader := bufio.NewReader(conn)
	_ = readServerMessage(t, conn, reader) // connected

	writeJSONLine(t, conn, map[string]any{
		"type":     "progress",
		"manga_id": "",
		"chapter":  3,
	})

	errMsg := readUntilType(t, conn, reader, "error")
	if errMsg.Error == "" {
		t.Fatalf("expected non-empty error message")
	}
}

func TestTCPPingReturnsPong(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, userService := setupStoreAndService(t)
	defer store.Close()

	addr := freeTCPAddr(t)
	server := NewServer(addr, userService)

	go func() {
		_ = server.ListenAndServe(ctx)
	}()

	conn := dialWithRetry(t, addr)
	defer conn.Close()
	reader := bufio.NewReader(conn)
	_ = readServerMessage(t, conn, reader) // connected

	writeJSONLine(t, conn, map[string]any{
		"type":       "ping",
		"request_id": "r-ping",
	})

	pong := readUntilType(t, conn, reader, "pong")
	if pong.RequestID != "r-ping" {
		t.Fatalf("expected request_id r-ping, got %s", pong.RequestID)
	}
}

func TestTCPUnsupportedMessageTypeReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, userService := setupStoreAndService(t)
	defer store.Close()

	addr := freeTCPAddr(t)
	server := NewServer(addr, userService)

	go func() {
		_ = server.ListenAndServe(ctx)
	}()

	conn := dialWithRetry(t, addr)
	defer conn.Close()
	reader := bufio.NewReader(conn)
	_ = readServerMessage(t, conn, reader) // connected

	writeJSONLine(t, conn, map[string]any{
		"type":       "status",
		"request_id": "r-status",
	})

	errMsg := readUntilType(t, conn, reader, "error")
	if errMsg.RequestID != "r-status" {
		t.Fatalf("expected request_id r-status, got %s", errMsg.RequestID)
	}
	if errMsg.Error == "" {
		t.Fatalf("expected error for unsupported message type")
	}
}

func TestTCPProgressWithInlineUserIDWithoutHello(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, userService := setupStoreAndService(t)
	defer store.Close()

	seedUserAndManga(t, store)

	addr := freeTCPAddr(t)
	server := NewServer(addr, userService)

	go func() {
		_ = server.ListenAndServe(ctx)
	}()

	conn := dialWithRetry(t, addr)
	defer conn.Close()
	reader := bufio.NewReader(conn)
	_ = readServerMessage(t, conn, reader) // connected

	writeJSONLine(t, conn, map[string]any{
		"type":       "progress",
		"request_id": "r-progress",
		"user_id":    "user-1",
		"manga_id":   "manga-1",
		"chapter":    9,
		"status":     "reading",
	})

	ack := readUntilType(t, conn, reader, "ack")
	if ack.RequestID != "r-progress" {
		t.Fatalf("expected request_id r-progress, got %s", ack.RequestID)
	}
	if ack.Progress == nil || ack.Progress.Chapter != 9 {
		t.Fatalf("expected ack progress chapter 9")
	}
}

func TestPublishProgressBroadcastsToAllClients(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, userService := setupStoreAndService(t)
	defer store.Close()

	addr := freeTCPAddr(t)
	server := NewServer(addr, userService)

	go func() {
		_ = server.ListenAndServe(ctx)
	}()

	connA := dialWithRetry(t, addr)
	defer connA.Close()
	readerA := bufio.NewReader(connA)
	_ = readServerMessage(t, connA, readerA) // connected

	connB := dialWithRetry(t, addr)
	defer connB.Close()
	readerB := bufio.NewReader(connB)
	_ = readServerMessage(t, connB, readerB) // connected

	server.PublishProgress(models.ProgressUpdate{
		UserID:  "user-1",
		MangaID: "manga-1",
		Chapter: 10,
	})

	broadcastA := readUntilType(t, connA, readerA, "progress_broadcast")
	broadcastB := readUntilType(t, connB, readerB, "progress_broadcast")

	if broadcastA.Progress == nil || broadcastA.Progress.Chapter != 10 {
		t.Fatalf("expected chapter 10 in client A broadcast")
	}
	if broadcastB.Progress == nil || broadcastB.Progress.Chapter != 10 {
		t.Fatalf("expected chapter 10 in client B broadcast")
	}
}

func setupStoreAndService(t *testing.T) (*database.Store, *user.Service) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "tcp-test.db")
	store, err := database.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}

	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	return store, user.NewService(store)
}

func seedUserAndManga(t *testing.T, store *database.Store) {
	t.Helper()

	if _, err := store.CreateUser(context.Background(), "user-1", "alice", "hash"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	err := store.InsertManga(context.Background(), models.Manga{
		ID:            "manga-1",
		Title:         "Test Manga",
		Author:        "Author",
		Genres:        []string{"Action"},
		Status:        "ongoing",
		TotalChapters: 100,
		Description:   "For TCP test",
		CoverURL:      "",
	})
	if err != nil {
		t.Fatalf("insert manga: %v", err)
	}
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve tcp addr: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

func dialWithRetry(t *testing.T, addr string) net.Conn {
	t.Helper()

	var conn net.Conn
	var err error
	for range 20 {
		conn, err = net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			return conn
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("dial tcp server: %v", err)
	return nil
}

func writeJSONLine(t *testing.T, conn net.Conn, payload any) {
	t.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write payload: %v", err)
	}
}

func readServerMessage(t *testing.T, conn net.Conn, reader *bufio.Reader) testServerMessage {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	defer conn.SetReadDeadline(time.Time{})

	line, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read tcp message: %v", err)
	}
	var msg testServerMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		t.Fatalf("unmarshal server message: %v", err)
	}
	return msg
}

func readUntilType(t *testing.T, conn net.Conn, reader *bufio.Reader, expectedType string) testServerMessage {
	t.Helper()

	for range 8 {
		msg := readServerMessage(t, conn, reader)
		if msg.Type == expectedType {
			return msg
		}
	}
	t.Fatalf("did not receive %s message", expectedType)
	return testServerMessage{}
}
