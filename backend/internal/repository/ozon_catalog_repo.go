package repository

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"ozon-manager/internal/model"
)

type OzonCatalogListQuery struct {
	ShopID            uint
	PageSize          int
	CursorListingDate *time.Time
	CursorID          uint
	Visibility        string
	OfferIDs          []string
	ProductIDs        []int64
	ListedFrom        *time.Time
	ListedTo          *time.Time
	ListingDateSource string
}

type OzonCatalogRepository struct {
	db *gorm.DB
}

func NewOzonCatalogRepository(db *gorm.DB) *OzonCatalogRepository {
	return &OzonCatalogRepository{db: db}
}

func (r *OzonCatalogRepository) ListWithFilters(query OzonCatalogListQuery) ([]model.OzonProductCatalogItem, int64, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	baseQuery := r.db.Model(&model.OzonProductCatalogItem{}).
		Where("shop_id = ?", query.ShopID)

	if query.Visibility != "" {
		baseQuery = baseQuery.Where("visibility = ?", query.Visibility)
	}
	if len(query.OfferIDs) > 0 {
		baseQuery = baseQuery.Where("offer_id IN ?", query.OfferIDs)
	}
	if len(query.ProductIDs) > 0 {
		baseQuery = baseQuery.Where("ozon_product_id IN ?", query.ProductIDs)
	}
	if query.ListingDateSource != "" && query.ListingDateSource != "all" {
		baseQuery = baseQuery.Where("listing_date_source = ?", query.ListingDateSource)
	}
	if query.ListedFrom != nil {
		baseQuery = baseQuery.Where("listing_date >= ?", *query.ListedFrom)
	}
	if query.ListedTo != nil {
		baseQuery = baseQuery.Where("listing_date < ?", query.ListedTo.Add(24*time.Hour))
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	listQuery := baseQuery
	if query.CursorListingDate != nil && query.CursorID > 0 {
		listQuery = listQuery.Where("(listing_date < ?) OR (listing_date = ? AND id < ?)", *query.CursorListingDate, *query.CursorListingDate, query.CursorID)
	}

	items := make([]model.OzonProductCatalogItem, 0, query.PageSize+1)
	err := listQuery.
		Order("listing_date DESC NULLS LAST").
		Order("id DESC").
		Limit(query.PageSize + 1).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *OzonCatalogRepository) UpsertBatch(items []model.OzonProductCatalogItem) error {
	if len(items) == 0 {
		return nil
	}

	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "shop_id"},
			{Name: "ozon_product_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"offer_id",
			"sku",
			"name",
			"primary_image_url",
			"price",
			"old_price",
			"min_price",
			"marketing_price",
			"currency",
			"visibility",
			"status",
			"stock_total",
			"stock_fbo",
			"stock_fbs",
			"listing_date",
			"listing_date_source",
			"sync_token",
			"payload",
			"last_remote_synced_at",
			"updated_at",
		}),
	}).Create(&items).Error
}

func (r *OzonCatalogRepository) DeleteStaleBySyncToken(shopID uint, syncToken string) error {
	return r.db.
		Where("shop_id = ? AND (sync_token IS NULL OR sync_token <> ?)", shopID, syncToken).
		Delete(&model.OzonProductCatalogItem{}).Error
}

func (r *OzonCatalogRepository) FindExistingByProductIDs(shopID uint, productIDs []int64) (map[int64]model.OzonProductCatalogItem, error) {
	result := make(map[int64]model.OzonProductCatalogItem)
	if len(productIDs) == 0 {
		return result, nil
	}

	items := make([]model.OzonProductCatalogItem, 0, len(productIDs))
	if err := r.db.Where("shop_id = ? AND ozon_product_id IN ?", shopID, productIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.OzonProductID] = item
	}
	return result, nil
}

func (r *OzonCatalogRepository) GetLatestSyncedAt(shopID uint) (*time.Time, error) {
	var item model.OzonProductCatalogItem
	err := r.db.
		Where("shop_id = ? AND last_remote_synced_at IS NOT NULL", shopID).
		Order("last_remote_synced_at DESC").
		First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return item.LastRemoteSyncedAt, nil
}

func (r *OzonCatalogRepository) ListByListingDate(shopID uint, targetDate time.Time) ([]model.OzonProductCatalogItem, error) {
	start := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	end := start.Add(24 * time.Hour)

	items := make([]model.OzonProductCatalogItem, 0)
	err := r.db.Where("shop_id = ? AND listing_date >= ? AND listing_date < ?", shopID, start, end).
		Order("listing_date ASC, id ASC").
		Find(&items).Error
	return items, err
}
