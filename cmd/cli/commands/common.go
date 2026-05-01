package commands

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func getSessionID() string {
	if customPath := os.Getenv("MANGAHUB_TOKEN_PATH"); customPath != "" {
		return ""
	}

	if sessionID := os.Getenv("TERM_SESSION_ID"); sessionID != "" {
		return sessionID
	}

	ppid := os.Getppid()
	return "session_" + strconv.Itoa(ppid)
}

func getTokenPath() string {
	if customPath := os.Getenv("MANGAHUB_TOKEN_PATH"); customPath != "" {
		return customPath
	}

	sessionID := getSessionID()
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".mangahub", sessionID)
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "token")
}

func saveToken(token string) error {
	return os.WriteFile(getTokenPath(), []byte(token), 0600)
}

func loadToken() string {
	data, err := os.ReadFile(getTokenPath())
	if err != nil {
		return ""
	}
	return string(data)
}

func deleteToken() error {
	err := os.Remove(getTokenPath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func printRespBody(body io.ReadCloser) {
	b, _ := io.ReadAll(body)
	fmt.Println(string(b))
}

func getAuthClient() *http.Client {
	return &http.Client{}
}

func doAuthReq(method, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(loadToken())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	return getAuthClient().Do(req)
}
