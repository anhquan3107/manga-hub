package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"mangahub/pkg/models"
)

type Store struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	store := &Store{db: db}
	if err := store.Ping(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return s.db.PingContext(ctx)
}

func (s *Store) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
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

func (s *Store) CreateUser(ctx context.Context, userID, username, passwordHash string) (models.User, error) {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)`,
		userID,
		username,
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

func (s *Store) ListManga(ctx context.Context, query models.MangaQuery) ([]models.Manga, error) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 4)

	if query.Query != "" {
		clauses = append(clauses, "(LOWER(title) LIKE ? OR LOWER(author) LIKE ?)")
		search := "%" + strings.ToLower(query.Query) + "%"
		args = append(args, search, search)
	}
	if query.Genre != "" {
		clauses = append(clauses, "LOWER(genres) LIKE ?")
		args = append(args, "%"+strings.ToLower(query.Genre)+"%")
	}
	if query.Status != "" {
		clauses = append(clauses, "LOWER(status) = ?")
		args = append(args, strings.ToLower(query.Status))
	}

	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	args = append(args, limit)

	rows, err := s.db.QueryContext(
		ctx,
		fmt.Sprintf(
			`SELECT id, title, author, genres, status, total_chapters, description, cover_url
			FROM manga
			WHERE %s
			ORDER BY title ASC
			LIMIT ?`,
			strings.Join(clauses, " AND "),
		),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list manga: %w", err)
	}
	defer rows.Close()

	var results []models.Manga
	for rows.Next() {
		manga, err := scanManga(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, manga)
	}

	return results, rows.Err()
}

func (s *Store) GetMangaByID(ctx context.Context, mangaID string) (models.Manga, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, title, author, genres, status, total_chapters, description, cover_url
		FROM manga WHERE id = ?`,
		mangaID,
	)

	manga, err := scanManga(row)
	if err != nil {
		return models.Manga{}, fmt.Errorf("get manga by id: %w", err)
	}
	return manga, nil
}

func (s *Store) UpsertLibraryEntry(ctx context.Context, userID string, entry models.LibraryEntry) (models.LibraryEntry, error) {
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

func (s *Store) GetUserLibrary(ctx context.Context, userID string) ([]models.LibraryEntry, error) {
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

type mangaScanner interface {
	Scan(dest ...any) error
}

func scanManga(scanner mangaScanner) (models.Manga, error) {
	var manga models.Manga
	var genresRaw string

	err := scanner.Scan(
		&manga.ID,
		&manga.Title,
		&manga.Author,
		&genresRaw,
		&manga.Status,
		&manga.TotalChapters,
		&manga.Description,
		&manga.CoverURL,
	)
	if err != nil {
		return models.Manga{}, err
	}

	if err := json.Unmarshal([]byte(genresRaw), &manga.Genres); err != nil {
		return models.Manga{}, fmt.Errorf("unmarshal genres: %w", err)
	}

	return manga, nil
}
