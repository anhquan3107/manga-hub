package auth

import (
	"testing"
	"time"

	"mangahub/pkg/models"
)

func TestIssueAndParseToken(t *testing.T) {
	service := &Service{jwtSecret: []byte("test-secret")}

	token, err := service.IssueToken(models.User{
		ID:        1,
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

	if claims.UserID != 1 {
		t.Fatalf("expected user id 1, got %d", claims.UserID)
	}
	if claims.Username != "demo" {
		t.Fatalf("expected username demo, got %s", claims.Username)
	}
}
