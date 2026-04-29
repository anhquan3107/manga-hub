package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type Service struct {
	store     *database.Store
	jwtSecret []byte
	revokedMu sync.RWMutex
	revoked   map[string]time.Time
}

func NewService(store *database.Store, jwtSecret string) *Service {
	return &Service{
		store:     store,
		jwtSecret: []byte(jwtSecret),
		revoked:   make(map[string]time.Time),
	}
}

func (s *Service) Register(ctx context.Context, req models.RegisterRequest) (models.AuthResponse, error) {
	if _, _, err := s.store.GetUserByUsername(ctx, req.Username); err == nil {
		return models.AuthResponse{}, errors.New("username already exists")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return models.AuthResponse{}, fmt.Errorf("hash password: %w", err)
	}

	userID := fmt.Sprintf("user-%d", time.Now().UnixNano())
	user, err := s.store.CreateUser(ctx, userID, req.Username, req.Email, string(passwordHash))
	if err != nil {
		return models.AuthResponse{}, err
	}

	token, err := s.IssueToken(user)
	if err != nil {
		return models.AuthResponse{}, err
	}

	return models.AuthResponse{Token: token, User: user}, nil
}

func (s *Service) Login(ctx context.Context, req models.LoginRequest) (models.AuthResponse, error) {
	user, passwordHash, err := s.store.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return models.AuthResponse{}, errors.New("account not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return models.AuthResponse{}, errors.New("invalid credentials")
	}

	token, err := s.IssueToken(user)
	if err != nil {
		return models.AuthResponse{}, err
	}

	return models.AuthResponse{Token: token, User: user}, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, passwordHash, err := s.store.GetUserByIDWithPassword(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword)); err != nil {
		return errors.New("invalid current password")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.store.UpdateUserPassword(ctx, user.ID, string(newHash)); err != nil {
		return err
	}

	return nil
}

func (s *Service) IssueToken(user models.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signedToken, nil
}

func (s *Service) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	if s.isRevoked(tokenString) {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func (s *Service) Logout(tokenString string) error {
	claims, err := s.ParseToken(tokenString)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	s.revokedMu.Lock()
	s.revoked[tokenString] = expiresAt
	s.revokedMu.Unlock()

	return nil
}

func (s *Service) isRevoked(tokenString string) bool {
	now := time.Now()

	s.revokedMu.Lock()
	defer s.revokedMu.Unlock()

	for token, exp := range s.revoked {
		if now.After(exp) {
			delete(s.revoked, token)
		}
	}

	exp, exists := s.revoked[tokenString]
	if !exists {
		return false
	}

	if now.After(exp) {
		delete(s.revoked, tokenString)
		return false
	}

	return true
}
