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

func TestResolveAutoPromotionTargetDate(t *testing.T) {
	t.Parallel()

	reference := time.Date(2026, 3, 22, 15, 30, 0, 0, time.FixedZone("CST", 8*3600))
	tests := []struct {
		name       string
		mode       string
		targetDate string
		wantMode   string
		wantDate   string
	}{
		{name: "yesterday mode", mode: model.AutoPromotionTargetDateModeYesterday, wantMode: model.AutoPromotionTargetDateModeYesterday, wantDate: "2026-03-21"},
		{name: "today mode", mode: model.AutoPromotionTargetDateModeToday, wantMode: model.AutoPromotionTargetDateModeToday, wantDate: "2026-03-22"},
		{name: "custom mode", mode: model.AutoPromotionTargetDateModeCustom, targetDate: "2026-03-05", wantMode: model.AutoPromotionTargetDateModeCustom, wantDate: "2026-03-05"},
		{name: "legacy request without mode falls back to custom", mode: "", targetDate: "2026-03-09", wantMode: model.AutoPromotionTargetDateModeCustom, wantDate: "2026-03-09"},
		{name: "empty request falls back to yesterday", mode: "", targetDate: "", wantMode: model.AutoPromotionTargetDateModeYesterday, wantDate: "2026-03-21"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotMode, gotDate, err := resolveAutoPromotionTargetDate(tt.mode, tt.targetDate, reference)
			if err != nil {
				t.Fatalf("resolveAutoPromotionTargetDate() error = %v", err)
			}
			if gotMode != tt.wantMode {
				t.Fatalf("resolveAutoPromotionTargetDate() mode = %s, want %s", gotMode, tt.wantMode)
			}
			if got := gotDate.Format("2006-01-02"); got != tt.wantDate {
				t.Fatalf("resolveAutoPromotionTargetDate() date = %s, want %s", got, tt.wantDate)
			}
		})
	}
}

func TestResolveAutoPromotionTargetDateErrors(t *testing.T) {
	t.Parallel()

	reference := time.Date(2026, 3, 22, 15, 30, 0, 0, time.UTC)
	tests := []struct {
		name       string
		mode       string
		targetDate string
	}{
		{name: "invalid mode", mode: "weekly", targetDate: ""},
		{name: "custom without date", mode: model.AutoPromotionTargetDateModeCustom, targetDate: ""},
		{name: "custom invalid date", mode: model.AutoPromotionTargetDateModeCustom, targetDate: "2026/03/05"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, _, err := resolveAutoPromotionTargetDate(tt.mode, tt.targetDate, reference); err == nil {
				t.Fatalf("resolveAutoPromotionTargetDate() expected error")
			}
		})
	}
}

func TestValidateAutoPromotionConfigTargetDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mode       string
		targetDate string
		wantMode   string
		wantDate   string
	}{
		{name: "yesterday keeps nil target date", mode: model.AutoPromotionTargetDateModeYesterday, wantMode: model.AutoPromotionTargetDateModeYesterday},
		{name: "today keeps nil target date", mode: model.AutoPromotionTargetDateModeToday, wantMode: model.AutoPromotionTargetDateModeToday},
		{name: "custom stores explicit date", mode: model.AutoPromotionTargetDateModeCustom, targetDate: "2026-03-05", wantMode: model.AutoPromotionTargetDateModeCustom, wantDate: "2026-03-05"},
		{name: "legacy config without mode stays custom", mode: "", targetDate: "2026-03-09", wantMode: model.AutoPromotionTargetDateModeCustom, wantDate: "2026-03-09"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotMode, gotDate, err := validateAutoPromotionConfigTargetDate(tt.mode, tt.targetDate)
			if err != nil {
				t.Fatalf("validateAutoPromotionConfigTargetDate() error = %v", err)
			}
			if gotMode != tt.wantMode {
				t.Fatalf("validateAutoPromotionConfigTargetDate() mode = %s, want %s", gotMode, tt.wantMode)
			}
			if tt.wantDate == "" {
				if gotDate != nil {
					t.Fatalf("validateAutoPromotionConfigTargetDate() date = %v, want nil", gotDate)
				}
				return
			}
			if gotDate == nil {
				t.Fatalf("validateAutoPromotionConfigTargetDate() date = nil, want %s", tt.wantDate)
			}
			if got := gotDate.Format("2006-01-02"); got != tt.wantDate {
				t.Fatalf("validateAutoPromotionConfigTargetDate() date = %s, want %s", got, tt.wantDate)
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
