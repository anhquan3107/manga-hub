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
	"mangahub/internal/manga"
	"mangahub/internal/user"
	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

func TestUserHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "user-handler-test.db")
	store, err := database.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	userService := user.NewService(store)
	mangaService := manga.NewService(store)
	h := New(Dependencies{
		UserService:  userService,
		MangaService: mangaService,
	})

	// Seed user and manga
	_, _ = store.CreateUser(context.Background(), "user-test-1", "testuser", "test@example.com", "hash")
	_, _ = mangaService.Create(context.Background(), models.CreateMangaRequest{
		ID:            "manga-test-1",
		Title:         "Test Manga",
		Author:        "Test Author",
		Genres:        []string{"Action"},
		Status:        "ongoing",
		Description:   "A test manga",
		TotalChapters: 10,
	})

	// Mock auth middleware by directly setting the ContextUserIDKey in Gin
	mockAuthMiddleware := func(c *gin.Context) {
		c.Set(auth.ContextUserIDKey, "user-test-1")
		c.Next()
	}

	r := gin.New()
	protected := r.Group("/")
	protected.Use(mockAuthMiddleware)
	protected.GET("/me", h.GetMe)
	protected.POST("/library", h.AddToLibrary)
	protected.GET("/library", h.GetLibrary)

	t.Run("GetMe - Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}
	})

	t.Run("AddToLibrary - Success", func(t *testing.T) {
		reqBody := models.AddLibraryRequest{
			MangaID: "manga-test-1",
			Status:  "reading",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/library", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201 Created, got %d", w.Code)
		}
	})

	t.Run("GetLibrary - Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/library", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}
		
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		if resp["items"] == nil {
			t.Error("Expected items in response")
		}
	})
}
