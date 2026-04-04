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
	Status      string                    `json:"status"`
	Error       string                    `json:"error,omitempty"`
	Message     string                    `json:"message,omitempty"`
	Diagnostics *SearchCPOStepDiagnostics `json:"diagnostics,omitempty"`
}

type SearchCPOStepDiagnostics struct {
	RequestedSKU           string   `json:"requested_sku,omitempty"`
	ParserRevision         string   `json:"parser_revision,omitempty"`
	BuildRevision          string   `json:"build_revision,omitempty"`
	ResponseRootKeys       []string `json:"response_root_keys,omitempty"`
	SampleResponseKeys     []string `json:"sample_response_keys,omitempty"`
	ResponseHTTPStatus     int      `json:"response_http_status,omitempty"`
	ResponseHTTPStatusText string   `json:"response_http_status_text,omitempty"`
	ResponseContentType    string   `json:"response_content_type,omitempty"`
	ResponseParseError     string   `json:"response_parse_error,omitempty"`
	ResponseExcerpt        string   `json:"response_excerpt,omitempty"`
	ResponseLength         int      `json:"response_length,omitempty"`
	ResponseKind           string   `json:"response_kind,omitempty"`
	ScriptResultType       string   `json:"script_result_type,omitempty"`
	ResponseItemCount      int      `json:"response_item_count,omitempty"`
}

type SearchCPOAvailabilityDiagnostics struct {
	RequestedSKU            string   `json:"requested_sku,omitempty"`
	ParserRevision          string   `json:"parser_revision,omitempty"`
	BuildRevision           string   `json:"build_revision,omitempty"`
	ResponseRootKeys        []string `json:"response_root_keys,omitempty"`
	SampleResponseKeys      []string `json:"sample_response_keys,omitempty"`
	AvailabilityMapKeyCount int      `json:"availability_map_key_count,omitempty"`
	ReasonMapKeyCount       int      `json:"reason_map_key_count,omitempty"`
	ResponseHTTPStatus      int      `json:"response_http_status,omitempty"`
	ResponseHTTPStatusText  string   `json:"response_http_status_text,omitempty"`
	ResponseContentType     string   `json:"response_content_type,omitempty"`
	ResponseParseError      string   `json:"response_parse_error,omitempty"`
	ResponseExcerpt         string   `json:"response_excerpt,omitempty"`
	ResponseLength          int      `json:"response_length,omitempty"`
	ResponseKind            string   `json:"response_kind,omitempty"`
	ScriptResultType        string   `json:"script_result_type,omitempty"`
	UnavailableReason       string   `json:"unavailable_reason,omitempty"`
}

type SearchCPOAutomationRunItemResponse struct {
	ID                      uint                              `json:"id"`
	ProductCacheID          *uint                             `json:"product_cache_id,omitempty"`
	SourceSKU               string                            `json:"source_sku"`
	SKU                     string                            `json:"sku"`
	Title                   string                            `json:"title"`
	SearchPromoStatus       string                            `json:"search_promo_status"`
	CarrotsStatus           string                            `json:"carrots_status"`
	AvailabilityPromo       *bool                             `json:"availability_promo,omitempty"`
	RuleStateBefore         string                            `json:"rule_state_before"`
	RuleStateAfter          string                            `json:"rule_state_after"`
	OverallStatus           string                            `json:"overall_status"`
	InitialStatus           string                            `json:"initial_status"`
	EnableStatus            string                            `json:"enable_status"`
	ExitStatus              string                            `json:"exit_status"`
	MorkovskStatus          string                            `json:"morkovsk_status"`
	InitialResults          []SearchCPORunActionResult        `json:"initial_results"`
	ExitResults             []SearchCPORunActionResult        `json:"exit_results"`
	EnableResult            SearchCPOAutomationStepResult     `json:"enable_result"`
	MorkovskResult          SearchCPOAutomationStepResult     `json:"morkovsk_result"`
	AvailabilityCheckedAt   string                            `json:"availability_checked_at,omitempty"`
	AvailabilityDiagnostics *SearchCPOAvailabilityDiagnostics `json:"availability_diagnostics,omitempty"`
	Message                 string                            `json:"message,omitempty"`
}

type SearchCPOAutomationRunSummaryResponse struct {
	ID                 uint   `json:"id"`
	TriggerMode        string `json:"trigger_mode"`
	TriggerDate        string `json:"trigger_date"`
	Status             string `json:"status"`
	TotalFetched   int    `json:"total_fetched"`
	TotalState1    int    `json:"total_state1"`
	TotalState2    int    `json:"total_state2"`
	TotalState3    int    `json:"total_state3"`
	TotalState4    int    `json:"total_state4"`
	TotalProcessed int    `json:"total_processed"`
	SuccessItems   int    `json:"success_items"`
	FailedItems    int    `json:"failed_items"`
	SkippedItems   int    `json:"skipped_items"`
	ErrorMessage   string `json:"error_message,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	CompletedAt    string `json:"completed_at,omitempty"`
	CreatedAt      string `json:"created_at"`
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
