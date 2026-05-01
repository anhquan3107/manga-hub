package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"mangahub/pkg/models"
)

// Helper types for nullable scan results
type sqlNullInt struct {
	Int64 int64
	Valid bool
}

type sqlNullTime struct {
	Time  time.Time
	Valid bool
}

// Implement Scan methods for sqlNullInt and sqlNullTime by delegating to sql.NullInt64 and sql.NullTime
func (n *sqlNullInt) Scan(src interface{}) error {
	switch v := src.(type) {
	case int64:
		n.Int64 = v
		n.Valid = true
	case int:
		n.Int64 = int64(v)
		n.Valid = true
	case nil:
		n.Valid = false
	default:
		return fmt.Errorf("unsupported scan type for sqlNullInt: %T", src)
	}
	return nil
}

func (n *sqlNullTime) Scan(src interface{}) error {
	switch v := src.(type) {
	case time.Time:
		n.Time = v
		n.Valid = true
	case nil:
		n.Valid = false
	default:
		return fmt.Errorf("unsupported scan type for sqlNullTime: %T", src)
	}
	return nil
}

func nullableInt(v int) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

func nullableTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}

func (s *Store) UpsertLibraryEntry(ctx context.Context, userID string, entry models.LibraryEntry) (models.LibraryEntry, error) {
	// Use COALESCE for started_at so that when not provided it defaults to CURRENT_TIMESTAMP on insert
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO user_progress (user_id, manga_id, current_chapter, status, updated_at, rating, started_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, ?, COALESCE(?, CURRENT_TIMESTAMP))
		ON CONFLICT(user_id, manga_id) DO UPDATE SET
			current_chapter = excluded.current_chapter,
			status = excluded.status,
			updated_at = CURRENT_TIMESTAMP,
			rating = excluded.rating`,
		userID,
		entry.MangaID,
		entry.CurrentChapter,
		entry.Status,
		nullableInt(entry.Rating),
		nullableTime(entry.StartedAt),
	)
	if err != nil {
		return models.LibraryEntry{}, fmt.Errorf("upsert library entry: %w", err)
	}

	items, err := s.GetUserLibrary(ctx, userID)
	if err != nil {
		return models.LibraryEntry{}, err
	}

	for _, item := range items {
		if item.MangaID == entry.MangaID {
			return item, nil
		}
	}

	return models.LibraryEntry{}, errors.New("library entry not found after update")
}

func (s *Store) GetUserLibrary(ctx context.Context, userID string) ([]models.LibraryEntry, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT up.user_id, up.manga_id, m.title, up.current_chapter, up.status, up.updated_at, up.rating, up.started_at
		FROM user_progress up
		JOIN manga m ON m.id = up.manga_id
		WHERE up.user_id = ?
		ORDER BY up.updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get user library: %w", err)
	}
	defer rows.Close()

	var results []models.LibraryEntry
	for rows.Next() {
		var item models.LibraryEntry
		var rating sqlNullInt
		var started sqlNullTime
		if err := rows.Scan(
			&item.UserID,
			&item.MangaID,
			&item.Title,
			&item.CurrentChapter,
			&item.Status,
			&item.UpdatedAt,
			&rating,
			&started,
		); err != nil {
			return nil, fmt.Errorf("scan library entry: %w", err)
		}
		if rating.Valid {
			item.Rating = int(rating.Int64)
		}
		if started.Valid {
			item.StartedAt = started.Time
		}
		results = append(results, item)
	}

	return results, rows.Err()
}

func (s *Store) DeleteLibraryEntry(ctx context.Context, userID, mangaID string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM user_progress WHERE user_id = ? AND manga_id = ?`, userID, mangaID)
	if err != nil {
		return fmt.Errorf("delete library entry: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete library rows affected: %w", err)
	}
	if affected == 0 {
		return errors.New("library entry not found")
	}
	return nil
}
