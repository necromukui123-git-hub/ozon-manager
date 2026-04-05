package ozon

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGetProductListV3RetriesTransientTransportTimeout(t *testing.T) {
	t.Parallel()

	attempts := 0
	client := NewClient("100", "test-key")
	client.httpClient = &http.Client{
		Timeout: 200 * time.Millisecond,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return nil, &timeoutError{message: "net/http: TLS handshake timeout"}
			}
			resp := `{"result":{"items":[{"product_id":12345,"offer_id":"A-1"}],"last_id":"","total":1}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(resp)),
			}, nil
		}),
	}

	resp, err := client.GetProductListV3(50, "", "ALL")
	if err != nil {
		t.Fatalf("GetProductListV3() error = %v", err)
	}
	if resp == nil || len(resp.Result.Items) != 1 {
		t.Fatalf("GetProductListV3() items len = %d, want 1", len(resp.Result.Items))
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

type timeoutError struct {
	message string
}

func (e *timeoutError) Error() string {
	return e.message
}

func (e *timeoutError) Timeout() bool {
	return true
}

func (e *timeoutError) Temporary() bool {
	return true
}
