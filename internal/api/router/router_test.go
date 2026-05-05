package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"mangahub/internal/auth"
	"mangahub/internal/chat"
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
	chatService := chat.NewService(store)
	mangaService := manga.NewService(store)
	userService := user.NewService(store)
	hub := chatws.NewHub()

	cfg := config.Config{AllowedOrigin: "*"}
	return NewRouter(cfg, authService, chatService, mangaService, userService, hub, nil)
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
		"email":    username + "@example.com",
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

func loginAndGetToken(t *testing.T, router http.Handler, username, password string) string {
	t.Helper()

	rr := performJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]any{
		"username": username,
		"password": password,
	}, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("login status = %d, body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login response: %v", err)
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
		"email":    "alice@example.com",
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

func TestAuthLogoutRevokesToken(t *testing.T) {
	router := setupRouterForTest(t)
	token := registerAndGetToken(t, router, "logout-user")

	logoutResp := performJSONRequest(t, router, http.MethodPost, "/auth/logout", nil, token)
	if logoutResp.Code != http.StatusOK {
		t.Fatalf("logout status = %d, body=%s", logoutResp.Code, logoutResp.Body.String())
	}

	protectedResp := performJSONRequest(t, router, http.MethodGet, "/users/library", nil, token)
	if protectedResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d, body=%s", protectedResp.Code, protectedResp.Body.String())
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

	var createdManga struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &createdManga); err != nil {
		t.Fatalf("decode create manga response: %v", err)
	}
	if createdManga.ID == "" {
		t.Fatalf("expected non-empty manga id")
	}

	mangaPath := fmt.Sprintf("/manga/%s", createdManga.ID)

	getResp := performJSONRequest(t, router, http.MethodGet, mangaPath, nil, "")
	if getResp.Code != http.StatusOK {
		t.Fatalf("get manga status = %d, body=%s", getResp.Code, getResp.Body.String())
	}

	updateResp := performJSONRequest(t, router, http.MethodPut, mangaPath, map[string]any{
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
		"manga_id":        createdManga.ID,
		"current_chapter": 1,
		"status":          "reading",
	}, token)
	if addLibraryResp.Code != http.StatusCreated {
		t.Fatalf("add library status = %d, body=%s", addLibraryResp.Code, addLibraryResp.Body.String())
	}

	updateLibraryResp := performJSONRequest(t, router, http.MethodPut, fmt.Sprintf("/users/library/%s", createdManga.ID), map[string]any{
		"status": "completed",
		"rating": 10,
	}, token)
	if updateLibraryResp.Code != http.StatusOK {
		t.Fatalf("update library status = %d, body=%s", updateLibraryResp.Code, updateLibraryResp.Body.String())
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
		"manga_id":        createdManga.ID,
		"current_chapter": 5,
		"status":          "reading",
	}, token)
	if updateProgressResp.Code != http.StatusOK {
		t.Fatalf("update progress status = %d, body=%s", updateProgressResp.Code, updateProgressResp.Body.String())
	}

	deleteResp := performJSONRequest(t, router, http.MethodDelete, mangaPath, nil, token)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete manga status = %d, body=%s", deleteResp.Code, deleteResp.Body.String())
	}
}

