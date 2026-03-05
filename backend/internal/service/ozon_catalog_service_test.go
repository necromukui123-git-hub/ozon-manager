package service

import (
	"testing"
	"time"

	"ozon-manager/internal/model"
)

func TestResolveListingDate(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"status": map[string]interface{}{
			"created_at": "2026-03-01T10:20:30Z",
		},
	}

	got, ok := resolveListingDate(raw)
	if !ok {
		t.Fatalf("expected listing date to be resolved")
	}
	if got.Year() != 2026 || got.Month() != 3 || got.Day() != 1 {
		t.Fatalf("unexpected listing date: %s", got.Format(time.RFC3339))
	}
}

func TestEncodeDecodeOzonCatalogCursor(t *testing.T) {
	t.Parallel()

	date := time.Date(2026, 3, 4, 12, 30, 0, 0, time.UTC)
	token := encodeOzonCatalogCursor(model.OzonProductCatalogItem{
		ID:          88,
		ListingDate: &date,
	})

	decodedDate, decodedID, err := decodeOzonCatalogCursor(token)
	if err != nil {
		t.Fatalf("decode cursor error: %v", err)
	}
	if decodedID != 88 {
		t.Fatalf("decoded ID=%d, want 88", decodedID)
	}
	if decodedDate == nil || !decodedDate.Equal(date) {
		t.Fatalf("decoded date=%v, want %v", decodedDate, date)
	}
}

func TestResolveCatalogVisibilityPrefersInfoVisible(t *testing.T) {
	t.Parallel()

	got := resolveCatalogVisibility("", map[string]interface{}{"visible": true}, false, true)
	if got != "VISIBLE" {
		t.Fatalf("visibility=%q, want %q", got, "VISIBLE")
	}
}

func TestResolveCatalogVisibilitySupportsInfoInvisible(t *testing.T) {
	t.Parallel()

	got := resolveCatalogVisibility("", map[string]interface{}{"visible": false}, false, false)
	if got != "INVISIBLE" {
		t.Fatalf("visibility=%q, want %q", got, "INVISIBLE")
	}
}

func TestResolveCatalogVisibilityUsesArchivedFallback(t *testing.T) {
	t.Parallel()

	got := resolveCatalogVisibility("", nil, false, true)
	if got != "ARCHIVED" {
		t.Fatalf("visibility=%q, want %q", got, "ARCHIVED")
	}
}

func TestResolveCatalogVisibilityDefaultsToAll(t *testing.T) {
	t.Parallel()

	got := resolveCatalogVisibility("", nil, false, false)
	if got != "ALL" {
		t.Fatalf("visibility=%q, want %q", got, "ALL")
	}
}
