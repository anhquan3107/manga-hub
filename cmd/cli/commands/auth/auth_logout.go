package auth

import (
	"fmt"
	"net/http"
	"strings"

	shared "mangahub/cmd/cli/commands/shared"
)

func handleLogout() {
	token := strings.TrimSpace(shared.LoadToken())
	if token == "" {
		fmt.Println("No active session found")
		return
	}
	req, _ := http.NewRequest(http.MethodPost, shared.APIURL("/auth/logout"), nil)
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
}