func TestRESTAPIAdditionalCoverage(t *testing.T) {
	router := setupRouterForTest(t)
	aliceToken := registerAndGetToken(t, router, "alice2")
	_ = registerAndGetToken(t, router, "bob2")

	// Public health endpoint
	healthResp := performJSONRequest(t, router, http.MethodGet, "/health", nil, "")
	if healthResp.Code != http.StatusOK {
		t.Fatalf("health status = %d, body=%s", healthResp.Code, healthResp.Body.String())
	}

	// Protected manga creation requires auth
	unauthCreate := performJSONRequest(t, router, http.MethodPost, "/manga", map[string]any{
		"id":          "no-auth",
		"title":       "No Auth",
		"author":      "N/A",
		"genres":      []string{"Action"},
		"status":      "ongoing",
		"description": "Should fail",
	}, "")
	if unauthCreate.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for protected create manga, got %d", unauthCreate.Code)
	}

	// Invalid payload should be rejected
	invalidCreate := performJSONRequest(t, router, http.MethodPost, "/manga", map[string]any{
		"id": "bad-manga",
	}, aliceToken)
	if invalidCreate.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid create manga payload, got %d", invalidCreate.Code)
	}

	// Valid manga creation and list/get/update/delete branches
	createResp := performJSONRequest(t, router, http.MethodPost, "/manga", map[string]any{
		"id":             "api-extra-manga",
		"title":          "API Extra Manga",
		"author":         "Author",
		"genres":         []string{"Action"},
		"status":         "ongoing",
		"total_chapters": 30,
		"description":    "extra coverage",
	}, aliceToken)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create manga status = %d, body=%s", createResp.Code, createResp.Body.String())
	}

	listResp := performJSONRequest(t, router, http.MethodGet, "/manga?limit=5&genre=Action&status=ongoing", nil, "")
	if listResp.Code != http.StatusOK {
		t.Fatalf("list manga status = %d, body=%s", listResp.Code, listResp.Body.String())
	}

	updateMissing := performJSONRequest(t, router, http.MethodPut, "/manga/does-not-exist", map[string]any{
		"title":          "Missing",
		"author":         "Missing",
		"genres":         []string{"Action"},
		"status":         "ongoing",
		"total_chapters": 1,
		"description":    "missing",
	}, aliceToken)
	if updateMissing.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for update missing manga, got %d", updateMissing.Code)
	}

	deleteMissing := performJSONRequest(t, router, http.MethodDelete, "/manga/does-not-exist", nil, aliceToken)
	if deleteMissing.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for delete missing manga, got %d", deleteMissing.Code)
	}

	// User endpoints
	meResp := performJSONRequest(t, router, http.MethodGet, "/users/me", nil, aliceToken)
	if meResp.Code != http.StatusOK {
		t.Fatalf("get me status = %d, body=%s", meResp.Code, meResp.Body.String())
	}

	addMissingLibrary := performJSONRequest(t, router, http.MethodPost, "/users/library", map[string]any{
		"manga_id": "missing-manga",
		"status":   "reading",
	}, aliceToken)
	if addMissingLibrary.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when adding missing manga to library, got %d", addMissingLibrary.Code)
	}

	addLibrary := performJSONRequest(t, router, http.MethodPost, "/users/library", map[string]any{
		"manga_id":        "api-extra-manga",
		"current_chapter": 1,
		"status":          "reading",
	}, aliceToken)
	if addLibrary.Code != http.StatusCreated {
		t.Fatalf("add library status = %d, body=%s", addLibrary.Code, addLibrary.Body.String())
	}

	badProgress := performJSONRequest(t, router, http.MethodPut, "/users/progress", map[string]any{
		"manga_id":        "api-extra-manga",
		"current_chapter": -1,
	}, aliceToken)
	if badProgress.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid progress update, got %d", badProgress.Code)
	}

	progressResp := performJSONRequest(t, router, http.MethodPut, "/users/progress", map[string]any{
		"manga_id":        "api-extra-manga",
		"current_chapter": 3,
		"status":          "reading",
	}, aliceToken)
	if progressResp.Code != http.StatusOK {
		t.Fatalf("progress update status = %d, body=%s", progressResp.Code, progressResp.Body.String())
	}

	historyResp := performJSONRequest(t, router, http.MethodGet, "/users/progress/history?manga_id=api-extra-manga", nil, aliceToken)
	if historyResp.Code != http.StatusOK {
		t.Fatalf("progress history status = %d, body=%s", historyResp.Code, historyResp.Body.String())
	}

	updateLibraryBadRating := performJSONRequest(t, router, http.MethodPut, "/users/library/api-extra-manga", map[string]any{
		"status": "completed",
		"rating": 11,
	}, aliceToken)
	if updateLibraryBadRating.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid library update rating, got %d", updateLibraryBadRating.Code)
	}

	removeLibrary := performJSONRequest(t, router, http.MethodDelete, "/users/library/api-extra-manga", nil, aliceToken)
	if removeLibrary.Code != http.StatusOK {
		t.Fatalf("remove library status = %d, body=%s", removeLibrary.Code, removeLibrary.Body.String())
	}

	removeLibraryAgain := performJSONRequest(t, router, http.MethodDelete, "/users/library/api-extra-manga", nil, aliceToken)
	if removeLibraryAgain.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when removing non-existing library entry, got %d", removeLibraryAgain.Code)
	}

	// PM and room endpoints
	sendPM := performJSONRequest(t, router, http.MethodPost, "/users/pm", map[string]any{
		"recipient_username": "bob2",
		"message":            "hello bob",
	}, aliceToken)
	if sendPM.Code != http.StatusCreated {
		t.Fatalf("send pm status = %d, body=%s", sendPM.Code, sendPM.Body.String())
	}

	sendPMToSelf := performJSONRequest(t, router, http.MethodPost, "/users/pm", map[string]any{
		"recipient_username": "alice2",
		"message":            "self message",
	}, aliceToken)
	if sendPMToSelf.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for self PM, got %d", sendPMToSelf.Code)
	}

	roomUsersUnauth := performJSONRequest(t, router, http.MethodGet, "/rooms/users", nil, "")
	if roomUsersUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for room users without token, got %d", roomUsersUnauth.Code)
	}

	roomUsers := performJSONRequest(t, router, http.MethodGet, "/rooms/users", nil, aliceToken)
	if roomUsers.Code != http.StatusOK {
		t.Fatalf("room users status = %d, body=%s", roomUsers.Code, roomUsers.Body.String())
	}

	roomHistoryInvalidLimit := performJSONRequest(t, router, http.MethodGet, "/rooms/general/history?limit=bad", nil, aliceToken)
	if roomHistoryInvalidLimit.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid room history limit, got %d", roomHistoryInvalidLimit.Code)
	}

	roomHistory := performJSONRequest(t, router, http.MethodGet, "/rooms/general/history?limit=10", nil, aliceToken)
	if roomHistory.Code != http.StatusOK {
		t.Fatalf("room history status = %d, body=%s", roomHistory.Code, roomHistory.Body.String())
	}

	// Change password path and token revocation behavior
	changePassword := performJSONRequest(t, router, http.MethodPost, "/auth/change-password", map[string]any{
		"current_password": "secret123",
		"new_password":     "newsecret123",
	}, aliceToken)
	if changePassword.Code != http.StatusOK {
		t.Fatalf("change password status = %d, body=%s", changePassword.Code, changePassword.Body.String())
	}

	oldTokenAfterPasswordChange := performJSONRequest(t, router, http.MethodGet, "/users/me", nil, aliceToken)
	if oldTokenAfterPasswordChange.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for revoked token after password change, got %d", oldTokenAfterPasswordChange.Code)
	}

	newToken := loginAndGetToken(t, router, "alice2", "newsecret123")
	meWithNewToken := performJSONRequest(t, router, http.MethodGet, "/users/me", nil, newToken)
	if meWithNewToken.Code != http.StatusOK {
		t.Fatalf("expected 200 with new token after password change, got %d", meWithNewToken.Code)
	}
}
