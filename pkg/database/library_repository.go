package database

import (
	"context"
	"errors"
	"fmt"

	"mangahub/pkg/models"
)

func (s *Store) UpsertLibraryEntry(ctx context.Context, userID int64, entry models.LibraryEntry) (models.LibraryEntry, error) {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO user_progress (user_id, manga_id, current_chapter, status, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, manga_id) DO UPDATE SET
			current_chapter = excluded.current_chapter,
			status = excluded.status,
			updated_at = CURRENT_TIMESTAMP`,
		userID,
		entry.MangaID,
		entry.CurrentChapter,
		entry.Status,
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

func (s *Store) GetUserLibrary(ctx context.Context, userID int64) ([]models.LibraryEntry, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT up.user_id, up.manga_id, m.title, up.current_chapter, up.status, up.updated_at
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
		if err := rows.Scan(
			&item.UserID,
			&item.MangaID,
			&item.Title,
			&item.CurrentChapter,
			&item.Status,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan library entry: %w", err)
		}
		results = append(results, item)
	}

	return results, rows.Err()
}
