package shared

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

func GetSessionID() string {
	if customPath := os.Getenv("MANGAHUB_TOKEN_PATH"); customPath != "" {
		return ""
	}
	if sessionID := os.Getenv("TERM_SESSION_ID"); sessionID != "" {
		return sessionID
	}
	return "session_" + strconv.Itoa(os.Getppid())
}

func GetTokenPath() string {
	if customPath := os.Getenv("MANGAHUB_TOKEN_PATH"); customPath != "" {
		return customPath
	}
	sessionID := GetSessionID()
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".mangahub", sessionID)
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "token")
}

func SaveToken(token string) error {
	tokenPath := GetTokenPath()
	return os.WriteFile(tokenPath, []byte(token), 0o600)
}

func LoadToken() string {
	data, err := os.ReadFile(GetTokenPath())
	if err != nil {
		return ""
	}
	return string(data)
}

func DeleteToken() error {
	err := os.Remove(GetTokenPath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func PrintRespBody(body io.ReadCloser) {
	b, _ := io.ReadAll(body)
	fmt.Println(string(b))
}

func GetAuthClient() *http.Client { return &http.Client{} }

func DoAuthReq(method, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(LoadToken())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	return GetAuthClient().Do(req)
}

func Truncate(s string, length int) string {
	if len(s) > length {
		return s[:length-1] + "…"
	}
	return s
}

func NonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func FormatNumber(v int) string {
	s := strconv.Itoa(v)
	if len(s) <= 3 {
		return s
	}
	first := len(s) % 3
	if first == 0 {
		first = 3
	}
	parts := []string{s[:first]}
	for i := first; i < len(s); i += 3 {
		parts = append(parts, s[i:i+3])
	}
	return strings.Join(parts, ",")
}

func WrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}
	lines := make([]string, 0, len(words)/2)
	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) <= width {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	lines = append(lines, current)
	return lines
}

func CenterText(s string, width int) string {
	r := []rune(s)
	if len(r) > width {
		r = r[:width]
	}
	trimmed := string(r)
	visibleLen := utf8.RuneCountInString(trimmed)
	if visibleLen >= width {
		return trimmed
	}
	pad := width - visibleLen
	left := pad / 2
	right := pad - left
	return strings.Repeat(" ", left) + trimmed + strings.Repeat(" ", right)
}
