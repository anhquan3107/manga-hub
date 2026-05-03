package database

import (
	"context"
	"fmt"

	"mangahub/pkg/models"
)

func (s *Store) InsertProgressHistory(ctx context.Context, entry models.ProgressHistoryEntry) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO user_progress_history
		(user_id, manga_id, previous_chapter, current_chapter, previous_volume, current_volume, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)` ,
		entry.UserID,
		entry.MangaID,
		entry.PreviousChapter,
		entry.CurrentChapter,
		entry.PreviousVolume,
		entry.CurrentVolume,
		entry.Notes,
	)
	if err != nil {
		return fmt.Errorf("insert progress history: %w", err)
	}
	return nil
}

func (s *Store) GetProgressHistory(ctx context.Context, userID, mangaID string) ([]models.ProgressHistoryEntry, error) {
	query := `SELECT id, user_id, manga_id, previous_chapter, current_chapter, previous_volume, current_volume, notes, created_at
		FROM user_progress_history
		WHERE user_id = ?`
	args := []any{userID}
	if mangaID != "" {
		query += " AND manga_id = ?"
		args = append(args, mangaID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get progress history: %w", err)
	}
	defer rows.Close()

	results := make([]models.ProgressHistoryEntry, 0)
	for rows.Next() {
		var entry models.ProgressHistoryEntry
		if err := rows.Scan(
			&entry.ID,
			&entry.UserID,
			&entry.MangaID,
			&entry.PreviousChapter,
			&entry.CurrentChapter,
			&entry.PreviousVolume,
			&entry.CurrentVolume,
			&entry.Notes,
			&entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan progress history: %w", err)
		}
		results = append(results, entry)
	}

	return results, rows.Err()
}
