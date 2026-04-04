package service

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
)

func mustJSON[T any](t *testing.T, value T) []byte {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return raw
}

func boolPtr(value bool) *bool {
	return &value
}

func openSearchCPOServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(): %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.Shop{}, &model.PromotionAction{}, &model.SearchCPOConfig{}); err != nil {
		t.Fatalf("AutoMigrate(): %v", err)
	}

	return db
}

func newSearchCPOConfigTestService(db *gorm.DB) *SearchCPOService {
	return &SearchCPOService{
		repo:          repository.NewSearchCPORepository(db),
		promotionRepo: repository.NewPromotionRepository(db),
	}
}

func createSearchCPOServiceTestShop(t *testing.T, db *gorm.DB) *model.Shop {
	t.Helper()

	user := &model.User{
		Username:     "search-cpo-owner-" + strings.ReplaceAll(t.Name(), "/", "-"),
		PasswordHash: "hash",
		DisplayName:  "owner",
		Role:         model.RoleShopAdmin,
		Status:       "active",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("db.Create(user): %v", err)
	}

	shop := &model.Shop{
		Name:                "search-cpo-shop",
		ClientID:            "client-" + strings.ReplaceAll(t.Name(), "/", "-"),
		ApiKey:              "api-key",
		IsActive:            true,
		ExecutionEngineMode: model.ShopExecutionEngineAuto,
		OwnerID:             user.ID,
	}
	if err := db.Create(shop).Error; err != nil {
		t.Fatalf("db.Create(shop): %v", err)
	}

	return shop
}

func createSearchCPOServiceTestAction(t *testing.T, db *gorm.DB, shopID uint, source string, actionID int64, sourceActionID string) *model.PromotionAction {
	t.Helper()

	action := &model.PromotionAction{
		ShopID:         shopID,
		ActionID:       actionID,
		Source:         source,
		SourceActionID: sourceActionID,
		Title:          source + "-action",
		Status:         "active",
	}
	if err := repository.NewPromotionRepository(db).CreatePromotionAction(action); err != nil {
		t.Fatalf("CreatePromotionAction(): %v", err)
	}
	return action
}

func TestToSearchCPOConfigDTOIncludesExitActionIDs(t *testing.T) {
	t.Parallel()

	config := &model.SearchCPOConfig{
		OfficialActionIDs:     mustJSON(t, []uint{11}),
		ShopActionIDs:         mustJSON(t, []uint{22}),
		ExitOfficialActionIDs: mustJSON(t, []uint{33}),
		ExitShopActionIDs:     mustJSON(t, []uint{44}),
		AutoEnabled:           true,
		ScheduleTime:          "09:05",
	}

	dto := toSearchCPOConfigDTO(config)
	if dto == nil {
		t.Fatal("toSearchCPOConfigDTO() = nil")
	}
	if !reflect.DeepEqual(dto.ExitOfficialActionIDs, []uint{33}) {
		t.Fatalf("ExitOfficialActionIDs = %#v", dto.ExitOfficialActionIDs)
	}
	if !reflect.DeepEqual(dto.ExitShopActionIDs, []uint{44}) {
		t.Fatalf("ExitShopActionIDs = %#v", dto.ExitShopActionIDs)
	}
}

func TestSearchCPOServiceGetConfigReturnsEmptyExitActionIDsByDefault(t *testing.T) {
	t.Parallel()

	db := openSearchCPOServiceTestDB(t)
	shop := createSearchCPOServiceTestShop(t, db)
	service := newSearchCPOConfigTestService(db)

	config, err := service.GetConfig(shop.ID)
	if err != nil {
		t.Fatalf("GetConfig(): %v", err)
	}

	if !reflect.DeepEqual(config.OfficialActionIDs, []uint{}) {
		t.Fatalf("OfficialActionIDs = %#v, want empty slice", config.OfficialActionIDs)
	}
	if !reflect.DeepEqual(config.ShopActionIDs, []uint{}) {
		t.Fatalf("ShopActionIDs = %#v, want empty slice", config.ShopActionIDs)
	}
	if !reflect.DeepEqual(config.ExitOfficialActionIDs, []uint{}) {
		t.Fatalf("ExitOfficialActionIDs = %#v, want empty slice", config.ExitOfficialActionIDs)
	}
	if !reflect.DeepEqual(config.ExitShopActionIDs, []uint{}) {
		t.Fatalf("ExitShopActionIDs = %#v, want empty slice", config.ExitShopActionIDs)
	}
}

