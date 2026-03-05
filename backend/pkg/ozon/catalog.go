package ozon

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ProductListV3Request v3 商品列表请求
type ProductListV3Request struct {
	Filter ProductFilter `json:"filter"`
	LastID string        `json:"last_id,omitempty"`
	Limit  int           `json:"limit"`
}

// ProductListV3Response v3 商品列表响应
type ProductListV3Response struct {
	Result struct {
		Items  []ProductListV3Item `json:"items"`
		LastID string              `json:"last_id"`
		Total  int                 `json:"total"`
	} `json:"result"`
}

type ProductListV3Item struct {
	ProductID    int64                  `json:"product_id"`
	OfferID      string                 `json:"offer_id"`
	HasFBOStocks bool                   `json:"has_fbo_stocks"`
	HasFBSStocks bool                   `json:"has_fbs_stocks"`
	Archived     bool                   `json:"archived"`
	IsDiscounted bool                   `json:"is_discounted"`
	Quants       []ProductListV3Quant   `json:"quants"`
	Visibility   string                 `json:"visibility"` // compatibility with older response variants
	Raw          map[string]interface{} `json:"-"`
}

type ProductListV3Quant struct {
	QuantCode string `json:"quant_code"`
	QuantSize int64  `json:"quant_size"`
}

func (p *ProductListV3Item) UnmarshalJSON(data []byte) error {
	type alias ProductListV3Item
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*p = ProductListV3Item(v)

	raw := map[string]interface{}{}
	if err := json.Unmarshal(data, &raw); err == nil {
		p.Raw = raw
	}
	return nil
}

// GetProductListV3 获取 v3 商品列表
func (c *Client) GetProductListV3(limit int, lastID string, visibility string) (*ProductListV3Response, error) {
	if limit <= 0 {
		limit = 1000
	}
	if visibility == "" {
		visibility = "ALL"
	}

	req := ProductListV3Request{
		Limit:  limit,
		LastID: lastID,
		Filter: ProductFilter{
			Visibility: visibility,
		},
	}

	respBody, err := c.doRequest("POST", "/v3/product/list", req)
	if err != nil {
		return nil, err
	}

	var resp ProductListV3Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// ProductInfoListRequest v3 商品详情批量查询请求
type ProductInfoListRequest struct {
	OfferID   []string `json:"offer_id,omitempty"`
	ProductID []string `json:"product_id,omitempty"`
	SKU       []string `json:"sku,omitempty"`
}

// ProductInfoListResponse v3 商品详情批量查询响应
type ProductInfoListResponse struct {
	Items  []ProductInfoListItem `json:"items"`
	Result struct {
		Items []ProductInfoListItem `json:"items"`
	} `json:"result"`
}

func (r *ProductInfoListResponse) ItemsList() []ProductInfoListItem {
	if r == nil {
		return nil
	}
	if len(r.Items) > 0 {
		return r.Items
	}
	return r.Result.Items
}

type ProductInfoListItem struct {
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
	Status         struct {
		State string `json:"state"`
	} `json:"status"`
	PrimaryImage string                 `json:"-"`
	Images       []string               `json:"images"`
	CurrencyCode string                 `json:"currency_code"`
	CreatedAt    string                 `json:"created_at"`
	Created      string                 `json:"created"`
	Raw          map[string]interface{} `json:"-"`
}

func (p *ProductInfoListItem) UnmarshalJSON(data []byte) error {
	type alias ProductInfoListItem
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*p = ProductInfoListItem(v)

	var extra struct {
		PrimaryImage json.RawMessage `json:"primary_image"`
		Statuses     struct {
			Status string `json:"status"`
		} `json:"statuses"`
	}
	if err := json.Unmarshal(data, &extra); err == nil {
		p.PrimaryImage = parsePrimaryImage(extra.PrimaryImage)
		if strings.TrimSpace(p.Status.State) == "" {
			p.Status.State = strings.TrimSpace(extra.Statuses.Status)
		}
	}

	raw := map[string]interface{}{}
	if err := json.Unmarshal(data, &raw); err == nil {
		p.Raw = raw
	}
	return nil
}

func parsePrimaryImage(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var image string
	if err := json.Unmarshal(raw, &image); err == nil {
		return strings.TrimSpace(image)
	}

	var images []string
	if err := json.Unmarshal(raw, &images); err == nil {
		for _, item := range images {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				return trimmed
			}
		}
		return ""
	}

	var imageObject map[string]interface{}
	if err := json.Unmarshal(raw, &imageObject); err == nil {
		return parsePrimaryImageObject(imageObject)
	}

	var values []interface{}
	if err := json.Unmarshal(raw, &values); err == nil {
		for _, item := range values {
			if parsed := parsePrimaryImageValue(item); parsed != "" {
				return parsed
			}
		}
	}

	return ""
}

