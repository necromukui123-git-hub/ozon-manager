package service

import (
	"testing"

	"ozon-manager/pkg/ozon"
)

func TestResolveOfficialActionProductID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		item ozon.ActionProduct
		want int64
	}{
		{
			name: "prefer product_id when available",
			item: ozon.ActionProduct{
				ID:        123,
				ProductID: 456,
			},
			want: 456,
		},
		{
			name: "fallback to id when product_id missing",
			item: ozon.ActionProduct{
				ID:        789,
				ProductID: 0,
			},
			want: 789,
		},
		{
			name: "return zero when both missing",
			item: ozon.ActionProduct{
				ID:        0,
				ProductID: 0,
			},
			want: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resolveOfficialActionProductID(tc.item)
			if got != tc.want {
				t.Fatalf("resolveOfficialActionProductID(%+v) = %d, want %d", tc.item, got, tc.want)
			}
		})
	}
}

func TestOfficialDeactivateResultError(t *testing.T) {
	t.Parallel()

	t.Run("success when product id returned in product_ids", func(t *testing.T) {
		t.Parallel()
		resp := &ozon.DeactivateProductsResponse{}
		resp.Result.ProductIDs = []int64{14975}

		if errText := officialDeactivateResultError(resp, 14975); errText != "" {
			t.Fatalf("officialDeactivateResultError() = %q, want empty", errText)
		}
	})

	t.Run("rejected reason wins for matching product id", func(t *testing.T) {
		t.Parallel()
		resp := &ozon.DeactivateProductsResponse{}
		resp.Result.Rejected = []ozon.ActivateRejectedItem{{
			ProductID: 14976,
			Reason:    "product already archived",
		}}

		if errText := officialDeactivateResultError(resp, 14976); errText != "product already archived" {
			t.Fatalf("officialDeactivateResultError() = %q, want %q", errText, "product already archived")
		}
	})

	t.Run("unknown result when response has no success and no rejection", func(t *testing.T) {
		t.Parallel()
		resp := &ozon.DeactivateProductsResponse{}

		if errText := officialDeactivateResultError(resp, 14977); errText != "官方活动返回未知结果" {
			t.Fatalf("officialDeactivateResultError() = %q, want %q", errText, "官方活动返回未知结果")
		}
	})
}
