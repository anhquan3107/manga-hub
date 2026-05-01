package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"mangahub/pkg/models"
)

func (s *Store) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		email TEXT NOT NULL,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS manga (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		genres TEXT NOT NULL,
		status TEXT NOT NULL,
		total_chapters INTEGER NOT NULL,
		description TEXT NOT NULL,
		cover_url TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS user_progress (
		user_id TEXT NOT NULL,
		manga_id TEXT NOT NULL,
		current_chapter INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		rating INTEGER,
		started_at TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, manga_id),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS chat_messages (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		username TEXT NOT NULL,
		room_id TEXT NOT NULL,
		message TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_chat_messages_room ON chat_messages(room_id, created_at DESC);
	`

	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}

	return nil
}

func (s *Store) SeedMangaFromJSON(ctx context.Context, path string) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM manga`).Scan(&count); err != nil {
		return fmt.Errorf("count manga: %w", err)
	}
	if count > 0 {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read seed file: %w", err)
	}

	var mangaList []models.Manga
	if err := json.Unmarshal(data, &mangaList); err != nil {
		return fmt.Errorf("parse seed file: %w", err)
	}

	for _, manga := range mangaList {
		if err := s.InsertManga(ctx, manga); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) InsertManga(ctx context.Context, manga models.Manga) error {
	genres, err := json.Marshal(manga.Genres)
	if err != nil {
		return fmt.Errorf("marshal genres: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT OR REPLACE INTO manga (id, title, author, genres, status, total_chapters, description, cover_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		manga.ID,
		manga.Title,
		manga.Author,
		string(genres),
		manga.Status,
		manga.TotalChapters,
		manga.Description,
		manga.CoverURL,
	)
	if err != nil {
		return fmt.Errorf("insert manga: %w", err)
	}

	return nil
}
