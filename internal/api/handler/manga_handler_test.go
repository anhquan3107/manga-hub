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

	"mangahub/internal/manga"
	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

func TestMangaHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "manga-handler-test.db")
	store, err := database.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	mangaService := manga.NewService(store)
	h := New(Dependencies{
		MangaService: mangaService,
	})

	r := gin.New()
	r.GET("/manga/:id", h.GetManga)
	r.POST("/manga", h.CreateManga)

	t.Run("CreateManga - Success", func(t *testing.T) {
		reqBody := models.CreateMangaRequest{
			ID:            "manga-test-1",
			Title:         "Test Manga",
			Author:        "Test Author",
			Genres:        []string{"Action"},
			Status:        "ongoing",
			TotalChapters: 10,
			Description:   "A test manga",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/manga", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201 Created, got %d", w.Code)
		}
	})

	t.Run("GetManga - Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/manga/manga-test-1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}

		var resp models.Manga
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		if resp.Title != "Test Manga" {
			t.Errorf("Expected title 'Test Manga', got '%s'", resp.Title)
		}
	})

	t.Run("GetManga - Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/manga/non-existent-manga", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 Not Found, got %d", w.Code)
		}
	})
}
