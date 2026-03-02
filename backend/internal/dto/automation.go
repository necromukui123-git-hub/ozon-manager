package dto

type CreateAutomationJobRequest struct {
	ShopID               uint                      `json:"shop_id" binding:"required"`
	JobType              string                    `json:"job_type" binding:"required,oneof=remove_reprice_readd"`
	DryRun               bool                      `json:"dry_run"`
	RequiresConfirmation bool                      `json:"requires_confirmation"`
	RateLimit            int                       `json:"rate_limit" binding:"omitempty,min=1,max=600"`
	Items                []AutomationJobCreateItem `json:"items" binding:"required,min=1,dive"`
}

type AutomationJobCreateItem struct {
	SourceSKU   string  `json:"source_sku" binding:"required"`
	TargetPrice float64 `json:"target_price" binding:"required,gt=0"`
}

type AutomationJobListRequest struct {
	ShopID   uint   `form:"shop_id" binding:"required"`
	Status   string `form:"status"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}

type AutomationJobListResponse struct {
	Total int64                  `json:"total"`
	Items []AutomationJobSummary `json:"items"`
}

type AutomationJobSummary struct {
	ID           uint    `json:"id"`
	ShopID       uint    `json:"shop_id"`
	CreatedBy    uint    `json:"created_by"`
	JobType      string  `json:"job_type"`
	Status       string  `json:"status"`
	DryRun       bool    `json:"dry_run"`
	TotalItems   int     `json:"total_items"`
	SuccessItems int     `json:"success_items"`
	FailedItems  int     `json:"failed_items"`
	CreatedAt    string  `json:"created_at"`
	CompletedAt  *string `json:"completed_at,omitempty"`
}

type AutomationJobItemDetail struct {
	ID                uint    `json:"id"`
	ProductID         *uint   `json:"product_id,omitempty"`
	SourceSKU         string  `json:"source_sku"`
	TargetPrice       float64 `json:"target_price"`
	OverallStatus     string  `json:"overall_status"`
	StepExitStatus    string  `json:"step_exit_status"`
	StepRepriceStatus string  `json:"step_reprice_status"`
	StepReaddStatus   string  `json:"step_readd_status"`
	StepExitError     string  `json:"step_exit_error,omitempty"`
	StepRepriceError  string  `json:"step_reprice_error,omitempty"`
	StepReaddError    string  `json:"step_readd_error,omitempty"`
}

type AutomationJobDetailResponse struct {
	ID                   uint                      `json:"id"`
	ShopID               uint                      `json:"shop_id"`
	CreatedBy            uint                      `json:"created_by"`
	JobType              string                    `json:"job_type"`
	Status               string                    `json:"status"`
	DryRun               bool                      `json:"dry_run"`
	RequiresConfirmation bool                      `json:"requires_confirmation"`
	RateLimit            int                       `json:"rate_limit"`
	TotalItems           int                       `json:"total_items"`
	SuccessItems         int                       `json:"success_items"`
	FailedItems          int                       `json:"failed_items"`
	ErrorMessage         string                    `json:"error_message,omitempty"`
	CreatedAt            string                    `json:"created_at"`
	UpdatedAt            string                    `json:"updated_at"`
	StartedAt            *string                   `json:"started_at,omitempty"`
	CompletedAt          *string                   `json:"completed_at,omitempty"`
	Items                []AutomationJobItemDetail `json:"items"`
}

type AgentHeartbeatRequest struct {
	AgentKey     string                 `json:"agent_key" binding:"required"`
	Name         string                 `json:"name" binding:"required"`
	Hostname     string                 `json:"hostname"`
	Capabilities map[string]interface{} `json:"capabilities"`
}

type AgentPollRequest struct {
	AgentKey string `json:"agent_key" binding:"required"`
}

type AgentPollResponse struct {
	Job *AgentJobPayload `json:"job,omitempty"`
}

type AgentJobPayload struct {
	JobID     uint                      `json:"job_id"`
	ShopID    uint                      `json:"shop_id"`
	JobType   string                    `json:"job_type"`
	DryRun    bool                      `json:"dry_run"`
	RateLimit int                       `json:"rate_limit"`
	Items     []AutomationJobCreateItem `json:"items"`
	Meta      map[string]interface{}    `json:"meta,omitempty"`
}

type AgentReportRequest struct {
	AgentKey string                 `json:"agent_key" binding:"required"`
	JobID    uint                   `json:"job_id" binding:"required"`
	Status   string                 `json:"status" binding:"required,oneof=success partial_success failed"`
	Results  []AgentItemResult      `json:"results" binding:"required,min=1,dive"`
	Meta     map[string]interface{} `json:"meta"`
}

type AgentItemResult struct {
	SourceSKU         string `json:"source_sku" binding:"required"`
	OverallStatus     string `json:"overall_status" binding:"required,oneof=success failed skipped"`
	StepExitStatus    string `json:"step_exit_status" binding:"required,oneof=success failed skipped"`
	StepRepriceStatus string `json:"step_reprice_status" binding:"required,oneof=success failed skipped"`
	StepReaddStatus   string `json:"step_readd_status" binding:"required,oneof=success failed skipped"`
	StepExitError     string `json:"step_exit_error"`
	StepRepriceError  string `json:"step_reprice_error"`
	StepReaddError    string `json:"step_readd_error"`
}

type ConfirmAutomationJobRequest struct {
	ShopID uint `json:"shop_id" binding:"required"`
}

type CancelAutomationJobRequest struct {
	ShopID uint `json:"shop_id" binding:"required"`
}

type RetryFailedAutomationJobRequest struct {
	ShopID uint `json:"shop_id" binding:"required"`
}

type AutomationEventListRequest struct {
	ShopID   uint `form:"shop_id" binding:"required"`
	JobID    uint `form:"job_id"`
	Page     int  `form:"page,default=1"`
	PageSize int  `form:"page_size,default=20"`
}

type AutomationEventItem struct {
	ID        uint   `json:"id"`
	JobID     uint   `json:"job_id"`
	EventType string `json:"event_type"`
	Message   string `json:"message"`
	CreatedBy *uint  `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at"`
}

type AutomationEventListResponse struct {
	Total int64                 `json:"total"`
	Items []AutomationEventItem `json:"items"`
}

type AgentStatusItem struct {
	ID              uint    `json:"id"`
	AgentKey        string  `json:"agent_key"`
	Name            string  `json:"name"`
	Hostname        string  `json:"hostname"`
	Status          string  `json:"status"`
	LastHeartbeatAt *string `json:"last_heartbeat_at,omitempty"`
	UpdatedAt       string  `json:"updated_at"`
}
