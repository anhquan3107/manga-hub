package review

import (
	"context"
	"testing"

	"mangahub/internal/cache"
	"mangahub/pkg/database"
	"mangahub/pkg/models"

	"github.com/alicebob/miniredis/v2"
)

func setupReviewTest(t *testing.T) (*database.Store, *Service) {
	t.Helper()

	store, err := database.NewSQLiteStore(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	ctx := context.Background()
	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}
	service := NewService(store)
	return store, service
}

func TestListReviewsCacheInvalidation(t *testing.T) {
	store, service := setupReviewTest(t)
	defer store.Close()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	redisCache, err := cache.NewRedis(mr.Addr(), "", 0)
	if err != nil {
		t.Fatalf("cache.NewRedis: %v", err)
	}
	defer redisCache.Close()

	service.SetCache(redisCache)
	ctx := context.Background()

	// insert manga required by review service
	if err := store.InsertManga(ctx, models.Manga{ID: "manga-1", Title: "A"}); err != nil {
		t.Fatalf("InsertManga: %v", err)
	}

	// create users referenced by reviews
	if _, err := store.CreateUser(ctx, "user-a", "alice", "alice@example.com", "hash"); err != nil {
		t.Fatalf("CreateUser a: %v", err)
	}
	if _, err := store.CreateUser(ctx, "user-b", "bob", "bob@example.com", "hash"); err != nil {
		t.Fatalf("CreateUser b: %v", err)
	}
	if _, err := store.CreateUser(ctx, "user-c", "carol", "carol@example.com", "hash"); err != nil {
		t.Fatalf("CreateUser c: %v", err)
	}

	// insert initial review via store (bypassing service) - should be observed after first List
	if _, err := store.UpsertReview(ctx, models.Review{UserID: "user-a", MangaID: "manga-1", Rating: 8, Text: "good"}); err != nil {
		t.Fatalf("UpsertReview store: %v", err)
	}

	list1, err := service.ListReviews(ctx, "manga-1", 10, "recent")
	if err != nil {
		t.Fatalf("ListReviews first: %v", err)
	}
	if len(list1) != 1 {
		t.Fatalf("expected 1 review, got %d", len(list1))
	}

	// add another review directly to DB (simulating external change)
	if _, err := store.UpsertReview(ctx, models.Review{UserID: "user-b", MangaID: "manga-1", Rating: 9, Text: "great"}); err != nil {
		t.Fatalf("UpsertReview store 2: %v", err)
	}

	// cached result should still be the old one
	list2, err := service.ListReviews(ctx, "manga-1", 10, "recent")
	if err != nil {
		t.Fatalf("ListReviews second: %v", err)
	}
	if len(list2) != 1 {
		t.Fatalf("expected cached list to have 1 review, got %d", len(list2))
	}

	// now use the service to upsert (will invalidate)
	if _, err := service.UpsertReview(ctx, "user-c", "manga-1", models.CreateReviewRequest{Rating: 7, Text: "ok"}); err != nil {
		t.Fatalf("service.UpsertReview: %v", err)
	}

	list3, err := service.ListReviews(ctx, "manga-1", 10, "recent")
	if err != nil {
		t.Fatalf("ListReviews third: %v", err)
	}
	if len(list3) != 3 {
		t.Fatalf("expected invalidated list to have 3 reviews, got %d", len(list3))
	}
}
