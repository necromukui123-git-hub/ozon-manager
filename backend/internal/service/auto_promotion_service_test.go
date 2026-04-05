package service

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"gorm.io/datatypes"
	"ozon-manager/internal/model"
)

func mustAutoPromotionJSON[T any](t *testing.T, value T) datatypes.JSON {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return raw
}

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

func TestResolveAutoPromotionTargetDateRange(t *testing.T) {
	t.Parallel()

	reference := time.Date(2026, 3, 22, 15, 30, 0, 0, time.FixedZone("CST", 8*3600))
	tests := []struct {
		name        string
		mode        string
		targetStart string
		targetEnd   string
		legacyDate  string
		wantMode    string
		wantStart   string
		wantEnd     string
	}{
		{name: "yesterday mode", mode: model.AutoPromotionTargetDateModeYesterday, wantMode: model.AutoPromotionTargetDateModeYesterday, wantStart: "2026-03-21", wantEnd: "2026-03-21"},
		{name: "today mode", mode: model.AutoPromotionTargetDateModeToday, wantMode: model.AutoPromotionTargetDateModeToday, wantStart: "2026-03-22", wantEnd: "2026-03-22"},
		{name: "custom same day range", mode: model.AutoPromotionTargetDateModeCustom, targetStart: "2026-03-05", targetEnd: "2026-03-05", wantMode: model.AutoPromotionTargetDateModeCustom, wantStart: "2026-03-05", wantEnd: "2026-03-05"},
		{name: "custom multi day range", mode: model.AutoPromotionTargetDateModeCustom, targetStart: "2026-03-05", targetEnd: "2026-03-08", wantMode: model.AutoPromotionTargetDateModeCustom, wantStart: "2026-03-05", wantEnd: "2026-03-08"},
		{name: "legacy request without mode falls back to custom single day", mode: "", legacyDate: "2026-03-09", wantMode: model.AutoPromotionTargetDateModeCustom, wantStart: "2026-03-09", wantEnd: "2026-03-09"},
		{name: "empty request falls back to yesterday", mode: "", wantMode: model.AutoPromotionTargetDateModeYesterday, wantStart: "2026-03-21", wantEnd: "2026-03-21"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotMode, gotStart, gotEnd, err := resolveAutoPromotionTargetDateRange(tt.mode, tt.targetStart, tt.targetEnd, tt.legacyDate, reference)
			if err != nil {
				t.Fatalf("resolveAutoPromotionTargetDateRange() error = %v", err)
			}
			if gotMode != tt.wantMode {
				t.Fatalf("resolveAutoPromotionTargetDateRange() mode = %s, want %s", gotMode, tt.wantMode)
			}
			if got := gotStart.Format("2006-01-02"); got != tt.wantStart {
				t.Fatalf("resolveAutoPromotionTargetDateRange() start = %s, want %s", got, tt.wantStart)
			}
			if got := gotEnd.Format("2006-01-02"); got != tt.wantEnd {
				t.Fatalf("resolveAutoPromotionTargetDateRange() end = %s, want %s", got, tt.wantEnd)
			}
		})
	}
}

func TestResolveAutoPromotionTargetDateRangeErrors(t *testing.T) {
	t.Parallel()

	reference := time.Date(2026, 3, 22, 15, 30, 0, 0, time.UTC)
	tests := []struct {
		name        string
		mode        string
		targetStart string
		targetEnd   string
		legacyDate  string
	}{
		{name: "invalid mode", mode: "weekly"},
		{name: "custom without start", mode: model.AutoPromotionTargetDateModeCustom, targetEnd: "2026-03-05"},
		{name: "custom without end", mode: model.AutoPromotionTargetDateModeCustom, targetStart: "2026-03-05"},
		{name: "custom invalid start", mode: model.AutoPromotionTargetDateModeCustom, targetStart: "2026/03/05", targetEnd: "2026-03-08"},
		{name: "custom end before start", mode: model.AutoPromotionTargetDateModeCustom, targetStart: "2026-03-08", targetEnd: "2026-03-05"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, _, _, err := resolveAutoPromotionTargetDateRange(tt.mode, tt.targetStart, tt.targetEnd, tt.legacyDate, reference); err == nil {
				t.Fatalf("resolveAutoPromotionTargetDateRange() expected error")
			}
		})
	}
}

