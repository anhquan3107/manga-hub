package auth

import (
	"testing"
	"time"

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
