package user

import (
	"context"
	"path/filepath"
	"testing"

	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

func TestAddToLibraryAddsValidManga(t *testing.T) {
	store, service := setupUserTest(t)
	defer store.Close()

	ctx := context.Background()

	// Create user and manga
	mustCreateUser(t, store, ctx, "user-1", "alice", "alice@example.com", "hash")
	mustInsertManga(t, store, ctx, models.Manga{
		ID:     "manga-1",
		Title:  "Test Manga",
		Author: "Author",
		Status: "ongoing",
	})

	req := models.AddLibraryRequest{
		MangaID:        "manga-1",
		CurrentChapter: 5,
		CurrentVolume:  2,
		Status:         "reading",
		Rating:         8,
		Notes:          "Great manga",
	}

	entry, err := service.AddToLibrary(ctx, "user-1", req)
	if err != nil {
		t.Fatalf("AddToLibrary returned error: %v", err)
	}

	if entry.MangaID != "manga-1" {
		t.Fatalf("expected manga-1, got %s", entry.MangaID)
	}
	if entry.Status != "reading" {
		t.Fatalf("expected status reading, got %s", entry.Status)
	}
}

func TestAddToLibraryFailsForNonexistentManga(t *testing.T) {
	store, service := setupUserTest(t)
	defer store.Close()

	ctx := context.Background()
	mustCreateUser(t, store, ctx, "user-1", "alice", "alice@example.com", "hash")

	req := models.AddLibraryRequest{
		MangaID: "nonexistent",
		Status:  "reading",
	}

	_, err := service.AddToLibrary(ctx, "user-1", req)
	if err == nil {
		t.Fatalf("expected error for nonexistent manga")
	}
}

func TestUpdateProgressIncreasesChapter(t *testing.T) {
	store, service := setupUserTest(t)
	defer store.Close()

	ctx := context.Background()

	// Setup
	mustCreateUser(t, store, ctx, "user-1", "alice", "alice@example.com", "hash")
	mustInsertManga(t, store, ctx, models.Manga{
		ID:            "manga-1",
		Title:         "Test",
		Author:        "Author",
		Status:        "ongoing",
		TotalChapters: 100,
	})
	mustUpsertLibraryEntry(t, store, ctx, "user-1", models.LibraryEntry{
		MangaID:        "manga-1",
		Status:         "reading",
		CurrentChapter: 5,
	})

	// Update progress
	req := models.UpdateProgressRequest{
		MangaID:        "manga-1",
		CurrentChapter: 10,
		Status:         "reading",
	}

	result, err := service.UpdateProgress(ctx, "user-1", req)
	if err != nil {
		t.Fatalf("UpdateProgress returned error: %v", err)
	}

	if result.Entry.CurrentChapter != 10 {
		t.Fatalf("expected chapter 10, got %d", result.Entry.CurrentChapter)
	}
	if result.PreviousChapter != 5 {
		t.Fatalf("expected previous chapter 5, got %d", result.PreviousChapter)
	}
}

func TestUpdateProgressFailsForDecreasingChapter(t *testing.T) {
	store, service := setupUserTest(t)
	defer store.Close()

	ctx := context.Background()

	// Setup
	mustCreateUser(t, store, ctx, "user-1", "alice", "alice@example.com", "hash")
	mustInsertManga(t, store, ctx, models.Manga{
		ID:            "manga-1",
		Title:         "Test",
		Author:        "Author",
		Status:        "ongoing",
		TotalChapters: 100,
	})
	mustUpsertLibraryEntry(t, store, ctx, "user-1", models.LibraryEntry{
		MangaID:        "manga-1",
		Status:         "reading",
		CurrentChapter: 10,
	})

	// Try to decrease chapter
	req := models.UpdateProgressRequest{
		MangaID:        "manga-1",
		CurrentChapter: 5,
		Status:         "reading",
		Force:          false,
	}

	_, err := service.UpdateProgress(ctx, "user-1", req)
	if err == nil {
		t.Fatalf("expected error when decreasing chapter without force")
	}
}

func TestUpdateProgressWithForceOverride(t *testing.T) {
	store, service := setupUserTest(t)
	defer store.Close()

	ctx := context.Background()

	// Setup
	mustCreateUser(t, store, ctx, "user-1", "alice", "alice@example.com", "hash")
	mustInsertManga(t, store, ctx, models.Manga{
		ID:            "manga-1",
		Title:         "Test",
		Author:        "Author",
		Status:        "ongoing",
		TotalChapters: 100,
	})
	mustUpsertLibraryEntry(t, store, ctx, "user-1", models.LibraryEntry{
		MangaID:        "manga-1",
		Status:         "reading",
		CurrentChapter: 10,
	})

	// Decrease with force
	req := models.UpdateProgressRequest{
		MangaID:        "manga-1",
		CurrentChapter: 5,
		Status:         "reading",
		Force:          true,
	}

	result, err := service.UpdateProgress(ctx, "user-1", req)
	if err != nil {
		t.Fatalf("UpdateProgress with force returned error: %v", err)
	}

	if result.Entry.CurrentChapter != 5 {
		t.Fatalf("expected chapter 5 with force, got %d", result.Entry.CurrentChapter)
	}
}

func TestUpdateProgressFailsExceedingTotalChapters(t *testing.T) {
	store, service := setupUserTest(t)
	defer store.Close()

	ctx := context.Background()

	// Setup with limited chapters
	mustCreateUser(t, store, ctx, "user-1", "alice", "alice@example.com", "hash")
	mustInsertManga(t, store, ctx, models.Manga{
		ID:            "manga-1",
		Title:         "Test",
		Author:        "Author",
		Status:        "completed",
		TotalChapters: 50,
	})
	mustUpsertLibraryEntry(t, store, ctx, "user-1", models.LibraryEntry{
		MangaID:        "manga-1",
		Status:         "reading",
		CurrentChapter: 30,
	})

	// Try to exceed
	req := models.UpdateProgressRequest{
		MangaID:        "manga-1",
		CurrentChapter: 100,
		Status:         "reading",
	}

	_, err := service.UpdateProgress(ctx, "user-1", req)
	if err == nil {
		t.Fatalf("expected error when exceeding total chapters")
	}
}

func TestGetLibraryReturnsUsersManga(t *testing.T) {
	store, service := setupUserTest(t)
	defer store.Close()

	ctx := context.Background()

	// Setup
	mustCreateUser(t, store, ctx, "user-1", "alice", "alice@example.com", "hash")
	mustInsertManga(t, store, ctx, models.Manga{
		ID:     "manga-1",
		Title:  "Manga 1",
		Author: "Author",
		Status: "ongoing",
	})
	mustInsertManga(t, store, ctx, models.Manga{
		ID:     "manga-2",
		Title:  "Manga 2",
		Author: "Author",
		Status: "ongoing",
	})

	mustUpsertLibraryEntry(t, store, ctx, "user-1", models.LibraryEntry{
		MangaID: "manga-1",
		Status:  "reading",
	})
	mustUpsertLibraryEntry(t, store, ctx, "user-1", models.LibraryEntry{
		MangaID: "manga-2",
		Status:  "completed",
	})

	library, err := service.GetLibrary(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetLibrary returned error: %v", err)
	}

	if len(library) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(library))
	}
}

