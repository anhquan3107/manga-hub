package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	shared "mangahub/cmd/cli/commands/shared"
)

func handleStatus() {
	token := strings.TrimSpace(shared.LoadToken())
	if token == "" {
		fmt.Println("✗ Not logged in")
		fmt.Println("Use: mangahub auth login --username <username> to login")
		return
	}
	req, _ := http.NewRequest(http.MethodGet, shared.APIURL("/health"), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("⚠ Server unreachable")
		fmt.Println("Token exists but cannot verify with server")
		return
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		fmt.Println("✗ Session expired or invalid")
		_ = shared.DeleteToken()
		fmt.Println("Token has been cleared")
		return
	}
	var userInfo struct {
		ID, Username, Email string
		CreatedAt           time.Time
	}
	req2, _ := http.NewRequest(http.MethodGet, shared.APIURL("/users/me"), nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	resp2, err := http.DefaultClient.Do(req2)
	if err == nil && resp2.StatusCode == 200 {
		json.NewDecoder(resp2.Body).Decode(&userInfo)
		resp2.Body.Close()
	}
	fmt.Println("✓ Logged in")
	fmt.Println("User Information:")
	if userInfo.Username != "" {
		fmt.Printf(" Username: %s\n", userInfo.Username)
	}
	if userInfo.ID != "" {
		fmt.Printf(" User ID: %s\n", userInfo.ID)
	}
	if userInfo.Email != "" {
		fmt.Printf(" Email: %s\n", userInfo.Email)
	}
	sessionID := shared.GetSessionID()
	if sessionID != "" {
		fmt.Printf(" Session ID: %s\n", sessionID)
	}
	expiresAt := time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02 15:04:05 UTC")
	fmt.Printf(" Token expires: %s (24 hours)\n", expiresAt)
	fmt.Println("Session Status: Active")
}
