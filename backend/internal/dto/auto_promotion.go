package dto

type AutoPromotionConfigRequest struct {
	ShopID            uint   `json:"shop_id" binding:"required"`
	Enabled           bool   `json:"enabled"`
	ScheduleTime      string `json:"schedule_time"`
	TargetDate        string `json:"target_date" binding:"required"`
	OfficialActionIDs []uint `json:"official_action_ids"`
	ShopActionIDs     []uint `json:"shop_action_ids"`
}

type AutoPromotionConfigResponse struct {
	ID                uint   `json:"id,omitempty"`
	ShopID            uint   `json:"shop_id"`
	Enabled           bool   `json:"enabled"`
	ScheduleTime      string `json:"schedule_time"`
	TargetDate        string `json:"target_date"`
	OfficialActionIDs []uint `json:"official_action_ids"`
	ShopActionIDs     []uint `json:"shop_action_ids"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

type AutoPromotionRunRequest struct {
	ShopID            uint   `json:"shop_id" binding:"required"`
	TargetDate        string `json:"target_date" binding:"required"`
	OfficialActionIDs []uint `json:"official_action_ids"`
	ShopActionIDs     []uint `json:"shop_action_ids"`
}

type AutoPromotionRunListRequest struct {
	ShopID   uint `form:"shop_id" binding:"required"`
	Page     int  `form:"page,default=1"`
	PageSize int  `form:"page_size,default=20"`
}

type AutoPromotionActionResult struct {
	PromotionActionID uint    `json:"promotion_action_id"`
	ActionID          int64   `json:"action_id,omitempty"`
	SourceActionID    string  `json:"source_action_id,omitempty"`
	Title             string  `json:"title"`
	Source            string  `json:"source"`
	Status            string  `json:"status"`
	Error             string  `json:"error,omitempty"`
	ActionPrice       float64 `json:"action_price,omitempty"`
	MaxActionPrice    float64 `json:"max_action_price,omitempty"`
}

type AutoPromotionRunItemResponse struct {
	ID              uint                      `json:"id"`
	ProductID       *uint                     `json:"product_id,omitempty"`
	OzonProductID   int64                     `json:"ozon_product_id"`
	SourceSKU       string                    `json:"source_sku"`
	ProductName     string                    `json:"product_name"`
	ListingDate     string                    `json:"listing_date"`
	OverallStatus   string                    `json:"overall_status"`
	OfficialStatus  string                    `json:"official_status"`
	ShopStatus      string                    `json:"shop_status"`
	OfficialResults []AutoPromotionActionResult `json:"official_results"`
	ShopResults     []AutoPromotionActionResult `json:"shop_results"`
}

type AutoPromotionRunSummaryResponse struct {
	ID              uint   `json:"id"`
	TriggerMode     string `json:"trigger_mode"`
	TriggerDate     string `json:"trigger_date"`
	TargetDate      string `json:"target_date"`
	Status          string `json:"status"`
	TotalCandidates int    `json:"total_candidates"`
	TotalSelected   int    `json:"total_selected"`
	TotalProcessed  int    `json:"total_processed"`
	SuccessItems    int    `json:"success_items"`
	FailedItems     int    `json:"failed_items"`
	SkippedItems    int    `json:"skipped_items"`
	ErrorMessage    string `json:"error_message,omitempty"`
	StartedAt       string `json:"started_at,omitempty"`
	CompletedAt     string `json:"completed_at,omitempty"`
	CreatedAt       string `json:"created_at"`
}

type AutoPromotionRunListResponse struct {
	Total int64                             `json:"total"`
	Items []AutoPromotionRunSummaryResponse `json:"items"`
}

type AutoPromotionRunDetailResponse struct {
	AutoPromotionRunSummaryResponse
	ShopID            uint                      `json:"shop_id"`
	ConfigID          *uint                     `json:"config_id,omitempty"`
	TriggeredBy       *uint                     `json:"triggered_by,omitempty"`
	ScheduleTime      string                    `json:"schedule_time,omitempty"`
	OfficialActionIDs []uint                    `json:"official_action_ids"`
	ShopActionIDs     []uint                    `json:"shop_action_ids"`
	Items             []AutoPromotionRunItemResponse `json:"items"`
}
