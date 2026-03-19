package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
	"ozon-manager/pkg/ozon"
)

const (
	searchCPORefreshTimeout  = 90 * time.Second
	searchCPOShopWaitTimeout = 5 * time.Minute
)

type SearchCPOService struct {
	repo              *repository.SearchCPORepository
	productRepo       *repository.ProductRepository
	promotionRepo     *repository.PromotionRepository
	shopRepo          *repository.ShopRepository
	ozonCatalogRepo   *repository.OzonCatalogRepository
	automationService *AutomationService
	promotionService  *PromotionService
}

type searchCPOProductsSnapshot struct {
	Items []searchCPOProductSnapshotItem `json:"items"`
}

type searchCPOProductSnapshotItem struct {
	SKU               string          `json:"sku"`
	SourceSKU         string          `json:"source_sku"`
	ImageURL          string          `json:"image_url"`
	Title             string          `json:"title"`
	CategoryName      string          `json:"category_name"`
	Price             float64         `json:"price"`
	IsInStock         bool            `json:"is_in_stock"`
	SearchPromoStatus string          `json:"search_promo_status"`
	CarrotsStatus     string          `json:"carrots_status"`
	IsFavorite        bool            `json:"is_favorite"`
	Orders            int64           `json:"orders"`
	Spent             float64         `json:"spent"`
	Clicks            int64           `json:"clicks"`
	CTRPercent        float64         `json:"ctr_percent"`
	StockTotal        int64           `json:"stock_total"`
	Payload           json.RawMessage `json:"payload"`
}

type searchCPORunActionSnapshot struct {
	OfficialActionIDs []uint `json:"official_action_ids"`
	ShopActionIDs     []uint `json:"shop_action_ids"`
}

type searchCPORunFilterSnapshot struct {
	SourceSKUs []string `json:"source_skus"`
}

type searchCPORunItemState struct {
	CacheItem       *model.SearchCPOProduct
	LocalProduct    *model.Product
	CatalogItem     *model.OzonProductCatalogItem
	OfficialResults []dto.SearchCPORunActionResult
	ShopResults     []dto.SearchCPORunActionResult
}

func NewSearchCPOService(
	repo *repository.SearchCPORepository,
	productRepo *repository.ProductRepository,
	promotionRepo *repository.PromotionRepository,
	shopRepo *repository.ShopRepository,
	ozonCatalogRepo *repository.OzonCatalogRepository,
	automationService *AutomationService,
	promotionService *PromotionService,
) *SearchCPOService {
	return &SearchCPOService{
		repo:              repo,
		productRepo:       productRepo,
		promotionRepo:     promotionRepo,
		shopRepo:          shopRepo,
		ozonCatalogRepo:   ozonCatalogRepo,
		automationService: automationService,
		promotionService:  promotionService,
	}
}

func (s *SearchCPOService) GetConfig(shopID uint) (*dto.SearchCPOConfigResponse, error) {
	config, err := s.repo.FindConfigByShopID(shopID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &dto.SearchCPOConfigResponse{
				ShopID:            shopID,
				OfficialActionIDs: []uint{},
				ShopActionIDs:     []uint{},
				AutoEnabled:       false,
				ScheduleTime:      searchCPODefaultScheduleTime,
				EnableStep:        true,
			}, nil
		}
		return nil, err
	}
	return toSearchCPOConfigDTO(config), nil
}

