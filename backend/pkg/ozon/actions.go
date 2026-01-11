package ozon

import (
	"encoding/json"
	"fmt"
)

// Action 促销活动
type Action struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	ActionType      string `json:"action_type"`
	Description     string `json:"description"`
	DateStart       string `json:"date_start"`
	DateEnd         string `json:"date_end"`
	FreezeDate      string `json:"freeze_date"`
	PotentialProducts int  `json:"potential_products_count"`
	ParticipatingProducts int `json:"participating_products_count"`
	IsParticipating bool   `json:"is_participating"`
	BannedProducts  int    `json:"banned_products_count"`
	WithTargeting   bool   `json:"with_targeting"`
}

// ActionsResponse 促销活动列表响应
type ActionsResponse struct {
	Result []Action `json:"result"`
}

// GetActions 获取所有促销活动
func (c *Client) GetActions() (*ActionsResponse, error) {
	respBody, err := c.doRequest("GET", "/v1/actions", nil)
	if err != nil {
		return nil, err
	}

	var resp ActionsResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// ActionCandidatesRequest 可参与促销的商品请求
type ActionCandidatesRequest struct {
	ActionID int64  `json:"action_id"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
}

// ActionCandidatesResponse 可参与促销的商品响应
type ActionCandidatesResponse struct {
	Result struct {
		Products []ActionProduct `json:"products"`
		Total    int             `json:"total"`
	} `json:"result"`
}

type ActionProduct struct {
	ID               int64   `json:"id"`
	ProductID        int64   `json:"product_id"`
	Price            float64 `json:"price"`
	ActionPrice      float64 `json:"action_price"`
	MaxActionPrice   float64 `json:"max_action_price"`
	AddMode          string  `json:"add_mode"`
	MinStock         int     `json:"min_stock"`
	Stock            int     `json:"stock"`
}

// GetActionCandidates 获取可参与促销的商品
func (c *Client) GetActionCandidates(actionID int64, limit, offset int) (*ActionCandidatesResponse, error) {
	req := ActionCandidatesRequest{
		ActionID: actionID,
		Limit:    limit,
		Offset:   offset,
	}

	respBody, err := c.doRequest("POST", "/v1/actions/candidates", req)
	if err != nil {
		return nil, err
	}

	var resp ActionCandidatesResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// ActionProductsRequest 已参与促销的商品请求
type ActionProductsRequest struct {
	ActionID int64 `json:"action_id"`
	Limit    int   `json:"limit"`
	Offset   int   `json:"offset"`
}

// ActionProductsResponse 已参与促销的商品响应
type ActionProductsResponse struct {
	Result struct {
		Products []ActionProduct `json:"products"`
		Total    int             `json:"total"`
	} `json:"result"`
}

// GetActionProducts 获取已参与促销的商品
func (c *Client) GetActionProducts(actionID int64, limit, offset int) (*ActionProductsResponse, error) {
	req := ActionProductsRequest{
		ActionID: actionID,
		Limit:    limit,
		Offset:   offset,
	}

	respBody, err := c.doRequest("POST", "/v1/actions/products", req)
	if err != nil {
		return nil, err
	}

	var resp ActionProductsResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// ActivateProductsRequest 添加商品到促销活动
type ActivateProductsRequest struct {
	ActionID int64                    `json:"action_id"`
	Products []ActivateProductItem    `json:"products"`
}

type ActivateProductItem struct {
	ProductID   int64   `json:"product_id"`
	ActionPrice float64 `json:"action_price"`
	Stock       int     `json:"stock,omitempty"`
}

// ActivateProductsResponse 添加商品到促销响应
type ActivateProductsResponse struct {
	Result struct {
		ProductIDs []int64 `json:"product_ids"`
	} `json:"result"`
}

// ActivateProducts 添加商品到促销活动
func (c *Client) ActivateProducts(actionID int64, products []ActivateProductItem) (*ActivateProductsResponse, error) {
	req := ActivateProductsRequest{
		ActionID: actionID,
		Products: products,
	}

	respBody, err := c.doRequest("POST", "/v1/actions/products/activate", req)
	if err != nil {
		return nil, err
	}

	var resp ActivateProductsResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// DeactivateProductsRequest 从促销活动移除商品
type DeactivateProductsRequest struct {
	ActionID   int64   `json:"action_id"`
	ProductIDs []int64 `json:"product_ids"`
}

// DeactivateProductsResponse 移除商品响应
type DeactivateProductsResponse struct {
	Result struct {
		ProductIDs []int64 `json:"product_ids"`
	} `json:"result"`
}

// DeactivateProducts 从促销活动移除商品
func (c *Client) DeactivateProducts(actionID int64, productIDs []int64) (*DeactivateProductsResponse, error) {
	req := DeactivateProductsRequest{
		ActionID:   actionID,
		ProductIDs: productIDs,
	}

	respBody, err := c.doRequest("POST", "/v1/actions/products/deactivate", req)
	if err != nil {
		return nil, err
	}

	var resp DeactivateProductsResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}
