package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	SearchCPORunStatusPending        = "pending"
	SearchCPORunStatusRunning        = "running"
	SearchCPORunStatusSuccess        = "success"
	SearchCPORunStatusPartialSuccess = "partial_success"
	SearchCPORunStatusFailed         = "failed"

	SearchCPOItemStatusPending        = "pending"
	SearchCPOItemStatusSuccess        = "success"
	SearchCPOItemStatusPartialSuccess = "partial_success"
	SearchCPOItemStatusFailed         = "failed"
	SearchCPOItemStatusSkipped        = "skipped"
)

type SearchCPOConfig struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	ShopID            uint           `gorm:"not null;uniqueIndex" json:"shop_id"`
	OfficialActionIDs datatypes.JSON `gorm:"type:jsonb;not null" json:"official_action_ids"`
	ShopActionIDs     datatypes.JSON `gorm:"type:jsonb;not null" json:"shop_action_ids"`
	CreatedAt         time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (SearchCPOConfig) TableName() string {
	return "search_cpo_configs"
}

type SearchCPOProduct struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	ShopID            uint           `gorm:"not null;index;uniqueIndex:idx_search_cpo_shop_source_sku" json:"shop_id"`
	SKU               string         `gorm:"size:120" json:"sku"`
	SourceSKU         string         `gorm:"size:120;not null;uniqueIndex:idx_search_cpo_shop_source_sku;index" json:"source_sku"`
	ImageURL          string         `gorm:"type:text" json:"image_url"`
	Title             string         `gorm:"size:500" json:"title"`
	CategoryName      string         `gorm:"size:300" json:"category_name"`
	Price             float64        `gorm:"type:decimal(12,2)" json:"price"`
	IsInStock         bool           `gorm:"default:false" json:"is_in_stock"`
	SearchPromoStatus string         `gorm:"size:80;index" json:"search_promo_status"`
	IsFavorite        bool           `gorm:"default:false" json:"is_favorite"`
	Orders            int64          `gorm:"default:0" json:"orders"`
	Spent             float64        `gorm:"type:decimal(12,2)" json:"spent"`
	Clicks            int64          `gorm:"default:0" json:"clicks"`
	CTRPercent        float64        `gorm:"type:decimal(8,4)" json:"ctr_percent"`
	StockTotal        int64          `gorm:"default:0" json:"stock_total"`
	Payload           datatypes.JSON `gorm:"type:jsonb" json:"payload"`
	LastSyncedAt      *time.Time     `json:"last_synced_at"`
	CreatedAt         time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (SearchCPOProduct) TableName() string {
	return "search_cpo_products"
}

type SearchCPORun struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	ShopID         uint           `gorm:"not null;index" json:"shop_id"`
	TriggeredBy    *uint          `gorm:"index" json:"triggered_by"`
	Status         string         `gorm:"size:30;not null;default:pending;index" json:"status"`
	FilterSnapshot datatypes.JSON `gorm:"type:jsonb" json:"filter_snapshot"`
	ActionSnapshot datatypes.JSON `gorm:"type:jsonb" json:"action_snapshot"`
	TotalFetched   int            `gorm:"default:0" json:"total_fetched"`
	TotalSelected  int            `gorm:"default:0" json:"total_selected"`
	TotalProcessed int            `gorm:"default:0" json:"total_processed"`
	SuccessItems   int            `gorm:"default:0" json:"success_items"`
	FailedItems    int            `gorm:"default:0" json:"failed_items"`
	SkippedItems   int            `gorm:"default:0" json:"skipped_items"`
	ErrorMessage   string         `gorm:"type:text" json:"error_message"`
	StartedAt      *time.Time     `json:"started_at"`
	CompletedAt    *time.Time     `json:"completed_at"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	RunItems []SearchCPORunItem `gorm:"foreignKey:RunID" json:"run_items,omitempty"`
}

func (SearchCPORun) TableName() string {
	return "search_cpo_runs"
}

type SearchCPORunItem struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	RunID             uint           `gorm:"not null;index;uniqueIndex:idx_search_cpo_run_item_source_sku" json:"run_id"`
	ProductCacheID    *uint          `gorm:"index" json:"product_cache_id"`
	SourceSKU         string         `gorm:"size:120;not null;uniqueIndex:idx_search_cpo_run_item_source_sku" json:"source_sku"`
	SKU               string         `gorm:"size:120" json:"sku"`
	Title             string         `gorm:"size:500" json:"title"`
	SearchPromoStatus string         `gorm:"size:80" json:"search_promo_status"`
	OverallStatus     string         `gorm:"size:20;not null;default:pending" json:"overall_status"`
	OfficialStatus    string         `gorm:"size:20;not null;default:pending" json:"official_status"`
	ShopStatus        string         `gorm:"size:20;not null;default:pending" json:"shop_status"`
	OfficialResults   datatypes.JSON `gorm:"type:jsonb;not null" json:"official_results"`
	ShopResults       datatypes.JSON `gorm:"type:jsonb;not null" json:"shop_results"`
	CreatedAt         time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (SearchCPORunItem) TableName() string {
	return "search_cpo_run_items"
}
