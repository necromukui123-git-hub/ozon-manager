package dto

import "time"

// 认证相关请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,len=64,hexadecimal"` // SHA-256 哈希固定64位十六进制
}

type LoginResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

type UserInfo struct {
	ID          uint       `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	Shops       []ShopInfo `json:"shops,omitempty"`
}

type ShopInfo struct {
	ID                  uint   `json:"id"`
	Name                string `json:"name"`
	IsActive            bool   `json:"is_active,omitempty"`
	ExecutionEngineMode string `json:"execution_engine_mode,omitempty"`
}

// 用户管理相关请求
type CreateUserRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required,len=64,hexadecimal"` // SHA-256 哈希
	DisplayName string `json:"display_name" binding:"required,max=100"`
	ShopIDs     []uint `json:"shop_ids"`
}

type UpdateUserStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active disabled"`
}

type UpdateUserPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,len=64,hexadecimal"` // SHA-256 哈希
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,len=64,hexadecimal"` // SHA-256 哈希
	NewPassword string `json:"new_password" binding:"required,len=64,hexadecimal"` // SHA-256 哈希
}

type UpdateUserShopsRequest struct {
	ShopIDs []uint `json:"shop_ids" binding:"required"`
}

// 店铺管理相关请求
type CreateShopRequest struct {
	Name     string `json:"name" binding:"required,max=100"`
	ClientID string `json:"client_id" binding:"required,max=50"`
	ApiKey   string `json:"api_key" binding:"required,max=200"`
}

type UpdateShopRequest struct {
	Name                string `json:"name" binding:"max=100"`
	ClientID            string `json:"client_id" binding:"max=50"`
	ApiKey              string `json:"api_key" binding:"max=200"`
	IsActive            *bool  `json:"is_active"`
	ExecutionEngineMode string `json:"execution_engine_mode" binding:"omitempty,oneof=auto extension agent"`
}

type UpdateShopExecutionEngineRequest struct {
	ExecutionEngineMode string `json:"execution_engine_mode" binding:"required,oneof=auto extension agent"`
}

type ShopExecutionEngineResponse struct {
	ShopID              uint   `json:"shop_id"`
	ExecutionEngineMode string `json:"execution_engine_mode"`
}

// 商品相关
type ProductListRequest struct {
	ShopID     uint   `form:"shop_id"`
	IsLoss     *bool  `form:"is_loss"`
	IsPromoted *bool  `form:"is_promoted"`
	Keyword    string `form:"keyword"`
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"page_size,default=20"`
}

type ProductListResponse struct {
	Total int64         `json:"total"`
	Items []ProductItem `json:"items"`
}

type ProductItem struct {
	ID           uint            `json:"id"`
	SourceSKU    string          `json:"source_sku"`
	Name         string          `json:"name"`
	CurrentPrice float64         `json:"current_price"`
	IsLoss       bool            `json:"is_loss"`
	IsPromoted   bool            `json:"is_promoted"`
	Promotions   []PromotionInfo `json:"promotions"`
	LossInfo     *LossInfo       `json:"loss_info,omitempty"`
}

type PromotionInfo struct {
	ActionID int64  `json:"action_id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
}

type LossInfo struct {
	LossDate      string  `json:"loss_date"`
	OriginalPrice float64 `json:"original_price"`
	NewPrice      float64 `json:"new_price"`
}

type SyncProductsRequest struct {
	ShopID uint `json:"shop_id" binding:"required"`
}

// ========== 三层角色系统相关 ==========

// 店铺管理员信息（系统管理员视角）
type ShopAdminInfo struct {
	ID          uint       `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Status      string     `json:"status"`
	ShopCount   int64      `json:"shop_count"`
	StaffCount  int64      `json:"staff_count"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// 店铺管理员详情（系统管理员视角）
type ShopAdminDetail struct {
	ID          uint       `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	Shops       []ShopInfo `json:"shops"`
	Staff       []UserInfo `json:"staff"`
}

// 创建店铺管理员请求
type CreateShopAdminRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required,len=64,hexadecimal"` // SHA-256 哈希
	DisplayName string `json:"display_name" binding:"required,max=100"`
}

// 创建员工请求（店铺管理员使用）
type CreateStaffRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required,len=64,hexadecimal"` // SHA-256 哈希
	DisplayName string `json:"display_name" binding:"required,max=100"`
	ShopIDs     []uint `json:"shop_ids"`
}

