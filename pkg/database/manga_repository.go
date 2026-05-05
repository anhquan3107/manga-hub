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
	args := make([]any, 0, 8)
	from := "manga"
	filters := query.Filters

	if query.Query != "" {
		from = "manga JOIN manga_fts ON manga_fts.rowid = manga.rowid"
		clauses = append(clauses, "manga_fts MATCH ?")
		args = append(args, query.Query)
	}
	if len(filters.Genres) > 0 {
		for _, genre := range filters.Genres {
			genre = strings.TrimSpace(genre)
			if genre == "" {
				continue
			}
			clauses = append(clauses, "LOWER(manga.genres) LIKE ?")
			args = append(args, "%"+strings.ToLower(genre)+"%")
		}
	}
	if filters.Status != "" {
			clauses = append(clauses, "LOWER(manga.status) = ?")
		args = append(args, strings.ToLower(filters.Status))
	}
	if filters.YearRange[0] > 0 {
			clauses = append(clauses, "manga.year >= ?")
		args = append(args, filters.YearRange[0])
	}
	if filters.YearRange[1] > 0 {
			clauses = append(clauses, "manga.year <= ?")
		args = append(args, filters.YearRange[1])
	}
	if filters.Rating > 0 {
			clauses = append(clauses, "manga.rating >= ?")
		args = append(args, filters.Rating)
	}

	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	args = append(args, limit)
	orderBy := buildMangaOrder(filters.SortBy)

	rows, err := s.db.QueryContext(
		ctx,
		fmt.Sprintf(
			`SELECT manga.id, manga.title, manga.author, manga.genres, manga.status, manga.year, manga.rating, manga.popularity, manga.total_chapters, manga.description, manga.cover_url
			FROM %s
			WHERE %s
			ORDER BY %s
			LIMIT ?`,
			from,
			strings.Join(clauses, " AND "),
			orderBy,
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
		`SELECT id, title, author, genres, status, year, rating, popularity, total_chapters, description, cover_url
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
		SET title = ?, author = ?, genres = ?, status = ?, year = ?, rating = ?, popularity = ?, total_chapters = ?, description = ?, cover_url = ?
		WHERE id = ?`,
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
		&manga.Year,
		&manga.Rating,
		&manga.Popularity,
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

func buildMangaOrder(sortBy string) string {
	switch strings.ToLower(strings.TrimSpace(sortBy)) {
	case "popularity":
		return "manga.popularity DESC, manga.rating DESC, manga.title ASC"
	case "rating":
		return "manga.rating DESC, manga.popularity DESC, manga.title ASC"
	case "recent":
		return "manga.year DESC, manga.title ASC"
	default:
		return "manga.title ASC"
	}
}
