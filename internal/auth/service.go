package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type Service struct {
	store     *database.Store
	jwtSecret []byte
}

func NewService(store *database.Store, jwtSecret string) *Service {
	return &Service{
		store:     store,
		jwtSecret: []byte(jwtSecret),
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

	user, err := s.store.CreateUser(ctx, req.Username, string(passwordHash))
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
		return models.AuthResponse{}, errors.New("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return models.AuthResponse{}, errors.New("invalid username or password")
	}

	token, err := s.IssueToken(user)
	if err != nil {
		return models.AuthResponse{}, err
	}

	return models.AuthResponse{Token: token, User: user}, nil
}

func (s *Service) IssueToken(user models.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", user.ID),
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

	return claims, nil
}