// 系统概览响应
type SystemOverviewResponse struct {
	ShopAdminCount int64 `json:"shop_admin_count"`
	ShopCount      int64 `json:"shop_count"`
	StaffCount     int64 `json:"staff_count"`
}

type ExtensionStatusItem struct {
	ShopID           uint    `json:"shop_id"`
	ShopName         string  `json:"shop_name"`
	ExecutionMode    string  `json:"execution_engine_mode"`
	ExtensionAgentID *uint   `json:"extension_agent_id,omitempty"`
	AgentKey         string  `json:"agent_key,omitempty"`
	AgentStatus      string  `json:"agent_status"`
	LastHeartbeatAt  *string `json:"last_heartbeat_at,omitempty"`
	LastError        string  `json:"last_error,omitempty"`
	LastRunAt        *string `json:"last_run_at,omitempty"`
	LatestJobID      *uint   `json:"latest_job_id,omitempty"`
	LatestJobType    string  `json:"latest_job_type,omitempty"`
	LatestJobStatus  string  `json:"latest_job_status,omitempty"`
}

// ========== 促销活动管理相关 ==========

// 获取促销活动列表请求
type GetActionsRequest struct {
	ShopID uint `form:"shop_id" binding:"required"`
}

// 手动创建促销活动请求
type CreateManualActionRequest struct {
	ShopID   uint   `json:"shop_id" binding:"required"`
	ActionID int64  `json:"action_id" binding:"required"`
	Title    string `json:"title"`
}

// 删除促销活动请求
type DeleteActionRequest struct {
	ShopID uint `json:"shop_id" binding:"required"`
}

// 同步促销活动请求
type SyncActionsRequest struct {
	ShopID uint `json:"shop_id" binding:"required"`
}

// 促销活动列表项响应
type PromotionActionItem struct {
	ID                 uint    `json:"id"`
	ShopID             uint    `json:"shop_id"`
	ActionID           int64   `json:"action_id"`
	Source             string  `json:"source"`
	SourceActionID     string  `json:"source_action_id"`
	Title              string  `json:"title"`
	ActionType         string  `json:"action_type"`
	DateStart          *string `json:"date_start"`
	DateEnd            *string `json:"date_end"`
	ParticipatingCount int     `json:"participating_products_count"`
	PotentialCount     int     `json:"potential_products_count"`
	IsManual           bool    `json:"is_manual"`
	Status             string  `json:"status"`
	LastSyncedAt       *string `json:"last_synced_at"`
}

type ActionProductsRequest struct {
	ShopID       uint `form:"shop_id" binding:"required"`
	Page         int  `form:"page,default=1"`
	PageSize     int  `form:"page_size,default=20"`
	ForceRefresh bool `form:"force_refresh"`
}

