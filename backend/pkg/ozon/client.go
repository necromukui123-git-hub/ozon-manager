package ozon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	BaseURL = "https://api-seller.ozon.ru"
)

// Client Ozon API客户端
type Client struct {
	clientID   string
	apiKey     string
	httpClient *http.Client
}

// NewClient 创建Ozon API客户端
func NewClient(clientID, apiKey string) *Client {
	return &Client{
		clientID: clientID,
		apiKey:   apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest 执行HTTP请求
func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Id", c.clientID)
	req.Header.Set("Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Product 商品信息
type Product struct {
	ProductID      int64   `json:"product_id"`
	OfferID        string  `json:"offer_id"` // 这就是source_sku
	Name           string  `json:"name"`
	SKU            int64   `json:"sku"`
	Price          string  `json:"price"`
	OldPrice       string  `json:"old_price"`
	MarketingPrice string  `json:"marketing_price"`
	MinPrice       string  `json:"min_price"`
	Visible        bool    `json:"visible"`
}

// ProductListRequest 商品列表请求
type ProductListRequest struct {
	Filter     ProductFilter `json:"filter,omitempty"`
	LastID     string        `json:"last_id,omitempty"`
	Limit      int           `json:"limit"`
}

type ProductFilter struct {
	OfferID    []string `json:"offer_id,omitempty"`
	ProductID  []int64  `json:"product_id,omitempty"`
	Visibility string   `json:"visibility,omitempty"`
}

// ProductListResponse 商品列表响应
type ProductListResponse struct {
	Result struct {
		Items  []ProductListItem `json:"items"`
		LastID string            `json:"last_id"`
		Total  int               `json:"total"`
	} `json:"result"`
}

type ProductListItem struct {
	ProductID int64  `json:"product_id"`
	OfferID   string `json:"offer_id"`
}

// GetProductList 获取商品列表
func (c *Client) GetProductList(limit int, lastID string) (*ProductListResponse, error) {
	req := ProductListRequest{
		Limit:  limit,
		LastID: lastID,
		Filter: ProductFilter{
			Visibility: "ALL",
		},
	}

	respBody, err := c.doRequest("POST", "/v2/product/list", req)
	if err != nil {
		return nil, err
	}

	var resp ProductListResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// ProductInfoRequest 商品详情请求
type ProductInfoRequest struct {
	OfferID   []string `json:"offer_id,omitempty"`
	ProductID []int64  `json:"product_id,omitempty"`
	SKU       []int64  `json:"sku,omitempty"`
}

// ProductInfoResponse 商品详情响应
type ProductInfoResponse struct {
	Result struct {
		Items []ProductInfo `json:"items"`
	} `json:"result"`
}

type ProductInfo struct {
	ID             int64  `json:"id"`
	ProductID      int64  `json:"product_id"`
	OfferID        string `json:"offer_id"`
	Name           string `json:"name"`
	SKU            int64  `json:"sku"`
	MarketingPrice string `json:"marketing_price"`
	Price          string `json:"price"`
	OldPrice       string `json:"old_price"`
	MinPrice       string `json:"min_price"`
	Visible        bool   `json:"visible"`
}

// GetProductInfo 获取商品详情
func (c *Client) GetProductInfo(productIDs []int64) (*ProductInfoResponse, error) {
	req := ProductInfoRequest{
		ProductID: productIDs,
	}

	respBody, err := c.doRequest("POST", "/v3/product/info/list", req)
	if err != nil {
		return nil, err
	}

	var resp ProductInfoResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}
