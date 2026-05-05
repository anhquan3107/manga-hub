package config

import (

	"testing"
)

func TestLoadSuccess(t *testing.T) {
	// Setup env vars for test
	expectedVars := map[string]string{
		"HTTP_ADDR":      ":8080",
		"TCP_ADDR":       ":9090",
		"UDP_ADDR":       ":9091",
		"GRPC_ADDR":      ":9092",
		"DB_PATH":        "./test.db",
		"SEED_FILE":      "./test.json",
		"JWT_SECRET":     "test-secret",
		"ALLOWED_ORIGIN": "*",
	}

	for k, v := range expectedVars {
		t.Setenv(k, v)
	}

	// Make sure we are not picking up a local .env file by changing dir
	// or godotenv will load it. But t.Setenv overrides.
	cfg := Load()

	if cfg.HTTPAddr != expectedVars["HTTP_ADDR"] {
		t.Errorf("Expected HTTPAddr %s, got %s", expectedVars["HTTP_ADDR"], cfg.HTTPAddr)
	}
	if cfg.TCPAddr != expectedVars["TCP_ADDR"] {
		t.Errorf("Expected TCPAddr %s, got %s", expectedVars["TCP_ADDR"], cfg.TCPAddr)
	}
	if cfg.UDPAddr != expectedVars["UDP_ADDR"] {
		t.Errorf("Expected UDPAddr %s, got %s", expectedVars["UDP_ADDR"], cfg.UDPAddr)
	}
	if cfg.GRPCAddr != expectedVars["GRPC_ADDR"] {
		t.Errorf("Expected GRPCAddr %s, got %s", expectedVars["GRPC_ADDR"], cfg.GRPCAddr)
	}
	if cfg.DatabasePath != expectedVars["DB_PATH"] {
		t.Errorf("Expected DatabasePath %s, got %s", expectedVars["DB_PATH"], cfg.DatabasePath)
	}
	if cfg.SeedFile != expectedVars["SEED_FILE"] {
		t.Errorf("Expected SeedFile %s, got %s", expectedVars["SEED_FILE"], cfg.SeedFile)
	}
	if cfg.JWTSecret != expectedVars["JWT_SECRET"] {
		t.Errorf("Expected JWTSecret %s, got %s", expectedVars["JWT_SECRET"], cfg.JWTSecret)
	}
	if cfg.AllowedOrigin != expectedVars["ALLOWED_ORIGIN"] {
		t.Errorf("Expected AllowedOrigin %s, got %s", expectedVars["ALLOWED_ORIGIN"], cfg.AllowedOrigin)
	}
}

// We cannot easily test os.Exit(1) without complex subprocesses, 
// so we skip the failure case for mustEnv to keep the suite simple and fast.