func parsePrimaryImageValue(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]interface{}:
		return parsePrimaryImageObject(typed)
	default:
		return ""
	}
}

func parsePrimaryImageObject(obj map[string]interface{}) string {
	keys := []string{"url", "src", "link", "image", "path"}
	for _, key := range keys {
		value, exists := obj[key]
		if !exists {
			continue
		}
		if parsed := parsePrimaryImageValue(value); parsed != "" {
			return parsed
		}
	}
	return ""
}

// GetProductInfoList 获取 v3 商品详情
func (c *Client) GetProductInfoList(productIDs []int64, offerIDs []string) (*ProductInfoListResponse, error) {
	productIDStrings := make([]string, 0, len(productIDs))
	for _, id := range productIDs {
		if id <= 0 {
			continue
		}
		productIDStrings = append(productIDStrings, fmt.Sprintf("%d", id))
	}

	req := ProductInfoListRequest{
		ProductID: productIDStrings,
		OfferID:   offerIDs,
	}

	respBody, err := c.doRequest("POST", "/v3/product/info/list", req)
	if err != nil {
		return nil, err
	}

	var resp ProductInfoListResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// ProductStocksRequest 商品库存请求
type ProductStocksRequest struct {
	Filter ProductFilter `json:"filter"`
	LastID string        `json:"last_id,omitempty"`
	Limit  int           `json:"limit"`
}

// ProductStocksResponse 商品库存响应
type ProductStocksResponse struct {
	Result struct {
		Items  []ProductStocksItem `json:"items"`
		LastID string              `json:"last_id"`
		Total  int                 `json:"total"`
	} `json:"result"`
}

type ProductStocksItem struct {
	ProductID int64                  `json:"product_id"`
	OfferID   string                 `json:"offer_id"`
	Stocks    []ProductStockDetail   `json:"stocks"`
	Raw       map[string]interface{} `json:"-"`
}

type ProductStockDetail struct {
	Type     string `json:"type"`
	Present  int    `json:"present"`
	Reserved int    `json:"reserved"`
}

func (p *ProductStocksItem) UnmarshalJSON(data []byte) error {
	type alias ProductStocksItem
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*p = ProductStocksItem(v)

	raw := map[string]interface{}{}
	if err := json.Unmarshal(data, &raw); err == nil {
		p.Raw = raw
	}
	return nil
}

// GetProductStocks 获取商品库存信息
func (c *Client) GetProductStocks(productIDs []int64, offerIDs []string, limit int, lastID string) (*ProductStocksResponse, error) {
	if limit <= 0 {
		limit = 1000
	}
	req := ProductStocksRequest{
		Filter: ProductFilter{
			ProductID: productIDs,
			OfferID:   offerIDs,
		},
		LastID: lastID,
		Limit:  limit,
	}

	// /v3/product/info/stocks has been deprecated by Ozon and replaced by /v4/product/info/stocks.
	// Try v4 first and keep v3 as compatibility fallback for older environments.
	respBody, err := c.doRequest("POST", "/v4/product/info/stocks", req)
	if err != nil && isNotFoundAPIError(err) {
		respBody, err = c.doRequest("POST", "/v3/product/info/stocks", req)
	}
	if err != nil {
		return nil, err
	}

	var resp ProductStocksResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

func isNotFoundAPIError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "API error (status 404)")
}
