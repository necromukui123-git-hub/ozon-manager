package service

import (
	"testing"
	"time"

	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
)

func TestDeriveSearchCPORuleState(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC)
	falseValue := false
	trueValue := true

	t.Run("state1 when disabled and availability false", func(t *testing.T) {
		t.Parallel()
		state, detectedAt := deriveSearchCPORuleState(model.SearchCPOProduct{
			SearchPromoStatus: "SEARCH_PROMO_STATUS_DISABLED",
			CarrotsStatus:     "CARROTS_STATUS_DISABLED",
			AvailabilityPromo: &falseValue,
		}, "", now)
		if state != model.SearchCPORuleStateState1 {
			t.Fatalf("deriveSearchCPORuleState() = %q, want %q", state, model.SearchCPORuleStateState1)
		}
		if detectedAt != nil {
			t.Fatalf("expected no detectedAt for state1")
		}
	})

	t.Run("state2 when disabled and availability true", func(t *testing.T) {
		t.Parallel()
		state, detectedAt := deriveSearchCPORuleState(model.SearchCPOProduct{
			SearchPromoStatus: "SEARCH_PROMO_STATUS_DISABLED",
			CarrotsStatus:     "CARROTS_STATUS_DISABLED",
			AvailabilityPromo: &trueValue,
		}, "", now)
		if state != model.SearchCPORuleStateState2 {
			t.Fatalf("deriveSearchCPORuleState() = %q, want %q", state, model.SearchCPORuleStateState2)
		}
		if detectedAt == nil || !detectedAt.Equal(now) {
			t.Fatalf("expected detectedAt to be %v, got %v", now, detectedAt)
		}
	})

	t.Run("state2 keeps existing detectedAt", func(t *testing.T) {
		t.Parallel()
		previous := now.Add(-2 * time.Hour)
		state, detectedAt := deriveSearchCPORuleState(model.SearchCPOProduct{
			SearchPromoStatus: "SEARCH_PROMO_STATUS_DISABLED",
			CarrotsStatus:     "CARROTS_STATUS_DISABLED",
			AvailabilityPromo: &trueValue,
			State2DetectedAt:  &previous,
		}, model.SearchCPORuleStateState2, now)
		if state != model.SearchCPORuleStateState2 {
			t.Fatalf("deriveSearchCPORuleState() = %q, want %q", state, model.SearchCPORuleStateState2)
		}
		if detectedAt == nil || !detectedAt.Equal(previous) {
			t.Fatalf("expected detectedAt to stay %v, got %v", previous, detectedAt)
		}
	})

	t.Run("state3 trigger when enabled after state2", func(t *testing.T) {
		t.Parallel()
		previous := now.Add(-time.Hour)
		state, detectedAt := deriveSearchCPORuleState(model.SearchCPOProduct{
			SearchPromoStatus: "SEARCH_PROMO_STATUS_ENABLED",
			State2DetectedAt:  &previous,
		}, model.SearchCPORuleStateState2, now)
		if state != model.SearchCPORuleStateState3Trigger {
			t.Fatalf("deriveSearchCPORuleState() = %q, want %q", state, model.SearchCPORuleStateState3Trigger)
		}
		if detectedAt == nil || !detectedAt.Equal(previous) {
			t.Fatalf("expected detectedAt to stay %v, got %v", previous, detectedAt)
		}
	})

	t.Run("joined wins over other states", func(t *testing.T) {
		t.Parallel()
		joinedAt := now.Add(-30 * time.Minute)
		state, _ := deriveSearchCPORuleState(model.SearchCPOProduct{
			MorkovskJoinedAt: &joinedAt,
			State2DetectedAt: &joinedAt,
		}, model.SearchCPORuleStateState3Trigger, now)
		if state != model.SearchCPORuleStateJoined {
			t.Fatalf("deriveSearchCPORuleState() = %q, want %q", state, model.SearchCPORuleStateJoined)
		}
	})
}

func TestBuildSearchCPOActionSyncFailureMessage(t *testing.T) {
	t.Parallel()

	t.Run("empty result means no failure", func(t *testing.T) {
		t.Parallel()
		if msg := buildSearchCPOActionSyncFailureMessage(&dto.SyncActionsResult{}); msg != "" {
			t.Fatalf("buildSearchCPOActionSyncFailureMessage() = %q, want empty", msg)
		}
	})

	t.Run("pending and partial errors are joined deterministically", func(t *testing.T) {
		t.Parallel()
		msg := buildSearchCPOActionSyncFailureMessage(&dto.SyncActionsResult{
			ShopSyncPending: true,
			PartialErrors: map[string]string{
				"shop":     "shop sync failed",
				"official": "official refresh failed",
			},
		})
		want := "店铺活动同步仍在后台进行; official: official refresh failed; shop: shop sync failed"
		if msg != want {
			t.Fatalf("buildSearchCPOActionSyncFailureMessage() = %q, want %q", msg, want)
		}
	})
}

func TestBuildSearchCPOSKUMeta(t *testing.T) {
	t.Parallel()

	meta := buildSearchCPOSKUMeta([]model.SearchCPOProduct{
		{SourceSKU: "offer-1", SKU: "123456"},
		{SourceSKU: "778899", SKU: ""},
		{SourceSKU: "offer-missing", SKU: ""},
	})
	if meta == nil {
		t.Fatalf("buildSearchCPOSKUMeta() returned nil")
	}

	skuMap, ok := meta["sku_map"].(map[string]string)
	if !ok {
		t.Fatalf("sku_map type = %T, want map[string]string", meta["sku_map"])
	}
	if got := skuMap["offer-1"]; got != "123456" {
		t.Fatalf("sku_map[offer-1] = %q, want %q", got, "123456")
	}
	if got := skuMap["778899"]; got != "778899" {
		t.Fatalf("sku_map[778899] = %q, want %q", got, "778899")
	}
	if _, exists := skuMap["offer-missing"]; exists {
		t.Fatalf("unexpected mapping for offer-missing")
	}
}

func TestBuildSearchCPOSKUMetaFromStates(t *testing.T) {
	t.Parallel()

	states := map[string]*searchCPOAutomationItemState{
		"offer-1": {Product: model.SearchCPOProduct{SourceSKU: "offer-1", SKU: "123456"}},
		"445566":  {Product: model.SearchCPOProduct{SourceSKU: "445566"}},
	}

	meta := buildSearchCPOSKUMetaFromStates([]string{"offer-1", "445566", "missing"}, states)
	if meta == nil {
		t.Fatalf("buildSearchCPOSKUMetaFromStates() returned nil")
	}

	skuMap, ok := meta["sku_map"].(map[string]string)
	if !ok {
		t.Fatalf("sku_map type = %T, want map[string]string", meta["sku_map"])
	}
	if got := skuMap["offer-1"]; got != "123456" {
		t.Fatalf("sku_map[offer-1] = %q, want %q", got, "123456")
	}
	if got := skuMap["445566"]; got != "445566" {
		t.Fatalf("sku_map[445566] = %q, want %q", got, "445566")
	}
	if _, exists := skuMap["missing"]; exists {
		t.Fatalf("unexpected mapping for missing state")
	}
}
