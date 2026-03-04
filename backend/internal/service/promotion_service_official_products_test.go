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
