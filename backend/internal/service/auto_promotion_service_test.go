package service

import (
	"testing"
	"time"

	"ozon-manager/internal/model"
)

func TestChooseOfficialActionPrice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		current    float64
		candidate  float64
		maxAllowed float64
		want       float64
	}{
		{name: "prefer candidate price", current: 500, candidate: 320, maxAllowed: 400, want: 320},
		{name: "fallback to current when within max", current: 280, candidate: 0, maxAllowed: 300, want: 280},
		{name: "cap by max price", current: 320, candidate: 0, maxAllowed: 300, want: 300},
		{name: "fallback to current only", current: 280, candidate: 0, maxAllowed: 0, want: 280},
		{name: "fallback to max only", current: 0, candidate: 0, maxAllowed: 260, want: 260},
		{name: "zero when nothing valid", current: 0, candidate: 0, maxAllowed: 0, want: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := chooseOfficialActionPrice(tt.current, tt.candidate, tt.maxAllowed); got != tt.want {
				t.Fatalf("chooseOfficialActionPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectEligibleItemsRequiresAllChosenActions(t *testing.T) {
	t.Parallel()

	targetDate := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	service := &AutoPromotionService{}
	catalogItems := []model.OzonProductCatalogItem{
		{OzonProductID: 101, ListingDate: &targetDate},
		{OzonProductID: 202, ListingDate: &targetDate},
	}
	localProducts := map[int64]model.Product{
		101: {ID: 1, OzonProductID: 101, SourceSKU: "SKU-101", Name: "A"},
		202: {ID: 2, OzonProductID: 202, SourceSKU: "SKU-202", Name: "B"},
	}
	officialActions := []model.PromotionAction{{ID: 11, ActionID: 9001, Source: "official", Title: "弹性"}}
	shopActions := []model.PromotionAction{{ID: 22, Source: "shop", SourceActionID: "shop-28", Title: "28"}}

	officialCandidates := []model.PromotionActionCandidate{
		{PromotionActionID: 11, SourceSKU: "SKU-101", Status: model.PromotionActionCandidateStatusCandidate, ActionPrice: 150, MaxActionPrice: 180},
		{PromotionActionID: 11, SourceSKU: "SKU-202", Status: model.PromotionActionCandidateStatusCandidate, ActionPrice: 150, MaxActionPrice: 180},
	}
	shopCandidates := []model.PromotionActionCandidate{
		{PromotionActionID: 22, SourceSKU: "SKU-101", Status: model.PromotionActionCandidateStatusCandidate},
	}

	selected := service.selectEligibleItems(catalogItems, localProducts, officialActions, shopActions, officialCandidates, shopCandidates, nil)
	if len(selected) != 1 {
		t.Fatalf("selected len = %d, want 1", len(selected))
	}

	state, exists := selected["SKU-101"]
	if !exists {
		t.Fatalf("expected SKU-101 to be selected")
	}
	if len(state.OfficialResults) != 1 || len(state.ShopResults) != 1 {
		t.Fatalf("expected both official and shop results to be recorded")
	}
}