func (s *SearchCPOService) UpdateConfig(req *dto.SearchCPOConfigRequest) (*dto.SearchCPOConfigResponse, error) {
	officialIDs := uniqueUints(req.OfficialActionIDs)
	shopIDs := uniqueUints(req.ShopActionIDs)
	if len(officialIDs)+len(shopIDs) > 0 {
		if err := s.validateSelectedActions(req.ShopID, officialIDs, shopIDs); err != nil {
			return nil, err
		}
	}

	existing, err := s.repo.FindConfigByShopID(req.ShopID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	autoEnabled := false
	scheduleSeed := searchCPODefaultScheduleTime
	if existing != nil {
		autoEnabled = existing.AutoEnabled
		scheduleSeed = firstNonEmptyServiceTrimmed(existing.ScheduleTime, searchCPODefaultScheduleTime)
	}
	if req.AutoEnabled != nil {
		autoEnabled = *req.AutoEnabled
	}
	scheduleInput := strings.TrimSpace(req.ScheduleTime)
	if scheduleInput == "" {
		scheduleInput = scheduleSeed
	}
	scheduleTime, err := normalizeScheduleTime(scheduleInput)
	if err != nil {
		return nil, err
	}

	officialBytes, _ := json.Marshal(officialIDs)
	shopBytes, _ := json.Marshal(shopIDs)
	config := &model.SearchCPOConfig{
		ShopID:            req.ShopID,
		OfficialActionIDs: officialBytes,
		ShopActionIDs:     shopBytes,
		AutoEnabled:       autoEnabled,
		ScheduleTime:      scheduleTime,
		EnableStep:        true,
	}
	if err := s.repo.UpsertConfig(config); err != nil {
		return nil, err
	}
	return s.GetConfig(req.ShopID)
}

func (s *SearchCPOService) RefreshProducts(userID, shopID uint) (*dto.SearchCPORefreshResponse, error) {
	if s.automationService == nil {
		return nil, fmt.Errorf("automation service unavailable")
	}

	job, err := s.automationService.CreateSyncSearchCPOProductsJob(userID, shopID)
	if err != nil {
		return nil, err
	}

	waitedJob, waitErr := s.automationService.WaitForJobCompletion(job.ID, searchCPORefreshTimeout)
	if waitErr != nil {
		return nil, fmt.Errorf("刷新 CPO 商品失败: %w", waitErr)
	}
	if waitedJob.Status != model.AutomationJobStatusSuccess && waitedJob.Status != model.AutomationJobStatusPartialSuccess {
		return nil, fmt.Errorf("刷新 CPO 商品失败: %s", automationJobFailureMessage(waitedJob, "extension sync failed"))
	}

	artifact, err := s.automationService.GetLatestArtifact(waitedJob.ID, "search_cpo_products_snapshot")
	if err != nil {
		return nil, err
	}

	snapshot := searchCPOProductsSnapshot{}
	if err := json.Unmarshal(artifact.Meta, &snapshot); err != nil {
		return nil, err
	}

	products := make([]model.SearchCPOProduct, 0, len(snapshot.Items))
	for _, item := range snapshot.Items {
		sourceSKU := strings.TrimSpace(item.SourceSKU)
		if sourceSKU == "" {
			sourceSKU = strings.TrimSpace(item.SKU)
		}
		if sourceSKU == "" {
			continue
		}

		payload := item.Payload
		if len(payload) == 0 {
			payload, _ = json.Marshal(item)
		}

		products = append(products, model.SearchCPOProduct{
			SKU:               strings.TrimSpace(item.SKU),
			SourceSKU:         sourceSKU,
			ImageURL:          strings.TrimSpace(item.ImageURL),
			Title:             strings.TrimSpace(item.Title),
			CategoryName:      strings.TrimSpace(item.CategoryName),
			Price:             item.Price,
			IsInStock:         item.IsInStock,
			SearchPromoStatus: strings.TrimSpace(item.SearchPromoStatus),
			CarrotsStatus:     strings.TrimSpace(item.CarrotsStatus),
			IsFavorite:        item.IsFavorite,
			Orders:            item.Orders,
			Spent:             item.Spent,
			Clicks:            item.Clicks,
			CTRPercent:        item.CTRPercent,
			StockTotal:        item.StockTotal,
			Payload:           datatypes.JSON(payload),
		})
	}

	if err := s.repo.ReplaceProducts(shopID, products); err != nil {
		return nil, err
	}

	items, err := s.repo.ListProducts(shopID)
	if err != nil {
		return nil, err
	}

	lastSynced := latestSearchCPOSyncTime(items)
	return &dto.SearchCPORefreshResponse{
		Total:      len(items),
		LastSynced: formatOptionalTime(lastSynced),
	}, nil
}

func (s *SearchCPOService) ListProducts(shopID uint) (*dto.SearchCPOProductsResponse, error) {
	items, err := s.repo.ListProducts(shopID)
	if err != nil {
		return nil, err
	}

	respItems := make([]dto.SearchCPOProductItem, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, dto.SearchCPOProductItem{
			ID:                    item.ID,
			SKU:                   item.SKU,
			SourceSKU:             item.SourceSKU,
			ImageURL:              item.ImageURL,
			Title:                 item.Title,
			CategoryName:          item.CategoryName,
			Price:                 item.Price,
			IsInStock:             item.IsInStock,
			SearchPromoStatus:     item.SearchPromoStatus,
			CarrotsStatus:         item.CarrotsStatus,
			AvailabilityPromo:     item.AvailabilityPromo,
			RuleState:             item.RuleState,
			IsFavorite:            item.IsFavorite,
			Orders:                item.Orders,
			Spent:                 item.Spent,
			Clicks:                item.Clicks,
			CTRPercent:            item.CTRPercent,
			StockTotal:            item.StockTotal,
			AvailabilityCheckedAt: formatOptionalTime(item.AvailabilityCheckedAt),
			State2DetectedAt:      formatOptionalTime(item.State2DetectedAt),
			MorkovskJoinedAt:      formatOptionalTime(item.MorkovskJoinedAt),
			LastSyncedAt:          formatOptionalTime(item.LastSyncedAt),
		})
	}

	return &dto.SearchCPOProductsResponse{
		Total:      len(respItems),
		LastSynced: formatOptionalTime(latestSearchCPOSyncTime(items)),
		Items:      respItems,
	}, nil
}

