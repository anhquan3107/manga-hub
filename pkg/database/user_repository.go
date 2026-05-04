package database

import (
	"context"
	"fmt"

	"mangahub/pkg/models"
)

func (s *Store) CreateUser(ctx context.Context, userID, username, email, passwordHash string) (models.User, error) {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO users (id, username, email, password_hash) VALUES (?, ?, ?, ?)`,
		userID,
		username,
		email,
		passwordHash,
	)
	if err != nil {
		return models.User{}, fmt.Errorf("create user: %w", err)
	}

	return s.GetUserByID(ctx, userID)
}

func (s *Store) GetUserByID(ctx context.Context, userID string) (models.User, error) {
	var user models.User
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, username, email, created_at FROM users WHERE id = ?`,
		userID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)
	if err != nil {
		return models.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (models.User, error) {
	var user models.User
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, username, email, created_at FROM users WHERE username = ?`,
		username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)
	if err != nil {
		return models.User{}, fmt.Errorf("get user by username: %w", err)
	}

	return user, nil
}

func (s *Store) GetUserByUsernameWithPassword(ctx context.Context, username string) (models.User, string, error) {
	var user models.User
	var passwordHash string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, username, email, password_hash, created_at FROM users WHERE username = ?`,
		username,
	).Scan(&user.ID, &user.Username, &user.Email, &passwordHash, &user.CreatedAt)
	if err != nil {
		return models.User{}, "", fmt.Errorf("get user by username: %w", err)
	}

	return user, passwordHash, nil
}

func (s *Store) GetUserByIDWithPassword(ctx context.Context, userID string) (models.User, string, error) {
	var user models.User
	var passwordHash string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, username, email, password_hash, created_at FROM users WHERE id = ?`,
		userID,
	).Scan(&user.ID, &user.Username, &user.Email, &passwordHash, &user.CreatedAt)
	if err != nil {
		return models.User{}, "", fmt.Errorf("get user by id: %w", err)
	}

	return user, passwordHash, nil
}

func (s *Store) UpdateUserPassword(ctx context.Context, userID, passwordHash string) error {
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE users SET password_hash = ? WHERE id = ?`,
		passwordHash,
		userID,
	)
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check password update: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("update user password: no rows affected")
	}

	return nil
}
