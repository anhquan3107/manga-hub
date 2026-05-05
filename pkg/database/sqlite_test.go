package database

import (
	"context"
	"path/filepath"
	"testing"

	"mangahub/pkg/models"
)

func TestCreateUserStoresCredentials(t *testing.T) {
	store := setupDatabaseTest(t)
	defer store.Close()

	ctx := context.Background()
	user, err := store.CreateUser(ctx, "user-1", "alice", "alice@example.com", "hash123")
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	if user.ID != "user-1" {
		t.Fatalf("expected ID user-1, got %s", user.ID)
	}
	if user.Username != "alice" {
		t.Fatalf("expected username alice, got %s", user.Username)
	}
	if user.Email != "alice@example.com" {
		t.Fatalf("expected email alice@example.com, got %s", user.Email)
	}
}

func TestGetUserByID(t *testing.T) {
	store := setupDatabaseTest(t)
	defer store.Close()

	ctx := context.Background()
	mustCreateUser(t, store, ctx, "user-1", "alice", "alice@example.com", "hash")

	user, err := store.GetUserByID(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetUserByID returned error: %v", err)
	}

	if user.Username != "alice" {
		t.Fatalf("expected alice, got %s", user.Username)
	}
}

func TestGetUserByUsername(t *testing.T) {
	store := setupDatabaseTest(t)
	defer store.Close()

	ctx := context.Background()
	mustCreateUser(t, store, ctx, "user-1", "alice", "alice@example.com", "hash")

	user, err := store.GetUserByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetUserByUsername returned error: %v", err)
	}

	if user.ID != "user-1" {
		t.Fatalf("expected ID user-1, got %s", user.ID)
	}
}

func TestInsertMangaCreatesRecord(t *testing.T) {
	store := setupDatabaseTest(t)
	defer store.Close()

	ctx := context.Background()
	manga := models.Manga{
		ID:            "manga-1",
		Title:         "Test Manga",
		Author:        "Test Author",
		Genres:        []string{"Action", "Adventure"},
		Status:        "ongoing",
		Year:          2020,
		Rating:        8,
		Popularity:    85,
		TotalChapters: 150,
		Description:   "Test description",
		CoverURL:      "http://example.com/cover.jpg",
	}

	err := store.InsertManga(ctx, manga)
	if err != nil {
		t.Fatalf("InsertManga returned error: %v", err)
	}

	retrieved, err := store.GetMangaByID(ctx, "manga-1")
	if err != nil {
		t.Fatalf("GetMangaByID returned error: %v", err)
	}

	if retrieved.Title != "Test Manga" {
		t.Fatalf("expected Test Manga, got %s", retrieved.Title)
	}
}

func TestListMangaReturnsAllRecords(t *testing.T) {
	store := setupDatabaseTest(t)
	defer store.Close()

	ctx := context.Background()

	// Insert multiple manga
	for i := 1; i <= 3; i++ {
		manga := models.Manga{
			ID:     "manga-" + string(rune('0'+i)),
			Title:  "Manga " + string(rune('0'+i)),
			Author: "Author",
			Status: "ongoing",
		}
		mustInsertManga(t, store, ctx, manga)
	}

	list, err := store.ListManga(ctx, models.MangaQuery{})
	if err != nil {
		t.Fatalf("ListManga returned error: %v", err)
	}

	if len(list) != 3 {
		t.Fatalf("expected 3 manga, got %d", len(list))
	}
}

func TestUpdateMangaModifiesRecord(t *testing.T) {
	store := setupDatabaseTest(t)
	defer store.Close()

	ctx := context.Background()

	// Create manga
	manga := models.Manga{
		ID:            "manga-1",
		Title:         "Original",
		Author:        "Author",
		Status:        "ongoing",
		TotalChapters: 100,
	}
	mustInsertManga(t, store, ctx, manga)

	// Update
	updated := models.Manga{
		ID:            "manga-1",
		Title:         "Updated",
		Author:        "Updated Author",
		Status:        "completed",
		TotalChapters: 200,
	}
	result, err := store.UpdateMangaByID(ctx, "manga-1", updated)
	if err != nil {
		t.Fatalf("UpdateMangaByID returned error: %v", err)
	}

	if result.Title != "Updated" {
		t.Fatalf("expected Updated, got %s", result.Title)
	}
	if result.TotalChapters != 200 {
		t.Fatalf("expected 200 chapters, got %d", result.TotalChapters)
	}
}

func TestDeleteMangaRemovesRecord(t *testing.T) {
	store := setupDatabaseTest(t)
	defer store.Close()

	ctx := context.Background()

	// Create and delete
	manga := models.Manga{
		ID:     "manga-1",
		Title:  "Test",
		Author: "Author",
		Status: "ongoing",
	}
	mustInsertManga(t, store, ctx, manga)

	err := store.DeleteMangaByID(ctx, "manga-1")
	if err != nil {
		t.Fatalf("DeleteMangaByID returned error: %v", err)
	}

	_, err = store.GetMangaByID(ctx, "manga-1")
	if err == nil {
		t.Fatalf("expected error retrieving deleted manga")
	}
}

