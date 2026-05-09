package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr      string
	TCPAddr       string
	UDPAddr       string
	GRPCAddr      string
	TCPServerAddr string
	DatabasePath  string
	SeedFile      string
	JWTSecret     string
	AllowedOrigin string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

func Load() Config {
	_ = godotenv.Load()
	return Config{
		HTTPAddr:      mustEnv("HTTP_ADDR"),
		TCPAddr:       mustEnv("TCP_ADDR"),
		UDPAddr:       mustEnv("UDP_ADDR"),
		GRPCAddr:      mustEnv("GRPC_ADDR"),
		TCPServerAddr: mustEnv("TCP_SERVER_ADDR"),
		DatabasePath:  mustEnv("DB_PATH"),
		SeedFile:      mustEnv("SEED_FILE"),
		JWTSecret:     mustEnv("JWT_SECRET"),
		AllowedOrigin: mustEnv("ALLOWED_ORIGIN"),
		RedisAddr:     optionalEnv("REDIS_ADDR"),
		RedisPassword: optionalEnv("REDIS_PASSWORD"),
		RedisDB:       0,
	}
}

func optionalEnv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func mustEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		fmt.Fprintf(os.Stderr, "missing required environment variable: %s\n", key)
		os.Exit(1)
	}
	return value
}