func (s *SearchCPOService) StartRun(userID uint, req *dto.SearchCPORunRequest) (*dto.SearchCPORunSummaryResponse, error) {
	sourceSKUs := uniqueSourceSKUs(req.SourceSKUs)
	if len(sourceSKUs) == 0 {
		return nil, fmt.Errorf("请至少选择一个商品")
	}

	if activeRun, err := s.repo.FindActiveRunByShop(req.ShopID); err == nil && activeRun != nil {
		return nil, fmt.Errorf("已有 CPO 报名任务正在执行中")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	officialIDs := uniqueUints(req.OfficialActionIDs)
	shopIDs := uniqueUints(req.ShopActionIDs)
	if len(officialIDs)+len(shopIDs) == 0 {
		cfg, err := s.GetConfig(req.ShopID)
		if err != nil {
			return nil, err
		}
		officialIDs = uniqueUints(cfg.OfficialActionIDs)
		shopIDs = uniqueUints(cfg.ShopActionIDs)
	}
	if len(officialIDs)+len(shopIDs) == 0 {
		return nil, fmt.Errorf("请先保存默认活动，或在本次运行时选择活动")
	}

	actions, err := s.resolveActions(req.ShopID, officialIDs, shopIDs)
	if err != nil {
		return nil, err
	}
	officialActions, shopActions := splitActionsBySource(actions)

	filterSnapshot, _ := json.Marshal(searchCPORunFilterSnapshot{
		SourceSKUs: sourceSKUs,
	})
	actionSnapshot, _ := json.Marshal(searchCPORunActionSnapshot{
		OfficialActionIDs: officialIDs,
		ShopActionIDs:     shopIDs,
	})

	allItems, err := s.repo.ListProducts(req.ShopID)
	if err != nil {
		return nil, err
	}

	run := &model.SearchCPORun{
		ShopID:         req.ShopID,
		TriggeredBy:    &userID,
		Status:         model.SearchCPORunStatusPending,
		FilterSnapshot: filterSnapshot,
		ActionSnapshot: actionSnapshot,
		TotalFetched:   len(allItems),
		TotalSelected:  len(sourceSKUs),
		TotalProcessed: len(sourceSKUs),
	}
	if err := s.repo.CreateRun(run); err != nil {
		return nil, err
	}

	go s.executeRun(run.ID, req.ShopID, userID, sourceSKUs, officialActions, shopActions)
	return toSearchCPORunSummaryDTO(run), nil
}

func (s *SearchCPOService) executeRun(runID, shopID, userID uint, sourceSKUs []string, officialActions, shopActions []model.PromotionAction) {
	run, err := s.repo.FindRunByIDAndShop(runID, shopID)
	if err != nil {
		return
	}

	now := time.Now()
	run.Status = model.SearchCPORunStatusRunning
	run.StartedAt = &now
	run.ErrorMessage = ""
	_ = s.repo.UpdateRun(run)

	if execErr := s.runExecution(run, userID, sourceSKUs, officialActions, shopActions); execErr != nil {
		finishedAt := time.Now()
		run.Status = model.SearchCPORunStatusFailed
		run.ErrorMessage = execErr.Error()
		run.CompletedAt = &finishedAt
		_ = s.repo.UpdateRun(run)
	}
}

func (s *SearchCPOService) runExecution(run *model.SearchCPORun, userID uint, sourceSKUs []string, officialActions, shopActions []model.PromotionAction) error {
	states, err := s.buildRunStates(run.ShopID, sourceSKUs)
	if err != nil {
		return err
	}

	if err := s.executeOfficial(run.ShopID, sourceSKUs, states, officialActions); err != nil {
		return err
	}
	if err := s.executeShop(run.ShopID, userID, sourceSKUs, states, shopActions); err != nil {
		return err
	}

	runItems := make([]model.SearchCPORunItem, 0, len(sourceSKUs))
	successCount := 0
	failedCount := 0
	skippedCount := 0
	partialCount := 0
	for _, sku := range sourceSKUs {
		state := states[sku]
		if state == nil {
			continue
		}
		overallStatus, officialStatus, shopStatus := summarizeSearchCPORowStatus(state)
		switch overallStatus {
		case model.SearchCPOItemStatusFailed:
			failedCount++
		case model.SearchCPOItemStatusPartialSuccess:
			partialCount++
			successCount++
		case model.SearchCPOItemStatusSkipped:
			skippedCount++
		default:
			successCount++
		}

		officialBytes, _ := json.Marshal(state.OfficialResults)
		shopBytes, _ := json.Marshal(state.ShopResults)

		item := model.SearchCPORunItem{
			SourceSKU:       sku,
			OverallStatus:   overallStatus,
			OfficialStatus:  officialStatus,
			ShopStatus:      shopStatus,
			OfficialResults: officialBytes,
			ShopResults:     shopBytes,
		}
		if state.CacheItem != nil {
			item.ProductCacheID = &state.CacheItem.ID
			item.SKU = state.CacheItem.SKU
			item.Title = state.CacheItem.Title
			item.SearchPromoStatus = state.CacheItem.SearchPromoStatus
		}
		runItems = append(runItems, item)
	}

	if err := s.repo.ReplaceRunItems(run.ID, runItems); err != nil {
		return err
	}

	run.SuccessItems = successCount
	run.FailedItems = failedCount
	run.SkippedItems = skippedCount
	run.Status = summarizeSearchCPORunStatus(successCount, failedCount, skippedCount, partialCount)
	finishedAt := time.Now()
	run.CompletedAt = &finishedAt
	return s.repo.UpdateRun(run)
}

func (s *SearchCPOService) executeOfficial(shopID uint, sourceSKUs []string, states map[string]*searchCPORunItemState, actions []model.PromotionAction) error {
	if len(actions) == 0 {
		return nil
	}

	shop, err := s.shopRepo.GetWithCredentials(shopID)
	if err != nil {
		return err
	}
	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	for _, action := range actions {
		payload := make([]ozon.ActivateProductItem, 0, len(sourceSKUs))
		skuByProductID := make(map[int64]string)

		for _, sourceSKU := range sourceSKUs {
			state := states[sourceSKU]
			if state == nil {
				continue
			}
			result := dto.SearchCPORunActionResult{
				PromotionActionID: action.ID,
				ActionID:          action.ActionID,
				Title:             displayActionName(action),
				Source:            action.Source,
			}

			productID := resolveSearchCPOOfficialProductID(state)
			if productID <= 0 {
				result.Status = model.SearchCPOItemStatusFailed
				result.Error = "未匹配到 Ozon 商品ID"
				state.OfficialResults = append(state.OfficialResults, result)
				continue
			}

			actionPrice := resolveSearchCPOActionPrice(state)
			if actionPrice <= 0 {
				result.Status = model.SearchCPOItemStatusFailed
				result.Error = "未找到合法的活动价"
				state.OfficialResults = append(state.OfficialResults, result)
				continue
			}

			result.Status = model.SearchCPOItemStatusPending
			result.ActionPrice = actionPrice
			state.OfficialResults = append(state.OfficialResults, result)

			payload = append(payload, ozon.ActivateProductItem{
				ProductID:   productID,
				ActionPrice: actionPrice,
			})
			skuByProductID[productID] = sourceSKU
		}

		if len(payload) == 0 {
			continue
		}

		resp, err := client.ActivateProducts(action.ActionID, payload)
		if err != nil {
			for _, reqItem := range payload {
				sourceSKU := skuByProductID[reqItem.ProductID]
				if sourceSKU == "" {
					continue
				}
				if result := findSearchCPORunActionResult(states[sourceSKU].OfficialResults, action.ID); result != nil {
					result.Status = model.SearchCPOItemStatusFailed
					result.Error = err.Error()
				}
			}
			continue
		}

		successProductIDs := make(map[int64]struct{}, len(resp.Result.ProductIDs))
		for _, productID := range resp.Result.ProductIDs {
			successProductIDs[productID] = struct{}{}
		}
		rejectedByProductID := make(map[int64]string, len(resp.Result.Rejected))
		for _, rejected := range resp.Result.Rejected {
			rejectedByProductID[rejected.ProductID] = strings.TrimSpace(rejected.Reason)
		}

		for _, reqItem := range payload {
			sourceSKU := skuByProductID[reqItem.ProductID]
			if sourceSKU == "" {
				continue
			}
			result := findSearchCPORunActionResult(states[sourceSKU].OfficialResults, action.ID)
			if result == nil {
				continue
			}
			if reason, exists := rejectedByProductID[reqItem.ProductID]; exists {
				result.Status = model.SearchCPOItemStatusFailed
				result.Error = firstNonEmptyServiceTrimmed(reason, "官方活动报名失败")
				continue
			}
			if _, exists := successProductIDs[reqItem.ProductID]; exists {
				result.Status = model.SearchCPOItemStatusSuccess
				result.Error = ""
				continue
			}
			result.Status = model.SearchCPOItemStatusFailed
			result.Error = "官方活动返回未知结果"
		}
	}

	return nil
}

func (s *SearchCPOService) executeShop(shopID, userID uint, sourceSKUs []string, states map[string]*searchCPORunItemState, actions []model.PromotionAction) error {
	if len(actions) == 0 {
		return nil
	}

	eligibleSKUs := make([]string, 0, len(sourceSKUs))
	for _, sourceSKU := range sourceSKUs {
		state := states[sourceSKU]
		if state == nil {
			continue
		}
		eligibleSKUs = append(eligibleSKUs, sourceSKU)
	}

	if len(eligibleSKUs) == 0 {
		return nil
	}

	job, err := s.promotionService.CreateUnifiedShopActionsJob(userID, shopID, model.AutomationJobTypePromoUnifiedEnroll, actions, eligibleSKUs)
	if err != nil {
		s.markShopFailed(states, eligibleSKUs, actions, err.Error())
		return nil
	}

	waitedJob, waitErr := s.automationService.WaitForJobCompletion(job.ID, searchCPOShopWaitTimeout)
	if waitErr != nil {
		s.markShopFailed(states, eligibleSKUs, actions, fmt.Sprintf("店铺活动执行超时: %v", waitErr))
		return nil
	}

	if waitedJob.Status != model.AutomationJobStatusSuccess && waitedJob.Status != model.AutomationJobStatusPartialSuccess {
		s.markShopFailed(states, eligibleSKUs, actions, automationJobFailureMessage(waitedJob, "店铺活动执行失败"))
		return nil
	}

	itemBySKU := make(map[string]model.AutomationJobItem, len(waitedJob.Items))
	for _, item := range waitedJob.Items {
		itemBySKU[item.SourceSKU] = item
	}

	for _, sourceSKU := range eligibleSKUs {
		state := states[sourceSKU]
		if state == nil {
			continue
		}
		item, exists := itemBySKU[sourceSKU]
		itemSuccess := exists && (item.OverallStatus == model.AutomationStepStatusSuccess || item.OverallStatus == model.AutomationStepStatusSkipped)
		itemError := ""
		if exists {
			itemError = firstNonEmptyServiceTrimmed(item.StepExitError, item.StepRepriceError, item.StepReaddError)
		}
		if !exists {
			itemError = "店铺活动执行结果缺失"
		}

		for _, action := range actions {
			result := dto.SearchCPORunActionResult{
				PromotionActionID: action.ID,
				SourceActionID:    action.SourceActionID,
				Title:             displayActionName(action),
				Source:            action.Source,
				Status:            model.SearchCPOItemStatusSuccess,
			}
			if !itemSuccess {
				result.Status = model.SearchCPOItemStatusFailed
				result.Error = firstNonEmptyServiceTrimmed(itemError, "店铺活动执行失败")
			}
			state.ShopResults = append(state.ShopResults, result)
		}
	}

	return nil
}

func (s *SearchCPOService) markShopFailed(states map[string]*searchCPORunItemState, sourceSKUs []string, actions []model.PromotionAction, message string) {
	for _, sourceSKU := range sourceSKUs {
		state := states[sourceSKU]
		if state == nil {
			continue
		}
		for _, action := range actions {
			state.ShopResults = append(state.ShopResults, dto.SearchCPORunActionResult{
				PromotionActionID: action.ID,
				SourceActionID:    action.SourceActionID,
				Title:             displayActionName(action),
				Source:            action.Source,
				Status:            model.SearchCPOItemStatusFailed,
				Error:             firstNonEmptyServiceTrimmed(message, "店铺活动执行失败"),
			})
		}
	}
}

func (s *SearchCPOService) buildRunStates(shopID uint, sourceSKUs []string) (map[string]*searchCPORunItemState, error) {
	cachedItems, err := s.repo.FindProductsBySourceSKUs(shopID, sourceSKUs)
	if err != nil {
		return nil, err
	}
	cacheBySKU := make(map[string]*model.SearchCPOProduct, len(cachedItems))
	for i := range cachedItems {
		item := cachedItems[i]
		cacheBySKU[item.SourceSKU] = &item
	}

	catalogByOfferID := make(map[string]model.OzonProductCatalogItem)
	catalogBySKU := make(map[int64]model.OzonProductCatalogItem)
	if s.ozonCatalogRepo != nil {
		catalogByOfferID, err = s.ozonCatalogRepo.FindByOfferIDs(shopID, sourceSKUs)
		if err != nil {
			return nil, err
		}
		numericSKUs := collectSearchCPONumericSKUs(sourceSKUs, cacheBySKU)
		catalogBySKU, err = s.ozonCatalogRepo.FindBySKUs(shopID, numericSKUs)
		if err != nil {
			return nil, err
		}
	}

	localProductsBySKU, err := s.productRepo.FindBySourceSKUs(shopID, sourceSKUs)
	if err != nil {
		return nil, err
	}

	catalogBySourceSKU := make(map[string]*model.OzonProductCatalogItem, len(sourceSKUs))
	catalogProductIDs := make([]int64, 0, len(sourceSKUs))
	for _, sourceSKU := range sourceSKUs {
		cacheItem := cacheBySKU[sourceSKU]
		catalogItem := resolveSearchCPOCatalogItem(sourceSKU, cacheItem, catalogByOfferID, catalogBySKU)
		if catalogItem == nil {
			continue
		}
		catalogBySourceSKU[sourceSKU] = catalogItem
		if catalogItem.OzonProductID > 0 {
			catalogProductIDs = append(catalogProductIDs, catalogItem.OzonProductID)
		}
	}

	localProductsByProductID, err := s.productRepo.FindByOzonProductIDs(shopID, uniqueInt64s(catalogProductIDs))
	if err != nil {
		return nil, err
	}

	states := make(map[string]*searchCPORunItemState, len(sourceSKUs))
	for _, sourceSKU := range sourceSKUs {
		sku := strings.TrimSpace(sourceSKU)
		if sku == "" {
			continue
		}
		state := &searchCPORunItemState{}
		if cacheItem := cacheBySKU[sku]; cacheItem != nil {
			cacheCopy := *cacheItem
			state.CacheItem = &cacheCopy
		}
		if product, ok := localProductsBySKU[sku]; ok {
			productCopy := product
			state.LocalProduct = &productCopy
		}
		if catalogItem := catalogBySourceSKU[sku]; catalogItem != nil {
			catalogCopy := *catalogItem
			state.CatalogItem = &catalogCopy
			if state.LocalProduct == nil && catalogCopy.OzonProductID > 0 {
				if product, ok := localProductsByProductID[catalogCopy.OzonProductID]; ok {
					productCopy := product
					state.LocalProduct = &productCopy
				}
			}
		}
		states[sku] = state
	}

	return states, nil
}

func (s *SearchCPOService) ListRuns(req *dto.SearchCPORunListRequest) (*dto.SearchCPORunListResponse, error) {
	runs, total, err := s.repo.ListRunsByShop(req.ShopID, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	items := make([]dto.SearchCPORunSummaryResponse, 0, len(runs))
	for _, run := range runs {
		runCopy := run
		items = append(items, *toSearchCPORunSummaryDTO(&runCopy))
	}
	return &dto.SearchCPORunListResponse{
		Total: total,
		Items: items,
	}, nil
}

func (s *SearchCPOService) GetRunDetail(shopID, runID uint) (*dto.SearchCPORunDetailResponse, error) {
	run, err := s.repo.FindRunByIDAndShop(runID, shopID)
	if err != nil {
		return nil, err
	}

	filter := decodeSearchCPORunFilterSnapshot(run.FilterSnapshot)
	action := decodeSearchCPORunActionSnapshot(run.ActionSnapshot)

	items := make([]dto.SearchCPORunItemResponse, 0, len(run.RunItems))
	for _, item := range run.RunItems {
		items = append(items, dto.SearchCPORunItemResponse{
			ID:                item.ID,
			SourceSKU:         item.SourceSKU,
			SKU:               item.SKU,
			Title:             item.Title,
			SearchPromoStatus: item.SearchPromoStatus,
			OverallStatus:     item.OverallStatus,
			OfficialStatus:    item.OfficialStatus,
			ShopStatus:        item.ShopStatus,
			OfficialResults:   decodeSearchCPORunActionResults(item.OfficialResults),
			ShopResults:       decodeSearchCPORunActionResults(item.ShopResults),
		})
	}

	return &dto.SearchCPORunDetailResponse{
		SearchCPORunSummaryResponse: *toSearchCPORunSummaryDTO(run),
		ShopID:                      run.ShopID,
		TriggeredBy:                 run.TriggeredBy,
		OfficialActionIDs:           action.OfficialActionIDs,
		ShopActionIDs:               action.ShopActionIDs,
		SourceSKUs:                  filter.SourceSKUs,
		Items:                       items,
	}, nil
}

func (s *SearchCPOService) validateSelectedActions(shopID uint, officialIDs []uint, shopIDs []uint) error {
	_, err := s.resolveActions(shopID, officialIDs, shopIDs)
	return err
}

func (s *SearchCPOService) resolveActions(shopID uint, officialIDs []uint, shopIDs []uint) ([]model.PromotionAction, error) {
	selectedIDs := append([]uint{}, officialIDs...)
	selectedIDs = append(selectedIDs, shopIDs...)
	selectedIDs = uniqueUints(selectedIDs)
	if len(selectedIDs) == 0 {
		return nil, fmt.Errorf("未选择任何活动")
	}

	actions, err := s.promotionRepo.FindPromotionActionsByIDs(shopID, selectedIDs)
	if err != nil {
		return nil, err
	}
	if len(actions) != len(selectedIDs) {
		return nil, fmt.Errorf("存在无效或无权限的促销活动")
	}

	actionsByID := make(map[uint]model.PromotionAction, len(actions))
	for _, action := range actions {
		actionsByID[action.ID] = action
	}
	for _, id := range officialIDs {
		action, ok := actionsByID[id]
		if !ok || action.Source != "official" {
			return nil, fmt.Errorf("官方活动选择无效")
		}
	}
	for _, id := range shopIDs {
		action, ok := actionsByID[id]
		if !ok || action.Source != "shop" {
			return nil, fmt.Errorf("店铺活动选择无效")
		}
	}
	return actions, nil
}

func toSearchCPOConfigDTO(config *model.SearchCPOConfig) *dto.SearchCPOConfigResponse {
	if config == nil {
		return nil
	}
	return &dto.SearchCPOConfigResponse{
		ID:                config.ID,
		ShopID:            config.ShopID,
		OfficialActionIDs: decodeUintSlice(config.OfficialActionIDs),
		ShopActionIDs:     decodeUintSlice(config.ShopActionIDs),
		AutoEnabled:       config.AutoEnabled,
		ScheduleTime:      firstNonEmptyServiceTrimmed(config.ScheduleTime, searchCPODefaultScheduleTime),
		EnableStep:        true,
		UpdatedAt:         config.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func toSearchCPORunSummaryDTO(run *model.SearchCPORun) *dto.SearchCPORunSummaryResponse {
	if run == nil {
		return nil
	}
	return &dto.SearchCPORunSummaryResponse{
		ID:             run.ID,
		Status:         run.Status,
		TotalFetched:   run.TotalFetched,
		TotalSelected:  run.TotalSelected,
		TotalProcessed: run.TotalProcessed,
		SuccessItems:   run.SuccessItems,
		FailedItems:    run.FailedItems,
		SkippedItems:   run.SkippedItems,
		ErrorMessage:   strings.TrimSpace(run.ErrorMessage),
		StartedAt:      formatOptionalTime(run.StartedAt),
		CompletedAt:    formatOptionalTime(run.CompletedAt),
		CreatedAt:      run.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func decodeUintSlice(raw []byte) []uint {
	items := make([]uint, 0)
	if len(raw) == 0 {
		return items
	}
	_ = json.Unmarshal(raw, &items)
	return uniqueUints(items)
}

func decodeSearchCPORunFilterSnapshot(raw []byte) searchCPORunFilterSnapshot {
	out := searchCPORunFilterSnapshot{SourceSKUs: []string{}}
	_ = json.Unmarshal(raw, &out)
	out.SourceSKUs = uniqueSourceSKUs(out.SourceSKUs)
	return out
}

func decodeSearchCPORunActionSnapshot(raw []byte) searchCPORunActionSnapshot {
	out := searchCPORunActionSnapshot{
		OfficialActionIDs: []uint{},
		ShopActionIDs:     []uint{},
	}
	_ = json.Unmarshal(raw, &out)
	out.OfficialActionIDs = uniqueUints(out.OfficialActionIDs)
	out.ShopActionIDs = uniqueUints(out.ShopActionIDs)
	return out
}

func decodeSearchCPORunActionResults(raw []byte) []dto.SearchCPORunActionResult {
	out := make([]dto.SearchCPORunActionResult, 0)
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

func latestSearchCPOSyncTime(items []model.SearchCPOProduct) *time.Time {
	var latest *time.Time
	for _, item := range items {
		if item.LastSyncedAt == nil {
			continue
		}
		if latest == nil || item.LastSyncedAt.After(*latest) {
			t := *item.LastSyncedAt
			latest = &t
		}
	}
	return latest
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}

func uniqueSourceSKUs(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	output := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		output = append(output, trimmed)
	}
	return output
}

func collectSearchCPONumericSKUs(sourceSKUs []string, cacheBySKU map[string]*model.SearchCPOProduct) []int64 {
	values := make([]int64, 0, len(sourceSKUs)*2)
	for _, sourceSKU := range sourceSKUs {
		if parsed := parseInt64WithFallback(sourceSKU, 0); parsed > 0 {
			values = append(values, parsed)
		}
		if cacheItem := cacheBySKU[sourceSKU]; cacheItem != nil {
			if parsed := parseInt64WithFallback(cacheItem.SKU, 0); parsed > 0 {
				values = append(values, parsed)
			}
		}
	}
	return uniqueInt64s(values)
}

func resolveSearchCPOCatalogItem(
	sourceSKU string,
	cacheItem *model.SearchCPOProduct,
	catalogByOfferID map[string]model.OzonProductCatalogItem,
	catalogBySKU map[int64]model.OzonProductCatalogItem,
) *model.OzonProductCatalogItem {
	sourceSKU = strings.TrimSpace(sourceSKU)
	if sourceSKU == "" {
		return nil
	}
	if item, exists := catalogByOfferID[sourceSKU]; exists {
		itemCopy := item
		return &itemCopy
	}

	candidates := []string{sourceSKU}
	if cacheItem != nil {
		candidates = append([]string{strings.TrimSpace(cacheItem.SKU)}, candidates...)
	}
	for _, candidate := range candidates {
		parsed := parseInt64WithFallback(candidate, 0)
		if parsed <= 0 {
			continue
		}
		if item, exists := catalogBySKU[parsed]; exists {
			itemCopy := item
			return &itemCopy
		}
	}
	return nil
}

func resolveSearchCPOOfficialProductID(state *searchCPORunItemState) int64 {
	if state == nil {
		return 0
	}
	if state.LocalProduct != nil && state.LocalProduct.OzonProductID > 0 {
		return state.LocalProduct.OzonProductID
	}
	if state.CatalogItem != nil && state.CatalogItem.OzonProductID > 0 {
		return state.CatalogItem.OzonProductID
	}
	return 0
}

func resolveSearchCPOActionPrice(state *searchCPORunItemState) float64 {
	if state == nil {
		return 0
	}
	if state.LocalProduct != nil && state.LocalProduct.CurrentPrice > 0 {
		return state.LocalProduct.CurrentPrice
	}
	if state.CacheItem != nil && state.CacheItem.Price > 0 {
		return state.CacheItem.Price
	}
	if state.CatalogItem != nil && state.CatalogItem.Price > 0 {
		return state.CatalogItem.Price
	}
	return 0
}

func findSearchCPORunActionResult(results []dto.SearchCPORunActionResult, promotionActionID uint) *dto.SearchCPORunActionResult {
	for i := range results {
		if results[i].PromotionActionID == promotionActionID {
			return &results[i]
		}
	}
	return nil
}

func hasFailedResult(results []dto.SearchCPORunActionResult) bool {
	for _, result := range results {
		if result.Status == model.SearchCPOItemStatusFailed {
			return true
		}
	}
	return false
}

func summarizeSearchCPORowStatus(state *searchCPORunItemState) (string, string, string) {
	officialStatus := summarizeActionResultsStatus(state.OfficialResults)
	shopStatus := summarizeActionResultsStatus(state.ShopResults)
	if officialStatus == model.SearchCPOItemStatusSkipped && shopStatus == model.SearchCPOItemStatusSkipped {
		return model.SearchCPOItemStatusSkipped, officialStatus, shopStatus
	}
	if (officialStatus == model.SearchCPOItemStatusFailed && shopStatus == model.SearchCPOItemStatusSuccess) ||
		(officialStatus == model.SearchCPOItemStatusSuccess && shopStatus == model.SearchCPOItemStatusFailed) {
		return model.SearchCPOItemStatusPartialSuccess, officialStatus, shopStatus
	}
	if officialStatus == model.SearchCPOItemStatusFailed || shopStatus == model.SearchCPOItemStatusFailed {
		return model.SearchCPOItemStatusFailed, officialStatus, shopStatus
	}
	return model.SearchCPOItemStatusSuccess, officialStatus, shopStatus
}

func summarizeActionResultsStatus(results []dto.SearchCPORunActionResult) string {
	if len(results) == 0 {
		return model.SearchCPOItemStatusSkipped
	}
	failed := false
	success := false
	for _, result := range results {
		switch result.Status {
		case model.SearchCPOItemStatusFailed:
			failed = true
		case model.SearchCPOItemStatusSuccess:
			success = true
		}
	}
	if failed {
		return model.SearchCPOItemStatusFailed
	}
	if success {
		return model.SearchCPOItemStatusSuccess
	}
	return model.SearchCPOItemStatusSkipped
}

func summarizeSearchCPORunStatus(successCount, failedCount, skippedCount, partialCount int) string {
	if failedCount == 0 && partialCount == 0 {
		return model.SearchCPORunStatusSuccess
	}
	if partialCount > 0 {
		return model.SearchCPORunStatusPartialSuccess
	}
	if successCount > 0 || skippedCount > 0 {
		return model.SearchCPORunStatusPartialSuccess
	}
	return model.SearchCPORunStatusFailed
}

func parseInt64WithFallback(value interface{}, fallback int64) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return fallback
		}
		parsed, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}

func sortedStringKeys(m map[string]*searchCPORunItemState) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
