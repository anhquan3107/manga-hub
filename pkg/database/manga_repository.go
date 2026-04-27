package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"mangahub/pkg/models"
)

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

func (s *Store) CreateManga(ctx context.Context, manga models.Manga) (models.Manga, error) {
	genres, err := json.Marshal(manga.Genres)
	if err != nil {
		return models.Manga{}, fmt.Errorf("marshal genres: %w", err)
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
		return models.Manga{}, fmt.Errorf("create manga: %w", err)
	}

	return s.GetMangaByID(ctx, manga.ID)
}

func (s *Store) UpdateMangaByID(ctx context.Context, mangaID string, manga models.Manga) (models.Manga, error) {
	genres, err := json.Marshal(manga.Genres)
	if err != nil {
		return models.Manga{}, fmt.Errorf("marshal genres: %w", err)
	}

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE manga
		SET title = ?, author = ?, genres = ?, status = ?, total_chapters = ?, description = ?, cover_url = ?
		WHERE id = ?`,
		manga.Title,
		manga.Author,
		string(genres),
		manga.Status,
		manga.TotalChapters,
		manga.Description,
		manga.CoverURL,
		mangaID,
	)
	if err != nil {
		return models.Manga{}, fmt.Errorf("update manga: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return models.Manga{}, fmt.Errorf("update manga rows affected: %w", err)
	}
	if affected == 0 {
		return models.Manga{}, sql.ErrNoRows
	}

	return s.GetMangaByID(ctx, mangaID)
}

func (s *Store) DeleteMangaByID(ctx context.Context, mangaID string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM manga WHERE id = ?`, mangaID)
	if err != nil {
		return fmt.Errorf("delete manga: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete manga rows affected: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
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
