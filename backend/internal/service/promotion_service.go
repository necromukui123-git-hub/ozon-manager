package service

import (
	"fmt"
	"strconv"
	"time"

	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
	"ozon-manager/pkg/ozon"
)

type PromotionService struct {
	productRepo   *repository.ProductRepository
	promotionRepo *repository.PromotionRepository
	shopRepo      *repository.ShopRepository
}

func NewPromotionService(
	productRepo *repository.ProductRepository,
	promotionRepo *repository.PromotionRepository,
	shopRepo *repository.ShopRepository,
) *PromotionService {
	return &PromotionService{
		productRepo:   productRepo,
		promotionRepo: promotionRepo,
		shopRepo:      shopRepo,
	}
}

// 功能1: BatchEnrollPromotions 批量报名促销活动
func (s *PromotionService) BatchEnrollPromotions(req *dto.BatchEnrollRequest) (*dto.BatchEnrollResponse, error) {
	// 获取店铺凭证
	shop, err := s.shopRepo.GetWithCredentials(req.ShopID)
	if err != nil {
		return nil, fmt.Errorf("shop not found: %w", err)
	}

	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	// 获取符合条件的商品
	products, err := s.productRepo.FindEligible(req.ShopID, req.ExcludeLoss, req.ExcludePromoted)
	if err != nil {
		return nil, fmt.Errorf("failed to get eligible products: %w", err)
	}

	// 获取促销活动
	var elasticBoostAction, discount28Action *model.PromotionAction
	if req.EnrollElasticBoost {
		elasticBoostAction, _ = s.promotionRepo.FindElasticBoostAction(req.ShopID)
	}
	if req.EnrollDiscount28 {
		discount28Action, _ = s.promotionRepo.FindDiscount28Action(req.ShopID)
	}

	response := &dto.BatchEnrollResponse{
		Success: true,
		Details: make([]dto.EnrollDetail, 0),
	}

	// 批量处理商品
	for _, product := range products {
		detail := dto.EnrollDetail{
			ProductID: product.ID,
			SourceSKU: product.SourceSKU,
			Status:    "success",
		}

		// 添加到弹性提升活动
		if req.EnrollElasticBoost && elasticBoostAction != nil {
			err := s.enrollProductToAction(client, elasticBoostAction.ActionID, product, "elastic_boost")
			if err != nil {
				detail.Status = "failed"
				detail.Error = err.Error()
				response.FailedCount++
			}
		}

		// 添加到28%折扣活动
		if req.EnrollDiscount28 && discount28Action != nil {
			err := s.enrollProductToAction(client, discount28Action.ActionID, product, "discount_28")
			if err != nil {
				detail.Status = "failed"
				detail.Error = err.Error()
				response.FailedCount++
			}
		}

		if detail.Status == "success" {
			response.EnrolledCount++
			// 更新商品推广状态
			s.productRepo.UpdatePromotedStatus(product.ID, true)
		}

		response.Details = append(response.Details, detail)
	}

	return response, nil
}

// enrollProductToAction 添加商品到促销活动
func (s *PromotionService) enrollProductToAction(client *ozon.Client, actionID int64, product model.Product, promotionType string) error {
	items := []ozon.ActivateProductItem{
		{
			ProductID:   product.OzonProductID,
			ActionPrice: product.CurrentPrice * 0.72, // 28%折扣后的价格
		},
	}

	_, err := client.ActivateProducts(actionID, items)
	if err != nil {
		return err
	}

	// 记录推广信息
	promotedProduct := &model.PromotedProduct{
		ProductID:     product.ID,
		PromotionType: promotionType,
		ActionID:      actionID,
		ActionPrice:   items[0].ActionPrice,
		Status:        "active",
	}
	s.promotionRepo.CreatePromotedProduct(promotedProduct)

	return nil
}

