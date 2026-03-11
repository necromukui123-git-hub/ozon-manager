package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	AutoPromotionTriggerModeManual    = "manual"
	AutoPromotionTriggerModeScheduled = "scheduled"

	AutoPromotionRunStatusPending        = "pending"
	AutoPromotionRunStatusRunning        = "running"
	AutoPromotionRunStatusSuccess        = "success"
	AutoPromotionRunStatusPartialSuccess = "partial_success"
	AutoPromotionRunStatusFailed         = "failed"

	AutoPromotionItemStatusPending = "pending"
	AutoPromotionItemStatusSuccess = "success"
	AutoPromotionItemStatusFailed  = "failed"
	AutoPromotionItemStatusSkipped = "skipped"

	PromotionActionCandidateStatusCandidate     = "candidate"
	PromotionActionCandidateStatusActive        = "active"
	PromotionActionCandidateStatusInactive      = "inactive"
	PromotionActionCandidateStatusAlreadyActive = "already_active"
)

type PromotionActionCandidate struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	PromotionActionID uint           `gorm:"not null;index;uniqueIndex:idx_action_candidate_sku" json:"promotion_action_id"`
	ShopID            uint           `gorm:"not null;index" json:"shop_id"`
	OzonProductID     int64          `gorm:"index" json:"ozon_product_id"`
	SourceSKU         string         `gorm:"size:120;not null;uniqueIndex:idx_action_candidate_sku" json:"source_sku"`
	OfferID           string         `gorm:"size:120" json:"offer_id"`
	PlatformSKU       string         `gorm:"size:120" json:"platform_sku"`
	ActionPrice       float64        `gorm:"type:decimal(12,2)" json:"action_price"`
	MaxActionPrice    float64        `gorm:"type:decimal(12,2)" json:"max_action_price"`
	DiscountPercent   float64        `gorm:"type:decimal(6,2)" json:"discount_percent"`
	Stock             int            `json:"stock"`
	Status            string         `gorm:"size:30;default:candidate" json:"status"`
	Payload           datatypes.JSON `gorm:"type:jsonb" json:"payload"`
	LastSyncedAt      *time.Time     `json:"last_synced_at"`
	CreatedAt         time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (PromotionActionCandidate) TableName() string {
	return "promotion_action_candidates"
}

type AutoPromotionConfig struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	ShopID            uint           `gorm:"not null;uniqueIndex" json:"shop_id"`
	Enabled           bool           `gorm:"not null;default:false" json:"enabled"`
	ScheduleTime      string         `gorm:"size:5;not null;default:09:05" json:"schedule_time"`
	TargetDate        time.Time      `gorm:"type:date;not null" json:"target_date"`
	OfficialActionIDs datatypes.JSON `gorm:"type:jsonb;not null" json:"official_action_ids"`
	ShopActionIDs     datatypes.JSON `gorm:"type:jsonb;not null" json:"shop_action_ids"`
	CreatedAt         time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (AutoPromotionConfig) TableName() string {
	return "auto_promotion_configs"
}

type AutoPromotionRun struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	ConfigID        *uint          `gorm:"index" json:"config_id"`
	ShopID          uint           `gorm:"not null;index" json:"shop_id"`
	TriggeredBy     *uint          `gorm:"index" json:"triggered_by"`
	TriggerMode     string         `gorm:"size:20;not null;index" json:"trigger_mode"`
	TriggerDate     time.Time      `gorm:"type:date;not null;index" json:"trigger_date"`
	TargetDate      time.Time      `gorm:"type:date;not null" json:"target_date"`
	Status          string         `gorm:"size:30;not null;default:pending;index" json:"status"`
	TotalCandidates int            `gorm:"default:0" json:"total_candidates"`
	TotalSelected   int            `gorm:"default:0" json:"total_selected"`
	TotalProcessed  int            `gorm:"default:0" json:"total_processed"`
	SuccessItems    int            `gorm:"default:0" json:"success_items"`
	FailedItems     int            `gorm:"default:0" json:"failed_items"`
	SkippedItems    int            `gorm:"default:0" json:"skipped_items"`
	ConfigSnapshot  datatypes.JSON `gorm:"type:jsonb" json:"config_snapshot"`
	ErrorMessage    string         `gorm:"type:text" json:"error_message"`
	StartedAt       *time.Time     `json:"started_at"`
	CompletedAt     *time.Time     `json:"completed_at"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	RunItems []AutoPromotionRunItem `gorm:"foreignKey:RunID" json:"run_items,omitempty"`
}

func (AutoPromotionRun) TableName() string {
	return "auto_promotion_runs"
}

type AutoPromotionRunItem struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	RunID           uint           `gorm:"not null;index;uniqueIndex:idx_auto_promo_run_item_sku" json:"run_id"`
	ProductID       *uint          `gorm:"index" json:"product_id"`
	OzonProductID   int64          `gorm:"index" json:"ozon_product_id"`
	SourceSKU       string         `gorm:"size:120;not null;uniqueIndex:idx_auto_promo_run_item_sku" json:"source_sku"`
	ProductName     string         `gorm:"size:500" json:"product_name"`
	ListingDate     time.Time      `gorm:"type:date;not null" json:"listing_date"`
	OverallStatus   string         `gorm:"size:20;not null;default:pending" json:"overall_status"`
	OfficialStatus  string         `gorm:"size:20;not null;default:pending" json:"official_status"`
	ShopStatus      string         `gorm:"size:20;not null;default:pending" json:"shop_status"`
	OfficialResults datatypes.JSON `gorm:"type:jsonb" json:"official_results"`
	ShopResults     datatypes.JSON `gorm:"type:jsonb" json:"shop_results"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (AutoPromotionRunItem) TableName() string {
	return "auto_promotion_run_items"
}
