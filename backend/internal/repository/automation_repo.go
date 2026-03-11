package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"ozon-manager/internal/model"
)

type AutomationRepository struct {
	db *gorm.DB
}

func NewAutomationRepository(db *gorm.DB) *AutomationRepository {
	return &AutomationRepository{db: db}
}

func (r *AutomationRepository) CreateJobWithItems(job *model.AutomationJob, items []model.AutomationJobItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(job).Error; err != nil {
			return err
		}

		if len(items) == 0 {
			return nil
		}

		for index := range items {
			items[index].JobID = job.ID
		}

		return tx.CreateInBatches(items, 200).Error
	})
}

func (r *AutomationRepository) CreateJobEvent(event *model.AutomationJobEvent) error {
	return r.db.Create(event).Error
}

func (r *AutomationRepository) FindWithFilters(shopID uint, status string, page, pageSize int) ([]model.AutomationJob, int64, error) {
	var jobs []model.AutomationJob
	var total int64

	query := r.db.Model(&model.AutomationJob{}).Where("shop_id = ?", shopID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&jobs).Error

	return jobs, total, err
}

func (r *AutomationRepository) FindJobByIDAndShop(jobID, shopID uint) (*model.AutomationJob, error) {
	var job model.AutomationJob
	err := r.db.
		Where("id = ? AND shop_id = ?", jobID, shopID).
		Preload("AssignedAgent").
		Preload("Items").
		First(&job).Error
	if err != nil {
		return nil, err
	}

	return &job, nil
}

func (r *AutomationRepository) FindJobByID(jobID uint) (*model.AutomationJob, error) {
	var job model.AutomationJob
	err := r.db.
		Where("id = ?", jobID).
		Preload("Items").
		First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *AutomationRepository) UpsertAgentByKey(agentKey, name, hostname string, capabilities []byte) (*model.AutomationAgent, error) {
	now := time.Now()
	var agent model.AutomationAgent
	err := r.db.Where("agent_key = ?", agentKey).First(&agent).Error
	if err == gorm.ErrRecordNotFound {
		agent = model.AutomationAgent{
			AgentKey:        agentKey,
			Name:            name,
			Hostname:        hostname,
			Status:          model.AutomationAgentStatusOnline,
			Capabilities:    capabilities,
			LastHeartbeatAt: &now,
		}
		if createErr := r.db.Create(&agent).Error; createErr != nil {
			return nil, createErr
		}
		return &agent, nil
	}
	if err != nil {
		return nil, err
	}

	agent.Name = name
	agent.Hostname = hostname
	agent.Status = model.AutomationAgentStatusOnline
	agent.Capabilities = capabilities
	agent.LastHeartbeatAt = &now
	if err := r.db.Save(&agent).Error; err != nil {
		return nil, err
	}

	return &agent, nil
}