// 功能2: ProcessLossProducts 处理亏损商品
func (s *PromotionService) ProcessLossProducts(req *dto.ProcessLossRequest) (*dto.ProcessLossResponse, error) {
	shop, err := s.shopRepo.GetWithCredentials(req.ShopID)
	if err != nil {
		return nil, fmt.Errorf("shop not found: %w", err)
	}

	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	// 获取亏损商品记录
	lossProducts, err := s.promotionRepo.FindLossProductsByIDs(req.LossProductIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get loss products: %w", err)
	}

	response := &dto.ProcessLossResponse{
		Success: true,
		Steps: dto.ProcessSteps{},
	}

	for _, lp := range lossProducts {
		product := lp.Product

		// Step 1: 退出所有促销活动
		err := s.exitAllPromotions(client, req.ShopID, product)
		if err != nil {
			response.Steps.ExitPromotion.Failed++
		} else {
			response.Steps.ExitPromotion.Success++
			s.promotionRepo.UpdateLossProductStep(lp.ID, "promotion_exited", true)
		}

		// Step 2: 改价
		priceStr := strconv.FormatFloat(lp.NewPrice, 'f', 2, 64)
		err = client.UpdateSinglePrice(product.OzonProductID, priceStr, "", "")
		if err != nil {
			response.Steps.PriceUpdate.Failed++
		} else {
			response.Steps.PriceUpdate.Success++
			s.productRepo.UpdatePrice(product.ID, lp.NewPrice)
			s.promotionRepo.UpdateLossProductStep(lp.ID, "price_updated", true)
		}

		// Step 3: 重新报名28%折扣促销
		discount28Action, _ := s.promotionRepo.FindDiscount28Action(req.ShopID)
		if discount28Action != nil {
			err := s.enrollProductToAction(client, discount28Action.ActionID, product, "discount_28")
			if err != nil {
				response.Steps.RejoinDiscount28.Failed++
			} else {
				response.Steps.RejoinDiscount28.Success++
				s.promotionRepo.UpdateLossProductStep(lp.ID, "promotion_rejoined", true)
			}
		}

		// 标记处理完成
		s.promotionRepo.UpdateLossProductProcessed(lp.ID)
		response.ProcessedCount++
	}

	return response, nil
}

// exitAllPromotions 退出所有促销活动
func (s *PromotionService) exitAllPromotions(client *ozon.Client, shopID uint, product model.Product) error {
	// 获取商品参与的所有促销
	promotedProducts, err := s.promotionRepo.FindPromotedProductsByProductID(product.ID)
	if err != nil {
		return err
	}

	for _, pp := range promotedProducts {
		_, err := client.DeactivateProducts(pp.ActionID, []int64{product.OzonProductID})
		if err != nil {
			return err
		}
		s.promotionRepo.ExitPromotion(product.ID, pp.PromotionType)
	}

	// 更新商品推广状态
	s.productRepo.UpdatePromotedStatus(product.ID, false)

	return nil
}

// 功能4: RemoveRepricePromote 移除-改价-重新推广
func (s *PromotionService) RemoveRepricePromote(req *dto.RemoveRepricePromoteRequest) error {
	shop, err := s.shopRepo.GetWithCredentials(req.ShopID)
	if err != nil {
		return fmt.Errorf("shop not found: %w", err)
	}

	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	for _, item := range req.Products {
		// 查找商品
		product, err := s.productRepo.FindBySourceSKU(req.ShopID, item.SourceSKU)
		if err != nil {
			continue
		}

		// Step 1: 从所有促销活动中移除
		s.exitAllPromotions(client, req.ShopID, *product)

		// Step 2: 改价
		priceStr := strconv.FormatFloat(item.NewPrice, 'f', 2, 64)
		client.UpdateSinglePrice(product.OzonProductID, priceStr, "", "")
		s.productRepo.UpdatePrice(product.ID, item.NewPrice)

		// Step 3: 重新添加到所有促销活动
		elasticBoostAction, _ := s.promotionRepo.FindElasticBoostAction(req.ShopID)
		if elasticBoostAction != nil {
			s.enrollProductToAction(client, elasticBoostAction.ActionID, *product, "elastic_boost")
		}

		discount28Action, _ := s.promotionRepo.FindDiscount28Action(req.ShopID)
		if discount28Action != nil {
			s.enrollProductToAction(client, discount28Action.ActionID, *product, "discount_28")
		}

		// 更新推广状态
		s.productRepo.UpdatePromotedStatus(product.ID, true)
	}

	return nil
}

