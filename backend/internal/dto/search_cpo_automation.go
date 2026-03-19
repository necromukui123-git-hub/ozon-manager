package dto

type SearchCPOAutomationRunRequest struct {
	ShopID uint `json:"shop_id" binding:"required"`
}

type SearchCPOAutomationRunListRequest struct {
	ShopID   uint `form:"shop_id" binding:"required"`
	Page     int  `form:"page,default=1"`
	PageSize int  `form:"page_size,default=20"`
}

type SearchCPOAutomationStepResult struct {
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

type SearchCPOAutomationRunItemResponse struct {
	ID                uint                          `json:"id"`
	ProductCacheID    *uint                         `json:"product_cache_id,omitempty"`
	SourceSKU         string                        `json:"source_sku"`
	SKU               string                        `json:"sku"`
	Title             string                        `json:"title"`
	SearchPromoStatus string                        `json:"search_promo_status"`
	CarrotsStatus     string                        `json:"carrots_status"`
	AvailabilityPromo *bool                         `json:"availability_promo,omitempty"`
	RuleStateBefore   string                        `json:"rule_state_before"`
	RuleStateAfter    string                        `json:"rule_state_after"`
	OverallStatus     string                        `json:"overall_status"`
	InitialStatus     string                        `json:"initial_status"`
	EnableStatus      string                        `json:"enable_status"`
	ExitStatus        string                        `json:"exit_status"`
	MorkovskStatus    string                        `json:"morkovsk_status"`
	InitialResults    []SearchCPORunActionResult    `json:"initial_results"`
	ExitResults       []SearchCPORunActionResult    `json:"exit_results"`
	EnableResult      SearchCPOAutomationStepResult `json:"enable_result"`
	MorkovskResult    SearchCPOAutomationStepResult `json:"morkovsk_result"`
	Message           string                        `json:"message,omitempty"`
}

type SearchCPOAutomationRunSummaryResponse struct {
	ID                 uint   `json:"id"`
	TriggerMode        string `json:"trigger_mode"`
	TriggerDate        string `json:"trigger_date"`
	Status             string `json:"status"`
	TotalFetched       int    `json:"total_fetched"`
	TotalState1        int    `json:"total_state1"`
	TotalState2        int    `json:"total_state2"`
	TotalState3Trigger int    `json:"total_state3_trigger"`
	TotalProcessed     int    `json:"total_processed"`
	SuccessItems       int    `json:"success_items"`
	FailedItems        int    `json:"failed_items"`
	SkippedItems       int    `json:"skipped_items"`
	ErrorMessage       string `json:"error_message,omitempty"`
	StartedAt          string `json:"started_at,omitempty"`
	CompletedAt        string `json:"completed_at,omitempty"`
	CreatedAt          string `json:"created_at"`
}

type SearchCPOAutomationRunListResponse struct {
	Total int64                                   `json:"total"`
	Items []SearchCPOAutomationRunSummaryResponse `json:"items"`
}

type SearchCPOAutomationRunDetailResponse struct {
	SearchCPOAutomationRunSummaryResponse
	ShopID            uint                                 `json:"shop_id"`
	ConfigID          *uint                                `json:"config_id,omitempty"`
	TriggeredBy       *uint                                `json:"triggered_by,omitempty"`
	ScheduleTime      string                               `json:"schedule_time,omitempty"`
	EnableStep        bool                                 `json:"enable_step"`
	OfficialActionIDs []uint                               `json:"official_action_ids"`
	ShopActionIDs     []uint                               `json:"shop_action_ids"`
	Items             []SearchCPOAutomationRunItemResponse `json:"items"`
}
