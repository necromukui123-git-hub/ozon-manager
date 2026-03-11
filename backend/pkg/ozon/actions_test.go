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

func TestGetActionCandidatesUsesLastIDCursor(t *testing.T) {
	t.Parallel()

	client := NewClient("100", "test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/v1/actions/candidates" {
				t.Fatalf("unexpected path: %s", req.URL.Path)
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
			switch got := payload["last_id"].(type) {
			case float64:
				if got != 1366 {
					t.Fatalf("last_id = %#v, want 1366", got)
				}
			default:
				t.Fatalf("last_id type = %T, want float64", got)
			}

			resp := `{"result":{"products":[{"id":226,"price":250,"action_price":175,"max_action_price":175,"stock":0}],"total":1,"last_id":226}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(resp)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	resp, err := client.GetActionCandidates(63337, 100, "1366")
	if err != nil {
		t.Fatalf("GetActionCandidates returned error: %v", err)
	}
	if resp.Result.LastID != "226" {
		t.Fatalf("last_id = %q, want %q", resp.Result.LastID, "226")
	}
	if len(resp.Result.Products) != 1 {
		t.Fatalf("products len = %d, want 1", len(resp.Result.Products))
	}
	if resp.Result.Products[0].MaxActionPrice != 175 {
		t.Fatalf("max_action_price = %v, want 175", resp.Result.Products[0].MaxActionPrice)
	}
}

func TestActivateProductsParsesRejectedItems(t *testing.T) {
	t.Parallel()

	client := NewClient("100", "test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			resp := `{"result":{"product_ids":[1389],"rejected":[{"product_id":1390,"reason":"price exceeds max_action_price"}]}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(resp)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	resp, err := client.ActivateProducts(60564, []ActivateProductItem{{ProductID: 1389, ActionPrice: 356}})
	if err != nil {
		t.Fatalf("ActivateProducts returned error: %v", err)
	}
	if len(resp.Result.Rejected) != 1 {
		t.Fatalf("rejected len = %d, want 1", len(resp.Result.Rejected))
	}
	if resp.Result.Rejected[0].ProductID != 1390 {
		t.Fatalf("rejected product_id = %d, want 1390", resp.Result.Rejected[0].ProductID)
	}
	if resp.Result.Rejected[0].Reason != "price exceeds max_action_price" {
		t.Fatalf("rejected reason = %q", resp.Result.Rejected[0].Reason)
	}
}
