package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	
	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

func TestAuthMiddleware(t *testing.T) {
	// Set Gin to test mode so it doesn't clutter output
	gin.SetMode(gin.TestMode)

	// Setup database and service
	dbPath := filepath.Join(t.TempDir(), "auth-middleware-test.db")
	store, err := database.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	service := NewService(store, "test-secret")

	// Helper to create a test router with the middleware
	setupRouter := func() *gin.Engine {
		r := gin.New()
		r.Use(Middleware(service))
		r.GET("/protected", func(c *gin.Context) {
			userID, _ := c.Get(ContextUserIDKey)
			username, _ := c.Get(ContextUsernameKey)
			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"user_id": userID,
				"username": username,
			})
		})
		return r
	}

	t.Run("Missing Header", func(t *testing.T) {
		r := setupRouter()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Invalid Header Format", func(t *testing.T) {
		r := setupRouter()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "InvalidFormatToken")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized for bad format, got %d", w.Code)
		}
	})

	t.Run("Invalid Token Signature", func(t *testing.T) {
		r := setupRouter()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer fake-invalid-token.payload.signature")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized for bad token, got %d", w.Code)
		}
	})

	t.Run("Valid Token", func(t *testing.T) {
		// Generate a valid token directly from the service
		user := models.User{ID: "user-1", Username: "alice"}
		token, err := service.IssueToken(user)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		r := setupRouter()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK for valid token, got %d", w.Code)
		}
	})
}
