package repository

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"ozon-manager/internal/model"
)

type AutoPromotionRepository struct {
	db *gorm.DB
}

func NewAutoPromotionRepository(db *gorm.DB) *AutoPromotionRepository {
	return &AutoPromotionRepository{db: db}
}

func (r *AutoPromotionRepository) FindConfigByShopID(shopID uint) (*model.AutoPromotionConfig, error) {
	var config model.AutoPromotionConfig
	err := r.db.Where("shop_id = ?", shopID).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *AutoPromotionRepository) UpsertConfig(config *model.AutoPromotionConfig) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "shop_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled", "schedule_time", "target_date", "official_action_ids", "shop_action_ids", "updated_at"}),
	}).Create(config).Error
}

func (r *AutoPromotionRepository) ListEnabledConfigs() ([]model.AutoPromotionConfig, error) {
	configs := make([]model.AutoPromotionConfig, 0)
	err := r.db.Where("enabled = ?", true).Order("shop_id ASC").Find(&configs).Error
	return configs, err
}

func (r *AutoPromotionRepository) FindActiveRunByShop(shopID uint) (*model.AutoPromotionRun, error) {
	var run model.AutoPromotionRun
	err := r.db.Where("shop_id = ? AND status IN ?", shopID, []string{
		model.AutoPromotionRunStatusPending,
		model.AutoPromotionRunStatusRunning,
	}).Order("id DESC").First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *AutoPromotionRepository) FindScheduledRunByConfigAndDate(configID uint, triggerDate time.Time) (*model.AutoPromotionRun, error) {
	var run model.AutoPromotionRun
	err := r.db.Where("config_id = ? AND trigger_mode = ? AND trigger_date = ?", configID, model.AutoPromotionTriggerModeScheduled, triggerDate).
		Order("id DESC").
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *AutoPromotionRepository) CreateRun(run *model.AutoPromotionRun) error {
	return r.db.Create(run).Error
}

func (r *AutoPromotionRepository) UpdateRun(run *model.AutoPromotionRun) error {
	return r.db.Save(run).Error
}

func (r *AutoPromotionRepository) ReplaceRunItems(runID uint, items []model.AutoPromotionRunItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("run_id = ?", runID).Delete(&model.AutoPromotionRunItem{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for index := range items {
			items[index].RunID = runID
		}
		return tx.CreateInBatches(items, 200).Error
	})
}

func (r *AutoPromotionRepository) ListRunsByShop(shopID uint, page, pageSize int) ([]model.AutoPromotionRun, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	var runs []model.AutoPromotionRun
	var total int64

	query := r.db.Model(&model.AutoPromotionRun{}).Where("shop_id = ?", shopID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&runs).Error
	return runs, total, err
}

func (r *AutoPromotionRepository) FindRunByIDAndShop(runID, shopID uint) (*model.AutoPromotionRun, error) {
	var run model.AutoPromotionRun
	err := r.db.Where("id = ? AND shop_id = ?", runID, shopID).
		Preload("RunItems", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *AutoPromotionRepository) MarkStaleRunningRunsFailed(staleBefore time.Time) error {
	return r.db.Model(&model.AutoPromotionRun{}).
		Where("status = ? AND updated_at < ?", model.AutoPromotionRunStatusRunning, staleBefore).
		Updates(map[string]interface{}{
			"status":         model.AutoPromotionRunStatusFailed,
			"error_message":  "后台重启后将超时运行标记为失败",
			"completed_at":   time.Now(),
			"updated_at":     time.Now(),
		}).Error
}
