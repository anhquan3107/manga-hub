package database

import (
	"context"
	"fmt"

	"mangahub/pkg/models"
)

func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) (models.User, error) {
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO users (username, password_hash) VALUES (?, ?)`,
		username,
		passwordHash,
	)
	if err != nil {
		return models.User{}, fmt.Errorf("create user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return models.User{}, fmt.Errorf("create user last insert id: %w", err)
	}

	return s.GetUserByID(ctx, userID)
}

func (s *Store) GetUserByID(ctx context.Context, userID int64) (models.User, error) {
	var user models.User
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, username, created_at FROM users WHERE id = ?`,
		userID,
	).Scan(&user.ID, &user.Username, &user.CreatedAt)
	if err != nil {
		return models.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (models.User, string, error) {
	var user models.User
	var passwordHash string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, username, password_hash, created_at FROM users WHERE username = ?`,
		username,
	).Scan(&user.ID, &user.Username, &passwordHash, &user.CreatedAt)
	if err != nil {
		return models.User{}, "", fmt.Errorf("get user by username: %w", err)
	}

	return user, passwordHash, nil
}
