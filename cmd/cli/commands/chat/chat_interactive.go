package chat

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"mangahub/cmd/cli/commands/shared"
)

func handleChatCommand(input string, roomID string) (string, bool) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", false
	}

	switch parts[0] {
	case "/help":
		fmt.Println("\nChat Commands:")
		fmt.Println(" /help - Show this help")
		fmt.Println(" /users - List online users")
		fmt.Println(" /quit - Leave chat")
		fmt.Println(" /pm <user> <msg> - Private message")
		fmt.Println(" /manga <id> - Switch to manga chat")
		fmt.Println(" /history - Show recent history")
		fmt.Println(" /status - Connection status")
	case "/users":
		if err := printAllRoomsUsers(); err != nil {
			fmt.Printf("✗ Failed to query users: %v\n", err)
		}
		return "", false
	case "/quit":
		fmt.Println("Leaving chat...")
		fmt.Println("✓ Disconnected from chat server")
		os.Exit(0)
	case "/pm":
		if len(parts) < 3 {
			fmt.Println("✗ Usage: /pm <user> <message>")
			return "", false
		}
		recipient := parts[1]
		message := strings.Join(parts[2:], " ")
		if err := sendPrivateMessage(recipient, message); err != nil {
			fmt.Printf("✗ Failed to send PM to %s: %v\n", recipient, err)
		} else {
			fmt.Printf("✓ Private message sent to %s\n", recipient)
		}
		return "", false
	case "/manga":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /manga <manga-id>")
			return "", false
		}
		return strings.TrimSpace(parts[1]), true
	case "/history":
		messages, err := fetchRoomHistory(roomID, 50)
		if err != nil {
			fmt.Printf("✗ Failed to load history: %v\n", err)
			return "", false
		}
		printRoomHistory(roomID, messages)
		return "", false
	case "/status":
		fmt.Println("\nConnection Status:")
		fmt.Printf("✓ Connected to %s\n", shared.WebSocketURL("/ws/chat"))
		fmt.Println("Room: #" + roomID)
		fmt.Println("Status: Online")
		return "", false
	default:
		fmt.Printf("✗ Unknown command: %s\n", parts[0])
		return "", false
	}

	return "", false
}

func sendPrivateMessage(recipient, message string) error {
	token := strings.TrimSpace(shared.LoadToken())
	if token == "" {
		return fmt.Errorf("not authenticated")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Create JSON payload
	escapedMsg := strings.ReplaceAll(message, `"`, `\"`)
	jsonData := fmt.Sprintf(`{"recipient_username":"%s","message":"%s"}`, recipient, escapedMsg)

	req, err := http.NewRequest("POST", shared.APIURL("/users/pm"), bytes.NewReader([]byte(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PM send failed: status %d", resp.StatusCode)
	}

	return nil
}
