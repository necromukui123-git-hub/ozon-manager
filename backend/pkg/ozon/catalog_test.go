package ozon

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetProductListV3UsesExpectedPayload(t *testing.T) {
	t.Parallel()

	client := NewClient("100", "test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", req.Method)
			}
			if req.URL.Path != "/v3/product/list" {
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}

			payload := map[string]interface{}{}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("failed to unmarshal payload: %v", err)
			}

			if _, exists := payload["current_page"]; exists {
				t.Fatalf("payload should not contain current_page: %s", string(body))
			}
			if got := int(payload["limit"].(float64)); got != 50 {
				t.Fatalf("limit=%d, want 50", got)
			}
			if got := payload["last_id"].(string); got != "cursor-abc" {
				t.Fatalf("last_id=%q, want %q", got, "cursor-abc")
			}

			filter, _ := payload["filter"].(map[string]interface{})
			if got := filter["visibility"].(string); got != "VISIBLE" {
				t.Fatalf("visibility=%q, want %q", got, "VISIBLE")
			}

			resp := `{"result":{"items":[{"product_id":12345,"offer_id":"A-1","has_fbo_stocks":false,"has_fbs_stocks":true,"archived":false,"is_discounted":true,"quants":[{"quant_code":"kg","quant_size":2}]}],"last_id":"n1","total":1}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(resp)),
			}, nil
		}),
	}

	resp, err := client.GetProductListV3(50, "cursor-abc", "VISIBLE")
	if err != nil {
		t.Fatalf("GetProductListV3 error: %v", err)
	}
	if len(resp.Result.Items) != 1 {
		t.Fatalf("items len=%d, want 1", len(resp.Result.Items))
	}
	if resp.Result.Items[0].ProductID != 12345 {
		t.Fatalf("product_id=%d, want 12345", resp.Result.Items[0].ProductID)
	}
	if resp.Result.Items[0].HasFBOStocks {
		t.Fatalf("has_fbo_stocks=%v, want false", resp.Result.Items[0].HasFBOStocks)
	}
	if !resp.Result.Items[0].HasFBSStocks {
		t.Fatalf("has_fbs_stocks=%v, want true", resp.Result.Items[0].HasFBSStocks)
	}
	if !resp.Result.Items[0].IsDiscounted {
		t.Fatalf("is_discounted=%v, want true", resp.Result.Items[0].IsDiscounted)
	}
	if len(resp.Result.Items[0].Quants) != 1 {
		t.Fatalf("quants len=%d, want 1", len(resp.Result.Items[0].Quants))
	}
	if resp.Result.Items[0].Quants[0].QuantCode != "kg" {
		t.Fatalf("quant_code=%q, want %q", resp.Result.Items[0].Quants[0].QuantCode, "kg")
	}
	if resp.Result.Items[0].Quants[0].QuantSize != 2 {
		t.Fatalf("quant_size=%d, want %d", resp.Result.Items[0].Quants[0].QuantSize, 2)
	}
}

func TestGetProductListV3SupportsVisibilityCompatibility(t *testing.T) {
	t.Parallel()

	client := NewClient("100", "test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			resp := `{"result":{"items":[{"product_id":12345,"offer_id":"A-1","visibility":"VISIBLE"}],"last_id":"n1","total":1}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(resp)),
			}, nil
		}),
	}

	resp, err := client.GetProductListV3(50, "cursor-abc", "VISIBLE")
	if err != nil {
		t.Fatalf("GetProductListV3 error: %v", err)
	}
	if len(resp.Result.Items) != 1 {
		t.Fatalf("items len=%d, want 1", len(resp.Result.Items))
	}
	if resp.Result.Items[0].Visibility != "VISIBLE" {
		t.Fatalf("visibility=%q, want %q", resp.Result.Items[0].Visibility, "VISIBLE")
	}
}

func TestGetProductInfoListUsesExpectedPathAndPayload(t *testing.T) {
	t.Parallel()

	client := NewClient("100", "test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", req.Method)
			}
			if req.URL.Path != "/v3/product/info/list" {
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}
			payload := map[string]interface{}{}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("failed to decode payload: %v", err)
			}
			productIDs, _ := payload["product_id"].([]interface{})
			if len(productIDs) != 1 {
				t.Fatalf("product_id len=%d, want 1", len(productIDs))
			}
			if got, _ := productIDs[0].(string); got != "12345" {
				t.Fatalf("product_id[0]=%q, want %q", got, "12345")
			}

			resp := `{"items":[{"product_id":12345,"offer_id":"A-1","name":"Test Product","sku":9988,"price":"100.00"}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(resp)),
			}, nil
		}),
	}

	resp, err := client.GetProductInfoList([]int64{12345}, nil)
	if err != nil {
		t.Fatalf("GetProductInfoList error: %v", err)
	}
	items := resp.ItemsList()
	if len(items) != 1 {
		t.Fatalf("items len=%d, want 1", len(items))
	}
	if items[0].Name != "Test Product" {
		t.Fatalf("name=%q, want %q", items[0].Name, "Test Product")
	}
}

func TestGetProductInfoListSupportsResultItemsFallback(t *testing.T) {
	t.Parallel()

	client := NewClient("100", "test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			resp := `{"result":{"items":[{"product_id":54321,"offer_id":"B-1","name":"Fallback Product"}]}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(resp)),
			}, nil
		}),
	}

	resp, err := client.GetProductInfoList([]int64{54321}, nil)
	if err != nil {
		t.Fatalf("GetProductInfoList error: %v", err)
	}
	items := resp.ItemsList()
	if len(items) != 1 {
		t.Fatalf("items len=%d, want 1", len(items))
	}
	if items[0].Name != "Fallback Product" {
		t.Fatalf("name=%q, want %q", items[0].Name, "Fallback Product")
	}
}

func TestGetProductStocksUsesExpectedPathAndPayload(t *testing.T) {
	t.Parallel()

	client := NewClient("100", "test-key")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", req.Method)
			}
			if req.URL.Path != "/v3/product/info/stocks" {
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}
			payload := map[string]interface{}{}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("failed to parse payload: %v", err)
			}
			if got := int(payload["limit"].(float64)); got != 200 {
				t.Fatalf("limit=%d, want 200", got)
			}

			filter, _ := payload["filter"].(map[string]interface{})
			productIDs, _ := filter["product_id"].([]interface{})
			if len(productIDs) != 2 {
				t.Fatalf("product_id len=%d, want 2", len(productIDs))
			}

			resp := `{"result":{"items":[{"product_id":12345,"offer_id":"A-1","stocks":[{"type":"fbo","present":4,"reserved":1},{"type":"fbs","present":6,"reserved":2}]}],"last_id":"","total":1}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(resp)),
			}, nil
		}),
	}

	resp, err := client.GetProductStocks([]int64{12345, 67890}, nil, 200, "")
	if err != nil {
		t.Fatalf("GetProductStocks error: %v", err)
	}
	if len(resp.Result.Items) != 1 {
		t.Fatalf("items len=%d, want 1", len(resp.Result.Items))
	}
	if len(resp.Result.Items[0].Stocks) != 2 {
		t.Fatalf("stocks len=%d, want 2", len(resp.Result.Items[0].Stocks))
	}
}
