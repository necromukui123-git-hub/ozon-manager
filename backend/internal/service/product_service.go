package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
	"ozon-manager/pkg/ozon"
)

type ProductService struct {
	productRepo   *repository.ProductRepository
	shopRepo      *repository.ShopRepository
	promotionRepo *repository.PromotionRepository
}

func NewProductService(
	productRepo *repository.ProductRepository,
	shopRepo *repository.ShopRepository,
	promotionRepo *repository.PromotionRepository,
) *ProductService {
	return &ProductService{
		productRepo:   productRepo,
		shopRepo:      shopRepo,
		promotionRepo: promotionRepo,
	}
}

// GetProducts 获取商品列表
func (s *ProductService) GetProducts(req *dto.ProductListRequest) (*dto.ProductListResponse, error) {
	products, total, err := s.productRepo.FindWithFilters(
		req.ShopID,
		req.IsLoss,
		req.IsPromoted,
		req.Keyword,
		req.Page,
		req.PageSize,
	)
	if err != nil {
		return nil, err
	}

	items := make([]dto.ProductItem, 0, len(products))
	for _, product := range products {
		item := dto.ProductItem{
			ID:           product.ID,
			SourceSKU:    product.SourceSKU,
			Name:         product.Name,
			CurrentPrice: product.CurrentPrice,
			IsLoss:       product.IsLoss,
			IsPromoted:   product.IsPromoted,
			Promotions:   make([]dto.PromotionInfo, 0),
		}

		// 获取促销信息
		for _, pp := range product.PromotedProducts {
			if pp.Status == "active" {
				item.Promotions = append(item.Promotions, dto.PromotionInfo{
					ActionID: pp.ActionID,
					Type:     pp.PromotionType,
					Title:    getPromotionTitle(pp.PromotionType),
				})
			}
		}

		// 获取亏损信息
		if product.IsLoss {
			lossProducts, _ := s.promotionRepo.FindUnprocessedLossProducts(req.ShopID)
			for _, lp := range lossProducts {
				if lp.ProductID == product.ID {
					item.LossInfo = &dto.LossInfo{
						LossDate:      lp.LossDate.Format("2006-01-02"),
						OriginalPrice: lp.OriginalPrice,
						NewPrice:      lp.NewPrice,
					}
					break
				}
			}
		}

		items = append(items, item)
	}

	return &dto.ProductListResponse{
		Total: total,
		Items: items,
	}, nil
}

