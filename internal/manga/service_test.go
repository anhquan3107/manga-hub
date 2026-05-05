package manga

import (
	"context"
	"path/filepath"
	"testing"

	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

func TestListMangaReturnsAllManga(t *testing.T) {
	store, service := setupMangaTest(t)
	defer store.Close()

	ctx := context.Background()

	// Create some test manga
	manga1 := models.Manga{
		ID:            "manga-1",
		Title:         "Attack on Titan",
		Author:        "Hajime Isayama",
		Genres:        []string{"Action", "Dark"},
		Status:        "completed",
		TotalChapters: 139,
		Description:   "Humanity fights giant monsters",
		CoverURL:      "http://example.com/aot.jpg",
	}
	manga2 := models.Manga{
		ID:            "manga-2",
		Title:         "Death Note",
		Author:        "Tsugumi Ohba",
		Genres:        []string{"Thriller", "Supernatural"},
		Status:        "completed",
		TotalChapters: 108,
		Description:   "A notebook that kills people",
		CoverURL:      "http://example.com/dn.jpg",
	}

	store.InsertManga(ctx, manga1)
	store.InsertManga(ctx, manga2)

	list, err := service.List(ctx, models.MangaQuery{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 manga, got %d", len(list))
	}
}

func TestGetByIDReturnsSpecificManga(t *testing.T) {
	store, service := setupMangaTest(t)
	defer store.Close()

	ctx := context.Background()

	manga := models.Manga{
		ID:            "manga-1",
		Title:         "One Piece",
		Author:        "Eiichiro Oda",
		Genres:        []string{"Adventure", "Action"},
		Status:        "ongoing",
		TotalChapters: 1000,
		Description:   "Pirates seeking treasure",
		CoverURL:      "http://example.com/op.jpg",
	}
	store.InsertManga(ctx, manga)

	result, err := service.GetByID(ctx, "manga-1")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}

	if result.ID != "manga-1" {
		t.Fatalf("expected ID manga-1, got %s", result.ID)
	}
	if result.Title != "One Piece" {
		t.Fatalf("expected title One Piece, got %s", result.Title)
	}
	if result.TotalChapters != 1000 {
		t.Fatalf("expected 1000 chapters, got %d", result.TotalChapters)
	}
}

func TestCreateMangaInsertsNewRecord(t *testing.T) {
	store, service := setupMangaTest(t)
	defer store.Close()

	ctx := context.Background()

	req := models.CreateMangaRequest{
		ID:            "manga-new",
		Title:         "Naruto",
		Author:        "Masashi Kishimoto",
		Genres:        []string{"Action", "Adventure"},
		Status:        "completed",
		Year:          2002,
		Rating:        8,
		Popularity:    95,
		TotalChapters: 700,
		Description:   "Ninja story",
		CoverURL:      "http://example.com/naruto.jpg",
	}

	created, err := service.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if created.ID != "manga-new" {
		t.Fatalf("expected ID manga-new, got %s", created.ID)
	}

	retrieved, err := service.GetByID(ctx, "manga-new")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}

	if retrieved.Title != "Naruto" {
		t.Fatalf("expected title Naruto, got %s", retrieved.Title)
	}
}

func TestUpdateMangaModifiesExistingRecord(t *testing.T) {
	store, service := setupMangaTest(t)
	defer store.Close()

	ctx := context.Background()

	// Create manga
	manga := models.Manga{
		ID:            "manga-1",
		Title:         "Original Title",
		Author:        "Author",
		Genres:        []string{"Action"},
		Status:        "ongoing",
		TotalChapters: 100,
		Description:   "Original description",
		CoverURL:      "http://example.com/old.jpg",
	}
	store.InsertManga(ctx, manga)

	// Update manga
	updateReq := models.UpdateMangaRequest{
		Title:         "Updated Title",
		Author:        "Updated Author",
		Status:        "completed",
		TotalChapters: 200,
		Description:   "Updated description",
		Rating:        9.0,
	}

	updated, err := service.Update(ctx, "manga-1", updateReq)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if updated.Title != "Updated Title" {
		t.Fatalf("expected updated title, got %s", updated.Title)
	}
	if updated.TotalChapters != 200 {
		t.Fatalf("expected 200 chapters, got %d", updated.TotalChapters)
	}
}

func TestDeleteMangaRemovesRecord(t *testing.T) {
	store, service := setupMangaTest(t)
	defer store.Close()

	ctx := context.Background()

	manga := models.Manga{
		ID:     "manga-1",
		Title:  "Test Manga",
		Author: "Author",
		Status: "ongoing",
	}
	store.InsertManga(ctx, manga)

	err := service.Delete(ctx, "manga-1")
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	_, err = service.GetByID(ctx, "manga-1")
	if err == nil {
		t.Fatalf("expected error when getting deleted manga")
	}
}

func TestSearchMangaByQuery(t *testing.T) {
	store, service := setupMangaTest(t)
	defer store.Close()

	ctx := context.Background()

	// Insert test manga
	store.InsertManga(ctx, models.Manga{
		ID:     "manga-1",
		Title:  "Attack on Titan",
		Author: "Hajime",
		Status: "completed",
	})
	store.InsertManga(ctx, models.Manga{
		ID:     "manga-2",
		Title:  "Death Note",
		Author: "Tsugumi",
		Status: "completed",
	})

	// Search by title
	results, err := service.List(ctx, models.MangaQuery{
		Query: "Attack",
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'Attack', got %d", len(results))
	}
	if results[0].Title != "Attack on Titan" {
		t.Fatalf("expected Attack on Titan, got %s", results[0].Title)
	}
}

func setupMangaTest(t *testing.T) (*database.Store, *Service) {
	t.Helper()

	store, err := database.NewSQLiteStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore returned error: %v", err)
	}

	ctx := context.Background()
	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema returned error: %v", err)
	}

	service := NewService(store)
	return store, service
}
