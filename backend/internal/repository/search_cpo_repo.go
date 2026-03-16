package repository

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"ozon-manager/internal/model"
)

type SearchCPORepository struct {
	db *gorm.DB
}

func NewSearchCPORepository(db *gorm.DB) *SearchCPORepository {
	return &SearchCPORepository{db: db}
}

func (r *SearchCPORepository) FindConfigByShopID(shopID uint) (*model.SearchCPOConfig, error) {
	var config model.SearchCPOConfig
	err := r.db.Where("shop_id = ?", shopID).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *SearchCPORepository) UpsertConfig(config *model.SearchCPOConfig) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "shop_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"official_action_ids",
			"shop_action_ids",
			"updated_at",
		}),
	}).Create(config).Error
}

func (r *SearchCPORepository) ReplaceProducts(shopID uint, products []model.SearchCPOProduct) error {
	now := time.Now()
	for i := range products {
		products[i].ShopID = shopID
		products[i].LastSyncedAt = &now
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		sourceSKUs := make([]string, 0, len(products))
		for i := range products {
			product := products[i]
			sourceSKUs = append(sourceSKUs, product.SourceSKU)
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "shop_id"},
					{Name: "source_sku"},
				},
				DoUpdates: clause.AssignmentColumns([]string{
					"sku",
					"image_url",
					"title",
					"category_name",
					"price",
					"is_in_stock",
					"search_promo_status",
					"is_favorite",
					"orders",
					"spent",
					"clicks",
					"ctr_percent",
					"stock_total",
					"payload",
					"last_synced_at",
					"updated_at",
				}),
			}).Create(&product).Error; err != nil {
				return err
			}
		}

		if len(sourceSKUs) == 0 {
			return tx.Where("shop_id = ?", shopID).Delete(&model.SearchCPOProduct{}).Error
		}

		return tx.Where("shop_id = ? AND source_sku NOT IN ?", shopID, sourceSKUs).Delete(&model.SearchCPOProduct{}).Error
	})
}

func (r *SearchCPORepository) ListProducts(shopID uint) ([]model.SearchCPOProduct, error) {
	items := make([]model.SearchCPOProduct, 0)
	err := r.db.Where("shop_id = ?", shopID).Order("source_sku ASC").Find(&items).Error
	return items, err
}

func (r *SearchCPORepository) FindProductsBySourceSKUs(shopID uint, sourceSKUs []string) ([]model.SearchCPOProduct, error) {
	items := make([]model.SearchCPOProduct, 0)
	if len(sourceSKUs) == 0 {
		return items, nil
	}
	err := r.db.Where("shop_id = ? AND source_sku IN ?", shopID, sourceSKUs).Find(&items).Error
	return items, err
}

func (r *SearchCPORepository) CreateRun(run *model.SearchCPORun) error {
	return r.db.Create(run).Error
}

func (r *SearchCPORepository) UpdateRun(run *model.SearchCPORun) error {
	return r.db.Save(run).Error
}

func (r *SearchCPORepository) ReplaceRunItems(runID uint, items []model.SearchCPORunItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("run_id = ?", runID).Delete(&model.SearchCPORunItem{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for i := range items {
			items[i].RunID = runID
		}
		return tx.CreateInBatches(items, 200).Error
	})
}

func (r *SearchCPORepository) ListRunsByShop(shopID uint, page, pageSize int) ([]model.SearchCPORun, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	var runs []model.SearchCPORun
	var total int64
	query := r.db.Model(&model.SearchCPORun{}).Where("shop_id = ?", shopID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&runs).Error
	return runs, total, err
}

func (r *SearchCPORepository) FindRunByIDAndShop(runID, shopID uint) (*model.SearchCPORun, error) {
	var run model.SearchCPORun
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

func (r *SearchCPORepository) FindActiveRunByShop(shopID uint) (*model.SearchCPORun, error) {
	var run model.SearchCPORun
	err := r.db.Where("shop_id = ? AND status IN ?", shopID, []string{
		model.SearchCPORunStatusPending,
		model.SearchCPORunStatusRunning,
	}).Order("id DESC").First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}
