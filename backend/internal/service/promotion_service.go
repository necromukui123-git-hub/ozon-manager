package service

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"
	"time"

	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
	"ozon-manager/pkg/ozon"
)

type PromotionService struct {
	productRepo       *repository.ProductRepository
	promotionRepo     *repository.PromotionRepository
	shopRepo          *repository.ShopRepository
	automationService *AutomationService
}

func NewPromotionService(
	productRepo *repository.ProductRepository,
	promotionRepo *repository.PromotionRepository,
	shopRepo *repository.ShopRepository,
	automationService ...*AutomationService,
) *PromotionService {
	var autoSvc *AutomationService
	if len(automationService) > 0 {
		autoSvc = automationService[0]
	}
	return &PromotionService{
		productRepo:       productRepo,
		promotionRepo:     promotionRepo,
		shopRepo:          shopRepo,
		automationService: autoSvc,
	}
}

// 功能1: BatchEnrollPromotions 批量报名促销活动
func (s *PromotionService) BatchEnrollPromotions(req *dto.BatchEnrollRequest) (*dto.BatchEnrollResponse, error) {
	shop, err := s.shopRepo.GetWithCredentials(req.ShopID)
	if err != nil {
		return nil, fmt.Errorf("shop not found: %w", err)
	}

	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	products, err := s.productRepo.FindEligible(req.ShopID, req.ExcludeLoss, req.ExcludePromoted)
	if err != nil {
		return nil, fmt.Errorf("failed to get eligible products: %w", err)
	}

	actions, err := s.promotionRepo.FindActivePromotionActions(req.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}
	if len(actions) == 0 {
		return nil, fmt.Errorf("no active actions found")
	}

	response := &dto.BatchEnrollResponse{
		Success: true,
		Details: make([]dto.EnrollDetail, 0),
	}

	for _, product := range products {
		detail := dto.EnrollDetail{
			ProductID: product.ID,
			SourceSKU: product.SourceSKU,
			Status:    "success",
		}

		hasSuccess := false
		for _, action := range actions {
			err := s.enrollProductToAction(client, action.ActionID, product, "custom")
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

func (s *PromotionService) enrollProductToAction(client *ozon.Client, actionID int64, product model.Product, promotionType string) error {
	items := []ozon.ActivateProductItem{
		{
			ProductID:   product.OzonProductID,
			ActionPrice: product.CurrentPrice,
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

	lossProducts, err := s.promotionRepo.FindLossProductsByIDs(req.LossProductIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get loss products: %w", err)
	}

	actions, err := s.promotionRepo.FindActivePromotionActions(req.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}

	response := &dto.ProcessLossResponse{
		Success: true,
		Steps:   dto.ProcessSteps{},
	}

	for _, lp := range lossProducts {
		product := lp.Product

		err := s.exitAllPromotions(client, req.ShopID, product)
		if err != nil {
			response.Steps.ExitPromotion.Failed++
		} else {
			response.Steps.ExitPromotion.Success++
			s.promotionRepo.UpdateLossProductStep(lp.ID, "promotion_exited", true)
		}

		priceStr := strconv.FormatFloat(lp.NewPrice, 'f', 2, 64)
		err = client.UpdateSinglePrice(product.OzonProductID, priceStr, "", "")
		if err != nil {
			response.Steps.PriceUpdate.Failed++
		} else {
			response.Steps.PriceUpdate.Success++
			s.productRepo.UpdatePrice(product.ID, lp.NewPrice)
			s.promotionRepo.UpdateLossProductStep(lp.ID, "price_updated", true)
		}

		if len(actions) > 0 {
			stepFailed := false
			for _, action := range actions {
				err := s.enrollProductToAction(client, action.ActionID, product, "custom")
				if err != nil {
					stepFailed = true
				}
			}

			if stepFailed {
				response.Steps.RejoinPromotions.Failed++
			} else {
				response.Steps.RejoinPromotions.Success++
				s.promotionRepo.UpdateLossProductStep(lp.ID, "promotion_rejoined", true)
			}
		}

		s.promotionRepo.UpdateLossProductProcessed(lp.ID)
		response.ProcessedCount++
	}

	return response, nil
}

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
		product, err := s.productRepo.FindBySourceSKU(req.ShopID, item.SourceSKU)
		if err != nil {
			continue
		}

		s.exitAllPromotions(client, req.ShopID, *product)

		priceStr := strconv.FormatFloat(item.NewPrice, 'f', 2, 64)
		client.UpdateSinglePrice(product.OzonProductID, priceStr, "", "")
		s.productRepo.UpdatePrice(product.ID, item.NewPrice)

		actions, _ := s.promotionRepo.FindActivePromotionActions(req.ShopID)
		for _, action := range actions {
			s.enrollProductToAction(client, action.ActionID, *product, "custom")
		}

		if len(actions) > 0 {
			s.productRepo.UpdatePromotedStatus(product.ID, true)
		}
	}

	return nil
}

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

		if err := s.promotionRepo.UpsertPromotionAction(pa); err != nil {
			return nil, fmt.Errorf("failed to upsert action %d: %w", action.ID, err)
		}
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

			err := s.enrollProductToAction(client, rejoinAction.ActionID, product, promotionType)
			if err != nil {
				response.Steps.RejoinPromotions.Failed++
			} else {
				response.Steps.RejoinPromotions.Success++
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
			s.enrollProductToAction(client, action.ActionID, *product, promotionType)
		}

		// 更新推广状态
		if len(reenrollActions) > 0 {
			s.productRepo.UpdatePromotedStatus(product.ID, true)
		}
	}

	return nil
}

// UpdateActionDisplayName 更新促销活动显示名称
func (s *PromotionService) UpdateActionDisplayName(shopID uint, id uint, displayName string) error {
	action, err := s.promotionRepo.FindPromotionActionByID(id)
	if err != nil {
		return fmt.Errorf("活动不存在")
	}
	if action.ShopID != shopID {
		return fmt.Errorf("无权修改此活动")
	}
	return s.promotionRepo.UpdatePromotionActionDisplayName(id, displayName)
}

// UpdateActionsSortOrder 批量更新促销活动排序
func (s *PromotionService) UpdateActionsSortOrder(shopID uint, sortOrders []dto.SortOrderItem) error {
	// 转换为 map
	sortOrderMap := make(map[uint]int)
	for _, item := range sortOrders {
		sortOrderMap[item.ID] = item.SortOrder
	}
	return s.promotionRepo.UpdatePromotionActionsSortOrder(shopID, sortOrderMap)
}

type shopActionSnapshot struct {
	SourceActionID     string     `json:"source_action_id"`
	Title              string     `json:"title"`
	ActionType         string     `json:"action_type"`
	ParticipatingCount int        `json:"participating_products_count"`
	PotentialCount     int        `json:"potential_products_count"`
	DateStart          *time.Time `json:"date_start"`
	DateEnd            *time.Time `json:"date_end"`
}

type shopActionProductsSnapshot struct {
	Items []shopActionProductSnapshotItem `json:"items"`
}

type shopActionProductSnapshotItem struct {
	SourceSKU     string  `json:"source_sku"`
	OzonProductID int64   `json:"ozon_product_id"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	ActionPrice   float64 `json:"action_price"`
	Stock         int     `json:"stock"`
	Status        string  `json:"status"`
}

func (s *PromotionService) SyncPromotionActionsV2(shopID uint, userID uint) (*dto.SyncActionsResult, error) {
	baseActions, err := s.SyncPromotionActions(shopID)
	if err != nil {
		return nil, err
	}

	result := &dto.SyncActionsResult{
		Actions: baseActions,
		SyncSummary: dto.SyncActionsSummary{
			OfficialCount: len(baseActions),
			ShopCount:     0,
		},
		ShopSyncPending: false,
		PartialErrors:   map[string]string{},
	}

	if s.automationService == nil {
		result.PartialErrors["shop"] = "automation service unavailable"
		return result, nil
	}

	job, err := s.automationService.CreateSyncShopActionsJob(userID, shopID)
	if err != nil {
		result.PartialErrors["shop"] = err.Error()
		return result, nil
	}

	waitedJob, waitErr := s.automationService.WaitForJobCompletion(job.ID, 25*time.Second)
	if waitErr != nil {
		result.ShopSyncPending = true
		result.PartialErrors["shop"] = "shop actions sync still running in background"
		return result, nil
	}

	if waitedJob.Status != model.AutomationJobStatusSuccess && waitedJob.Status != model.AutomationJobStatusPartialSuccess {
		result.PartialErrors["shop"] = "shop actions sync failed"
		return result, nil
	}

	artifact, err := s.automationService.GetLatestArtifact(waitedJob.ID, "shop_actions_snapshot")
	if err != nil {
		result.PartialErrors["shop"] = err.Error()
		return result, nil
	}

	shopActions, err := parseShopActionsArtifact(artifact.Meta)
	if err != nil {
		result.PartialErrors["shop"] = err.Error()
		return result, nil
	}
	if len(shopActions) == 0 {
		result.PartialErrors["shop"] = "shop actions snapshot is empty"
		return result, nil
	}

	now := time.Now()
	for _, action := range shopActions {
		payloadBytes, _ := json.Marshal(action)
		pa := &model.PromotionAction{
			ShopID:             shopID,
			ActionID:           hashToActionID(action.SourceActionID),
			Source:             "shop",
			SourceActionID:     action.SourceActionID,
			Title:              action.Title,
			ActionType:         action.ActionType,
			ParticipatingCount: action.ParticipatingCount,
			PotentialCount:     action.PotentialCount,
			Status:             "active",
			SourcePayload:      payloadBytes,
			LastSyncedAt:       &now,
			DateStart:          action.DateStart,
			DateEnd:            action.DateEnd,
		}
		if upsertErr := s.promotionRepo.UpsertPromotionAction(pa); upsertErr == nil {
			result.SyncSummary.ShopCount++
		}
	}

	allActions, listErr := s.promotionRepo.FindPromotionActionsByShopID(shopID)
	if listErr == nil {
		result.Actions = allActions
	}

	return result, nil
}

func (s *PromotionService) GetActionProducts(actionID uint, req *dto.ActionProductsRequest, userID uint) (*dto.ActionProductsResponse, error) {
	action, err := s.promotionRepo.FindPromotionActionByIDAndShop(actionID, req.ShopID)
	if err != nil {
		return nil, fmt.Errorf("action not found")
	}

	shouldRefresh := req.ForceRefresh
	if !shouldRefresh {
		if action.LastProductsSyncedAt == nil {
			shouldRefresh = true
		} else if time.Since(*action.LastProductsSyncedAt) > 10*time.Minute {
			shouldRefresh = true
		}
	}

	if shouldRefresh {
		if action.Source == "official" {
			if err := s.refreshOfficialActionProducts(action); err != nil {
				return nil, err
			}
		} else if action.Source == "shop" {
			if err := s.refreshShopActionProducts(action, userID); err != nil {
				return nil, err
			}
		}
	}

	items, total, err := s.promotionRepo.ListActionProducts(req.ShopID, actionID, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	respItems := make([]dto.ActionProductItem, 0, len(items))
	for _, item := range items {
		lastSynced := ""
		if item.LastSyncedAt != nil {
			lastSynced = item.LastSyncedAt.Format("2006-01-02 15:04:05")
		}
		respItems = append(respItems, dto.ActionProductItem{
			ID:            item.ID,
			OzonProductID: item.OzonProductID,
			SourceSKU:     item.SourceSKU,
			Name:          item.Name,
			Price:         item.Price,
			ActionPrice:   item.ActionPrice,
			Stock:         item.Stock,
			Status:        item.Status,
			LastSyncedAt:  lastSynced,
		})
	}

	return &dto.ActionProductsResponse{
		ActionID:       action.ID,
		Source:         action.Source,
		SourceActionID: action.SourceActionID,
		Total:          total,
		Page:           req.Page,
		PageSize:       req.PageSize,
		Items:          respItems,
	}, nil
}

func (s *PromotionService) refreshOfficialActionProducts(action *model.PromotionAction) error {
	shop, err := s.shopRepo.GetWithCredentials(action.ShopID)
	if err != nil {
		return err
	}
	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	const pageSize = 200
	offset := 0
	products := make([]model.PromotionActionProduct, 0)

	for {
		resp, getErr := client.GetActionProducts(action.ActionID, pageSize, offset)
		if getErr != nil {
			return getErr
		}
		if len(resp.Result.Products) == 0 {
			break
		}

		for _, item := range resp.Result.Products {
			sourceSKU := strconv.FormatInt(item.ProductID, 10)
			if localProduct, findErr := s.productRepo.FindByOzonProductID(action.ShopID, item.ProductID); findErr == nil {
				sourceSKU = localProduct.SourceSKU
			}
			payload, _ := json.Marshal(item)
			products = append(products, model.PromotionActionProduct{
				OzonProductID: item.ProductID,
				SourceSKU:     sourceSKU,
				Name:          sourceSKU,
				Price:         item.Price,
				ActionPrice:   item.ActionPrice,
				Stock:         item.Stock,
				Status:        "active",
				Payload:       payload,
			})
		}

		offset += len(resp.Result.Products)
		if len(resp.Result.Products) < pageSize {
			break
		}
	}

	return s.promotionRepo.ReplaceActionProducts(action, products)
}

func (s *PromotionService) refreshShopActionProducts(action *model.PromotionAction, userID uint) error {
	if s.automationService == nil {
		return fmt.Errorf("automation service unavailable")
	}
	job, err := s.automationService.CreateSyncActionProductsJob(userID, action.ShopID, action.ID, action.SourceActionID)
	if err != nil {
		return err
	}
	waitedJob, waitErr := s.automationService.WaitForJobCompletion(job.ID, 25*time.Second)
	if waitErr != nil {
		return fmt.Errorf("shop action products sync timeout")
	}
	if waitedJob.Status != model.AutomationJobStatusSuccess && waitedJob.Status != model.AutomationJobStatusPartialSuccess {
		return fmt.Errorf("shop action products sync failed")
	}

	artifact, err := s.automationService.GetLatestArtifact(waitedJob.ID, "action_products_snapshot")
	if err != nil {
		return err
	}

	snapshot := shopActionProductsSnapshot{}
	if err := json.Unmarshal(artifact.Meta, &snapshot); err != nil {
		return err
	}

	products := make([]model.PromotionActionProduct, 0, len(snapshot.Items))
	for _, item := range snapshot.Items {
		payloadBytes, _ := json.Marshal(item)
		products = append(products, model.PromotionActionProduct{
			OzonProductID: item.OzonProductID,
			SourceSKU:     item.SourceSKU,
			Name:          item.Name,
			Price:         item.Price,
			ActionPrice:   item.ActionPrice,
			Stock:         item.Stock,
			Status:        item.Status,
			Payload:       payloadBytes,
		})
	}

	return s.promotionRepo.ReplaceActionProducts(action, products)
}

func parseShopActionsArtifact(meta []byte) ([]shopActionSnapshot, error) {
	payload := map[string]json.RawMessage{}
	if err := json.Unmarshal(meta, &payload); err != nil {
		return nil, err
	}

	if raw, ok := payload["actions"]; ok {
		items := make([]shopActionSnapshot, 0)
		if err := json.Unmarshal(raw, &items); err == nil {
			return items, nil
		}
	}

	direct := make([]shopActionSnapshot, 0)
	if err := json.Unmarshal(meta, &direct); err == nil {
		return direct, nil
	}

	return nil, fmt.Errorf("invalid shop actions artifact")
}

func hashToActionID(sourceActionID string) int64 {
	if sourceActionID == "" {
		return 0
	}
	if parsed, err := strconv.ParseInt(sourceActionID, 10, 64); err == nil {
		return parsed
	}
	return int64(crc32.ChecksumIEEE([]byte(sourceActionID)))
}

// ========== 统一促销操作（方案 B）==========

func splitActionsBySource(actions []model.PromotionAction) ([]model.PromotionAction, []model.PromotionAction) {
	official := make([]model.PromotionAction, 0)
	shop := make([]model.PromotionAction, 0)
	for _, action := range actions {
		if action.Source == "shop" {
			shop = append(shop, action)
			continue
		}
		official = append(official, action)
	}
	return official, shop
}

func uniqueSKUs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, raw := range values {
		sku := strings.TrimSpace(raw)
		if sku == "" {
			continue
		}
		if _, ok := seen[sku]; ok {
			continue
		}
		seen[sku] = struct{}{}
		result = append(result, sku)
	}
	return result
}

func (s *PromotionService) collectEligibleSKUs(shopID uint, excludeLoss bool, excludePromoted bool) ([]string, error) {
	products, err := s.productRepo.FindEligible(shopID, excludeLoss, excludePromoted)
	if err != nil {
		return nil, fmt.Errorf("failed to get eligible products: %w", err)
	}
	skus := make([]string, 0, len(products))
	for _, product := range products {
		skus = append(skus, product.SourceSKU)
	}
	return uniqueSKUs(skus), nil
}

func (s *PromotionService) removeFromOfficialActions(shopID uint, officialActions []model.PromotionAction, sourceSKUs []string) (*dto.BatchEnrollResponse, error) {
	shop, err := s.shopRepo.GetWithCredentials(shopID)
	if err != nil {
		return nil, fmt.Errorf("shop not found: %w", err)
	}
	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	result := &dto.BatchEnrollResponse{
		Success: true,
		Details: make([]dto.EnrollDetail, 0, len(sourceSKUs)),
	}

	for _, sku := range sourceSKUs {
		product, findErr := s.productRepo.FindBySourceSKU(shopID, sku)
		if findErr != nil {
			result.FailedCount++
			result.Details = append(result.Details, dto.EnrollDetail{
				SourceSKU: sku,
				Status:    "failed",
				Error:     "商品未找到",
			})
			continue
		}

		hasError := false
		for _, action := range officialActions {
			_, deErr := client.DeactivateProducts(action.ActionID, []int64{product.OzonProductID})
			if deErr != nil {
				hasError = true
			}
		}

		if hasError {
			result.FailedCount++
			result.Details = append(result.Details, dto.EnrollDetail{
				ProductID: product.ID,
				SourceSKU: sku,
				Status:    "failed",
				Error:     "部分活动退出失败",
			})
			continue
		}

		result.EnrolledCount++
		result.Details = append(result.Details, dto.EnrollDetail{
			ProductID: product.ID,
			SourceSKU: sku,
			Status:    "success",
		})
	}

	if result.FailedCount > 0 {
		result.Success = false
	}

	return result, nil
}

func (s *PromotionService) CreateUnifiedShopActionsJob(userID, shopID uint, jobType string, shopActions []model.PromotionAction, skus []string) (*model.AutomationJob, error) {
	if s.automationService == nil {
		return nil, fmt.Errorf("automation service unavailable")
	}

	items := make([]model.AutomationJobItem, 0, len(skus))
	for _, sku := range uniqueSKUs(skus) {
		items = append(items, model.AutomationJobItem{
			SourceSKU:         sku,
			TargetPrice:       0.01,
			OverallStatus:     model.AutomationStepStatusPending,
			StepExitStatus:    model.AutomationStepStatusPending,
			StepRepriceStatus: model.AutomationStepStatusPending,
			StepReaddStatus:   model.AutomationStepStatusPending,
		})
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("没有可处理的 SKU")
	}

	job := &model.AutomationJob{
		ShopID:     shopID,
		CreatedBy:  userID,
		JobType:    jobType,
		Status:     model.AutomationJobStatusPending,
		RateLimit:  1,
		TotalItems: len(items),
	}
	if err := s.automationService.CreateJobWithItems(job, items); err != nil {
		return nil, err
	}

	actionPayload := make([]map[string]interface{}, 0, len(shopActions))
	for _, action := range shopActions {
		title := action.DisplayName
		if strings.TrimSpace(title) == "" {
			title = action.Title
		}
		actionPayload = append(actionPayload, map[string]interface{}{
			"action_db_id":     action.ID,
			"source_action_id": action.SourceActionID,
			"title":            title,
		})
	}
	operation := "declare"
	if jobType == model.AutomationJobTypePromoUnifiedRemove {
		operation = "remove"
	}
	meta := map[string]interface{}{
		"operation":    operation,
		"shop_actions": actionPayload,
	}
	if err := s.automationService.CreateArtifact(job.ID, "promo_unified_meta", meta); err != nil {
		return nil, err
	}

	return s.automationService.FindJobByIDAndShop(job.ID, shopID)
}

// UnifiedEnroll 统一报名入口：根据活动 source 自动路由
func (s *PromotionService) UnifiedEnroll(userID uint, req *dto.UnifiedEnrollRequest) (*dto.UnifiedOperationResponse, error) {
	actions, err := s.promotionRepo.FindPromotionActionsByIDs(req.ShopID, req.ActionIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}
	if len(actions) == 0 {
		return nil, fmt.Errorf("未找到有效的促销活动")
	}

	officialActions, shopActions := splitActionsBySource(actions)
	officialActionIDs := make([]int64, 0, len(officialActions))
	for _, action := range officialActions {
		officialActionIDs = append(officialActionIDs, action.ActionID)
	}

	var officialResult *dto.BatchEnrollResponse
	if len(officialActionIDs) > 0 {
		enrollReq := &dto.BatchEnrollV2Request{
			ShopID:          req.ShopID,
			ActionIDs:       officialActionIDs,
			ExcludeLoss:     req.ExcludeLoss,
			ExcludePromoted: req.ExcludePromoted,
		}
		officialResult, err = s.BatchEnrollToActions(enrollReq)
		if err != nil {
			return nil, err
		}
	}

	if len(shopActions) == 0 {
		if officialResult == nil {
			return nil, fmt.Errorf("未找到可操作的活动")
		}
		return &dto.UnifiedOperationResponse{
			Mode:    "sync",
			Results: officialResult,
			Message: fmt.Sprintf("同步完成：%d 成功，%d 失败", officialResult.EnrolledCount, officialResult.FailedCount),
		}, nil
	}

	skus := uniqueSKUs(req.SourceSKUs)
	if len(skus) == 0 {
		skus, err = s.collectEligibleSKUs(req.ShopID, req.ExcludeLoss, req.ExcludePromoted)
		if err != nil {
			return nil, err
		}
	}
	if len(skus) == 0 {
		return nil, fmt.Errorf("没有找到可报名的商品")
	}

	job, err := s.CreateUnifiedShopActionsJob(userID, req.ShopID, model.AutomationJobTypePromoUnifiedEnroll, shopActions, skus)
	if err != nil {
		return nil, fmt.Errorf("创建店铺促销申报任务失败: %w", err)
	}

	msg := fmt.Sprintf("已创建店铺促销申报任务 #%d，等待 Agent 执行", job.ID)
	if officialResult != nil {
		msg = fmt.Sprintf("官方活动已同步完成（%d 成功，%d 失败）；店铺活动任务 #%d 已创建",
			officialResult.EnrolledCount, officialResult.FailedCount, job.ID)
	}
	return &dto.UnifiedOperationResponse{
		Mode:    "async",
		Results: officialResult,
		JobID:   &job.ID,
		Message: msg,
	}, nil
}

// UnifiedRemove 统一退出入口：根据活动 source 自动路由
func (s *PromotionService) UnifiedRemove(userID uint, req *dto.UnifiedRemoveRequest) (*dto.UnifiedOperationResponse, error) {
	actions, err := s.promotionRepo.FindPromotionActionsByIDs(req.ShopID, req.ActionIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}
	if len(actions) == 0 {
		return nil, fmt.Errorf("未找到有效的促销活动")
	}

	sourceSKUs := uniqueSKUs(req.SourceSKUs)
	if len(sourceSKUs) == 0 {
		return nil, fmt.Errorf("请提供需要退出的 SKU")
	}

	officialActions, shopActions := splitActionsBySource(actions)
	var officialResult *dto.BatchEnrollResponse
	if len(officialActions) > 0 {
		officialResult, err = s.removeFromOfficialActions(req.ShopID, officialActions, sourceSKUs)
		if err != nil {
			return nil, err
		}
	}

	if len(shopActions) == 0 {
		if officialResult == nil {
			return nil, fmt.Errorf("未找到可操作的活动")
		}
		return &dto.UnifiedOperationResponse{
			Mode:    "sync",
			Results: officialResult,
			Message: fmt.Sprintf("同步完成：%d 成功，%d 失败", officialResult.EnrolledCount, officialResult.FailedCount),
		}, nil
	}

	job, err := s.CreateUnifiedShopActionsJob(userID, req.ShopID, model.AutomationJobTypePromoUnifiedRemove, shopActions, sourceSKUs)
	if err != nil {
		return nil, fmt.Errorf("创建店铺促销退出任务失败: %w", err)
	}

	msg := fmt.Sprintf("已创建店铺促销退出任务 #%d，等待 Agent 执行", job.ID)
	if officialResult != nil {
		msg = fmt.Sprintf("官方活动已同步完成（%d 成功，%d 失败）；店铺活动任务 #%d 已创建",
			officialResult.EnrolledCount, officialResult.FailedCount, job.ID)
	}
	return &dto.UnifiedOperationResponse{
		Mode:    "async",
		Results: officialResult,
		JobID:   &job.ID,
		Message: msg,
	}, nil
}

// UnifiedProcessLoss 统一亏损处理入口
func (s *PromotionService) UnifiedProcessLoss(userID uint, req *dto.UnifiedProcessLossRequest) (*dto.UnifiedProcessLossResponse, error) {
	if len(req.RejoinActionIDs) == 0 {
		result, err := s.ProcessLossProductsV2(&dto.ProcessLossV2Request{
			ShopID:         req.ShopID,
			LossProductIDs: req.LossProductIDs,
			RejoinActionID: nil,
		})
		if err != nil {
			return nil, err
		}
		return &dto.UnifiedProcessLossResponse{
			Mode:    "sync",
			Result:  result,
			Message: "同步处理完成",
		}, nil
	}

	actions, err := s.promotionRepo.FindPromotionActionsByIDs(req.ShopID, req.RejoinActionIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}
	if len(actions) == 0 {
		return nil, fmt.Errorf("未找到有效的促销活动")
	}
	officialActions, shopActions := splitActionsBySource(actions)

	if len(shopActions) > 0 {
		lossProducts, listErr := s.promotionRepo.FindLossProductsByIDs(req.LossProductIDs)
		if listErr != nil {
			return nil, fmt.Errorf("failed to get loss products: %w", listErr)
		}
		inputs := make([]dto.RepriceItem, 0, len(lossProducts))
		for _, lossProduct := range lossProducts {
			inputs = append(inputs, dto.RepriceItem{
				SourceSKU: lossProduct.Product.SourceSKU,
				NewPrice:  lossProduct.NewPrice,
			})
		}
		job, createErr := s.createRemoveRepriceReaddJob(userID, req.ShopID, inputs, map[string]interface{}{
			"reason":            "unified_process_loss",
			"rejoin_action_ids": req.RejoinActionIDs,
		})
		if createErr != nil {
			return nil, createErr
		}
		return &dto.UnifiedProcessLossResponse{
			Mode:    "async",
			JobID:   &job.ID,
			Message: fmt.Sprintf("已创建统一亏损处理任务 #%d，等待 Agent 执行", job.ID),
		}, nil
	}

	var rejoinActionID *int64
	if len(officialActions) > 0 {
		rejoinActionID = &officialActions[0].ActionID
	}
	result, err := s.ProcessLossProductsV2(&dto.ProcessLossV2Request{
		ShopID:         req.ShopID,
		LossProductIDs: req.LossProductIDs,
		RejoinActionID: rejoinActionID,
	})
	if err != nil {
		return nil, err
	}
	return &dto.UnifiedProcessLossResponse{
		Mode:    "sync",
		Result:  result,
		Message: "同步处理完成",
	}, nil
}

func (s *PromotionService) createRemoveRepriceReaddJob(userID, shopID uint, products []dto.RepriceItem, meta map[string]interface{}) (*model.AutomationJob, error) {
	if s.automationService == nil {
		return nil, fmt.Errorf("automation service unavailable")
	}
	if len(products) == 0 {
		return nil, fmt.Errorf("没有可处理的商品")
	}

	items := make([]model.AutomationJobItem, 0, len(products))
	for _, product := range products {
		if strings.TrimSpace(product.SourceSKU) == "" {
			continue
		}
		items = append(items, model.AutomationJobItem{
			SourceSKU:         strings.TrimSpace(product.SourceSKU),
			TargetPrice:       product.NewPrice,
			OverallStatus:     model.AutomationStepStatusPending,
			StepExitStatus:    model.AutomationStepStatusPending,
			StepRepriceStatus: model.AutomationStepStatusPending,
			StepReaddStatus:   model.AutomationStepStatusPending,
		})
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("没有可处理的商品")
	}

	job := &model.AutomationJob{
		ShopID:     shopID,
		CreatedBy:  userID,
		JobType:    model.AutomationJobTypeRemoveRepriceReadd,
		Status:     model.AutomationJobStatusPending,
		RateLimit:  1,
		TotalItems: len(items),
	}
	if err := s.automationService.CreateJobWithItems(job, items); err != nil {
		return nil, err
	}
	if len(meta) > 0 {
		_ = s.automationService.CreateArtifact(job.ID, "remove_reprice_readd_meta", meta)
	}
	return s.automationService.FindJobByIDAndShop(job.ID, shopID)
}

// UnifiedRepricePromote 统一改价推广入口
func (s *PromotionService) UnifiedRepricePromote(userID uint, req *dto.UnifiedRepricePromoteRequest) (*dto.UnifiedRepricePromoteResponse, error) {
	if len(req.ReenrollActionIDs) == 0 {
		err := s.RemoveRepricePromoteV2(&dto.RemoveRepricePromoteV2Request{
			ShopID:            req.ShopID,
			Products:          req.Products,
			ReenrollActionIDs: []int64{},
		})
		if err != nil {
			return nil, err
		}
		return &dto.UnifiedRepricePromoteResponse{
			Mode: "sync",
			Result: &dto.UnifiedRepricePromoteResult{
				Success:          true,
				RemoveCount:      len(req.Products),
				PriceUpdateCount: len(req.Products),
				PromoteCount:     0,
				FailedCount:      0,
			},
			Message: "同步处理完成",
		}, nil
	}

	actions, err := s.promotionRepo.FindPromotionActionsByIDs(req.ShopID, req.ReenrollActionIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}
	if len(actions) == 0 {
		return nil, fmt.Errorf("未找到有效的促销活动")
	}
	officialActions, shopActions := splitActionsBySource(actions)

	if len(shopActions) > 0 {
		job, createErr := s.createRemoveRepriceReaddJob(userID, req.ShopID, req.Products, map[string]interface{}{
			"reason":              "unified_reprice_promote",
			"reenroll_action_ids": req.ReenrollActionIDs,
		})
		if createErr != nil {
			return nil, createErr
		}
		return &dto.UnifiedRepricePromoteResponse{
			Mode:    "async",
			JobID:   &job.ID,
			Message: fmt.Sprintf("已创建统一改价推广任务 #%d，等待 Agent 执行", job.ID),
		}, nil
	}

	officialActionIDs := make([]int64, 0, len(officialActions))
	for _, action := range officialActions {
		officialActionIDs = append(officialActionIDs, action.ActionID)
	}
	if err := s.RemoveRepricePromoteV2(&dto.RemoveRepricePromoteV2Request{
		ShopID:            req.ShopID,
		Products:          req.Products,
		ReenrollActionIDs: officialActionIDs,
	}); err != nil {
		return nil, err
	}
	return &dto.UnifiedRepricePromoteResponse{
		Mode: "sync",
		Result: &dto.UnifiedRepricePromoteResult{
			Success:          true,
			RemoveCount:      len(req.Products),
			PriceUpdateCount: len(req.Products),
			PromoteCount:     len(req.Products),
			FailedCount:      0,
		},
		Message: "同步处理完成",
	}, nil
}

// CreateShopActionJob 创建店铺促销操作的 automation job
func (s *PromotionService) CreateShopActionJob(userID, shopID uint, jobType string, sourceActionID string, skus []string) (*model.AutomationJob, error) {
	if s.automationService == nil {
		return nil, fmt.Errorf("automation service unavailable")
	}

	job := &model.AutomationJob{
		ShopID:     shopID,
		CreatedBy:  userID,
		JobType:    jobType,
		Status:     model.AutomationJobStatusPending,
		RateLimit:  1,
		TotalItems: len(skus),
	}

	items := make([]model.AutomationJobItem, 0, len(skus))
	for _, sku := range skus {
		items = append(items, model.AutomationJobItem{
			SourceSKU:         sku,
			TargetPrice:       0.01, // placeholder
			OverallStatus:     model.AutomationStepStatusPending,
			StepExitStatus:    model.AutomationStepStatusPending,
			StepRepriceStatus: model.AutomationStepStatusPending,
			StepReaddStatus:   model.AutomationStepStatusPending,
		})
	}

	if err := s.automationService.CreateJobWithItems(job, items); err != nil {
		return nil, err
	}

	// 通过 artifact 存储 meta，Agent 轮询时会读取
	meta := map[string]interface{}{
		"source_action_id": sourceActionID,
	}
	if err := s.automationService.CreateArtifact(job.ID, "shop_action_meta", meta); err != nil {
		return nil, err
	}

	return s.automationService.FindJobByIDAndShop(job.ID, shopID)
}
