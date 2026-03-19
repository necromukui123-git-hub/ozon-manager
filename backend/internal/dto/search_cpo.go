package dto

type SearchCPOConfigRequest struct {
	ShopID            uint   `json:"shop_id" binding:"required"`
	OfficialActionIDs []uint `json:"official_action_ids"`
	ShopActionIDs     []uint `json:"shop_action_ids"`
	AutoEnabled       *bool  `json:"auto_enabled"`
	ScheduleTime      string `json:"schedule_time"`
	EnableStep        *bool  `json:"enable_step"`
}

type SearchCPOConfigResponse struct {
	ID                uint   `json:"id,omitempty"`
	ShopID            uint   `json:"shop_id"`
	OfficialActionIDs []uint `json:"official_action_ids"`
	ShopActionIDs     []uint `json:"shop_action_ids"`
	AutoEnabled       bool   `json:"auto_enabled"`
	ScheduleTime      string `json:"schedule_time"`
	EnableStep        bool   `json:"enable_step"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

type SearchCPOProductsResponse struct {
	Total      int                    `json:"total"`
	LastSynced string                 `json:"last_synced,omitempty"`
	Items      []SearchCPOProductItem `json:"items"`
}

type SearchCPOProductItem struct {
	ID                    uint    `json:"id"`
	SKU                   string  `json:"sku"`
	SourceSKU             string  `json:"source_sku"`
	ImageURL              string  `json:"image_url"`
	Title                 string  `json:"title"`
	CategoryName          string  `json:"category_name"`
	Price                 float64 `json:"price"`
	IsInStock             bool    `json:"is_in_stock"`
	SearchPromoStatus     string  `json:"search_promo_status"`
	CarrotsStatus         string  `json:"carrots_status"`
	AvailabilityPromo     *bool   `json:"availability_promo,omitempty"`
	RuleState             string  `json:"rule_state"`
	IsFavorite            bool    `json:"is_favorite"`
	Orders                int64   `json:"orders"`
	Spent                 float64 `json:"spent"`
	Clicks                int64   `json:"clicks"`
	CTRPercent            float64 `json:"ctr_percent"`
	StockTotal            int64   `json:"stock_total"`
	AvailabilityCheckedAt string  `json:"availability_checked_at,omitempty"`
	State2DetectedAt      string  `json:"state2_detected_at,omitempty"`
	MorkovskJoinedAt      string  `json:"morkovsk_joined_at,omitempty"`
	LastSyncedAt          string  `json:"last_synced_at,omitempty"`
}

type SearchCPORefreshRequest struct {
	ShopID uint `json:"shop_id" binding:"required"`
}

type SearchCPORefreshResponse struct {
	Total      int    `json:"total"`
	LastSynced string `json:"last_synced,omitempty"`
}

type SearchCPORunRequest struct {
	ShopID            uint     `json:"shop_id" binding:"required"`
	SourceSKUs        []string `json:"source_skus" binding:"required,min=1"`
	OfficialActionIDs []uint   `json:"official_action_ids"`
	ShopActionIDs     []uint   `json:"shop_action_ids"`
}

type SearchCPORunActionResult struct {
	PromotionActionID uint    `json:"promotion_action_id"`
	ActionID          int64   `json:"action_id,omitempty"`
	SourceActionID    string  `json:"source_action_id,omitempty"`
	Title             string  `json:"title"`
	Source            string  `json:"source"`
	Status            string  `json:"status"`
	Error             string  `json:"error,omitempty"`
	ActionPrice       float64 `json:"action_price,omitempty"`
}

type SearchCPORunItemResponse struct {
	ID                uint                       `json:"id"`
	SourceSKU         string                     `json:"source_sku"`
	SKU               string                     `json:"sku"`
	Title             string                     `json:"title"`
	SearchPromoStatus string                     `json:"search_promo_status"`
	OverallStatus     string                     `json:"overall_status"`
	OfficialStatus    string                     `json:"official_status"`
	ShopStatus        string                     `json:"shop_status"`
	OfficialResults   []SearchCPORunActionResult `json:"official_results"`
	ShopResults       []SearchCPORunActionResult `json:"shop_results"`
}

type SearchCPORunSummaryResponse struct {
	ID             uint   `json:"id"`
	Status         string `json:"status"`
	TotalFetched   int    `json:"total_fetched"`
	TotalSelected  int    `json:"total_selected"`
	TotalProcessed int    `json:"total_processed"`
	SuccessItems   int    `json:"success_items"`
	FailedItems    int    `json:"failed_items"`
	SkippedItems   int    `json:"skipped_items"`
	ErrorMessage   string `json:"error_message,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	CompletedAt    string `json:"completed_at,omitempty"`
	CreatedAt      string `json:"created_at"`
}

type SearchCPORunListRequest struct {
	ShopID   uint `form:"shop_id" binding:"required"`
	Page     int  `form:"page,default=1"`
	PageSize int  `form:"page_size,default=20"`
}

type SearchCPORunListResponse struct {
	Total int64                         `json:"total"`
	Items []SearchCPORunSummaryResponse `json:"items"`
}

type SearchCPORunDetailResponse struct {
	SearchCPORunSummaryResponse
	ShopID            uint                       `json:"shop_id"`
	TriggeredBy       *uint                      `json:"triggered_by,omitempty"`
	OfficialActionIDs []uint                     `json:"official_action_ids"`
	ShopActionIDs     []uint                     `json:"shop_action_ids"`
	SourceSKUs        []string                   `json:"source_skus"`
	Items             []SearchCPORunItemResponse `json:"items"`
}
