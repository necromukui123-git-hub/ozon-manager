package dto

// 认证相关请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
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
	Shops       []ShopInfo `json:"shops,omitempty"`
}

type ShopInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// 用户管理相关请求
type CreateUserRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required,min=6"`
	DisplayName string `json:"display_name" binding:"required,max=100"`
	ShopIDs     []uint `json:"shop_ids"`
}

type UpdateUserStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active disabled"`
}

type UpdateUserPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
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
	Name     string `json:"name" binding:"max=100"`
	ClientID string `json:"client_id" binding:"max=50"`
	ApiKey   string `json:"api_key" binding:"max=200"`
	IsActive *bool  `json:"is_active"`
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
	ID           uint              `json:"id"`
	SourceSKU    string            `json:"source_sku"`
	Name         string            `json:"name"`
	CurrentPrice float64           `json:"current_price"`
	IsLoss       bool              `json:"is_loss"`
	IsPromoted   bool              `json:"is_promoted"`
	Promotions   []PromotionInfo   `json:"promotions"`
	LossInfo     *LossInfo         `json:"loss_info,omitempty"`
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
