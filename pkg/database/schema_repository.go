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
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS manga (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		genres TEXT NOT NULL,
		status TEXT NOT NULL,
		total_chapters INTEGER NOT NULL,
		description TEXT NOT NULL,
		cover_url TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS user_progress (
		user_id INTEGER NOT NULL,
		manga_id INTEGER NOT NULL,
		current_chapter INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, manga_id),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE
	);
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
		`INSERT INTO manga (title, author, genres, status, total_chapters, description, cover_url)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
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
