package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	AutomationJobTypeRemoveRepriceReadd = "remove_reprice_readd"
	AutomationJobTypeSyncShopActions    = "sync_shop_actions"
	AutomationJobTypeSyncActionProducts = "sync_action_products"

	AutomationJobStatusPending         = "pending"
	AutomationJobStatusAwaitConfirm    = "await_confirm"
	AutomationJobStatusRunning         = "running"
	AutomationJobStatusSuccess         = "success"
	AutomationJobStatusPartialSuccess  = "partial_success"
	AutomationJobStatusFailed          = "failed"
	AutomationJobStatusCanceled        = "canceled"
	AutomationJobStatusDryRunCompleted = "dry_run_completed"

	AutomationStepStatusPending = "pending"
	AutomationStepStatusSkipped = "skipped"
	AutomationStepStatusSuccess = "success"
	AutomationStepStatusFailed  = "failed"

	AutomationAgentStatusOnline  = "online"
	AutomationAgentStatusOffline = "offline"
)

type AutomationJob struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	ShopID               uint       `gorm:"not null;index" json:"shop_id"`
	CreatedBy            uint       `gorm:"not null;index" json:"created_by"`
	AssignedAgentID      *uint      `gorm:"index" json:"assigned_agent_id"`
	JobType              string     `gorm:"size:50;not null;index" json:"job_type"`
	Status               string     `gorm:"size:30;not null;default:pending;index" json:"status"`
	DryRun               bool       `gorm:"default:false" json:"dry_run"`
	RequiresConfirmation bool       `gorm:"default:false" json:"requires_confirmation"`
	RateLimit            int        `gorm:"default:30" json:"rate_limit"`
	TotalItems           int        `gorm:"default:0" json:"total_items"`
	SuccessItems         int        `gorm:"default:0" json:"success_items"`
	FailedItems          int        `gorm:"default:0" json:"failed_items"`
	ErrorMessage         string     `gorm:"type:text" json:"error_message"`
	StartedAt            *time.Time `json:"started_at"`
	CompletedAt          *time.Time `json:"completed_at"`
	CreatedAt            time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	Shop          Shop                `gorm:"foreignKey:ShopID" json:"shop,omitempty"`
	Creator       User                `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	AssignedAgent *AutomationAgent    `gorm:"foreignKey:AssignedAgentID" json:"assigned_agent,omitempty"`
	Items         []AutomationJobItem `gorm:"foreignKey:JobID" json:"items,omitempty"`
}

func (AutomationJob) TableName() string {
	return "automation_jobs"
}

type AutomationJobItem struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	JobID             uint      `gorm:"not null;index;uniqueIndex:idx_automation_job_source_sku" json:"job_id"`
	ProductID         *uint     `gorm:"index" json:"product_id"`
	SourceSKU         string    `gorm:"size:100;not null;uniqueIndex:idx_automation_job_source_sku" json:"source_sku"`
	TargetPrice       float64   `gorm:"type:decimal(12,2);not null" json:"target_price"`
	OverallStatus     string    `gorm:"size:20;not null;default:pending;index" json:"overall_status"`
	StepExitStatus    string    `gorm:"size:20;not null;default:pending" json:"step_exit_status"`
	StepRepriceStatus string    `gorm:"size:20;not null;default:pending" json:"step_reprice_status"`
	StepReaddStatus   string    `gorm:"size:20;not null;default:pending" json:"step_readd_status"`
	StepExitError     string    `gorm:"type:text" json:"step_exit_error"`
	StepRepriceError  string    `gorm:"type:text" json:"step_reprice_error"`
	StepReaddError    string    `gorm:"type:text" json:"step_readd_error"`
	RetryCount        int       `gorm:"default:0" json:"retry_count"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Job     AutomationJob `gorm:"foreignKey:JobID" json:"job,omitempty"`
	Product *Product      `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

func (AutomationJobItem) TableName() string {
	return "automation_job_items"
}

type AutomationAgent struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	AgentKey        string         `gorm:"size:100;not null;uniqueIndex" json:"agent_key"`
	Name            string         `gorm:"size:100;not null" json:"name"`
	Hostname        string         `gorm:"size:200" json:"hostname"`
	Status          string         `gorm:"size:20;not null;default:offline" json:"status"`
	Capabilities    datatypes.JSON `gorm:"type:jsonb" json:"capabilities"`
	LastHeartbeatAt *time.Time     `json:"last_heartbeat_at"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (AutomationAgent) TableName() string {
	return "automation_agents"
}

type AutomationJobEvent struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	JobID     uint           `gorm:"not null;index" json:"job_id"`
	EventType string         `gorm:"size:50;not null" json:"event_type"`
	Message   string         `gorm:"type:text" json:"message"`
	Payload   datatypes.JSON `gorm:"type:jsonb" json:"payload"`
	CreatedBy *uint          `json:"created_by"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`

	Job AutomationJob `gorm:"foreignKey:JobID" json:"job,omitempty"`
}

func (AutomationJobEvent) TableName() string {
	return "automation_job_events"
}

type AutomationArtifact struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	JobID        uint           `gorm:"not null;index" json:"job_id"`
	JobItemID    *uint          `gorm:"index" json:"job_item_id"`
	ArtifactType string         `gorm:"size:50;not null" json:"artifact_type"`
	StoragePath  string         `gorm:"size:500;not null" json:"storage_path"`
	Checksum     string         `gorm:"size:128" json:"checksum"`
	Meta         datatypes.JSON `gorm:"type:jsonb" json:"meta"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`

	Job     AutomationJob      `gorm:"foreignKey:JobID" json:"job,omitempty"`
	JobItem *AutomationJobItem `gorm:"foreignKey:JobItemID" json:"job_item,omitempty"`
}

func (AutomationArtifact) TableName() string {
	return "automation_artifacts"
}