// SyncPromotionActions 同步促销活动并返回更新后的列表
func (s *PromotionService) SyncPromotionActions(shopID uint) ([]model.PromotionAction, error) {
	shop, err := s.shopRepo.GetWithCredentials(shopID)
	if err != nil {
		return nil, fmt.Errorf("shop not found: %w", err)
	}

	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	actionsResp, err := client.GetActions()
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}

	now := time.Now()
	for _, action := range actionsResp.Result {
		pa := &model.PromotionAction{
			ShopID:             shopID,
			ActionID:           action.ID,
			Title:              action.Title,
			ActionType:         action.ActionType,
			ParticipatingCount: action.ParticipatingProducts,
			PotentialCount:     action.PotentialProducts,
			Status:             "active",
			LastSyncedAt:       &now,
		}

		// 解析日期
		if action.DateStart != "" {
			if t, err := time.Parse(time.RFC3339, action.DateStart); err == nil {
				pa.DateStart = &t
			}
		}
		if action.DateEnd != "" {
			if t, err := time.Parse(time.RFC3339, action.DateEnd); err == nil {
				pa.DateEnd = &t
			}
		}

		// 检测是否是弹性提升活动
		if containsKeyword(action.Title, []string{"弹性", "elastic", "boost"}) {
			pa.IsElasticBoost = true
		}

		// 检测是否是28%折扣活动
		if containsKeyword(action.Title, []string{"28", "折扣"}) {
			pa.IsDiscount28 = true
		}

		s.promotionRepo.UpsertPromotionAction(pa)
	}

	// 返回更新后的活动列表
	return s.promotionRepo.FindPromotionActionsByShopID(shopID)
}

// ImportLossProducts 导入亏损商品
func (s *PromotionService) ImportLossProducts(shopID uint, items []struct {
	SourceSKU string
	NewPrice  float64
}) ([]uint, error) {
	var lossProductIDs []uint

	for _, item := range items {
		product, err := s.productRepo.FindBySourceSKU(shopID, item.SourceSKU)
		if err != nil {
			continue
		}

		// 标记商品为亏损
		s.productRepo.UpdateLossStatus(product.ID, true)

		// 创建亏损记录
		lp := &model.LossProduct{
			ProductID:     product.ID,
			LossDate:      time.Now(),
			OriginalPrice: product.CurrentPrice,
			NewPrice:      item.NewPrice,
		}

		if err := s.promotionRepo.CreateLossProduct(lp); err == nil {
			lossProductIDs = append(lossProductIDs, lp.ID)
		}
	}

	return lossProductIDs, nil
}

func containsKeyword(s string, keywords []string) bool {
	for _, kw := range keywords {
		if len(s) > 0 && len(kw) > 0 {
			for i := 0; i <= len(s)-len(kw); i++ {
				if s[i:i+len(kw)] == kw {
					return true
				}
			}
		}
	}
	return false
}

// GetPromotionActions 获取店铺的促销活动列表
func (s *PromotionService) GetPromotionActions(shopID uint) ([]model.PromotionAction, error) {
	return s.promotionRepo.FindPromotionActionsByShopID(shopID)
}

