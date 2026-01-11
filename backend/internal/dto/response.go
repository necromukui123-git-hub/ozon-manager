package dto

// 促销管理相关请求
type BatchEnrollRequest struct {
	ShopID             uint `json:"shop_id" binding:"required"`
	ExcludeLoss        bool `json:"exclude_loss"`
	ExcludePromoted    bool `json:"exclude_promoted"`
	EnrollElasticBoost bool `json:"enroll_elastic_boost"`
	EnrollDiscount28   bool `json:"enroll_discount_28"`
}

type BatchEnrollResponse struct {
	Success       bool           `json:"success"`
	EnrolledCount int            `json:"enrolled_count"`
	FailedCount   int            `json:"failed_count"`
	Details       []EnrollDetail `json:"details,omitempty"`
}

type EnrollDetail struct {
	ProductID uint   `json:"product_id"`
	SourceSKU string `json:"source_sku"`
	Status    string `json:"status"` // success / failed
	Error     string `json:"error,omitempty"`
}

type ProcessLossRequest struct {
	ShopID         uint   `json:"shop_id" binding:"required"`
	LossProductIDs []uint `json:"loss_product_ids" binding:"required"`
}

type ProcessLossResponse struct {
	Success        bool          `json:"success"`
	ProcessedCount int           `json:"processed_count"`
	Steps          ProcessSteps  `json:"steps"`
}

type ProcessSteps struct {
	ExitPromotion     StepResult `json:"exit_promotion"`
	PriceUpdate       StepResult `json:"price_update"`
	RejoinDiscount28  StepResult `json:"rejoin_discount_28"`
}

type StepResult struct {
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

type RemoveRepricePromoteRequest struct {
	ShopID   uint            `json:"shop_id" binding:"required"`
	Products []RepriceItem   `json:"products" binding:"required,dive"`
}

type RepriceItem struct {
	SourceSKU string  `json:"source_sku" binding:"required"`
	NewPrice  float64 `json:"new_price" binding:"required,gt=0"`
}

// Excel相关
type ImportLossRequest struct {
	ShopID uint `form:"shop_id" binding:"required"`
}

type ImportLossResponse struct {
	Success        bool   `json:"success"`
	ImportedCount  int    `json:"imported_count"`
	LossProductIDs []uint `json:"loss_product_ids"`
}

// 操作日志相关
type OperationLogListRequest struct {
	UserID        uint   `form:"user_id"`
	ShopID        uint   `form:"shop_id"`
	OperationType string `form:"operation_type"`
	DateFrom      string `form:"date_from"`
	DateTo        string `form:"date_to"`
	Page          int    `form:"page,default=1"`
	PageSize      int    `form:"page_size,default=20"`
}

type OperationLogListResponse struct {
	Total int64              `json:"total"`
	Items []OperationLogItem `json:"items"`
}

type OperationLogItem struct {
	ID              uint        `json:"id"`
	User            UserInfo    `json:"user"`
	Shop            *ShopInfo   `json:"shop"`
	OperationType   string      `json:"operation_type"`
	OperationDetail interface{} `json:"operation_detail"`
	AffectedCount   int         `json:"affected_count"`
	Status          string      `json:"status"`
	IPAddress       string      `json:"ip_address"`
	CreatedAt       string      `json:"created_at"`
}

// 通用响应
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PageResponse struct {
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Items    interface{} `json:"items"`
}

// 统计相关
type StatsOverview struct {
	TotalProducts      int64 `json:"total_products"`
	LossProducts       int64 `json:"loss_products"`
	PromotedProducts   int64 `json:"promoted_products"`
	PromotableProducts int64 `json:"promotable_products"`
	ElasticBoostCount  int64 `json:"elastic_boost_count"`
	Discount28Count    int64 `json:"discount_28_count"`
}