func TestValidateAutoPromotionConfigTargetDateRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		mode        string
		targetStart string
		targetEnd   string
		legacyDate  string
		wantMode    string
		wantStart   string
		wantEnd     string
		wantNil     bool
	}{
		{name: "yesterday keeps nil target range", mode: model.AutoPromotionTargetDateModeYesterday, wantMode: model.AutoPromotionTargetDateModeYesterday, wantNil: true},
		{name: "today keeps nil target range", mode: model.AutoPromotionTargetDateModeToday, wantMode: model.AutoPromotionTargetDateModeToday, wantNil: true},
		{name: "custom stores explicit range", mode: model.AutoPromotionTargetDateModeCustom, targetStart: "2026-03-05", targetEnd: "2026-03-08", wantMode: model.AutoPromotionTargetDateModeCustom, wantStart: "2026-03-05", wantEnd: "2026-03-08"},
		{name: "legacy config without mode stays custom single day", mode: "", legacyDate: "2026-03-09", wantMode: model.AutoPromotionTargetDateModeCustom, wantStart: "2026-03-09", wantEnd: "2026-03-09"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotMode, gotStart, gotEnd, err := validateAutoPromotionConfigTargetDateRange(tt.mode, tt.targetStart, tt.targetEnd, tt.legacyDate)
			if err != nil {
				t.Fatalf("validateAutoPromotionConfigTargetDateRange() error = %v", err)
			}
			if gotMode != tt.wantMode {
				t.Fatalf("validateAutoPromotionConfigTargetDateRange() mode = %s, want %s", gotMode, tt.wantMode)
			}
			if tt.wantNil {
				if gotStart != nil || gotEnd != nil {
					t.Fatalf("validateAutoPromotionConfigTargetDateRange() start/end = %v/%v, want nil/nil", gotStart, gotEnd)
				}
				return
			}
			if gotStart == nil || gotEnd == nil {
				t.Fatalf("validateAutoPromotionConfigTargetDateRange() start/end = %v/%v, want %s/%s", gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
			if got := gotStart.Format("2006-01-02"); got != tt.wantStart {
				t.Fatalf("validateAutoPromotionConfigTargetDateRange() start = %s, want %s", got, tt.wantStart)
			}
			if got := gotEnd.Format("2006-01-02"); got != tt.wantEnd {
				t.Fatalf("validateAutoPromotionConfigTargetDateRange() end = %s, want %s", got, tt.wantEnd)
			}
		})
	}
}

func TestToAutoPromotionConfigDTOUsesDateRange(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	config := &model.AutoPromotionConfig{
		ID:                1,
		ShopID:            9,
		Enabled:           true,
		ScheduleTime:      "09:05",
		TargetDateMode:    model.AutoPromotionTargetDateModeCustom,
		TargetDate:        &start,
		TargetDateEnd:     &end,
		OfficialActionIDs: mustAutoPromotionJSON(t, []uint{11}),
		ShopActionIDs:     mustAutoPromotionJSON(t, []uint{22}),
		UpdatedAt:         time.Date(2026, 4, 5, 10, 30, 0, 0, time.UTC),
	}

	got, err := toAutoPromotionConfigDTO(config)
	if err != nil {
		t.Fatalf("toAutoPromotionConfigDTO() error = %v", err)
	}
	if got.TargetDateStart != "2026-03-05" {
		t.Fatalf("TargetDateStart = %s, want 2026-03-05", got.TargetDateStart)
	}
	if got.TargetDateEnd != "2026-03-08" {
		t.Fatalf("TargetDateEnd = %s, want 2026-03-08", got.TargetDateEnd)
	}
	if !reflect.DeepEqual(got.OfficialActionIDs, []uint{11}) {
		t.Fatalf("OfficialActionIDs = %#v, want %#v", got.OfficialActionIDs, []uint{11})
	}
	if !reflect.DeepEqual(got.ShopActionIDs, []uint{22}) {
		t.Fatalf("ShopActionIDs = %#v, want %#v", got.ShopActionIDs, []uint{22})
	}
}

func TestToAutoPromotionRunSummaryDTOUsesDateRange(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	completedAt := time.Date(2026, 4, 5, 10, 5, 0, 0, time.UTC)
	run := &model.AutoPromotionRun{
		ID:             7,
		TriggerMode:    model.AutoPromotionTriggerModeManual,
		TriggerDate:    time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC),
		TargetDateMode: model.AutoPromotionTargetDateModeCustom,
		TargetDate:     time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
		TargetDateEnd:  time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC),
		Status:         model.AutoPromotionRunStatusSuccess,
		StartedAt:      &startedAt,
		CompletedAt:    &completedAt,
		CreatedAt:      time.Date(2026, 4, 5, 9, 55, 0, 0, time.UTC),
	}

	got := toAutoPromotionRunSummaryDTO(run)
	if got == nil {
		t.Fatal("toAutoPromotionRunSummaryDTO() = nil")
	}
	if got.TargetDateStart != "2026-03-05" {
		t.Fatalf("TargetDateStart = %s, want 2026-03-05", got.TargetDateStart)
	}
	if got.TargetDateEnd != "2026-03-08" {
		t.Fatalf("TargetDateEnd = %s, want 2026-03-08", got.TargetDateEnd)
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

func TestAutoPromotionShopCandidateWaitTimeoutAllowsSlowBrowserSync(t *testing.T) {
	t.Parallel()

	if autoPromotionShopCandidateWaitTimeout < 2*time.Minute {
		t.Fatalf("autoPromotionShopCandidateWaitTimeout = %s, want at least %s", autoPromotionShopCandidateWaitTimeout, 2*time.Minute)
	}
}
