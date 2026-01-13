package model

import (
	"time"

	"gorm.io/datatypes"
)

// Product 商品表
type Product struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ShopID        uint       `gorm:"not null;index" json:"shop_id"`
	OzonProductID int64      `gorm:"not null;uniqueIndex:idx_shop_ozon_product" json:"ozon_product_id"`
	OzonSKU       int64      `json:"ozon_sku"`
	SourceSKU     string     `gorm:"size:100;not null;uniqueIndex:idx_shop_source_sku" json:"source_sku"`
	Name          string     `gorm:"size:500" json:"name"`
	CurrentPrice  float64    `gorm:"type:decimal(12,2)" json:"current_price"`
	Status        string     `gorm:"size:20;default:active" json:"status"` // active / inactive / archived
	IsLoss        bool       `gorm:"default:false" json:"is_loss"`
	IsPromoted    bool       `gorm:"default:false" json:"is_promoted"`
	LastSyncedAt  *time.Time `json:"last_synced_at"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Shop             Shop              `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
	LossProducts     []LossProduct     `gorm:"foreignKey:ProductID" json:"loss_products,omitempty"`
	PromotedProducts []PromotedProduct `gorm:"foreignKey:ProductID" json:"promoted_products,omitempty"`
}

func (Product) TableName() string {
	return "products"
}

// LossProduct 亏损商品表
type LossProduct struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	ProductID        uint       `gorm:"not null;uniqueIndex:idx_product_loss_date" json:"product_id"`
	LossDate         time.Time  `gorm:"type:date;not null;uniqueIndex:idx_product_loss_date" json:"loss_date"`
	OriginalPrice    float64    `gorm:"type:decimal(12,2)" json:"original_price"`
	NewPrice         float64    `gorm:"type:decimal(12,2);not null" json:"new_price"`
	PriceUpdated     bool       `gorm:"default:false" json:"price_updated"`
	PromotionExited  bool       `gorm:"default:false" json:"promotion_exited"`
	PromotionRejoined bool      `gorm:"default:false" json:"promotion_rejoined"`
	ProcessedAt      *time.Time `json:"processed_at"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

func (LossProduct) TableName() string {
	return "loss_products"
}

// PromotedProduct 已推广商品表
type PromotedProduct struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ProductID     uint       `gorm:"not null;uniqueIndex:idx_product_promotion_action" json:"product_id"`
	PromotionType string     `gorm:"size:50;not null;uniqueIndex:idx_product_promotion_action" json:"promotion_type"`
	ActionID      int64      `gorm:"uniqueIndex:idx_product_promotion_action" json:"action_id"`
	ActionPrice   float64    `gorm:"type:decimal(12,2)" json:"action_price"`
	Status        string     `gorm:"size:20;default:active" json:"status"` // active / exited / pending
	PromotedAt    time.Time  `gorm:"autoCreateTime" json:"promoted_at"`
	ExitedAt      *time.Time `json:"exited_at"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

func (PromotedProduct) TableName() string {
	return "promoted_products"
}

// PromotionAction 促销活动缓存表
type PromotionAction struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	ShopID             uint       `gorm:"not null;uniqueIndex:idx_shop_action" json:"shop_id"`
	ActionID           int64      `gorm:"not null;uniqueIndex:idx_shop_action" json:"action_id"`
	Title              string     `gorm:"size:200" json:"title"`
	ActionType         string     `gorm:"size:50" json:"action_type"`
	DateStart          *time.Time `json:"date_start"`
	DateEnd            *time.Time `json:"date_end"`
	ParticipatingCount int        `gorm:"default:0" json:"participating_products_count"`
	PotentialCount     int        `gorm:"default:0" json:"potential_products_count"`
	IsManual           bool       `gorm:"default:false" json:"is_manual"`
	Status             string     `gorm:"size:20;default:active" json:"status"` // active / expired / disabled
	LastSyncedAt       *time.Time `json:"last_synced_at"`
	CreatedAt          time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Shop Shop `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
}

func (PromotionAction) TableName() string {
	return "promotion_actions"
}

// OperationLog 操作日志表
type OperationLog struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	UserID          uint           `gorm:"not null;index" json:"user_id"`
	ShopID          *uint          `gorm:"index" json:"shop_id"`
	OperationType   string         `gorm:"size:50;not null" json:"operation_type"`
	OperationDetail datatypes.JSON `gorm:"type:jsonb" json:"operation_detail"`
	AffectedCount   int            `gorm:"default:0" json:"affected_count"`
	Status          string         `gorm:"size:20;default:pending" json:"status"` // pending / success / failed
	ErrorMessage    string         `gorm:"type:text" json:"error_message"`
	IPAddress       string         `gorm:"size:45" json:"ip_address"`
	UserAgent       string         `gorm:"size:500" json:"user_agent"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"created_at"`
	CompletedAt     *time.Time     `json:"completed_at"`

	// 关联
	User User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Shop *Shop `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
}

func (OperationLog) TableName() string {
	return "operation_logs"
}