func TestSearchCPOServiceUpdateConfigPersistsAndUpdatesExitActionIDs(t *testing.T) {
	t.Parallel()

	db := openSearchCPOServiceTestDB(t)
	shop := createSearchCPOServiceTestShop(t, db)
	service := newSearchCPOConfigTestService(db)

	enterOfficial := createSearchCPOServiceTestAction(t, db, shop.ID, "official", 1001, "")
	enterShop := createSearchCPOServiceTestAction(t, db, shop.ID, "shop", 2001, "shop-enter")
	exitOfficialA := createSearchCPOServiceTestAction(t, db, shop.ID, "official", 1002, "")
	exitOfficialB := createSearchCPOServiceTestAction(t, db, shop.ID, "official", 1003, "")
	exitShopA := createSearchCPOServiceTestAction(t, db, shop.ID, "shop", 2002, "shop-exit-a")
	exitShopB := createSearchCPOServiceTestAction(t, db, shop.ID, "shop", 2003, "shop-exit-b")

	first, err := service.UpdateConfig(&dto.SearchCPOConfigRequest{
		ShopID:                shop.ID,
		OfficialActionIDs:     []uint{enterOfficial.ID, enterOfficial.ID},
		ShopActionIDs:         []uint{enterShop.ID, enterShop.ID},
		ExitOfficialActionIDs: []uint{exitOfficialA.ID, exitOfficialA.ID},
		ExitShopActionIDs:     []uint{exitShopA.ID, exitShopA.ID},
		AutoEnabled:           boolPtr(true),
		ScheduleTime:          "10:30",
	})
	if err != nil {
		t.Fatalf("UpdateConfig(first): %v", err)
	}

	if !reflect.DeepEqual(first.ExitOfficialActionIDs, []uint{exitOfficialA.ID}) {
		t.Fatalf("first ExitOfficialActionIDs = %#v, want %#v", first.ExitOfficialActionIDs, []uint{exitOfficialA.ID})
	}
	if !reflect.DeepEqual(first.ExitShopActionIDs, []uint{exitShopA.ID}) {
		t.Fatalf("first ExitShopActionIDs = %#v, want %#v", first.ExitShopActionIDs, []uint{exitShopA.ID})
	}

	second, err := service.UpdateConfig(&dto.SearchCPOConfigRequest{
		ShopID:                shop.ID,
		OfficialActionIDs:     []uint{enterOfficial.ID},
		ShopActionIDs:         []uint{enterShop.ID},
		ExitOfficialActionIDs: []uint{exitOfficialB.ID, exitOfficialB.ID},
		ExitShopActionIDs:     []uint{exitShopB.ID, exitShopB.ID},
		AutoEnabled:           boolPtr(true),
		ScheduleTime:          "10:30",
	})
	if err != nil {
		t.Fatalf("UpdateConfig(second): %v", err)
	}

	if !reflect.DeepEqual(second.ExitOfficialActionIDs, []uint{exitOfficialB.ID}) {
		t.Fatalf("second ExitOfficialActionIDs = %#v, want %#v", second.ExitOfficialActionIDs, []uint{exitOfficialB.ID})
	}
	if !reflect.DeepEqual(second.ExitShopActionIDs, []uint{exitShopB.ID}) {
		t.Fatalf("second ExitShopActionIDs = %#v, want %#v", second.ExitShopActionIDs, []uint{exitShopB.ID})
	}

	stored, err := service.GetConfig(shop.ID)
	if err != nil {
		t.Fatalf("GetConfig(): %v", err)
	}

	if !reflect.DeepEqual(stored.OfficialActionIDs, []uint{enterOfficial.ID}) {
		t.Fatalf("stored OfficialActionIDs = %#v, want %#v", stored.OfficialActionIDs, []uint{enterOfficial.ID})
	}
	if !reflect.DeepEqual(stored.ShopActionIDs, []uint{enterShop.ID}) {
		t.Fatalf("stored ShopActionIDs = %#v, want %#v", stored.ShopActionIDs, []uint{enterShop.ID})
	}
	if !reflect.DeepEqual(stored.ExitOfficialActionIDs, []uint{exitOfficialB.ID}) {
		t.Fatalf("stored ExitOfficialActionIDs = %#v, want %#v", stored.ExitOfficialActionIDs, []uint{exitOfficialB.ID})
	}
	if !reflect.DeepEqual(stored.ExitShopActionIDs, []uint{exitShopB.ID}) {
		t.Fatalf("stored ExitShopActionIDs = %#v, want %#v", stored.ExitShopActionIDs, []uint{exitShopB.ID})
	}
}

