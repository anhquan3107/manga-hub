package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	shared "mangahub/cmd/cli/commands/shared"
)

func handleChangePassword() {
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
	req, _ := http.NewRequest(http.MethodPost, shared.APIURL("/auth/change-password"), bytes.NewBuffer(data))
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
}
