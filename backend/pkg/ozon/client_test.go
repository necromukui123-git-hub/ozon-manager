package ozon

import (
	"errors"
	"testing"
)

func TestNormalizeClientID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      string
		shouldErr bool
	}{
		{
			name:      "trim spaces and keep number",
			input:     " 12345 ",
			want:      "12345",
			shouldErr: false,
		},
		{
			name:      "reject empty",
			input:     "   ",
			shouldErr: true,
		},
		{
			name:      "reject zero",
			input:     "0",
			shouldErr: true,
		},
		{
			name:      "reject non numeric",
			input:     "abc123",
			shouldErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeClientID(tc.input)
			if tc.shouldErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidClientID) {
					t.Fatalf("expected ErrInvalidClientID, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("normalizeClientID(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
