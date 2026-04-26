package router

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
	"mangahub/internal/config"
	"mangahub/internal/manga"
	"mangahub/internal/user"
	chatws "mangahub/internal/websocket"
	"mangahub/pkg/database"
)

func setupRouterForTest(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := database.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	authService := auth.NewService(store, "test-secret")
	mangaService := manga.NewService(store)
	userService := user.NewService(store)
	hub := chatws.NewHub()

	cfg := config.Config{AllowedOrigin: "*"}
	return NewRouter(cfg, authService, mangaService, userService, hub)
}

func performJSONRequest(t *testing.T, router http.Handler, method, path string, payload any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(data)
	}

	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func registerAndGetToken(t *testing.T, router http.Handler, username string) string {
	t.Helper()

	rr := performJSONRequest(t, router, http.MethodPost, "/auth/register", map[string]any{
		"username": username,
		"password": "secret123",
	}, "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if resp.Token == "" {
		t.Fatalf("expected non-empty token")
	}
	return resp.Token
}

func TestAuthRegisterAndLogin(t *testing.T) {
	router := setupRouterForTest(t)

	registerResp := performJSONRequest(t, router, http.MethodPost, "/auth/register", map[string]any{
		"username": "alice",
		"password": "secret123",
	}, "")
	if registerResp.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body=%s", registerResp.Code, registerResp.Body.String())
	}

	loginResp := performJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]any{
		"username": "alice",
		"password": "secret123",
	}, "")
	if loginResp.Code != http.StatusOK {
		t.Fatalf("login status = %d, body=%s", loginResp.Code, loginResp.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if resp.Token == "" {
		t.Fatalf("expected non-empty token")
	}
}

func TestProtectedLibraryRequiresAuth(t *testing.T) {
	router := setupRouterForTest(t)

	rr := performJSONRequest(t, router, http.MethodGet, "/users/library", nil, "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestMangaCRUDAndLibraryProgressFlow(t *testing.T) {
	router := setupRouterForTest(t)
	token := registerAndGetToken(t, router, "bob")

	createResp := performJSONRequest(t, router, http.MethodPost, "/manga", map[string]any{
		"id":             "test-manga",
		"title":          "Test Manga",
		"author":         "Tester",
		"genres":         []string{"Action", "Shounen"},
		"status":         "ongoing",
		"total_chapters": 10,
		"description":    "Testing manga CRUD",
		"cover_url":      "https://example.com/test.jpg",
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create manga status = %d, body=%s", createResp.Code, createResp.Body.String())
	}

	getResp := performJSONRequest(t, router, http.MethodGet, "/manga/test-manga", nil, "")
	if getResp.Code != http.StatusOK {
		t.Fatalf("get manga status = %d, body=%s", getResp.Code, getResp.Body.String())
	}

	updateResp := performJSONRequest(t, router, http.MethodPut, "/manga/test-manga", map[string]any{
		"title":          "Test Manga Updated",
		"author":         "Tester",
		"genres":         []string{"Action", "Adventure", "Shounen"},
		"status":         "ongoing",
		"total_chapters": 12,
		"description":    "Updated testing manga",
		"cover_url":      "https://example.com/test2.jpg",
	}, token)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update manga status = %d, body=%s", updateResp.Code, updateResp.Body.String())
	}

	addLibraryResp := performJSONRequest(t, router, http.MethodPost, "/users/library", map[string]any{
		"manga_id":        "test-manga",
		"current_chapter": 1,
		"status":          "reading",
	}, token)
	if addLibraryResp.Code != http.StatusCreated {
		t.Fatalf("add library status = %d, body=%s", addLibraryResp.Code, addLibraryResp.Body.String())
	}

	libraryResp := performJSONRequest(t, router, http.MethodGet, "/users/library", nil, token)
	if libraryResp.Code != http.StatusOK {
		t.Fatalf("get library status = %d, body=%s", libraryResp.Code, libraryResp.Body.String())
	}

	var libraryPayload map[string]any
	if err := json.Unmarshal(libraryResp.Body.Bytes(), &libraryPayload); err != nil {
		t.Fatalf("decode library response: %v", err)
	}
	if _, ok := libraryPayload["reading_lists"]; !ok {
		t.Fatalf("expected reading_lists in response")
	}

	updateProgressResp := performJSONRequest(t, router, http.MethodPut, "/users/progress", map[string]any{
		"manga_id":        "test-manga",
		"current_chapter": 5,
		"status":          "reading",
	}, token)
	if updateProgressResp.Code != http.StatusOK {
		t.Fatalf("update progress status = %d, body=%s", updateProgressResp.Code, updateProgressResp.Body.String())
	}

	deleteResp := performJSONRequest(t, router, http.MethodDelete, "/manga/test-manga", nil, token)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete manga status = %d, body=%s", deleteResp.Code, deleteResp.Body.String())
	}
}
