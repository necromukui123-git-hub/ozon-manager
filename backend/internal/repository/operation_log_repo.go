package repository

import (
	"time"

	"gorm.io/gorm"
	"ozon-manager/internal/model"
)

type OperationLogRepository struct {
	db *gorm.DB
}

func NewOperationLogRepository(db *gorm.DB) *OperationLogRepository {
	return &OperationLogRepository{db: db}
}

// Create 创建操作日志
func (r *OperationLogRepository) Create(log *model.OperationLog) error {
	return r.db.Create(log).Error
}

// FindByID 根据ID查找操作日志
func (r *OperationLogRepository) FindByID(id uint) (*model.OperationLog, error) {
	var log model.OperationLog
	err := r.db.Preload("User").Preload("Shop").First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// FindWithFilters 带筛选条件的操作日志列表
func (r *OperationLogRepository) FindWithFilters(
	userID uint,
	shopID uint,
	operationType string,
	dateFrom, dateTo time.Time,
	page, pageSize int,
) ([]model.OperationLog, int64, error) {
	var logs []model.OperationLog
	var total int64

	query := r.db.Model(&model.OperationLog{})

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	if shopID > 0 {
		query = query.Where("shop_id = ?", shopID)
	}

	if operationType != "" {
		query = query.Where("operation_type = ?", operationType)
	}

	if !dateFrom.IsZero() {
		query = query.Where("created_at >= ?", dateFrom)
	}

	if !dateTo.IsZero() {
		query = query.Where("created_at <= ?", dateTo)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Preload("User").Preload("Shop").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

// FindByUserID 获取用户的操作日志
func (r *OperationLogRepository) FindByUserID(userID uint, limit int) ([]model.OperationLog, error) {
	var logs []model.OperationLog
	err := r.db.Where("user_id = ?", userID).
		Preload("Shop").
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// UpdateStatus 更新操作日志状态
func (r *OperationLogRepository) UpdateStatus(id uint, status string, errorMessage string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":       status,
		"completed_at": &now,
	}
	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}
	return r.db.Model(&model.OperationLog{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateAffectedCount 更新影响数量
func (r *OperationLogRepository) UpdateAffectedCount(id uint, count int) error {
	return r.db.Model(&model.OperationLog{}).Where("id = ?", id).Update("affected_count", count).Error
}
