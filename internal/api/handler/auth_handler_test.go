package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"mangahub/internal/auth"
	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

func TestAuthHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup database and service
	dbPath := filepath.Join(t.TempDir(), "auth-handler-test.db")
	store, err := database.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	authService := auth.NewService(store, "test-secret")
	h := New(Dependencies{
		AuthService: authService,
	})

	r := gin.New()
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)

	t.Run("Register - Success", func(t *testing.T) {
		reqBody := models.RegisterRequest{
			Username: "testhandler",
			Email:    "test@example.com",
			Password: "StrongPassword123!",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201 Created, got %d", w.Code)
		}
	})

	t.Run("Register - Duplicate", func(t *testing.T) {
		reqBody := models.RegisterRequest{
			Username: "testhandler",
			Email:    "test2@example.com",
			Password: "StrongPassword123!",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("Expected 409 Conflict, got %d", w.Code)
		}
	})

	t.Run("Login - Success", func(t *testing.T) {
		reqBody := models.LoginRequest{
			Username: "testhandler",
			Password: "StrongPassword123!",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}
		
		var resp models.AuthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		if resp.Token == "" {
			t.Error("Expected token in response")
		}
	})
	
	t.Run("Login - Failure", func(t *testing.T) {
		reqBody := models.LoginRequest{
			Username: "testhandler",
			Password: "WrongPassword123!",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
		}
	})
}
