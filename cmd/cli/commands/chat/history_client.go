package chat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	shared "mangahub/cmd/cli/commands/shared"
)

type historyMessage struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

func chatRoomLabel(roomID string) string {
	if roomID == "general" {
		return "General Chat"
	}
	return fmt.Sprintf("%s Discussion", strings.Title(roomID))
}

func fetchRoomHistory(roomID string, limit int) ([]historyMessage, error) {
	token := shared.LoadToken()
	if token == "" {
		return nil, fmt.Errorf("you must login first")
	}

	reqURL := fmt.Sprintf("http://localhost:8080/rooms/%s/history?limit=%d", roomID, limit)
	req, _ := http.NewRequest(http.MethodGet, reqURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var body struct {
		Messages []historyMessage `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return body.Messages, nil
}

func printRoomHistory(roomID string, messages []historyMessage) {
	fmt.Printf("\nRecent messages in %s (showing %d):\n", chatRoomLabel(roomID), len(messages))
	if len(messages) == 0 {
		fmt.Println("(No messages yet)")
		return
	}
	for _, m := range messages {
		msgTime := time.Unix(m.Timestamp, 0).Format("15:04")
		fmt.Printf("[%s] %s: %s\n", msgTime, m.Username, m.Message)
	}
}
