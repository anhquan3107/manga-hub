package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	shared "mangahub/cmd/cli/commands/shared"
)

func handleLogin(username string) {
	if username == "" {
		fmt.Println("--username is required")
		return
	}
	existingToken := strings.TrimSpace(shared.LoadToken())
	if existingToken != "" {
		fmt.Println("⚠ You're already logged in")
		fmt.Println("Logging in again will replace your current session")
	}
	password := readPasswordPrompt("Password: ")
	if password == "" {
		fmt.Println("Password required")
		return
	}
	data, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := http.Post("http://localhost:8080/auth/login", "application/json", bytes.NewBuffer(data))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		var res struct {
			Token string `json:"token"`
			User  struct {
				Username string `json:"username"`
			} `json:"user"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			fmt.Println("Login succeeded but response could not be parsed")
			return
		}
		if res.Token == "" {
			fmt.Println("Login succeeded but token missing in response")
			return
		}
		_ = shared.SaveToken(res.Token)
		expiresAt := time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02 15:04:05 UTC")
		if res.User.Username == "" {
			res.User.Username = username
		}
		sessionID := shared.GetSessionID()
		fmt.Println("✓ Login successful!")
		fmt.Printf("Welcome back, %s!\n", res.User.Username)
		fmt.Println("Session Details:")
		fmt.Printf(" Token expires: %s (24 hours)\n", expiresAt)
		if sessionID != "" {
			fmt.Printf(" Session ID: %s\n", sessionID)
		}
		fmt.Println(" Permissions: read, write, sync")
		fmt.Println()
		fmt.Println("Auto-sync: enabled")
		fmt.Println("Notifications: enabled")
		fmt.Println("Ready to use MangaHub! Try:")
		fmt.Printf(" mangahub manga search \"your favorite manga\"\n")
	} else {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		message := strings.ToLower(strings.TrimSpace(errResp.Error))
		switch {
		case strings.Contains(message, "account not found"):
			printLoginError("Account not found", "Try: mangahub auth register --username johndoe --email john@example.com")
		case strings.Contains(message, "invalid credentials"):
			printLoginError("Invalid credentials", "Check your username and password")
		default:
			printLoginError("Server connection error", "Check server status: mangahub server status")
		}
	}
}