type ActionProductItem struct {
	ID            uint    `json:"id"`
	OzonProductID int64   `json:"ozon_product_id"`
	SourceSKU     string  `json:"source_sku"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	ActionPrice   float64 `json:"action_price"`
	Stock         int     `json:"stock"`
	Status        string  `json:"status"`
	LastSyncedAt  string  `json:"last_synced_at"`
}

type ActionProductsResponse struct {
	ActionID       uint                `json:"action_id"`
	Source         string              `json:"source"`
	SourceActionID string              `json:"source_action_id"`
	Total          int64               `json:"total"`
	Page           int                 `json:"page"`
	PageSize       int                 `json:"page_size"`
	Items          []ActionProductItem `json:"items"`
}

// 批量报名V2请求（支持选择具体活动）
type BatchEnrollV2Request struct {
	ShopID          uint    `json:"shop_id" binding:"required"`
	ActionIDs       []int64 `json:"action_ids" binding:"required,min=1"`
	ExcludeLoss     bool    `json:"exclude_loss"`
	ExcludePromoted bool    `json:"exclude_promoted"`
}

// 处理亏损商品V2请求（支持选择重新报名活动）
type ProcessLossV2Request struct {
	ShopID         uint   `json:"shop_id" binding:"required"`
	LossProductIDs []uint `json:"loss_product_ids" binding:"required"`
	RejoinActionID *int64 `json:"rejoin_action_id"`
}

// 改价推广V2请求（支持选择重新推广活动）
type RemoveRepricePromoteV2Request struct {
	ShopID            uint          `json:"shop_id" binding:"required"`
	Products          []RepriceItem `json:"products" binding:"required,dive"`
	ReenrollActionIDs []int64       `json:"reenroll_action_ids"`
}

// 更新促销活动显示名称请求
type UpdateActionDisplayNameRequest struct {
	ShopID      uint   `json:"shop_id"`
	DisplayName string `json:"display_name" binding:"max=200"`
}

// 更新促销活动排序请求
type UpdateActionsSortOrderRequest struct {
	ShopID     uint            `json:"shop_id" binding:"required"`
	SortOrders []SortOrderItem `json:"sort_orders" binding:"required,min=1,dive"`
}

// 排序项
type SortOrderItem struct {
	ID        uint `json:"id" binding:"required"`
	SortOrder int  `json:"sort_order" binding:"min=0"`
}

// ========== 统一促销操作 ==========

// UnifiedEnrollRequest 统一报名请求（自动判断官方/店铺）
type UnifiedEnrollRequest struct {
	ShopID          uint     `json:"shop_id" binding:"required"`
	ActionIDs       []uint   `json:"action_ids" binding:"required,min=1"` // promotion_actions 表 ID
	SourceSKUs      []string `json:"source_skus"`                         // 店铺活动用：指定 SKU 列表
	ExcludeLoss     bool     `json:"exclude_loss"`                        // 官方活动用
	ExcludePromoted bool     `json:"exclude_promoted"`                    // 官方活动用
}

// UnifiedRemoveRequest 统一退出请求
type UnifiedRemoveRequest struct {
	ShopID     uint     `json:"shop_id" binding:"required"`
	ActionIDs  []uint   `json:"action_ids" binding:"required,min=1"`
	SourceSKUs []string `json:"source_skus" binding:"required,min=1"`
}

// UnifiedOperationResponse 统一操作响应
type UnifiedOperationResponse struct {
	Mode    string               `json:"mode"`              // "sync" 或 "async"
	Results *BatchEnrollResponse `json:"results,omitempty"` // sync 模式下的结果
	JobID   *uint                `json:"job_id,omitempty"`  // async 模式下的 job ID
	Message string               `json:"message,omitempty"`
}

// UnifiedProcessLossRequest 统一亏损处理请求
type UnifiedProcessLossRequest struct {
	ShopID          uint   `json:"shop_id" binding:"required"`
	LossProductIDs  []uint `json:"loss_product_ids" binding:"required,min=1"`
	RejoinActionIDs []uint `json:"rejoin_action_ids"`
}

// UnifiedRepricePromoteRequest 统一改价推广请求
type UnifiedRepricePromoteRequest struct {
	ShopID            uint          `json:"shop_id" binding:"required"`
	Products          []RepriceItem `json:"products" binding:"required,min=1,dive"`
	ReenrollActionIDs []uint        `json:"reenroll_action_ids"`
}

// UnifiedProcessLossResponse 统一亏损处理响应
type UnifiedProcessLossResponse struct {
	Mode    string               `json:"mode"` // "sync" 或 "async"
	Result  *ProcessLossResponse `json:"result,omitempty"`
	JobID   *uint                `json:"job_id,omitempty"`
	Message string               `json:"message,omitempty"`
}

// UnifiedRepricePromoteResult 统一改价推广结果
type UnifiedRepricePromoteResult struct {
	Success          bool `json:"success"`
	RemoveCount      int  `json:"remove_count"`
	PriceUpdateCount int  `json:"price_update_count"`
	PromoteCount     int  `json:"promote_count"`
	FailedCount      int  `json:"failed_count"`
}

// UnifiedRepricePromoteResponse 统一改价推广响应
type UnifiedRepricePromoteResponse struct {
	Mode    string                       `json:"mode"` // "sync" 或 "async"
	Result  *UnifiedRepricePromoteResult `json:"result,omitempty"`
	JobID   *uint                        `json:"job_id,omitempty"`
	Message string                       `json:"message,omitempty"`
}
