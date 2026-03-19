package service

import (
	"testing"
	"time"

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
