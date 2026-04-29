package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

func TestIssueAndParseToken(t *testing.T) {
	service := &Service{jwtSecret: []byte("test-secret"), revoked: make(map[string]time.Time)}

	token, err := service.IssueToken(models.User{
		ID:        "user-1",
		Username:  "demo",
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	claims, err := service.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}

	if claims.UserID != "user-1" {
		t.Fatalf("expected user id user-1, got %s", claims.UserID)
	}
	if claims.Username != "demo" {
		t.Fatalf("expected username demo, got %s", claims.Username)
	}
}

func TestLogoutRevokesToken(t *testing.T) {
	service := &Service{jwtSecret: []byte("test-secret"), revoked: make(map[string]time.Time)}

	token, err := service.IssueToken(models.User{
		ID:        "user-1",
		Username:  "demo",
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	if err := service.Logout(token); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}

	if _, err := service.ParseToken(token); err == nil {
		t.Fatalf("expected revoked token to be invalid")
	}
}

func TestChangePasswordUpdatesStoredHash(t *testing.T) {
	store, err := database.NewSQLiteStore(filepath.Join(t.TempDir(), "mangahub.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore returned error: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema returned error: %v", err)
	}

	oldPassword := "oldPass123"
	hash, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}

	service := NewService(store, "test-secret")
	if _, err := store.CreateUser(ctx, "user-1", "demo", "demo@example.com", string(hash)); err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	if err := service.ChangePassword(ctx, "user-1", oldPassword, "newPass123"); err != nil {
		t.Fatalf("ChangePassword returned error: %v", err)
	}

	if err := service.ChangePassword(ctx, "user-1", oldPassword, "anotherPass123"); err == nil {
		t.Fatalf("expected old password to stop working after change")
	}
}
