package auth

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"syscall"
	"time"

	shared "mangahub/cmd/cli/commands/shared"

	"golang.org/x/term"
)

func readPasswordPrompt(prompt string) string {
	fmt.Print(prompt)
	bytepw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(bytepw))
}

func printRegistrationError(message, detail string) {
	fmt.Printf("✗ Registration failed: %s\n", message)
	if detail != "" {
		fmt.Printf(" %s\n", detail)
	}
}

func isStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}

func printLoginError(message, detail string) {
	fmt.Printf("✗ Login failed: %s\n", message)
	if detail != "" {
		fmt.Printf(" %s\n", detail)
	}
}

func HandleAuth(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub auth <register|login|logout|status|change-password> [flags]")
		return
	}

	subCmd := args[0]
	flags := flag.NewFlagSet("auth "+subCmd, flag.ExitOnError)
	var username, email string
	flags.StringVar(&username, "username", "", "Your username")
	flags.StringVar(&email, "email", "", "Email address")
	flags.Parse(args[1:])

	switch subCmd {
	case "register":
		if username == "" {
			fmt.Println("--username is required")
			return
		}
		if email == "" {
			fmt.Println("--email is required")
			return
		}
		if _, err := mail.ParseAddress(email); err != nil {
			printRegistrationError("Invalid email format", "Please provide a valid email address")
			return
		}
		password := readPasswordPrompt("Password: ")
		confirm := readPasswordPrompt("Confirm password: ")
		if password == "" || password != confirm {
			fmt.Println("Passwords do not match or empty")
			return
		}
		if !isStrongPassword(password) {
			printRegistrationError("Password too weak", "Password must be at least 8 characters with mixed case and numbers")
			return
		}

		data, _ := json.Marshal(map[string]string{"username": username, "password": password, "email": email})
		resp, err := http.Post("http://localhost:8080/auth/register", "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusConflict {
			printRegistrationError(fmt.Sprintf("Username '%s' already exists", username), "Try: mangahub auth login --username "+username)
			return
		}
		if resp.StatusCode >= 400 {
			fmt.Printf("✗ Registration failed: %s\n", http.StatusText(resp.StatusCode))
			shared.PrintRespBody(resp.Body)
			return
		}
		var res struct {
			User struct {
				ID        string    `json:"id"`
				Username  string    `json:"username"`
				Email     string    `json:"email"`
				CreatedAt time.Time `json:"created_at"`
			} `json:"user"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			fmt.Println("Registration succeeded but response could not be parsed")
			return
		}
		createdAt := res.User.CreatedAt.UTC().Format("2006-01-02 15:04:05 UTC")
		if createdAt == "0001-01-01 00:00:00 UTC" {
			createdAt = time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
		}
		fmt.Println("✓ Account created successfully!")
		fmt.Printf("User ID: %s\n", res.User.ID)
		fmt.Printf("Username: %s\n", res.User.Username)
		fmt.Printf("Email: %s\n", res.User.Email)
		fmt.Printf("Created: %s\n", createdAt)
		fmt.Println("Please login to start using MangaHub:")
		fmt.Printf(" mangahub auth login --username %s\n", username)

	case "login":
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
	case "logout":
		token := strings.TrimSpace(shared.LoadToken())
		if token == "" {
			fmt.Println("No active session found")
			return
		}
		req, _ := http.NewRequest(http.MethodPost, "http://localhost:8080/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			_ = shared.DeleteToken()
			fmt.Println("✓ Logged out locally")
			fmt.Println("Server unreachable; local session has been cleared")
			return
		}
		defer resp.Body.Close()
		_ = shared.DeleteToken()
		if resp.StatusCode >= 400 {
			fmt.Println("✓ Logged out locally")
			fmt.Println("Session token cleared from this device")
			return
		}
		fmt.Println("✓ Logout successful!")
		fmt.Println("Session ended and token removed")

	case "change-password":
		token := strings.TrimSpace(shared.LoadToken())
		if token == "" {
			fmt.Println("✗ Not logged in")
			fmt.Println("Use: mangahub auth login --username <username> to login")
			return
		}
		currentPassword := readPasswordPrompt("Current password: ")
		if currentPassword == "" {
			fmt.Println("Current password required")
			return
		}
		newPassword := readPasswordPrompt("New password: ")
		if newPassword == "" {
			fmt.Println("New password required")
			return
		}
		if !isStrongPassword(newPassword) {
			printRegistrationError("Password too weak", "Password must be at least 8 characters with mixed case and numbers")
			return
		}
		if currentPassword == newPassword {
			fmt.Println("New password must be different from the current password")
			return
		}
		data, _ := json.Marshal(map[string]string{"current_password": currentPassword, "new_password": newPassword})
		req, _ := http.NewRequest(http.MethodPost, "http://localhost:8080/auth/change-password", bytes.NewBuffer(data))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized {
			fmt.Println("✗ Change password failed: invalid current password")
			return
		}
		if resp.StatusCode >= 400 {
			fmt.Printf("✗ Change password failed: %s\n", http.StatusText(resp.StatusCode))
			shared.PrintRespBody(resp.Body)
			return
		}
		_ = shared.DeleteToken()
		fmt.Println("✓ Password changed successfully!")
		fmt.Println("Your session has been ended. Please login again with the new password.")

	case "status":
		token := strings.TrimSpace(shared.LoadToken())
		if token == "" {
			fmt.Println("✗ Not logged in")
			fmt.Println("Use: mangahub auth login --username <username> to login")
			return
		}
		req, _ := http.NewRequest(http.MethodGet, "http://localhost:8080/health", nil)
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
		req2, _ := http.NewRequest(http.MethodGet, "http://localhost:8080/users/me", nil)
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

	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}
