package service

import (
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
)

func openOzonCatalogRefreshTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(): %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.Shop{}); err != nil {
		t.Fatalf("AutoMigrate(): %v", err)
	}

	return db
}

func createOzonCatalogRefreshTestShop(t *testing.T, db *gorm.DB) uint {
	t.Helper()

	user := &model.User{
		Username:     "catalog-owner-" + strings.ReplaceAll(t.Name(), "/", "-"),
		PasswordHash: "hash",
		DisplayName:  "owner",
		Role:         model.RoleShopAdmin,
		Status:       "active",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("db.Create(user): %v", err)
	}

	shop := &model.Shop{
		Name:                "catalog-shop",
		ClientID:            "catalog-client-" + strings.ReplaceAll(t.Name(), "/", "-"),
		ApiKey:              "catalog-key",
		IsActive:            true,
		ExecutionEngineMode: model.ShopExecutionEngineAuto,
		OwnerID:             user.ID,
	}
	if err := db.Create(shop).Error; err != nil {
		t.Fatalf("db.Create(shop): %v", err)
	}

	return shop.ID
}

func TestRefreshShopCatalogSyncWaitsForExistingRefreshToFinish(t *testing.T) {
	t.Parallel()

	db := openOzonCatalogRefreshTestDB(t)
	shopID := createOzonCatalogRefreshTestShop(t, db)
	service := NewOzonCatalogService(repository.NewOzonCatalogRepository(db), repository.NewShopRepository(db))

	startedAt := time.Now()
	service.refreshStateBy[shopID] = &ozonCatalogRefreshState{
		Running:       true,
		LastStartedAt: &startedAt,
	}

	go func() {
		time.Sleep(80 * time.Millisecond)
		service.updateRefreshState(shopID, nil)
	}()

	if err := service.RefreshShopCatalogSync(shopID); err != nil {
		t.Fatalf("RefreshShopCatalogSync() error = %v", err)
	}
}
