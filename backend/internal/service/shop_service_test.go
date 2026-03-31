package service

import "testing"

func TestNormalizeShopClientID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      string
		shouldErr bool
	}{
		{
			name:      "normalize with spaces",
			input:     " 778899 ",
			want:      "778899",
			shouldErr: false,
		},
		{
			name:      "empty string is invalid",
			input:     "",
			shouldErr: true,
		},
		{
			name:      "zero is invalid",
			input:     "0",
			shouldErr: true,
		},
		{
			name:      "nonnumeric is invalid",
			input:     "client-1",
			shouldErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeShopClientID(tc.input)
			if tc.shouldErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("normalizeShopClientID(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeShopAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      string
		shouldErr bool
	}{
		{
			name:      "trim surrounding spaces and newlines",
			input:     "  test-api-key-123  \r\n",
			want:      "test-api-key-123",
			shouldErr: false,
		},
		{
			name:      "empty string is invalid",
			input:     "",
			shouldErr: true,
		},
		{
			name:      "whitespace only is invalid",
			input:     " \t\r\n ",
			shouldErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeShopAPIKey(tc.input)
			if tc.shouldErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("normalizeShopAPIKey(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
