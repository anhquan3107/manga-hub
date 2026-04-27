package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	HTTPAddr      string
	TCPAddr       string
	UDPAddr       string
	GRPCAddr      string
	DatabasePath  string
	SeedFile      string
	JWTSecret     string
	AllowedOrigin string
}

func Load() Config {
	databasePath := resolveProjectPath(getEnv("DB_PATH", "./data/mangahub.db"))
	seedFile := resolveProjectPath(getEnv("SEED_FILE", "./data/manga.sample.json"))

	return Config{
		HTTPAddr:      getEnv("HTTP_ADDR", ":8080"),
		TCPAddr:       getEnv("TCP_ADDR", ":9090"),
		UDPAddr:       getEnv("UDP_ADDR", ":9091"),
		GRPCAddr:      getEnv("GRPC_ADDR", ":9092"),
		DatabasePath:  databasePath,
		SeedFile:      seedFile,
		JWTSecret:     getEnv("JWT_SECRET", "change-this-secret"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "*"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func resolveProjectPath(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}

	root, ok := findProjectRoot()
	if !ok {
		return path
	}

	return filepath.Join(root, path)
}

func findProjectRoot() (string, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}

	current := wd
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, true
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}