func (r *AutomationRepository) FindAgentByKey(agentKey string) (*model.AutomationAgent, error) {
	var agent model.AutomationAgent
	err := r.db.Where("agent_key = ?", agentKey).First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

func (r *AutomationRepository) AcquirePendingJobForAgent(agentID uint) (*model.AutomationJob, error) {
	var job model.AutomationJob
	err := r.db.Transaction(func(tx *gorm.DB) error {
		findErr := tx.Where("status = ? AND dry_run = ?", model.AutomationJobStatusPending, false).
			Where("job_type IN ?", []string{
				model.AutomationJobTypeRemoveRepriceReadd,
				model.AutomationJobTypeSyncShopActions,
				model.AutomationJobTypeSyncActionCandidates,
				model.AutomationJobTypeSyncActionProducts,
				model.AutomationJobTypeShopActionDeclare,
				model.AutomationJobTypeShopActionRemove,
				model.AutomationJobTypePromoUnifiedEnroll,
				model.AutomationJobTypePromoUnifiedRemove,
			}).
			Order("created_at ASC").
			First(&job).Error
		if findErr != nil {
			return findErr
		}

		now := time.Now()
		updateResult := tx.Model(&model.AutomationJob{}).
			Where("id = ? AND status = ? AND dry_run = ?", job.ID, model.AutomationJobStatusPending, false).
			Updates(map[string]interface{}{
				"status":            model.AutomationJobStatusRunning,
				"assigned_agent_id": agentID,
				"started_at":        &now,
			})
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.FindJobByID(job.ID)
}

func (r *AutomationRepository) AcquirePendingJobForShop(shopID uint, jobTypes []string, assignedAgentID *uint) (*model.AutomationJob, error) {
	var job model.AutomationJob
	err := r.db.Transaction(func(tx *gorm.DB) error {
		query := tx.Where("shop_id = ? AND status = ? AND dry_run = ?", shopID, model.AutomationJobStatusPending, false)
		if len(jobTypes) > 0 {
			query = query.Where("job_type IN ?", jobTypes)
		}

		findErr := query.Order("created_at ASC").First(&job).Error
		if findErr != nil {
			return findErr
		}

		now := time.Now()
		updates := map[string]interface{}{
			"status":     model.AutomationJobStatusRunning,
			"started_at": &now,
		}
		if assignedAgentID != nil {
			updates["assigned_agent_id"] = *assignedAgentID
		}

		updateResult := tx.Model(&model.AutomationJob{}).
			Where("id = ? AND status = ? AND dry_run = ?", job.ID, model.AutomationJobStatusPending, false).
			Updates(updates)
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.FindJobByID(job.ID)
}

func (r *AutomationRepository) ListPendingJobsByTypes(jobTypes []string, limit int) ([]model.AutomationJob, error) {
	if limit <= 0 {
		limit = 50
	}

	query := r.db.Where("status = ? AND dry_run = ?", model.AutomationJobStatusPending, false)
	if len(jobTypes) > 0 {
		query = query.Where("job_type IN ?", jobTypes)
	}

	var jobs []model.AutomationJob
	err := query.Order("created_at ASC").Limit(limit).Find(&jobs).Error
	return jobs, err
}

func (r *AutomationRepository) AcquirePendingJobByIDForAgent(jobID uint, agentID uint) (*model.AutomationJob, error) {
	now := time.Now()
	updateResult := r.db.Model(&model.AutomationJob{}).
		Where("id = ? AND status = ? AND dry_run = ?", jobID, model.AutomationJobStatusPending, false).
		Updates(map[string]interface{}{
			"status":            model.AutomationJobStatusRunning,
			"assigned_agent_id": agentID,
			"started_at":        &now,
		})
	if updateResult.Error != nil {
		return nil, updateResult.Error
	}
	if updateResult.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return r.FindJobByID(jobID)
}

func (r *AutomationRepository) HasOnlineExtensionForShop(shopID uint, staleAfter time.Time) (bool, error) {
	var count int64
	pattern := fmt.Sprintf("ext:%%:%d:%%", shopID)
	err := r.db.Model(&model.AutomationAgent{}).
		Where("agent_key LIKE ?", pattern).
		Where("last_heartbeat_at IS NOT NULL AND last_heartbeat_at >= ?", staleAfter).
		Count(&count).Error
	return count > 0, err
}

func (r *AutomationRepository) FindLatestExtensionAgentByShop(shopID uint) (*model.AutomationAgent, error) {
	var agent model.AutomationAgent
	pattern := fmt.Sprintf("ext:%%:%d:%%", shopID)
	err := r.db.Where("agent_key LIKE ?", pattern).Order("updated_at DESC").First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

func (r *AutomationRepository) FindLatestJobByShopAndTypes(shopID uint, jobTypes []string) (*model.AutomationJob, error) {
	var job model.AutomationJob
	query := r.db.Where("shop_id = ?", shopID)
	if len(jobTypes) > 0 {
		query = query.Where("job_type IN ?", jobTypes)
	}

	err := query.Order("id DESC").First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *AutomationRepository) FindLatestJobByShopAndTypesAndStatuses(shopID uint, jobTypes []string, statuses []string) (*model.AutomationJob, error) {
	var job model.AutomationJob
	query := r.db.Where("shop_id = ?", shopID)
	if len(jobTypes) > 0 {
		query = query.Where("job_type IN ?", jobTypes)
	}
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}

	err := query.Order("id DESC").First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *AutomationRepository) FindLatestFailedItemError(jobID uint) (string, error) {
	var item model.AutomationJobItem
	err := r.db.
		Where("job_id = ? AND (overall_status = ? OR step_exit_status = ? OR step_reprice_status = ? OR step_readd_status = ?)",
			jobID,
			model.AutomationStepStatusFailed,
			model.AutomationStepStatusFailed,
			model.AutomationStepStatusFailed,
			model.AutomationStepStatusFailed,
		).
		Order("id DESC").
		First(&item).Error
	if err != nil {
		return "", err
	}

	if item.StepExitError != "" {
		return item.StepExitError, nil
	}
	if item.StepRepriceError != "" {
		return item.StepRepriceError, nil
	}
	if item.StepReaddError != "" {
		return item.StepReaddError, nil
	}
	return "", nil
}

func (r *AutomationRepository) FindPendingJobByTypeAndShop(jobType string, shopID uint) (*model.AutomationJob, error) {
	var job model.AutomationJob
	err := r.db.Where("job_type = ? AND shop_id = ? AND status IN ?", jobType, shopID, []string{model.AutomationJobStatusPending, model.AutomationJobStatusRunning}).Order("id DESC").First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *AutomationRepository) UpdateJobAndItemsByReport(jobID uint, status string, results []model.AutomationJobItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		successCount := 0
		failedCount := 0

		for _, result := range results {
			updates := map[string]interface{}{
				"overall_status":      result.OverallStatus,
				"step_exit_status":    result.StepExitStatus,
				"step_reprice_status": result.StepRepriceStatus,
				"step_readd_status":   result.StepReaddStatus,
				"step_exit_error":     result.StepExitError,
				"step_reprice_error":  result.StepRepriceError,
				"step_readd_error":    result.StepReaddError,
			}
			if err := tx.Model(&model.AutomationJobItem{}).
				Where("job_id = ? AND source_sku = ?", jobID, result.SourceSKU).
				Updates(updates).Error; err != nil {
				return err
			}

			if result.OverallStatus == model.AutomationStepStatusSuccess || result.OverallStatus == model.AutomationStepStatusSkipped {
				successCount++
			} else {
				failedCount++
			}
		}

		now := time.Now()
		jobUpdates := map[string]interface{}{
			"status":        status,
			"success_items": successCount,
			"failed_items":  failedCount,
			"completed_at":  &now,
		}
		return tx.Model(&model.AutomationJob{}).Where("id = ?", jobID).Updates(jobUpdates).Error
	})
}

func (r *AutomationRepository) UpdateJobStatus(jobID uint, status string) error {
	updates := map[string]interface{}{"status": status}
	if status == model.AutomationJobStatusCanceled || status == model.AutomationJobStatusSuccess || status == model.AutomationJobStatusPartialSuccess || status == model.AutomationJobStatusFailed {
		now := time.Now()
		updates["completed_at"] = &now
	}
	if status == model.AutomationJobStatusRunning {
		now := time.Now()
		updates["started_at"] = &now
	}
	return r.db.Model(&model.AutomationJob{}).Where("id = ?", jobID).Updates(updates).Error
}

func (r *AutomationRepository) ResetFailedItemsForRetry(jobID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AutomationJobItem{}).
			Where("job_id = ? AND overall_status = ?", jobID, model.AutomationStepStatusFailed).
			Updates(map[string]interface{}{
				"overall_status":      model.AutomationStepStatusPending,
				"step_exit_status":    model.AutomationStepStatusPending,
				"step_reprice_status": model.AutomationStepStatusPending,
				"step_readd_status":   model.AutomationStepStatusPending,
				"step_exit_error":     "",
				"step_reprice_error":  "",
				"step_readd_error":    "",
				"retry_count":         gorm.Expr("retry_count + 1"),
			}).Error; err != nil {
			return err
		}

		return tx.Model(&model.AutomationJob{}).Where("id = ?", jobID).Updates(map[string]interface{}{
			"status":            model.AutomationJobStatusPending,
			"failed_items":      0,
			"assigned_agent_id": nil,
			"started_at":        nil,
			"completed_at":      nil,
		}).Error
	})
}