func TestSearchCPOServiceUpdateConfigRejectsInvalidExitActionIDs(t *testing.T) {
	t.Parallel()

	db := openSearchCPOServiceTestDB(t)
	shop := createSearchCPOServiceTestShop(t, db)
	service := newSearchCPOConfigTestService(db)

	_, err := service.UpdateConfig(&dto.SearchCPOConfigRequest{
		ShopID:                shop.ID,
		ExitOfficialActionIDs: []uint{999999},
		AutoEnabled:           boolPtr(true),
		ScheduleTime:          "09:05",
	})
	if err == nil {
		t.Fatal("UpdateConfig() error = nil, want invalid exit action error")
	}
	if !strings.Contains(err.Error(), "无效") {
		t.Fatalf("UpdateConfig() error = %q, want invalid action message", err.Error())
	}
}

func TestResolveSearchCPOCatalogItem(t *testing.T) {
	t.Parallel()

	offerCatalog := map[string]model.OzonProductCatalogItem{
		"OFFER-1": {OzonProductID: 101, OfferID: "OFFER-1"},
	}
	skuCatalog := map[int64]model.OzonProductCatalogItem{
		998877: {OzonProductID: 202, SKU: 998877},
	}

	t.Run("prefer offer id match", func(t *testing.T) {
		t.Parallel()

		got := resolveSearchCPOCatalogItem("OFFER-1", &model.SearchCPOProduct{SKU: "998877"}, offerCatalog, skuCatalog)
		if got == nil || got.OzonProductID != 101 {
			t.Fatalf("resolveSearchCPOCatalogItem() = %+v, want offer-id catalog item", got)
		}
	})

	t.Run("fallback to numeric sku match", func(t *testing.T) {
		t.Parallel()

		got := resolveSearchCPOCatalogItem("998877", &model.SearchCPOProduct{SKU: "998877"}, map[string]model.OzonProductCatalogItem{}, skuCatalog)
		if got == nil || got.OzonProductID != 202 {
			t.Fatalf("resolveSearchCPOCatalogItem() = %+v, want sku catalog item", got)
		}
	})
}