func TestRemoveFromLibraryDeletesEntry(t *testing.T) {
	store, service := setupUserTest(t)
	defer store.Close()

	ctx := context.Background()

	// Setup
	mustCreateUser(t, store, ctx, "user-1", "alice", "alice@example.com", "hash")
	mustInsertManga(t, store, ctx, models.Manga{
		ID:     "manga-1",
		Title:  "Test",
		Author: "Author",
		Status: "ongoing",
	})
	mustUpsertLibraryEntry(t, store, ctx, "user-1", models.LibraryEntry{
		MangaID: "manga-1",
		Status:  "reading",
	})

	err := service.RemoveFromLibrary(ctx, "user-1", "manga-1")
	if err != nil {
		t.Fatalf("RemoveFromLibrary returned error: %v", err)
	}

	library, err := service.GetLibrary(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetLibrary returned error: %v", err)
	}

	if len(library) != 0 {
		t.Fatalf("expected empty library, got %d entries", len(library))
	}
}

func TestUpdateLibraryEntry(t *testing.T) {
	store, service := setupUserTest(t)
	defer store.Close()

	ctx := context.Background()

	// Setup
	mustCreateUser(t, store, ctx, "user-1", "alice", "alice@example.com", "hash")
	mustInsertManga(t, store, ctx, models.Manga{
		ID:     "manga-1",
		Title:  "Test",
		Author: "Author",
		Status: "ongoing",
	})
	mustUpsertLibraryEntry(t, store, ctx, "user-1", models.LibraryEntry{
		MangaID: "manga-1",
		Status:  "reading",
		Rating:  0,
	})

	// Update
	updateReq := models.UpdateLibraryRequest{
		Status: "completed",
		Rating: 9,
	}

	updated, err := service.UpdateLibrary(ctx, "user-1", "manga-1", updateReq)
	if err != nil {
		t.Fatalf("UpdateLibrary returned error: %v", err)
	}

	if updated.Status != "completed" {
		t.Fatalf("expected status completed, got %s", updated.Status)
	}
	if updated.Rating != 9 {
		t.Fatalf("expected rating 9, got %d", updated.Rating)
	}
}

