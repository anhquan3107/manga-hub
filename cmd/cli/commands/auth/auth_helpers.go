package auth

import (
	"fmt"
	"strings"
	"syscall"

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
