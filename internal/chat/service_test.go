package chat

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

func TestChatService(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "chat-service-test.db")
	store, err := database.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	service := NewService(store)
	ctx := context.Background()

	// Seed users
	_, _ = store.CreateUser(ctx, "user-1", "alice", "alice@example.com", "hash")
	_, _ = store.CreateUser(ctx, "user-2", "bob", "bob@example.com", "hash")

	t.Run("Save and Get Room History", func(t *testing.T) {
		msg := models.ChatMessage{
			UserID:    "user-1",
			Username:  "alice",
			Message:   "hello room 1",
			Timestamp: time.Now().Unix(),
		}
		if err := service.SaveMessage(ctx, msg, "room-1"); err != nil {
			t.Fatalf("SaveMessage failed: %v", err)
		}

		history, err := service.GetRoomHistory(ctx, "room-1", 10)
		if err != nil {
			t.Fatalf("GetRoomHistory failed: %v", err)
		}
		if len(history) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(history))
		}
		if history[0].Message != "hello room 1" {
			t.Errorf("Expected 'hello room 1', got '%s'", history[0].Message)
		}
	})

	t.Run("Send and Get Private Message", func(t *testing.T) {
		pm := models.PrivateMessage{
			SenderID:          "user-1",
			SenderUsername:    "alice",
			RecipientID:       "user-2",
			RecipientUsername: "bob",
			Message:           "hello bob",
			Timestamp:         time.Now().Unix(),
		}
		if err := service.SendPrivateMessage(ctx, pm); err != nil {
			t.Fatalf("SendPrivateMessage failed: %v", err)
		}

		received, err := service.GetReceivedMessages(ctx, "user-2", 10)
		if err != nil {
			t.Fatalf("GetReceivedMessages failed: %v", err)
		}
		if len(received) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(received))
		}
		if received[0].Message != "hello bob" {
			t.Errorf("Expected 'hello bob', got '%s'", received[0].Message)
		}
	})
}