// SyncProducts 从Ozon同步商品
func (s *ProductService) SyncProducts(shopID uint) (int, error) {
	// 获取店铺凭证
	shop, err := s.shopRepo.GetWithCredentials(shopID)
	if err != nil {
		return 0, err
	}

	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	// 获取所有商品
	var allProducts []ozon.ProductListV3Item
	lastID := ""
	seenCursor := map[string]struct{}{}
	for {
		resp, err := client.GetProductListV3(1000, lastID, "ALL")
		if err != nil {
			return 0, fmt.Errorf("failed to get product list from ozon: %w", err)
		}

		allProducts = append(allProducts, resp.Result.Items...)

		nextCursor := strings.TrimSpace(resp.Result.LastID)
		if nextCursor == "" || len(resp.Result.Items) < 1000 {
			break
		}
		if _, exists := seenCursor[nextCursor]; exists {
			break
		}
		seenCursor[nextCursor] = struct{}{}
		lastID = nextCursor
	}

	// 先写入基础字段，保证详情批次失败时也不至于全空
	now := time.Now()
	syncedIDs := make(map[int64]struct{}, len(allProducts))
	syncErrors := make([]string, 0)
	for _, p := range allProducts {
		if p.ProductID <= 0 {
			continue
		}

		sourceSKU := strings.TrimSpace(p.OfferID)
		if sourceSKU == "" {
			sourceSKU = strconv.FormatInt(p.ProductID, 10)
		}

		product := &model.Product{
			ShopID:        shopID,
			OzonProductID: p.ProductID,
			SourceSKU:     sourceSKU,
			Status:        "active",
			LastSyncedAt:  &now,
		}

		if err := s.productRepo.Upsert(product); err != nil {
			syncErrors = append(syncErrors, fmt.Sprintf("base upsert product_id=%d failed: %v", p.ProductID, err))
			continue
		}
		syncedIDs[p.ProductID] = struct{}{}
	}

	// 批量获取商品详情并保存
	batchSize := 100
	for i := 0; i < len(allProducts); i += batchSize {
		end := i + batchSize
		if end > len(allProducts) {
			end = len(allProducts)
		}

		batch := allProducts[i:end]
		productIDs := make([]int64, len(batch))
		for j, p := range batch {
			productIDs[j] = p.ProductID
		}

		infoResp, err := client.GetProductInfoList(productIDs, nil)
		if err != nil {
			syncErrors = append(syncErrors, fmt.Sprintf("info batch [%d,%d) failed: %v", i, end, err))
			continue
		}

		for _, info := range infoResp.ItemsList() {
			price, _ := strconv.ParseFloat(info.Price, 64)
			ozonProductID := info.ProductID
			if ozonProductID <= 0 {
				ozonProductID = info.ID
			}
			if ozonProductID <= 0 {
				continue
			}
			sourceSKU := strings.TrimSpace(info.OfferID)
			if sourceSKU == "" {
				sourceSKU = strconv.FormatInt(ozonProductID, 10)
			}

			product := &model.Product{
				ShopID:        shopID,
				OzonProductID: ozonProductID,
				OzonSKU:       info.SKU,
				SourceSKU:     sourceSKU,
				Name:          strings.TrimSpace(info.Name),
				CurrentPrice:  price,
				Status:        "active",
				LastSyncedAt:  &now,
			}

			if err := s.productRepo.Upsert(product); err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("detail upsert product_id=%d failed: %v", ozonProductID, err))
				continue
			}
			syncedIDs[ozonProductID] = struct{}{}
		}
	}

	syncedCount := len(syncedIDs)
	if len(allProducts) > 0 && syncedCount == 0 {
		return 0, fmt.Errorf("sync returned %d remote products but saved none", len(allProducts))
	}
	if len(syncErrors) > 0 {
		return syncedCount, fmt.Errorf("sync completed with %d errors: %s", len(syncErrors), summarizeSyncErrors(syncErrors))
	}

	return syncedCount, nil
}

func summarizeSyncErrors(errors []string) string {
	if len(errors) == 0 {
		return ""
	}
	limit := 3
	if len(errors) < limit {
		limit = len(errors)
	}
	return strings.Join(errors[:limit], "; ")
}

// GetProductByID 获取商品详情
func (s *ProductService) GetProductByID(productID uint) (*model.Product, error) {
	return s.productRepo.FindByID(productID)
}

// GetProductBySourceSKU 根据SourceSKU获取商品
func (s *ProductService) GetProductBySourceSKU(shopID uint, sourceSKU string) (*model.Product, error) {
	return s.productRepo.FindBySourceSKU(shopID, sourceSKU)
}

// GetPromotableProducts 获取可推广的商品
func (s *ProductService) GetPromotableProducts(shopID uint) ([]model.Product, error) {
	return s.productRepo.FindPromotable(shopID)
}

// GetStats 获取统计数据
func (s *ProductService) GetStats(shopID uint) (*dto.StatsOverview, error) {
	total, _ := s.productRepo.CountByShopID(shopID)
	loss, _ := s.productRepo.CountLossByShopID(shopID)
	promoted, _ := s.productRepo.CountPromotedByShopID(shopID)

	promotable := total - loss - promoted

	return &dto.StatsOverview{
		TotalProducts:      total,
		LossProducts:       loss,
		PromotedProducts:   promoted,
		PromotableProducts: promotable,
	}, nil
}

func getPromotionTitle(promotionType string) string {
	if promotionType == "" || promotionType == "custom" {
		return "Promotion"
	}
	return promotionType
}