func (r *AutomationRepository) ListEventsByShop(shopID, jobID uint, page, pageSize int) ([]model.AutomationJobEvent, int64, error) {
	var events []model.AutomationJobEvent
	var total int64

	query := r.db.Model(&model.AutomationJobEvent{}).
		Joins("JOIN automation_jobs ON automation_jobs.id = automation_job_events.job_id").
		Where("automation_jobs.shop_id = ?", shopID)

	if jobID > 0 {
		query = query.Where("automation_job_events.job_id = ?", jobID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Offset(offset).
		Limit(pageSize).
		Order("automation_job_events.created_at DESC").
		Find(&events).Error
	if err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (r *AutomationRepository) ListAgents() ([]model.AutomationAgent, error) {
	var agents []model.AutomationAgent
	err := r.db.Order("updated_at DESC").Find(&agents).Error
	return agents, err
}

func (r *AutomationRepository) MarkStaleAgentsOffline(staleBefore time.Time) error {
	return r.db.Model(&model.AutomationAgent{}).
		Where("last_heartbeat_at IS NULL OR last_heartbeat_at < ?", staleBefore).
		Update("status", model.AutomationAgentStatusOffline).Error
}

func (r *AutomationRepository) CreateArtifact(jobID uint, artifactType string, payload interface{}) error {
	metaBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	artifact := &model.AutomationArtifact{
		JobID:        jobID,
		ArtifactType: artifactType,
		StoragePath:  "inline://agent",
		Meta:         metaBytes,
	}
	return r.db.Create(artifact).Error
}

func (r *AutomationRepository) FindLatestArtifactByJob(jobID uint, artifactType string) (*model.AutomationArtifact, error) {
	var artifact model.AutomationArtifact
	err := r.db.Where("job_id = ? AND artifact_type = ?", jobID, artifactType).Order("id DESC").First(&artifact).Error
	if err != nil {
		return nil, err
	}
	return &artifact, nil
}
