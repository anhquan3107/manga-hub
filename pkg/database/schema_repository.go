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
		year INTEGER NOT NULL DEFAULT 0,
		rating REAL NOT NULL DEFAULT 0,
		popularity INTEGER NOT NULL DEFAULT 0,
		total_chapters INTEGER NOT NULL,
		description TEXT NOT NULL,
		cover_url TEXT NOT NULL
	);
	CREATE VIRTUAL TABLE IF NOT EXISTS manga_fts USING fts5(
		title,
		author,
		description,
		content='manga',
		content_rowid='rowid'
	);
	CREATE TRIGGER IF NOT EXISTS manga_ai AFTER INSERT ON manga BEGIN
		INSERT INTO manga_fts(rowid, title, author, description)
		VALUES (new.rowid, new.title, new.author, new.description);
	END;
	CREATE TRIGGER IF NOT EXISTS manga_ad AFTER DELETE ON manga BEGIN
		INSERT INTO manga_fts(manga_fts, rowid, title, author, description)
		VALUES('delete', old.rowid, old.title, old.author, old.description);
	END;
	CREATE TRIGGER IF NOT EXISTS manga_au AFTER UPDATE ON manga BEGIN
		INSERT INTO manga_fts(manga_fts, rowid, title, author, description)
		VALUES('delete', old.rowid, old.title, old.author, old.description);
		INSERT INTO manga_fts(rowid, title, author, description)
		VALUES (new.rowid, new.title, new.author, new.description);
	END;

	CREATE TABLE IF NOT EXISTS user_progress (
		user_id TEXT NOT NULL,
		manga_id TEXT NOT NULL,
		current_chapter INTEGER NOT NULL DEFAULT 0,
		current_volume INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		rating INTEGER,
		started_at TIMESTAMP,
		notes TEXT NOT NULL DEFAULT '',
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, manga_id),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS user_progress_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL,
		manga_id TEXT NOT NULL,
		previous_chapter INTEGER NOT NULL,
		current_chapter INTEGER NOT NULL,
		previous_volume INTEGER NOT NULL,
		current_volume INTEGER NOT NULL,
		notes TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS reviews (
		user_id TEXT NOT NULL,
		manga_id TEXT NOT NULL,
		rating INTEGER NOT NULL,
		text TEXT NOT NULL,
		timestamp INTEGER NOT NULL DEFAULT (strftime('%s','now')),
		helpful INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (user_id, manga_id),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_reviews_manga ON reviews(manga_id, timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_reviews_helpful ON reviews(manga_id, helpful DESC, timestamp DESC);

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

	CREATE TABLE IF NOT EXISTS private_messages (
		id TEXT PRIMARY KEY,
		sender_id TEXT NOT NULL,
		sender_username TEXT NOT NULL,
		recipient_id TEXT NOT NULL,
		recipient_username TEXT NOT NULL,
		message TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (recipient_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_private_messages_recipient ON private_messages(recipient_id, created_at DESC);
	`

	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}

	if err := s.ensureUsersEmailColumn(ctx); err != nil {
		return err
	}
	if err := s.ensureUserProgressColumns(ctx); err != nil {
		return err
	}
	if err := s.ensureMangaColumns(ctx); err != nil {
		return err
	}
	if err := s.rebuildMangaFTS(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Store) ensureUsersEmailColumn(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(users)`)
	if err != nil {
		return fmt.Errorf("inspect users schema: %w", err)
	}
	defer rows.Close()

	hasEmail := false
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return fmt.Errorf("scan users schema: %w", err)
		}
		if name == "email" {
			hasEmail = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate users schema: %w", err)
	}

	if hasEmail {
		return nil
	}

	if _, err := s.db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add users.email column: %w", err)
	}

	return nil
}

func (s *Store) ensureUserProgressColumns(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(user_progress)`)
	if err != nil {
		return fmt.Errorf("inspect user_progress schema: %w", err)
	}
	defer rows.Close()

	hasVolume := false
	hasNotes := false
	hasRating := false
	hasStartedAt := false
	for rows.Next() {
		var (
			cid       int
			name      string
			columnType string
			notNull   int
			defaultVal any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return fmt.Errorf("scan user_progress schema: %w", err)
		}
		switch name {
		case "current_volume":
			hasVolume = true
		case "notes":
			hasNotes = true
		case "rating":
			hasRating = true
		case "started_at":
			hasStartedAt = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate user_progress schema: %w", err)
	}

	if !hasVolume {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE user_progress ADD COLUMN current_volume INTEGER NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("add user_progress.current_volume column: %w", err)
		}
	}
	if !hasNotes {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE user_progress ADD COLUMN notes TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add user_progress.notes column: %w", err)
		}
	}
	if !hasRating {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE user_progress ADD COLUMN rating INTEGER`); err != nil {
			return fmt.Errorf("add user_progress.rating column: %w", err)
		}
	}
	if !hasStartedAt {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE user_progress ADD COLUMN started_at TIMESTAMP`); err != nil {
			return fmt.Errorf("add user_progress.started_at column: %w", err)
		}
	}

	return nil
}

func (s *Store) ensureMangaColumns(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(manga)`)
	if err != nil {
		return fmt.Errorf("inspect manga schema: %w", err)
	}
	defer rows.Close()

	hasYear := false
	hasRating := false
	hasPopularity := false
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return fmt.Errorf("scan manga schema: %w", err)
		}
		switch name {
		case "year":
			hasYear = true
		case "rating":
			hasRating = true
		case "popularity":
			hasPopularity = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate manga schema: %w", err)
	}

	if !hasYear {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE manga ADD COLUMN year INTEGER NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("add manga.year column: %w", err)
		}
	}
	if !hasRating {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE manga ADD COLUMN rating REAL NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("add manga.rating column: %w", err)
		}
	}
	if !hasPopularity {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE manga ADD COLUMN popularity INTEGER NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("add manga.popularity column: %w", err)
		}
	}

	return nil
}

func (s *Store) rebuildMangaFTS(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO manga_fts(manga_fts) VALUES('rebuild')`)
	if err != nil {
		return fmt.Errorf("rebuild manga fts: %w", err)
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
		`INSERT OR REPLACE INTO manga (id, title, author, genres, status, year, rating, popularity, total_chapters, description, cover_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		manga.ID,
		manga.Title,
		manga.Author,
		string(genres),
		manga.Status,
		manga.Year,
		manga.Rating,
		manga.Popularity,
		manga.TotalChapters,
		manga.Description,
		manga.CoverURL,
	)
	if err != nil {
		return fmt.Errorf("insert manga: %w", err)
	}

	return nil
}
