package ozon

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGetActionProductsUsesLastIDAndLanguageHeader(t *testing.T) {
	t.Parallel()

	client := NewClient(" 3676662 ", "test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", req.Method)
			}
			if req.URL.Path != "/v1/actions/products" {
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}
			if got := req.Header.Get("Language"); got != DefaultAPILanguage {
				t.Fatalf("Language header = %q, want %q", got, DefaultAPILanguage)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}

			payload := map[string]any{}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}

			if _, exists := payload["offset"]; exists {
				t.Fatalf("request body should not contain deprecated offset: %s", string(body))
			}
			if got, ok := payload["last_id"].(string); !ok || got != "cursor-1" {
				t.Fatalf("last_id = %#v, want %q", payload["last_id"], "cursor-1")
			}

			resp := `{"result":{"products":[{"id":28745,"price":99,"action_price":50,"stock":20}],"total":1,"last_id":"bnVsbA=="}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(resp)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	resp, err := client.GetActionProducts(12345, 100, "cursor-1")
	if err != nil {
		t.Fatalf("GetActionProducts returned error: %v", err)
	}
	if resp.Result.LastID != "bnVsbA==" {
		t.Fatalf("last_id = %q, want %q", resp.Result.LastID, "bnVsbA==")
	}
	if len(resp.Result.Products) != 1 {
		t.Fatalf("products len = %d, want 1", len(resp.Result.Products))
	}
	if resp.Result.Products[0].ID != 28745 {
		t.Fatalf("product id = %d, want 28745", resp.Result.Products[0].ID)
	}
}

func TestGetActionProductsParsesProductIDField(t *testing.T) {
	t.Parallel()

	client := NewClient("100", "test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			resp := `{"result":{"products":[{"product_id":9001,"price":120,"action_price":99}],"total":1,"last_id":""}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(resp)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	resp, err := client.GetActionProducts(1, 10, "")
	if err != nil {
		t.Fatalf("GetActionProducts returned error: %v", err)
	}
	if len(resp.Result.Products) != 1 {
		t.Fatalf("products len = %d, want 1", len(resp.Result.Products))
	}
	if resp.Result.Products[0].ProductID != 9001 {
		t.Fatalf("product_id = %d, want 9001", resp.Result.Products[0].ProductID)
	}
}
