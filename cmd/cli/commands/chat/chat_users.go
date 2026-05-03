package chat

import (
	"encoding/json"
	"fmt"
	"net/http"

	shared "mangahub/cmd/cli/commands/shared"
)

func printAllRoomsUsers() error {
	token := shared.LoadToken()
	if token == "" {
		return fmt.Errorf("you must login first")
	}

	req, _ := http.NewRequest(http.MethodGet, "http://localhost:8080/rooms/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var body struct {
		Rooms []struct {
			Room  string `json:"room"`
			Count int    `json:"count"`
			Users []struct {
				UserID   string `json:"user_id"`
				Username string `json:"username"`
			} `json:"users"`
		} `json:"rooms"`
		TotalUsers int `json:"total_users"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	fmt.Printf("\nOnline Users (%d):\n", body.TotalUsers)
	for _, room := range body.Rooms {
		roomLabel := chatRoomLabel(room.Room)
		for _, u := range room.Users {
			fmt.Printf("● %s (%s)\n", u.Username, roomLabel)
		}
	}

	return nil
}
