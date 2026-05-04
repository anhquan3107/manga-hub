package config

import (
	"fmt"
	"mangahub/pkg/utils"
	"os"
	"strings"
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
	utils.LoadEnv()
	return Config{
		HTTPAddr:      mustEnv("HTTP_ADDR"),
		TCPAddr:       mustEnv("TCP_ADDR"),
		UDPAddr:       mustEnv("UDP_ADDR"),
		GRPCAddr:      mustEnv("GRPC_ADDR"),
		DatabasePath:  mustEnv("DB_PATH"),
		SeedFile:      mustEnv("SEED_FILE"),
		JWTSecret:     mustEnv("JWT_SECRET"),
		AllowedOrigin: mustEnv("ALLOWED_ORIGIN"),
	}
}

func mustEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		fmt.Fprintf(os.Stderr, "missing required environment variable: %s\n", key)
		os.Exit(1)
	}
	return value
}