// CreateManualAction 手动创建促销活动
func (s *PromotionService) CreateManualAction(req *dto.CreateManualActionRequest) (*model.PromotionAction, error) {
	// 检查店铺是否存在
	_, err := s.shopRepo.GetWithCredentials(req.ShopID)
	if err != nil {
		return nil, fmt.Errorf("shop not found: %w", err)
	}

	// 检查是否已存在
	existing, _ := s.promotionRepo.FindPromotionActionByActionID(req.ShopID, req.ActionID)
	if existing != nil {
		return nil, fmt.Errorf("action %d already exists for this shop", req.ActionID)
	}

	now := time.Now()
	pa := &model.PromotionAction{
		ShopID:       req.ShopID,
		ActionID:     req.ActionID,
		Title:        req.Title,
		IsManual:     true,
		Status:       "active",
		LastSyncedAt: &now,
	}

	// 如果没有提供标题，尝试从API获取
	if pa.Title == "" {
		pa.Title = fmt.Sprintf("活动 #%d", req.ActionID)
	}

	if err := s.promotionRepo.CreatePromotionAction(pa); err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	return pa, nil
}

// DeletePromotionAction 删除促销活动
func (s *PromotionService) DeletePromotionAction(shopID uint, id uint) error {
	// 检查活动是否存在且属于该店铺
	action, err := s.promotionRepo.FindPromotionActionByID(id)
	if err != nil {
		return fmt.Errorf("action not found: %w", err)
	}

	if action.ShopID != shopID {
		return fmt.Errorf("action does not belong to this shop")
	}

	return s.promotionRepo.DeletePromotionAction(id)
}

// BatchEnrollToActions 批量报名到指定的促销活动
func (s *PromotionService) BatchEnrollToActions(req *dto.BatchEnrollV2Request) (*dto.BatchEnrollResponse, error) {
	// 获取店铺凭证
	shop, err := s.shopRepo.GetWithCredentials(req.ShopID)
	if err != nil {
		return nil, fmt.Errorf("shop not found: %w", err)
	}

	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	// 获取符合条件的商品
	products, err := s.productRepo.FindEligible(req.ShopID, req.ExcludeLoss, req.ExcludePromoted)
	if err != nil {
		return nil, fmt.Errorf("failed to get eligible products: %w", err)
	}

	// 获取指定的促销活动
	actions, err := s.promotionRepo.FindPromotionActionsByActionIDs(req.ShopID, req.ActionIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}

	if len(actions) == 0 {
		return nil, fmt.Errorf("no valid actions found")
	}

	response := &dto.BatchEnrollResponse{
		Success: true,
		Details: make([]dto.EnrollDetail, 0),
	}

	// 批量处理商品
	for _, product := range products {
		detail := dto.EnrollDetail{
			ProductID: product.ID,
			SourceSKU: product.SourceSKU,
			Status:    "success",
		}

		hasSuccess := false
		for _, action := range actions {
			// 确定促销类型
			promotionType := "custom"
			if action.IsElasticBoost {
				promotionType = "elastic_boost"
			} else if action.IsDiscount28 {
				promotionType = "discount_28"
			}

			err := s.enrollProductToAction(client, action.ActionID, product, promotionType)
			if err != nil {
				detail.Error = err.Error()
			} else {
				hasSuccess = true
			}
		}

		if hasSuccess {
			response.EnrolledCount++
			s.productRepo.UpdatePromotedStatus(product.ID, true)
		} else {
			detail.Status = "failed"
			response.FailedCount++
		}

		response.Details = append(response.Details, detail)
	}

	return response, nil
}