func TestGetProgressHistory(t *testing.T) {
	store, service := setupUserTest(t)
	defer store.Close()

	ctx := context.Background()

	// Setup
	mustCreateUser(t, store, ctx, "user-1", "alice", "alice@example.com", "hash")
	mustInsertManga(t, store, ctx, models.Manga{
		ID:            "manga-1",
		Title:         "Test",
		Author:        "Author",
		Status:        "ongoing",
		TotalChapters: 100,
	})
	mustUpsertLibraryEntry(t, store, ctx, "user-1", models.LibraryEntry{
		MangaID:        "manga-1",
		Status:         "reading",
		CurrentChapter: 0,
	})

	// Update progress multiple times
	if _, err := service.UpdateProgress(ctx, "user-1", models.UpdateProgressRequest{
		MangaID:        "manga-1",
		CurrentChapter: 10,
	}); err != nil {
		t.Fatalf("UpdateProgress returned error: %v", err)
	}
	if _, err := service.UpdateProgress(ctx, "user-1", models.UpdateProgressRequest{
		MangaID:        "manga-1",
		CurrentChapter: 20,
	}); err != nil {
		t.Fatalf("UpdateProgress returned error: %v", err)
	}

	history, err := service.GetProgressHistory(ctx, "user-1", "manga-1")
	if err != nil {
		t.Fatalf("GetProgressHistory returned error: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}
}

func setupUserTest(t *testing.T) (*database.Store, *Service) {
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

func mustCreateUser(t *testing.T, store *database.Store, ctx context.Context, id, username, email, passwordHash string) {
	t.Helper()
	if _, err := store.CreateUser(ctx, id, username, email, passwordHash); err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
}

func mustInsertManga(t *testing.T, store *database.Store, ctx context.Context, manga models.Manga) {
	t.Helper()
	if err := store.InsertManga(ctx, manga); err != nil {
		t.Fatalf("InsertManga returned error: %v", err)
	}
}

func mustUpsertLibraryEntry(t *testing.T, store *database.Store, ctx context.Context, userID string, entry models.LibraryEntry) {
	t.Helper()
	if _, err := store.UpsertLibraryEntry(ctx, userID, entry); err != nil {
		t.Fatalf("UpsertLibraryEntry returned error: %v", err)
	}
}
