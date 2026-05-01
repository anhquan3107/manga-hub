package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"time"

	shared "mangahub/cmd/cli/commands/shared"
)

func handleRegister(username, email string) {
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
}
