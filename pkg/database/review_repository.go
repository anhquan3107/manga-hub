package database

import (
	"context"
	"errors"
	"fmt"

	"mangahub/pkg/models"
)

func (s *Store) UpsertReview(ctx context.Context, review models.Review) (models.Review, error) {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO reviews (user_id, manga_id, rating, text, timestamp, helpful)
		VALUES (?, ?, ?, ?, ?, 0)
		ON CONFLICT(user_id, manga_id) DO UPDATE SET
			rating = excluded.rating,
			text = excluded.text,
			timestamp = excluded.timestamp`,
		review.UserID,
		review.MangaID,
		review.Rating,
		review.Text,
		review.Timestamp,
	)
	if err != nil {
		return models.Review{}, fmt.Errorf("upsert review: %w", err)
	}

	return s.GetReview(ctx, review.UserID, review.MangaID)
}

func (s *Store) GetReview(ctx context.Context, userID, mangaID string) (models.Review, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT user_id, manga_id, rating, text, timestamp, helpful
		FROM reviews
		WHERE user_id = ? AND manga_id = ?`,
		userID,
		mangaID,
	)

	var review models.Review
	if err := row.Scan(
		&review.UserID,
		&review.MangaID,
		&review.Rating,
		&review.Text,
		&review.Timestamp,
		&review.Helpful,
	); err != nil {
		return models.Review{}, fmt.Errorf("get review: %w", err)
	}

	return review, nil
}

func (s *Store) ListReviewsByManga(ctx context.Context, mangaID string, limit int, sortBy string) ([]models.Review, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	order := "timestamp DESC"
	if sortBy == "helpful" {
		order = "helpful DESC, timestamp DESC"
	}

	rows, err := s.db.QueryContext(
		ctx,
		fmt.Sprintf(
			`SELECT user_id, manga_id, rating, text, timestamp, helpful
			FROM reviews
			WHERE manga_id = ?
			ORDER BY %s
			LIMIT ?`,
			order,
		),
		mangaID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()

	results := make([]models.Review, 0)
	for rows.Next() {
		var review models.Review
		if err := rows.Scan(
			&review.UserID,
			&review.MangaID,
			&review.Rating,
			&review.Text,
			&review.Timestamp,
			&review.Helpful,
		); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		results = append(results, review)
	}

	return results, rows.Err()
}

func (s *Store) IncrementReviewHelpful(ctx context.Context, userID, mangaID string) (models.Review, error) {
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE reviews
		SET helpful = helpful + 1
		WHERE user_id = ? AND manga_id = ?`,
		userID,
		mangaID,
	)
	if err != nil {
		return models.Review{}, fmt.Errorf("increment review helpful: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return models.Review{}, fmt.Errorf("review helpful rows affected: %w", err)
	}
	if affected == 0 {
		return models.Review{}, errors.New("review not found")
	}

	return s.GetReview(ctx, userID, mangaID)
}
