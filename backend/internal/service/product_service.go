package service

import (
	"strconv"
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
	var allProducts []ozon.ProductListItem
	lastID := ""
	for {
		resp, err := client.GetProductList(1000, lastID)
		if err != nil {
			return 0, err
		}

		allProducts = append(allProducts, resp.Result.Items...)

		if resp.Result.LastID == "" || len(resp.Result.Items) < 1000 {
			break
		}
		lastID = resp.Result.LastID
	}

	// 批量获取商品详情并保存
	syncedCount := 0
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

		infoResp, err := client.GetProductInfo(productIDs)
		if err != nil {
			continue
		}

		for _, info := range infoResp.Result.Items {
			price, _ := strconv.ParseFloat(info.Price, 64)
			now := time.Now()

			product := &model.Product{
				ShopID:        shopID,
				OzonProductID: info.ProductID,
				OzonSKU:       info.SKU,
				SourceSKU:     info.OfferID,
				Name:          info.Name,
				CurrentPrice:  price,
				Status:        "active",
				LastSyncedAt:  &now,
			}

			if err := s.productRepo.Upsert(product); err == nil {
				syncedCount++
			}
		}
	}

	return syncedCount, nil
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
	elasticBoost, _ := s.promotionRepo.CountByPromotionType(shopID, "elastic_boost")
	discount28, _ := s.promotionRepo.CountByPromotionType(shopID, "discount_28")

	promotable := total - loss - promoted

	return &dto.StatsOverview{
		TotalProducts:      total,
		LossProducts:       loss,
		PromotedProducts:   promoted,
		PromotableProducts: promotable,
		ElasticBoostCount:  elasticBoost,
		Discount28Count:    discount28,
	}, nil
}

func getPromotionTitle(promotionType string) string {
	switch promotionType {
	case "elastic_boost":
		return "弹性提升"
	case "discount_28":
		return "折扣28%"
	default:
		return promotionType
	}
}
