package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func getTokenPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".mangahub")
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