// ProcessLossProductsV2 处理亏损商品（支持选择重新报名活动）
func (s *PromotionService) ProcessLossProductsV2(req *dto.ProcessLossV2Request) (*dto.ProcessLossResponse, error) {
	shop, err := s.shopRepo.GetWithCredentials(req.ShopID)
	if err != nil {
		return nil, fmt.Errorf("shop not found: %w", err)
	}

	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	// 获取亏损商品记录
	lossProducts, err := s.promotionRepo.FindLossProductsByIDs(req.LossProductIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get loss products: %w", err)
	}

	// 获取重新报名的活动
	var rejoinAction *model.PromotionAction
	if req.RejoinActionID != nil {
		rejoinAction, _ = s.promotionRepo.FindPromotionActionByActionID(req.ShopID, *req.RejoinActionID)
	}

	response := &dto.ProcessLossResponse{
		Success: true,
		Steps:   dto.ProcessSteps{},
	}

	for _, lp := range lossProducts {
		product := lp.Product

		// Step 1: 退出所有促销活动
		err := s.exitAllPromotions(client, req.ShopID, product)
		if err != nil {
			response.Steps.ExitPromotion.Failed++
		} else {
			response.Steps.ExitPromotion.Success++
			s.promotionRepo.UpdateLossProductStep(lp.ID, "promotion_exited", true)
		}

		// Step 2: 改价
		priceStr := strconv.FormatFloat(lp.NewPrice, 'f', 2, 64)
		err = client.UpdateSinglePrice(product.OzonProductID, priceStr, "", "")
		if err != nil {
			response.Steps.PriceUpdate.Failed++
		} else {
			response.Steps.PriceUpdate.Success++
			s.productRepo.UpdatePrice(product.ID, lp.NewPrice)
			s.promotionRepo.UpdateLossProductStep(lp.ID, "price_updated", true)
		}

		// Step 3: 重新报名指定活动
		if rejoinAction != nil {
			promotionType := "custom"
			if rejoinAction.IsDiscount28 {
				promotionType = "discount_28"
			} else if rejoinAction.IsElasticBoost {
				promotionType = "elastic_boost"
			}

			err := s.enrollProductToAction(client, rejoinAction.ActionID, product, promotionType)
			if err != nil {
				response.Steps.RejoinDiscount28.Failed++
			} else {
				response.Steps.RejoinDiscount28.Success++
				s.promotionRepo.UpdateLossProductStep(lp.ID, "promotion_rejoined", true)
			}
		}

		// 标记处理完成
		s.promotionRepo.UpdateLossProductProcessed(lp.ID)
		response.ProcessedCount++
	}

	return response, nil
}

// RemoveRepricePromoteV2 移除-改价-重新推广（支持选择活动）
func (s *PromotionService) RemoveRepricePromoteV2(req *dto.RemoveRepricePromoteV2Request) error {
	shop, err := s.shopRepo.GetWithCredentials(req.ShopID)
	if err != nil {
		return fmt.Errorf("shop not found: %w", err)
	}

	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	// 获取要重新报名的活动
	var reenrollActions []model.PromotionAction
	if len(req.ReenrollActionIDs) > 0 {
		reenrollActions, _ = s.promotionRepo.FindPromotionActionsByActionIDs(req.ShopID, req.ReenrollActionIDs)
	}

	for _, item := range req.Products {
		// 查找商品
		product, err := s.productRepo.FindBySourceSKU(req.ShopID, item.SourceSKU)
		if err != nil {
			continue
		}

		// Step 1: 从所有促销活动中移除
		s.exitAllPromotions(client, req.ShopID, *product)

		// Step 2: 改价
		priceStr := strconv.FormatFloat(item.NewPrice, 'f', 2, 64)
		client.UpdateSinglePrice(product.OzonProductID, priceStr, "", "")
		s.productRepo.UpdatePrice(product.ID, item.NewPrice)

		// Step 3: 重新添加到指定的促销活动
		for _, action := range reenrollActions {
			promotionType := "custom"
			if action.IsElasticBoost {
				promotionType = "elastic_boost"
			} else if action.IsDiscount28 {
				promotionType = "discount_28"
			}
			s.enrollProductToAction(client, action.ActionID, *product, promotionType)
		}

		// 更新推广状态
		if len(reenrollActions) > 0 {
			s.productRepo.UpdatePromotedStatus(product.ID, true)
		}
	}

	return nil
}
