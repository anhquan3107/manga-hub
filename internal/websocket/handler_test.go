package websocket

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"

	"mangahub/internal/auth"
	"mangahub/pkg/database"
)

func TestWebsocketHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "ws-test.db")
	store, err := database.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	authService := auth.NewService(store, "test-secret")
	hub := NewHub()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	r := gin.New()
	r.GET("/chat", Handler(hub, authService, nil))

	server := httptest.NewServer(r)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/chat"

	t.Run("Missing Token", func(t *testing.T) {
		_, resp, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
		if err == nil {
			t.Fatal("Expected error dialing with no token")
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", resp.StatusCode)
		}
	})

	t.Run("Invalid Token", func(t *testing.T) {
		_, resp, err := gorillaws.DefaultDialer.Dial(wsURL+"?token=bad-token", nil)
		if err == nil {
			t.Fatal("Expected error dialing with invalid token")
		}
		if resp == nil || resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("Expected 401 Unauthorized, got resp=%+v, err=%v", resp, err)
		}
	})

	t.Run("Valid Token - Join and Broadcast", func(t *testing.T) {
		user, _ := store.CreateUser(context.Background(), "user-1", "wsuser", "ws@example.com", "hash")
		token, err := authService.IssueToken(user)
		if err != nil {
			t.Fatalf("Failed to issue token: %v", err)
		}

		conn, resp, err := gorillaws.DefaultDialer.Dial(wsURL+"?token="+token+"&room=testroom", nil)
		if err != nil {
			t.Fatalf("Failed to dial: %v, resp: %+v", err, resp)
		}
		defer conn.Close()

		// Read the "joined the chat" message
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("Failed to read join message: %v", err)
		}
		if msg["message"] != "joined the chat" {
			t.Errorf("Expected 'joined the chat', got '%v'", msg["message"])
		}

		// Send a message
		if err := conn.WriteJSON(map[string]string{"message": "hello world"}); err != nil {
			t.Fatalf("Failed to send message: %v", err)
		}

		// Read the echoed message
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("Failed to read echoed message: %v", err)
		}
		if msg["message"] != "hello world" {
			t.Errorf("Expected 'hello world', got '%v'", msg["message"])
		}
	})

	t.Run("Default Room Is General", func(t *testing.T) {
		user, _ := store.CreateUser(context.Background(), "user-default", "wsgeneral", "general@example.com", "hash")
		token, err := authService.IssueToken(user)
		if err != nil {
			t.Fatalf("Failed to issue token: %v", err)
		}

		conn, resp, err := gorillaws.DefaultDialer.Dial(wsURL+"?token="+token, nil)
		if err != nil {
			t.Fatalf("Failed to dial: %v, resp: %+v", err, resp)
		}
		defer conn.Close()

		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("Failed to read join message: %v", err)
		}
		if msg["message"] != "joined the chat" {
			t.Errorf("Expected joined message, got '%v'", msg["message"])
		}

		deadline := time.Now().Add(1 * time.Second)
		for time.Now().Before(deadline) {
			if len(hub.GetRoomUsers("general")) > 0 {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}

		t.Fatal("Expected user to be tracked in general room")
	})

	t.Run("Room Isolation", func(t *testing.T) {
		userA, _ := store.CreateUser(context.Background(), "user-room-a", "rooma", "rooma@example.com", "hash")
		userB, _ := store.CreateUser(context.Background(), "user-room-b", "roomb", "roomb@example.com", "hash")

		tokenA, err := authService.IssueToken(userA)
		if err != nil {
			t.Fatalf("Failed to issue token A: %v", err)
		}
		tokenB, err := authService.IssueToken(userB)
		if err != nil {
			t.Fatalf("Failed to issue token B: %v", err)
		}

		connA, respA, err := gorillaws.DefaultDialer.Dial(wsURL+"?token="+tokenA+"&room=room-a", nil)
		if err != nil {
			t.Fatalf("Failed to dial room-a: %v, resp: %+v", err, respA)
		}
		defer connA.Close()

		connB, respB, err := gorillaws.DefaultDialer.Dial(wsURL+"?token="+tokenB+"&room=room-b", nil)
		if err != nil {
			t.Fatalf("Failed to dial room-b: %v, resp: %+v", err, respB)
		}
		defer connB.Close()

		_ = connA.SetReadDeadline(time.Now().Add(2 * time.Second))
		_ = connB.SetReadDeadline(time.Now().Add(2 * time.Second))

		var msgA map[string]interface{}
		if err := connA.ReadJSON(&msgA); err != nil {
			t.Fatalf("Failed to read room-a join message: %v", err)
		}

		var msgB map[string]interface{}
		if err := connB.ReadJSON(&msgB); err != nil {
			t.Fatalf("Failed to read room-b join message: %v", err)
		}

		if err := connA.WriteJSON(map[string]string{"message": "room-a-only"}); err != nil {
			t.Fatalf("Failed to send room-a message: %v", err)
		}

		_ = connA.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err := connA.ReadJSON(&msgA); err != nil {
			t.Fatalf("Failed to read room-a broadcast: %v", err)
		}
		if msgA["message"] != "room-a-only" {
			t.Fatalf("Expected room-a broadcast message, got '%v'", msgA["message"])
		}

		_ = connB.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		err = connB.ReadJSON(&msgB)
		if err == nil {
			t.Fatalf("Expected no cross-room message, got '%v'", msgB)
		}
		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			t.Fatalf("Expected timeout waiting for cross-room message, got %v", err)
		}
	})
}