func TestUpsertLibraryEntryCreatesNewEntry(t *testing.T) {
	store := setupDatabaseTest(t)
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

	// Upsert (insert)
	entry := models.LibraryEntry{
		MangaID:        "manga-1",
		CurrentChapter: 5,
		Status:         "reading",
		Rating:         8,
	}
	result, err := store.UpsertLibraryEntry(ctx, "user-1", entry)
	if err != nil {
		t.Fatalf("UpsertLibraryEntry returned error: %v", err)
	}

	if result.Status != "reading" {
		t.Fatalf("expected reading, got %s", result.Status)
	}
}

func TestUpsertLibraryEntryUpdatesExisting(t *testing.T) {
	store := setupDatabaseTest(t)
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

	// Initial upsert
	mustUpsertLibraryEntry(t, store, ctx, "user-1", models.LibraryEntry{
		MangaID:        "manga-1",
		CurrentChapter: 5,
		Status:         "reading",
	})

	// Update upsert
	updated, err := store.UpsertLibraryEntry(ctx, "user-1", models.LibraryEntry{
		MangaID:        "manga-1",
		CurrentChapter: 10,
		Status:         "completed",
	})
	if err != nil {
		t.Fatalf("UpsertLibraryEntry returned error: %v", err)
	}

	if updated.CurrentChapter != 10 {
		t.Fatalf("expected chapter 10, got %d", updated.CurrentChapter)
	}
}

func TestGetUserLibraryReturnsAllEntries(t *testing.T) {
	store := setupDatabaseTest(t)
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

	library, err := store.GetUserLibrary(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetUserLibrary returned error: %v", err)
	}

	if len(library) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(library))
	}
}

func TestGetLibraryEntry(t *testing.T) {
	store := setupDatabaseTest(t)
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
		MangaID:        "manga-1",
		Status:         "reading",
		CurrentChapter: 15,
	})

	entry, err := store.GetLibraryEntry(ctx, "user-1", "manga-1")
	if err != nil {
		t.Fatalf("GetLibraryEntry returned error: %v", err)
	}

	if entry.CurrentChapter != 15 {
		t.Fatalf("expected chapter 15, got %d", entry.CurrentChapter)
	}
}

func TestDeleteLibraryEntry(t *testing.T) {
	store := setupDatabaseTest(t)
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

	// Delete
	err := store.DeleteLibraryEntry(ctx, "user-1", "manga-1")
	if err != nil {
		t.Fatalf("DeleteLibraryEntry returned error: %v", err)
	}

	// Verify deleted
	_, err = store.GetLibraryEntry(ctx, "user-1", "manga-1")
	if err == nil {
		t.Fatalf("expected error retrieving deleted entry")
	}
}

func TestInsertProgressHistory(t *testing.T) {
	store := setupDatabaseTest(t)
	defer store.Close()

	ctx := context.Background()

	// Setup user and manga first
	mustCreateUser(t, store, ctx, "user-1", "alice", "alice@example.com", "hash")
	mustInsertManga(t, store, ctx, models.Manga{
		ID:     "manga-1",
		Title:  "Test",
		Author: "Author",
		Status: "ongoing",
	})

	entry := models.ProgressHistoryEntry{
		UserID:          "user-1",
		MangaID:         "manga-1",
		PreviousChapter: 5,
		CurrentChapter:  10,
		Notes:           "Great progress",
	}

	err := store.InsertProgressHistory(ctx, entry)
	if err != nil {
		t.Fatalf("InsertProgressHistory returned error: %v", err)
	}
}

func TestGetProgressHistory(t *testing.T) {
	store := setupDatabaseTest(t)
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

	// Insert history
	mustInsertProgressHistory(t, store, ctx, models.ProgressHistoryEntry{
		UserID:          "user-1",
		MangaID:         "manga-1",
		PreviousChapter: 0,
		CurrentChapter:  5,
	})
	mustInsertProgressHistory(t, store, ctx, models.ProgressHistoryEntry{
		UserID:          "user-1",
		MangaID:         "manga-1",
		PreviousChapter: 5,
		CurrentChapter:  10,
	})

	// Retrieve
	history, err := store.GetProgressHistory(ctx, "user-1", "manga-1")
	if err != nil {
		t.Fatalf("GetProgressHistory returned error: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}
}

func setupDatabaseTest(t *testing.T) *Store {
	t.Helper()

	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore returned error: %v", err)
	}

	ctx := context.Background()
	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema returned error: %v", err)
	}

	return store
}

func mustCreateUser(t *testing.T, store *Store, ctx context.Context, id, username, email, passwordHash string) {
	t.Helper()
	if _, err := store.CreateUser(ctx, id, username, email, passwordHash); err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
}

func mustInsertManga(t *testing.T, store *Store, ctx context.Context, manga models.Manga) {
	t.Helper()
	if err := store.InsertManga(ctx, manga); err != nil {
		t.Fatalf("InsertManga returned error: %v", err)
	}
}

func mustUpsertLibraryEntry(t *testing.T, store *Store, ctx context.Context, userID string, entry models.LibraryEntry) {
	t.Helper()
	if _, err := store.UpsertLibraryEntry(ctx, userID, entry); err != nil {
		t.Fatalf("UpsertLibraryEntry returned error: %v", err)
	}
}

func mustInsertProgressHistory(t *testing.T, store *Store, ctx context.Context, entry models.ProgressHistoryEntry) {
	t.Helper()
	if err := store.InsertProgressHistory(ctx, entry); err != nil {
		t.Fatalf("InsertProgressHistory returned error: %v", err)
	}
}
