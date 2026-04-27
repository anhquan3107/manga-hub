package config

import (
	"os"
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
	return Config{
		HTTPAddr:      getEnv("HTTP_ADDR", ":8080"),
		TCPAddr:       getEnv("TCP_ADDR", ":9090"),
		UDPAddr:       getEnv("UDP_ADDR", ":9091"),
		GRPCAddr:      getEnv("GRPC_ADDR", ":9092"),
		DatabasePath:  getEnv("DB_PATH", "./data/mangahub.db"),
		SeedFile:      getEnv("SEED_FILE", "./data/manga.sample.json"),
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