func TestResolveSearchCPOOfficialContext(t *testing.T) {
	t.Parallel()

	t.Run("prefer local product id and price", func(t *testing.T) {
		t.Parallel()

		state := &searchCPORunItemState{
			LocalProduct: &model.Product{
				OzonProductID: 555,
				CurrentPrice:  320.5,
			},
			CacheItem: &model.SearchCPOProduct{Price: 280},
			CatalogItem: &model.OzonProductCatalogItem{
				OzonProductID: 777,
				Price:         260,
			},
		}

		if got := resolveSearchCPOOfficialProductID(state); got != 555 {
			t.Fatalf("resolveSearchCPOOfficialProductID() = %d, want 555", got)
		}
		if got := resolveSearchCPOActionPrice(state); got != 320.5 {
			t.Fatalf("resolveSearchCPOActionPrice() = %v, want 320.5", got)
		}
	})

	t.Run("fallback to catalog id and catalog price", func(t *testing.T) {
		t.Parallel()

		state := &searchCPORunItemState{
			CacheItem: &model.SearchCPOProduct{Price: 0},
			CatalogItem: &model.OzonProductCatalogItem{
				OzonProductID: 888,
				Price:         199,
			},
		}

		if got := resolveSearchCPOOfficialProductID(state); got != 888 {
			t.Fatalf("resolveSearchCPOOfficialProductID() = %d, want 888", got)
		}
		if got := resolveSearchCPOActionPrice(state); got != 199 {
			t.Fatalf("resolveSearchCPOActionPrice() = %v, want 199", got)
		}
	})

	t.Run("prefer cache price when local current price is invalid", func(t *testing.T) {
		t.Parallel()

		state := &searchCPORunItemState{
			LocalProduct: &model.Product{
				OzonProductID: 999,
				CurrentPrice:  0,
			},
			CacheItem:   &model.SearchCPOProduct{Price: 145},
			CatalogItem: &model.OzonProductCatalogItem{Price: 199},
		}

		if got := resolveSearchCPOActionPrice(state); got != 145 {
			t.Fatalf("resolveSearchCPOActionPrice() = %v, want 145", got)
		}
	})
}

func TestSummarizeSearchCPORowStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		state    *searchCPORunItemState
		want     string
		wantOff  string
		wantShop string
	}{
		{
			name: "partial when official fails but shop succeeds",
			state: &searchCPORunItemState{
				OfficialResults: []dto.SearchCPORunActionResult{{Status: model.SearchCPOItemStatusFailed}},
				ShopResults:     []dto.SearchCPORunActionResult{{Status: model.SearchCPOItemStatusSuccess}},
			},
			want:     model.SearchCPOItemStatusPartialSuccess,
			wantOff:  model.SearchCPOItemStatusFailed,
			wantShop: model.SearchCPOItemStatusSuccess,
		},
		{
			name: "failed when only official side fails",
			state: &searchCPORunItemState{
				OfficialResults: []dto.SearchCPORunActionResult{{Status: model.SearchCPOItemStatusFailed}},
			},
			want:     model.SearchCPOItemStatusFailed,
			wantOff:  model.SearchCPOItemStatusFailed,
			wantShop: model.SearchCPOItemStatusSkipped,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, gotOff, gotShop := summarizeSearchCPORowStatus(tt.state)
			if got != tt.want || gotOff != tt.wantOff || gotShop != tt.wantShop {
				t.Fatalf("summarizeSearchCPORowStatus() = (%q, %q, %q), want (%q, %q, %q)", got, gotOff, gotShop, tt.want, tt.wantOff, tt.wantShop)
			}
		})
	}
}

func TestSummarizeSearchCPORunStatus(t *testing.T) {
	t.Parallel()

	if got := summarizeSearchCPORunStatus(1, 0, 0, 1); got != model.SearchCPORunStatusPartialSuccess {
		t.Fatalf("summarizeSearchCPORunStatus(partial only) = %q, want %q", got, model.SearchCPORunStatusPartialSuccess)
	}
	if got := summarizeSearchCPORunStatus(1, 1, 0, 0); got != model.SearchCPORunStatusPartialSuccess {
		t.Fatalf("summarizeSearchCPORunStatus(mixed) = %q, want %q", got, model.SearchCPORunStatusPartialSuccess)
	}
	if got := summarizeSearchCPORunStatus(2, 0, 0, 0); got != model.SearchCPORunStatusSuccess {
		t.Fatalf("summarizeSearchCPORunStatus(success) = %q, want %q", got, model.SearchCPORunStatusSuccess)
	}
}
