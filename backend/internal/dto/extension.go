package dto

type ExtensionRegisterRequest struct {
	ShopID      uint   `json:"shop_id" binding:"required"`
	ExtensionID string `json:"extension_id" binding:"required,max=120"`
	Name        string `json:"name" binding:"max=120"`
	Version     string `json:"version" binding:"max=60"`
}

type ExtensionRegisterResponse struct {
	AgentKey       string `json:"agent_key"`
	PollIntervalMS int    `json:"poll_interval_ms"`
}

type ExtensionPollRequest struct {
	ShopID      uint   `json:"shop_id" binding:"required"`
	ExtensionID string `json:"extension_id" binding:"required,max=120"`
}

type ExtensionPollResponse struct {
	Job *AgentJobPayload `json:"job,omitempty"`
}

type ExtensionReportRequest struct {
	ShopID      uint                   `json:"shop_id" binding:"required"`
	ExtensionID string                 `json:"extension_id" binding:"required,max=120"`
	JobID       uint                   `json:"job_id" binding:"required"`
	Status      string                 `json:"status" binding:"required,oneof=success partial_success failed"`
	Results     []AgentItemResult      `json:"results" binding:"required,min=1,dive"`
	Meta        map[string]interface{} `json:"meta"`
}

type ExtensionRepriceRequest struct {
	ShopID    uint    `json:"shop_id" binding:"required"`
	SourceSKU string  `json:"source_sku" binding:"required"`
	NewPrice  float64 `json:"new_price" binding:"required,gt=0"`
}

type ExtensionRepriceResponse struct {
	ShopID    uint    `json:"shop_id"`
	SourceSKU string  `json:"source_sku"`
	NewPrice  float64 `json:"new_price"`
}
