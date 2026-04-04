package service

import (
	"encoding/json"
	"reflect"
	"testing"

	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
)

func mustJSON[T any](t *testing.T, value T) []byte {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return raw
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
