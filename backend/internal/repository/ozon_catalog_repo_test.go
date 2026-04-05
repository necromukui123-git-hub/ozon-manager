package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ozon-manager/internal/model"
)

func openOzonCatalogRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(): %v", err)
	}

	if err := db.AutoMigrate(&model.OzonProductCatalogItem{}); err != nil {
		t.Fatalf("AutoMigrate(): %v", err)
	}

	return db
}

func TestOzonCatalogRepositoryListByListingDateRangeIncludesBothBoundaries(t *testing.T) {
	t.Parallel()

	db := openOzonCatalogRepositoryTestDB(t)
	repo := NewOzonCatalogRepository(db)

	start := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	outside := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)

	items := []model.OzonProductCatalogItem{
		{ShopID: 9, OzonProductID: 101, OfferID: "offer-101", ListingDate: &start},
		{ShopID: 9, OzonProductID: 102, OfferID: "offer-102", ListingDate: &end},
		{ShopID: 9, OzonProductID: 103, OfferID: "offer-103", ListingDate: &outside},
		{ShopID: 10, OzonProductID: 201, OfferID: "offer-201", ListingDate: &start},
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatalf("db.Create(items): %v", err)
	}

	got, err := repo.ListByListingDateRange(9, start, end)
	if err != nil {
		t.Fatalf("ListByListingDateRange(): %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(ListByListingDateRange()) = %d, want 2", len(got))
	}
	if got[0].OzonProductID != 101 || got[1].OzonProductID != 102 {
		t.Fatalf("product IDs = %d,%d, want 101,102", got[0].OzonProductID, got[1].OzonProductID)
	}
}
