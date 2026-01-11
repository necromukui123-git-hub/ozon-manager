package ozon

import (
	"encoding/json"
	"fmt"
)

// UpdatePriceRequest 更新价格请求
type UpdatePriceRequest struct {
	Prices []PriceItem `json:"prices"`
}

type PriceItem struct {
	ProductID    int64  `json:"product_id"`
	OfferID      string `json:"offer_id,omitempty"`
	Price        string `json:"price"`
	OldPrice     string `json:"old_price,omitempty"`
	MinPrice     string `json:"min_price,omitempty"`
	AutoActionEnabled bool `json:"auto_action_enabled,omitempty"`
}

// UpdatePriceResponse 更新价格响应
type UpdatePriceResponse struct {
	Result []PriceUpdateResult `json:"result"`
}

type PriceUpdateResult struct {
	ProductID int64  `json:"product_id"`
	OfferID   string `json:"offer_id"`
	Updated   bool   `json:"updated"`
	Errors    []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// UpdatePrices 更新商品价格
func (c *Client) UpdatePrices(prices []PriceItem) (*UpdatePriceResponse, error) {
	req := UpdatePriceRequest{
		Prices: prices,
	}

	respBody, err := c.doRequest("POST", "/v1/product/import/prices", req)
	if err != nil {
		return nil, err
	}

	var resp UpdatePriceResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// GetProductPricesRequest 获取商品价格请求
type GetProductPricesRequest struct {
	Filter struct {
		OfferID   []string `json:"offer_id,omitempty"`
		ProductID []int64  `json:"product_id,omitempty"`
		Visibility string  `json:"visibility,omitempty"`
	} `json:"filter"`
	LastID string `json:"last_id,omitempty"`
	Limit  int    `json:"limit"`
}

// GetProductPricesResponse 获取商品价格响应
type GetProductPricesResponse struct {
	Result struct {
		Items  []ProductPriceInfo `json:"items"`
		LastID string             `json:"last_id"`
		Total  int                `json:"total"`
	} `json:"result"`
}

type ProductPriceInfo struct {
	ProductID int64 `json:"product_id"`
	OfferID   string `json:"offer_id"`
	Price     struct {
		Price          string `json:"price"`
		OldPrice       string `json:"old_price"`
		MinPrice       string `json:"min_price"`
		MarketingPrice string `json:"marketing_price"`
	} `json:"price"`
	Commissions []struct {
		Percent float64 `json:"percent"`
		Value   float64 `json:"value"`
	} `json:"commissions"`
}

// GetProductPrices 获取商品价格信息
func (c *Client) GetProductPrices(productIDs []int64, limit int, lastID string) (*GetProductPricesResponse, error) {
	req := GetProductPricesRequest{
		Limit:  limit,
		LastID: lastID,
	}
	req.Filter.ProductID = productIDs
	req.Filter.Visibility = "ALL"

	respBody, err := c.doRequest("POST", "/v4/product/info/prices", req)
	if err != nil {
		return nil, err
	}

	var resp GetProductPricesResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// UpdateSinglePrice 更新单个商品价格的便捷方法
func (c *Client) UpdateSinglePrice(productID int64, price, oldPrice, minPrice string) error {
	prices := []PriceItem{
		{
			ProductID: productID,
			Price:     price,
			OldPrice:  oldPrice,
			MinPrice:  minPrice,
		},
	}

	resp, err := c.UpdatePrices(prices)
	if err != nil {
		return err
	}

	if len(resp.Result) > 0 && !resp.Result[0].Updated {
		if len(resp.Result[0].Errors) > 0 {
			return fmt.Errorf("price update failed: %s", resp.Result[0].Errors[0].Message)
		}
		return fmt.Errorf("price update failed for unknown reason")
	}

	return nil
}
