package shared

import (
	"bytes"
	"fmt"
	"io"
	"net"
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

func APIBaseURL() string { return "http://" + net.JoinHostPort(httpHostFromAddr(httpAddr()), httpPortFromAddr(httpAddr())) }

func APIURL(path string) string { return APIBaseURL() + ensureLeadingSlash(path) }

func WebSocketURL(path string) string { return "ws://" + net.JoinHostPort(httpHostFromAddr(httpAddr()), httpPortFromAddr(httpAddr())) + ensureLeadingSlash(path) }

func TCPAddr() string { return clientAddrFromListenAddr(mustEnv("TCP_ADDR"), httpHostFromAddr(httpAddr()), "TCP_ADDR") }

func UDPAddr() string { return clientAddrFromListenAddr(mustEnv("UDP_ADDR"), httpHostFromAddr(httpAddr()), "UDP_ADDR") }

func GRPCAddr() string { return clientAddrFromListenAddr(mustEnv("GRPC_ADDR"), httpHostFromAddr(httpAddr()), "GRPC_ADDR") }

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

func mustEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		fmt.Fprintf(os.Stderr, "missing required environment variable: %s\n", key)
		os.Exit(1)
	}
	return value
}

func httpAddr() string { return mustEnv("HTTP_ADDR") }

func httpHostFromAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid HTTP_ADDR %q: %v\n", addr, err)
		os.Exit(1)
	}
	if strings.TrimSpace(host) == "" {
		return "localhost"
	}
	if host == "0.0.0.0" || host == "::" || host == "[::]" {
		return "localhost"
	}
	return host
}

func httpPortFromAddr(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid HTTP_ADDR %q: %v\n", addr, err)
		os.Exit(1)
	}
	if strings.TrimSpace(port) == "" {
		fmt.Fprintf(os.Stderr, "invalid HTTP_ADDR %q: missing port\n", addr)
		os.Exit(1)
	}
	return port
}

func clientAddrFromListenAddr(listenAddr, fallbackHost, envName string) string {
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid %s %q: %v\n", envName, listenAddr, err)
		os.Exit(1)
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = fallbackHost
	}
	return net.JoinHostPort(host, port)
}

func ensureLeadingSlash(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}
