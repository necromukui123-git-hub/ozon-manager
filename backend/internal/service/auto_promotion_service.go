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
	autoPromotionDefaultScheduleTime       = "09:05"
	autoPromotionSchedulerInterval         = time.Minute
	autoPromotionRunStaleAfter             = 2 * time.Hour
	autoPromotionOfficialCandidatePageSize = 200
	autoPromotionShopCandidateWaitTimeout  = 60 * time.Second
	autoPromotionShopActionWaitTimeout     = 5 * time.Minute
)

type AutoPromotionService struct {
	autoRepo           *repository.AutoPromotionRepository
	productRepo        *repository.ProductRepository
	promotionRepo      *repository.PromotionRepository
	shopRepo           *repository.ShopRepository
	ozonCatalogRepo    *repository.OzonCatalogRepository
	ozonCatalogService *OzonCatalogService
	automationService  *AutomationService
	promotionService   *PromotionService
}

type autoPromotionConfigSnapshot struct {
	ScheduleTime      string `json:"schedule_time,omitempty"`
	TargetDate        string `json:"target_date"`
	OfficialActionIDs []uint `json:"official_action_ids"`
	ShopActionIDs     []uint `json:"shop_action_ids"`
}

type autoPromotionCandidateSnapshot struct {
	Items []autoPromotionCandidateSnapshotItem `json:"items"`
}

type autoPromotionCandidateSnapshotItem struct {
	SourceSKU       string  `json:"source_sku"`
	OfferID         string  `json:"offer_id"`
	PlatformSKU     string  `json:"platform_sku"`
	OzonProductID   int64   `json:"ozon_product_id"`
	ActionPrice     float64 `json:"action_price"`
	MaxActionPrice  float64 `json:"max_action_price"`
	DiscountPercent float64 `json:"discount_percent"`
	Stock           int     `json:"stock"`
	Status          string  `json:"status"`
}

type autoPromotionRunInput struct {
	RunID             uint
	ConfigID          *uint
	ShopID            uint
	TriggeredBy       *uint
	TriggerMode       string
	TriggerDate       time.Time
	TargetDate        time.Time
	ScheduleTime      string
	OfficialActionIDs []uint
	ShopActionIDs     []uint
}

type autoPromotionItemState struct {
	Product         model.Product
	CatalogItem     model.OzonProductCatalogItem
	OfficialResults []dto.AutoPromotionActionResult
	ShopResults     []dto.AutoPromotionActionResult
	Blocked         bool
	HasExecutedStep bool
}

func NewAutoPromotionService(
	autoRepo *repository.AutoPromotionRepository,
	productRepo *repository.ProductRepository,
	promotionRepo *repository.PromotionRepository,
	shopRepo *repository.ShopRepository,
	ozonCatalogRepo *repository.OzonCatalogRepository,
	ozonCatalogService *OzonCatalogService,
	automationService *AutomationService,
	promotionService *PromotionService,
) *AutoPromotionService {
	return &AutoPromotionService{
		autoRepo:           autoRepo,
		productRepo:        productRepo,
		promotionRepo:      promotionRepo,
		shopRepo:           shopRepo,
		ozonCatalogRepo:    ozonCatalogRepo,
		ozonCatalogService: ozonCatalogService,
		automationService:  automationService,
		promotionService:   promotionService,
	}
}

func (s *AutoPromotionService) StartScheduler() {
	_ = s.autoRepo.MarkStaleRunningRunsFailed(time.Now().Add(-autoPromotionRunStaleAfter))

	go func() {
		s.scanDueConfigs(time.Now())
		ticker := time.NewTicker(autoPromotionSchedulerInterval)
		defer ticker.Stop()

		for now := range ticker.C {
			s.scanDueConfigs(now)
		}
	}()
}

func (s *AutoPromotionService) GetConfig(shopID uint) (*dto.AutoPromotionConfigResponse, error) {
	config, err := s.autoRepo.FindConfigByShopID(shopID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			yesterday := time.Now().AddDate(0, 0, -1)
			return &dto.AutoPromotionConfigResponse{
				ShopID:            shopID,
				Enabled:           false,
				ScheduleTime:      autoPromotionDefaultScheduleTime,
				TargetDate:        yesterday.Format("2006-01-02"),
				OfficialActionIDs: []uint{},
				ShopActionIDs:     []uint{},
			}, nil
		}
		return nil, err
	}
	return toAutoPromotionConfigDTO(config)
}

func (s *AutoPromotionService) UpdateConfig(req *dto.AutoPromotionConfigRequest) (*dto.AutoPromotionConfigResponse, error) {
	targetDate, err := parseDateOnly(req.TargetDate)
	if err != nil || targetDate == nil {
		return nil, fmt.Errorf("invalid target_date, expected YYYY-MM-DD")
	}

	scheduleTime, err := normalizeScheduleTime(req.ScheduleTime)
	if err != nil {
		return nil, err
	}

	officialIDs := uniqueUints(req.OfficialActionIDs)
	shopIDs := uniqueUints(req.ShopActionIDs)
	if req.Enabled && len(officialIDs)+len(shopIDs) == 0 {
		return nil, fmt.Errorf("启用自动加促销前至少选择一个活动")
	}
	if len(officialIDs)+len(shopIDs) > 0 {
		if err := s.validateSelectedActions(req.ShopID, officialIDs, shopIDs); err != nil {
			return nil, err
		}
	}

	officialBytes, _ := json.Marshal(officialIDs)
	shopBytes, _ := json.Marshal(shopIDs)
	config := &model.AutoPromotionConfig{
		ShopID:            req.ShopID,
		Enabled:           req.Enabled,
		ScheduleTime:      scheduleTime,
		TargetDate:        dateOnlyValue(*targetDate),
		OfficialActionIDs: officialBytes,
		ShopActionIDs:     shopBytes,
	}
	if err := s.autoRepo.UpsertConfig(config); err != nil {
		return nil, err
	}

	saved, err := s.autoRepo.FindConfigByShopID(req.ShopID)
	if err != nil {
		return nil, err
	}
	return toAutoPromotionConfigDTO(saved)
}

func (s *AutoPromotionService) StartManualRun(userID uint, req *dto.AutoPromotionRunRequest) (*dto.AutoPromotionRunSummaryResponse, error) {
	targetDate, err := parseDateOnly(req.TargetDate)
	if err != nil || targetDate == nil {
		return nil, fmt.Errorf("invalid target_date, expected YYYY-MM-DD")
	}

	officialIDs := uniqueUints(req.OfficialActionIDs)
	shopIDs := uniqueUints(req.ShopActionIDs)
	if len(officialIDs)+len(shopIDs) == 0 {
		return nil, fmt.Errorf("请至少选择一个促销活动")
	}
	if err := s.validateSelectedActions(req.ShopID, officialIDs, shopIDs); err != nil {
		return nil, err
	}

	if activeRun, err := s.autoRepo.FindActiveRunByShop(req.ShopID); err == nil && activeRun != nil {
		return nil, fmt.Errorf("已有自动加促销任务正在执行中")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now()
	input := autoPromotionRunInput{
		ShopID:            req.ShopID,
		TriggeredBy:       &userID,
		TriggerMode:       model.AutoPromotionTriggerModeManual,
		TriggerDate:       dateOnlyValue(now),
		TargetDate:        dateOnlyValue(*targetDate),
		OfficialActionIDs: officialIDs,
		ShopActionIDs:     shopIDs,
	}

	run, err := s.createRun(input)
	if err != nil {
		return nil, err
	}
	input.RunID = run.ID
	go s.executeRun(input)

	return toAutoPromotionRunSummaryDTO(run), nil
}

func (s *AutoPromotionService) ListRuns(req *dto.AutoPromotionRunListRequest) (*dto.AutoPromotionRunListResponse, error) {
	runs, total, err := s.autoRepo.ListRunsByShop(req.ShopID, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	items := make([]dto.AutoPromotionRunSummaryResponse, 0, len(runs))
	for _, run := range runs {
		runCopy := run
		items = append(items, *toAutoPromotionRunSummaryDTO(&runCopy))
	}

	return &dto.AutoPromotionRunListResponse{
		Total: total,
		Items: items,
	}, nil
}

func (s *AutoPromotionService) GetRunDetail(shopID uint, runID uint) (*dto.AutoPromotionRunDetailResponse, error) {
	run, err := s.autoRepo.FindRunByIDAndShop(runID, shopID)
	if err != nil {
		return nil, err
	}

	snapshot := decodeAutoPromotionConfigSnapshot(run.ConfigSnapshot)
	items := make([]dto.AutoPromotionRunItemResponse, 0, len(run.RunItems))
	for _, item := range run.RunItems {
		items = append(items, dto.AutoPromotionRunItemResponse{
			ID:              item.ID,
			ProductID:       item.ProductID,
			OzonProductID:   item.OzonProductID,
			SourceSKU:       item.SourceSKU,
			ProductName:     item.ProductName,
			ListingDate:     item.ListingDate.Format("2006-01-02"),
			OverallStatus:   item.OverallStatus,
			OfficialStatus:  item.OfficialStatus,
			ShopStatus:      item.ShopStatus,
			OfficialResults: decodeActionResults(item.OfficialResults),
			ShopResults:     decodeActionResults(item.ShopResults),
		})
	}

	return &dto.AutoPromotionRunDetailResponse{
		AutoPromotionRunSummaryResponse: *toAutoPromotionRunSummaryDTO(run),
		ShopID:                          run.ShopID,
		ConfigID:                        run.ConfigID,
		TriggeredBy:                     run.TriggeredBy,
		ScheduleTime:                    snapshot.ScheduleTime,
		OfficialActionIDs:               snapshot.OfficialActionIDs,
		ShopActionIDs:                   snapshot.ShopActionIDs,
		Items:                           items,
	}, nil
}

func (s *AutoPromotionService) scanDueConfigs(now time.Time) {
	configs, err := s.autoRepo.ListEnabledConfigs()
	if err != nil {
		return
	}

	currentMinute := now.Format("15:04")
	for _, config := range configs {
		if strings.TrimSpace(config.ScheduleTime) != currentMinute {
			continue
		}

		triggerDate := dateOnlyValue(now)
		if existing, err := s.autoRepo.FindScheduledRunByConfigAndDate(config.ID, triggerDate); err == nil && existing != nil {
			continue
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}

		if activeRun, err := s.autoRepo.FindActiveRunByShop(config.ShopID); err == nil && activeRun != nil {
			continue
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}

		input := autoPromotionRunInput{
			ConfigID:          &config.ID,
			ShopID:            config.ShopID,
			TriggerMode:       model.AutoPromotionTriggerModeScheduled,
			TriggerDate:       triggerDate,
			TargetDate:        dateOnlyValue(config.TargetDate),
			ScheduleTime:      strings.TrimSpace(config.ScheduleTime),
			OfficialActionIDs: decodeActionIDs(config.OfficialActionIDs),
			ShopActionIDs:     decodeActionIDs(config.ShopActionIDs),
		}

		run, err := s.createRun(input)
		if err != nil {
			continue
		}
		input.RunID = run.ID
		go s.executeRun(input)
	}
}

func (s *AutoPromotionService) createRun(input autoPromotionRunInput) (*model.AutoPromotionRun, error) {
	snapshotBytes, _ := json.Marshal(autoPromotionConfigSnapshot{
		ScheduleTime:      input.ScheduleTime,
		TargetDate:        input.TargetDate.Format("2006-01-02"),
		OfficialActionIDs: input.OfficialActionIDs,
		ShopActionIDs:     input.ShopActionIDs,
	})

	run := &model.AutoPromotionRun{
		ConfigID:       input.ConfigID,
		ShopID:         input.ShopID,
		TriggeredBy:    input.TriggeredBy,
		TriggerMode:    input.TriggerMode,
		TriggerDate:    input.TriggerDate,
		TargetDate:     input.TargetDate,
		Status:         model.AutoPromotionRunStatusPending,
		ConfigSnapshot: snapshotBytes,
	}
	if err := s.autoRepo.CreateRun(run); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *AutoPromotionService) executeRun(input autoPromotionRunInput) {
	run, err := s.autoRepo.FindRunByIDAndShop(input.RunID, input.ShopID)
	if err != nil {
		return
	}

	now := time.Now()
	run.Status = model.AutoPromotionRunStatusRunning
	run.StartedAt = &now
	run.ErrorMessage = ""
	_ = s.autoRepo.UpdateRun(run)

	if input.TriggeredBy == nil {
		shop, shopErr := s.shopRepo.FindByID(input.ShopID)
		if shopErr == nil {
			input.TriggeredBy = &shop.OwnerID
		}
	}

	if execErr := s.runExecution(run, input); execErr != nil {
		finishedAt := time.Now()
		run.Status = model.AutoPromotionRunStatusFailed
		run.ErrorMessage = execErr.Error()
		run.CompletedAt = &finishedAt
		_ = s.autoRepo.UpdateRun(run)
	}
}

func (s *AutoPromotionService) runExecution(run *model.AutoPromotionRun, input autoPromotionRunInput) error {
	actions, err := s.resolveActions(input.ShopID, input.OfficialActionIDs, input.ShopActionIDs)
	if err != nil {
		return err
	}
	officialActions, shopActions := splitActionsBySource(actions)

	if err := s.ozonCatalogService.RefreshShopCatalogSync(input.ShopID); err != nil {
		return fmt.Errorf("刷新 Ozon 商品目录失败: %w", err)
	}

	for _, action := range officialActions {
		actionCopy := action
		if err := s.refreshOfficialCandidates(&actionCopy); err != nil {
			return fmt.Errorf("刷新官方活动候选商品失败: %s: %w", displayActionName(action), err)
		}
		if err := s.promotionService.refreshOfficialActionProducts(&actionCopy); err != nil {
			return fmt.Errorf("刷新官方活动已报名商品失败: %s: %w", displayActionName(action), err)
		}
	}

	triggerUserID := uint(0)
	if input.TriggeredBy != nil {
		triggerUserID = *input.TriggeredBy
	}
	for _, action := range shopActions {
		actionCopy := action
		if err := s.refreshShopCandidates(&actionCopy, triggerUserID); err != nil {
			return fmt.Errorf("刷新店铺活动候选商品失败: %s: %w", displayActionName(action), err)
		}
	}

	catalogItems, err := s.ozonCatalogRepo.ListByListingDate(input.ShopID, input.TargetDate)
	if err != nil {
		return fmt.Errorf("按日期查询目录商品失败: %w", err)
	}

	localProducts, err := s.productRepo.FindByOzonProductIDs(input.ShopID, collectCatalogProductIDs(catalogItems))
	if err != nil {
		return fmt.Errorf("查询本地商品失败: %w", err)
	}

	run.TotalCandidates = 0
	for _, item := range catalogItems {
		if _, exists := localProducts[item.OzonProductID]; exists {
			run.TotalCandidates++
		}
	}

	sourceSKUs := make([]string, 0, run.TotalCandidates)
	for _, item := range catalogItems {
		if product, exists := localProducts[item.OzonProductID]; exists {
			sourceSKUs = append(sourceSKUs, product.SourceSKU)
		}
	}
	sourceSKUs = uniqueStrings(sourceSKUs)

	officialActionIDs := actionIDsForActions(officialActions)
	shopActionIDs := actionIDsForActions(shopActions)

	officialCandidates, err := s.promotionRepo.ListActionCandidatesByActionIDsAndSourceSKUs(input.ShopID, officialActionIDs, sourceSKUs)
	if err != nil {
		return fmt.Errorf("查询官方活动候选缓存失败: %w", err)
	}
	shopCandidates, err := s.promotionRepo.ListActionCandidatesByActionIDsAndSourceSKUs(input.ShopID, shopActionIDs, sourceSKUs)
	if err != nil {
		return fmt.Errorf("查询店铺活动候选缓存失败: %w", err)
	}
	officialExisting, err := s.promotionRepo.ListActionProductsByActionIDsAndSourceSKUs(input.ShopID, officialActionIDs, sourceSKUs)
	if err != nil {
		return fmt.Errorf("查询官方活动已报名缓存失败: %w", err)
	}

	selectedStates := s.selectEligibleItems(catalogItems, localProducts, officialActions, shopActions, officialCandidates, shopCandidates, officialExisting)
	run.TotalSelected = len(selectedStates)
	run.TotalProcessed = len(selectedStates)

	if len(selectedStates) == 0 {
		finishedAt := time.Now()
		run.Status = model.AutoPromotionRunStatusSuccess
		run.CompletedAt = &finishedAt
		if err := s.autoRepo.ReplaceRunItems(run.ID, []model.AutoPromotionRunItem{}); err != nil {
			return err
		}
		return s.autoRepo.UpdateRun(run)
	}

	if err := s.executeOfficialActions(input.ShopID, officialActions, selectedStates); err != nil {
		return err
	}
	if err := s.executeShopActions(input.ShopID, triggerUserID, shopActions, selectedStates); err != nil {
		return err
	}

	runItems := make([]model.AutoPromotionRunItem, 0, len(selectedStates))
	runKeys := sortedStateKeys(selectedStates)
	successCount := 0
	failedCount := 0
	skippedCount := 0
	for _, sku := range runKeys {
		state := selectedStates[sku]
		overallStatus, officialStatus, shopStatus := summarizeItemStatuses(state)
		switch overallStatus {
		case model.AutoPromotionItemStatusFailed:
			failedCount++
		case model.AutoPromotionItemStatusSkipped:
			skippedCount++
		default:
			successCount++
		}

		officialBytes, _ := json.Marshal(state.OfficialResults)
		shopBytes, _ := json.Marshal(state.ShopResults)
		runItems = append(runItems, model.AutoPromotionRunItem{
			ProductID:       &state.Product.ID,
			OzonProductID:   state.Product.OzonProductID,
			SourceSKU:       state.Product.SourceSKU,
			ProductName:     firstNonEmpty(strings.TrimSpace(state.Product.Name), strings.TrimSpace(state.CatalogItem.Name), state.Product.SourceSKU),
			ListingDate:     dateOnlyValue(*state.CatalogItem.ListingDate),
			OverallStatus:   overallStatus,
			OfficialStatus:  officialStatus,
			ShopStatus:      shopStatus,
			OfficialResults: officialBytes,
			ShopResults:     shopBytes,
		})
	}

	if err := s.autoRepo.ReplaceRunItems(run.ID, runItems); err != nil {
		return err
	}

	run.SuccessItems = successCount
	run.FailedItems = failedCount
	run.SkippedItems = skippedCount
	run.Status = summarizeRunStatus(successCount, failedCount, skippedCount)
	finishedAt := time.Now()
	run.CompletedAt = &finishedAt
	return s.autoRepo.UpdateRun(run)
}

func (s *AutoPromotionService) validateSelectedActions(shopID uint, officialIDs []uint, shopIDs []uint) error {
	_, err := s.resolveActions(shopID, officialIDs, shopIDs)
	return err
}

func (s *AutoPromotionService) resolveActions(shopID uint, officialIDs []uint, shopIDs []uint) ([]model.PromotionAction, error) {
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

	actionByID := make(map[uint]model.PromotionAction, len(actions))
	for _, action := range actions {
		actionByID[action.ID] = action
	}

	for _, id := range officialIDs {
		action, exists := actionByID[id]
		if !exists || action.Source != "official" {
			return nil, fmt.Errorf("官方活动选择无效")
		}
	}
	for _, id := range shopIDs {
		action, exists := actionByID[id]
		if !exists || action.Source != "shop" {
			return nil, fmt.Errorf("店铺活动选择无效")
		}
	}

	return actions, nil
}

func (s *AutoPromotionService) refreshOfficialCandidates(action *model.PromotionAction) error {
	shop, err := s.shopRepo.GetWithCredentials(action.ShopID)
	if err != nil {
		return err
	}
	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	seenLastIDs := make(map[string]struct{})
	lastID := ""
	rawCandidates := make([]ozon.ActionCandidate, 0)
	productIDs := make([]int64, 0)

	for {
		resp, err := client.GetActionCandidates(action.ActionID, autoPromotionOfficialCandidatePageSize, lastID)
		if err != nil {
			return err
		}

		for _, item := range resp.Result.Products {
			productID := resolveOfficialCandidateProductID(item)
			if productID <= 0 {
				continue
			}
			rawCandidates = append(rawCandidates, item)
			productIDs = append(productIDs, productID)
		}

		nextLastID := strings.TrimSpace(resp.Result.LastID)
		if nextLastID == "" || len(resp.Result.Products) == 0 {
			break
		}
		if _, exists := seenLastIDs[nextLastID]; exists {
			break
		}
		seenLastIDs[nextLastID] = struct{}{}
		lastID = nextLastID
	}

	localProducts, err := s.productRepo.FindByOzonProductIDs(action.ShopID, uniqueInt64s(productIDs))
	if err != nil {
		return err
	}

	candidates := make([]model.PromotionActionCandidate, 0, len(rawCandidates))
	for _, item := range rawCandidates {
		productID := resolveOfficialCandidateProductID(item)
		if productID <= 0 {
			continue
		}

		sourceSKU := strconv.FormatInt(productID, 10)
		offerID := sourceSKU
		platformSKU := ""
		if localProduct, exists := localProducts[productID]; exists {
			if strings.TrimSpace(localProduct.SourceSKU) != "" {
				sourceSKU = strings.TrimSpace(localProduct.SourceSKU)
				offerID = sourceSKU
			}
			if localProduct.OzonSKU > 0 {
				platformSKU = strconv.FormatInt(localProduct.OzonSKU, 10)
			}
		}

		payload, _ := json.Marshal(item)
		candidates = append(candidates, model.PromotionActionCandidate{
			OzonProductID:   productID,
			SourceSKU:       sourceSKU,
			OfferID:         offerID,
			PlatformSKU:     platformSKU,
			ActionPrice:     item.ActionPrice,
			MaxActionPrice:  item.MaxActionPrice,
			DiscountPercent: calculateDiscountPercent(item.Price, item.ActionPrice),
			Stock:           item.Stock,
			Status:          model.PromotionActionCandidateStatusCandidate,
			Payload:         payload,
		})
	}

	return s.promotionRepo.ReplaceActionCandidates(action, dedupeCandidates(candidates))
}

func (s *AutoPromotionService) refreshShopCandidates(action *model.PromotionAction, userID uint) error {
	if s.automationService == nil {
		return fmt.Errorf("automation service unavailable")
	}
	if userID == 0 {
		shop, err := s.shopRepo.FindByID(action.ShopID)
		if err != nil {
			return err
		}
		userID = shop.OwnerID
	}

	job, err := s.automationService.CreateSyncActionCandidatesJob(userID, action.ShopID, action.ID, action.SourceActionID)
	if err != nil {
		return err
	}

	waitedJob, waitErr := s.automationService.WaitForJobCompletion(job.ID, autoPromotionShopCandidateWaitTimeout)
	if waitErr != nil {
		return fmt.Errorf("shop action candidates sync timeout")
	}
	if waitedJob.Status != model.AutomationJobStatusSuccess && waitedJob.Status != model.AutomationJobStatusPartialSuccess {
		return fmt.Errorf("shop action candidates sync failed")
	}

	artifact, err := s.automationService.GetLatestArtifact(waitedJob.ID, "action_candidates_snapshot")
	if err != nil {
		return err
	}

	snapshot := autoPromotionCandidateSnapshot{}
	if err := json.Unmarshal(artifact.Meta, &snapshot); err != nil {
		return err
	}

	candidates := make([]model.PromotionActionCandidate, 0, len(snapshot.Items))
	for _, item := range snapshot.Items {
		sourceSKU := strings.TrimSpace(item.SourceSKU)
		if sourceSKU == "" {
			sourceSKU = strings.TrimSpace(item.OfferID)
		}
		if sourceSKU == "" && item.OzonProductID > 0 {
			sourceSKU = strconv.FormatInt(item.OzonProductID, 10)
		}
		if sourceSKU == "" {
			continue
		}

		payload, _ := json.Marshal(item)
		candidates = append(candidates, model.PromotionActionCandidate{
			OzonProductID:   item.OzonProductID,
			SourceSKU:       sourceSKU,
			OfferID:         firstNonEmpty(item.OfferID, sourceSKU),
			PlatformSKU:     strings.TrimSpace(item.PlatformSKU),
			ActionPrice:     item.ActionPrice,
			MaxActionPrice:  item.MaxActionPrice,
			DiscountPercent: item.DiscountPercent,
			Stock:           item.Stock,
			Status:          normalizeCandidateStatus(item.Status),
			Payload:         payload,
		})
	}

	return s.promotionRepo.ReplaceActionCandidates(action, dedupeCandidates(candidates))
}

func (s *AutoPromotionService) selectEligibleItems(
	catalogItems []model.OzonProductCatalogItem,
	localProducts map[int64]model.Product,
	officialActions []model.PromotionAction,
	shopActions []model.PromotionAction,
	officialCandidates []model.PromotionActionCandidate,
	shopCandidates []model.PromotionActionCandidate,
	officialExisting []model.PromotionActionProduct,
) map[string]*autoPromotionItemState {
	officialCandidateMap := groupCandidatesByActionAndSKU(officialCandidates)
	shopCandidateMap := groupCandidatesByActionAndSKU(shopCandidates)
	officialExistingMap := groupActionProductsByActionAndSKU(officialExisting)

	states := make(map[string]*autoPromotionItemState)
	for _, catalogItem := range catalogItems {
		product, exists := localProducts[catalogItem.OzonProductID]
		if !exists || catalogItem.ListingDate == nil {
			continue
		}
		sku := strings.TrimSpace(product.SourceSKU)
		if sku == "" {
			continue
		}

		state := &autoPromotionItemState{
			Product:     product,
			CatalogItem: catalogItem,
		}

		eligible := true
		for _, action := range officialActions {
			if actionItems, ok := officialExistingMap[action.ID]; ok {
				if _, exists := actionItems[sku]; exists {
					state.OfficialResults = append(state.OfficialResults, dto.AutoPromotionActionResult{
						PromotionActionID: action.ID,
						ActionID:          action.ActionID,
						Title:             displayActionName(action),
						Source:            action.Source,
						Status:            model.PromotionActionCandidateStatusAlreadyActive,
					})
					continue
				}
			}

			actionCandidates, ok := officialCandidateMap[action.ID]
			if !ok {
				eligible = false
				break
			}
			candidate, ok := actionCandidates[sku]
			if !ok {
				eligible = false
				break
			}
			state.OfficialResults = append(state.OfficialResults, dto.AutoPromotionActionResult{
				PromotionActionID: action.ID,
				ActionID:          action.ActionID,
				Title:             displayActionName(action),
				Source:            action.Source,
				Status:            model.PromotionActionCandidateStatusCandidate,
				ActionPrice:       candidate.ActionPrice,
				MaxActionPrice:    candidate.MaxActionPrice,
			})
		}
		if !eligible {
			continue
		}

		for _, action := range shopActions {
			actionCandidates, ok := shopCandidateMap[action.ID]
			if !ok {
				eligible = false
				break
			}
			candidate, ok := actionCandidates[sku]
			if !ok {
				eligible = false
				break
			}

			resultStatus := model.PromotionActionCandidateStatusCandidate
			if candidate.Status == model.PromotionActionCandidateStatusActive {
				resultStatus = model.PromotionActionCandidateStatusAlreadyActive
			}
			state.ShopResults = append(state.ShopResults, dto.AutoPromotionActionResult{
				PromotionActionID: action.ID,
				SourceActionID:    action.SourceActionID,
				Title:             displayActionName(action),
				Source:            action.Source,
				Status:            resultStatus,
				ActionPrice:       candidate.ActionPrice,
				MaxActionPrice:    candidate.MaxActionPrice,
			})
		}
		if !eligible {
			continue
		}

		states[sku] = state
	}

	return states
}

func (s *AutoPromotionService) executeOfficialActions(shopID uint, actions []model.PromotionAction, states map[string]*autoPromotionItemState) error {
	if len(actions) == 0 || len(states) == 0 {
		return nil
	}

	shop, err := s.shopRepo.GetWithCredentials(shopID)
	if err != nil {
		return err
	}
	client := ozon.NewClient(shop.ClientID, shop.ApiKey)

	for _, action := range actions {
		payload := make([]ozon.ActivateProductItem, 0)
		skusByProductID := make(map[int64]string)
		orderedSKUs := sortedStateKeys(states)

		for _, sku := range orderedSKUs {
			state := states[sku]
			if state.Blocked {
				markActionSkipped(state, &action, "official")
				continue
			}

			result := findActionResultByID(state.OfficialResults, action.ID)
			if result == nil || result.Status == model.PromotionActionCandidateStatusAlreadyActive {
				continue
			}

			actionPrice := chooseOfficialActionPrice(state.Product.CurrentPrice, result.ActionPrice, result.MaxActionPrice)
			if actionPrice <= 0 {
				result.Status = model.AutoPromotionItemStatusFailed
				result.Error = "未找到合法的官方活动价"
				state.Blocked = true
				continue
			}

			result.ActionPrice = actionPrice
			payload = append(payload, ozon.ActivateProductItem{
				ProductID:   state.Product.OzonProductID,
				ActionPrice: actionPrice,
			})
			skusByProductID[state.Product.OzonProductID] = sku
		}

		if len(payload) == 0 {
			continue
		}

		resp, err := client.ActivateProducts(action.ActionID, payload)
		if err != nil {
			for _, item := range payload {
				if sku := skusByProductID[item.ProductID]; sku != "" {
					if state := states[sku]; state != nil {
						if result := findActionResultByID(state.OfficialResults, action.ID); result != nil {
							result.Status = model.AutoPromotionItemStatusFailed
							result.Error = err.Error()
						}
						state.Blocked = true
					}
				}
			}
			continue
		}

		successProductIDs := make(map[int64]struct{}, len(resp.Result.ProductIDs))
		for _, productID := range resp.Result.ProductIDs {
			successProductIDs[productID] = struct{}{}
		}
		rejectedByProductID := make(map[int64]string, len(resp.Result.Rejected))
		for _, item := range resp.Result.Rejected {
			rejectedByProductID[item.ProductID] = strings.TrimSpace(item.Reason)
		}

		for _, item := range payload {
			sku := skusByProductID[item.ProductID]
			state := states[sku]
			if state == nil {
				continue
			}
			result := findActionResultByID(state.OfficialResults, action.ID)
			if result == nil {
				continue
			}

			if reason, exists := rejectedByProductID[item.ProductID]; exists {
				result.Status = model.AutoPromotionItemStatusFailed
				result.Error = firstNonEmpty(reason, "官方活动拒绝加入")
				state.Blocked = true
				continue
			}
			if _, exists := successProductIDs[item.ProductID]; exists {
				result.Status = model.AutoPromotionItemStatusSuccess
				state.HasExecutedStep = true
				continue
			}

			result.Status = model.AutoPromotionItemStatusFailed
			result.Error = "官方活动未返回明确成功结果"
			state.Blocked = true
		}
	}

	return nil
}

func (s *AutoPromotionService) executeShopActions(shopID uint, userID uint, actions []model.PromotionAction, states map[string]*autoPromotionItemState) error {
	if len(actions) == 0 || len(states) == 0 {
		return nil
	}

	if userID == 0 {
		shop, err := s.shopRepo.FindByID(shopID)
		if err != nil {
			return err
		}
		userID = shop.OwnerID
	}

	for _, action := range actions {
		actionSKUs := make([]string, 0)
		for _, sku := range sortedStateKeys(states) {
			state := states[sku]
			if state.Blocked {
				markActionSkipped(state, &action, "shop")
				continue
			}

			result := findActionResultBySourceActionID(state.ShopResults, action.ID, action.SourceActionID)
			if result == nil || result.Status == model.PromotionActionCandidateStatusAlreadyActive {
				continue
			}
			actionSKUs = append(actionSKUs, sku)
		}

		if len(actionSKUs) == 0 {
			continue
		}

		job, err := s.promotionService.CreateShopActionJob(userID, shopID, model.AutomationJobTypeShopActionDeclare, action.SourceActionID, actionSKUs)
		if err != nil {
			for _, sku := range actionSKUs {
				if state := states[sku]; state != nil {
					if result := findActionResultBySourceActionID(state.ShopResults, action.ID, action.SourceActionID); result != nil {
						result.Status = model.AutoPromotionItemStatusFailed
						result.Error = err.Error()
					}
					state.Blocked = true
				}
			}
			continue
		}

		waitedJob, waitErr := s.automationService.WaitForJobCompletion(job.ID, autoPromotionShopActionWaitTimeout)
		if waitErr != nil {
			for _, sku := range actionSKUs {
				if state := states[sku]; state != nil {
					if result := findActionResultBySourceActionID(state.ShopResults, action.ID, action.SourceActionID); result != nil {
						result.Status = model.AutoPromotionItemStatusFailed
						result.Error = "店铺活动执行超时"
					}
					state.Blocked = true
				}
			}
			continue
		}

		itemBySKU := make(map[string]model.AutomationJobItem, len(waitedJob.Items))
		for _, item := range waitedJob.Items {
			itemBySKU[item.SourceSKU] = item
		}

		for _, sku := range actionSKUs {
			state := states[sku]
			if state == nil {
				continue
			}
			result := findActionResultBySourceActionID(state.ShopResults, action.ID, action.SourceActionID)
			if result == nil {
				continue
			}

			jobItem, exists := itemBySKU[sku]
			if !exists {
				result.Status = model.AutoPromotionItemStatusFailed
				result.Error = "店铺活动未返回商品执行结果"
				state.Blocked = true
				continue
			}

			if jobItem.OverallStatus == model.AutomationStepStatusSuccess || jobItem.OverallStatus == model.AutomationStepStatusSkipped {
				result.Status = model.AutoPromotionItemStatusSuccess
				state.HasExecutedStep = true
				continue
			}

			result.Status = model.AutoPromotionItemStatusFailed
			result.Error = firstNonEmpty(jobItem.StepReaddError, jobItem.StepExitError, jobItem.StepRepriceError, waitedJob.ErrorMessage, "店铺活动执行失败")
			state.Blocked = true
		}
	}

	return nil
}

func toAutoPromotionConfigDTO(config *model.AutoPromotionConfig) (*dto.AutoPromotionConfigResponse, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	return &dto.AutoPromotionConfigResponse{
		ID:                config.ID,
		ShopID:            config.ShopID,
		Enabled:           config.Enabled,
		ScheduleTime:      strings.TrimSpace(config.ScheduleTime),
		TargetDate:        config.TargetDate.Format("2006-01-02"),
		OfficialActionIDs: decodeActionIDs(config.OfficialActionIDs),
		ShopActionIDs:     decodeActionIDs(config.ShopActionIDs),
		UpdatedAt:         config.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func toAutoPromotionRunSummaryDTO(run *model.AutoPromotionRun) *dto.AutoPromotionRunSummaryResponse {
	if run == nil {
		return nil
	}

	startedAt := ""
	if formatted := FormatAutomationTime(run.StartedAt); formatted != nil {
		startedAt = *formatted
	}
	completedAt := ""
	if formatted := FormatAutomationTime(run.CompletedAt); formatted != nil {
		completedAt = *formatted
	}

	return &dto.AutoPromotionRunSummaryResponse{
		ID:              run.ID,
		TriggerMode:     run.TriggerMode,
		TriggerDate:     run.TriggerDate.Format("2006-01-02"),
		TargetDate:      run.TargetDate.Format("2006-01-02"),
		Status:          run.Status,
		TotalCandidates: run.TotalCandidates,
		TotalSelected:   run.TotalSelected,
		TotalProcessed:  run.TotalProcessed,
		SuccessItems:    run.SuccessItems,
		FailedItems:     run.FailedItems,
		SkippedItems:    run.SkippedItems,
		ErrorMessage:    strings.TrimSpace(run.ErrorMessage),
		StartedAt:       startedAt,
		CompletedAt:     completedAt,
		CreatedAt:       run.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func decodeAutoPromotionConfigSnapshot(raw datatypes.JSON) autoPromotionConfigSnapshot {
	snapshot := autoPromotionConfigSnapshot{}
	if len(raw) == 0 {
		return snapshot
	}
	_ = json.Unmarshal(raw, &snapshot)
	return snapshot
}

func decodeActionIDs(raw datatypes.JSON) []uint {
	if len(raw) == 0 {
		return []uint{}
	}
	result := make([]uint, 0)
	_ = json.Unmarshal(raw, &result)
	return uniqueUints(result)
}

func decodeActionResults(raw datatypes.JSON) []dto.AutoPromotionActionResult {
	if len(raw) == 0 {
		return []dto.AutoPromotionActionResult{}
	}
	result := make([]dto.AutoPromotionActionResult, 0)
	_ = json.Unmarshal(raw, &result)
	return result
}

func normalizeScheduleTime(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return autoPromotionDefaultScheduleTime, nil
	}
	parsed, err := time.Parse("15:04", trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid schedule_time, expected HH:MM")
	}
	return parsed.Format("15:04"), nil
}

func dateOnlyValue(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func uniqueUints(values []uint) []uint {
	seen := make(map[uint]struct{}, len(values))
	result := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func uniqueInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func displayActionName(action model.PromotionAction) string {
	if strings.TrimSpace(action.DisplayName) != "" {
		return strings.TrimSpace(action.DisplayName)
	}
	if strings.TrimSpace(action.Title) != "" {
		return strings.TrimSpace(action.Title)
	}
	if action.Source == "shop" && strings.TrimSpace(action.SourceActionID) != "" {
		return "店铺活动 #" + strings.TrimSpace(action.SourceActionID)
	}
	return fmt.Sprintf("活动 #%d", action.ActionID)
}

func collectCatalogProductIDs(items []model.OzonProductCatalogItem) []int64 {
	productIDs := make([]int64, 0, len(items))
	for _, item := range items {
		if item.OzonProductID > 0 {
			productIDs = append(productIDs, item.OzonProductID)
		}
	}
	return uniqueInt64s(productIDs)
}

func groupCandidatesByActionAndSKU(items []model.PromotionActionCandidate) map[uint]map[string]model.PromotionActionCandidate {
	result := make(map[uint]map[string]model.PromotionActionCandidate)
	for _, item := range items {
		if _, exists := result[item.PromotionActionID]; !exists {
			result[item.PromotionActionID] = make(map[string]model.PromotionActionCandidate)
		}
		result[item.PromotionActionID][item.SourceSKU] = item
	}
	return result
}

func groupActionProductsByActionAndSKU(items []model.PromotionActionProduct) map[uint]map[string]model.PromotionActionProduct {
	result := make(map[uint]map[string]model.PromotionActionProduct)
	for _, item := range items {
		if _, exists := result[item.PromotionActionID]; !exists {
			result[item.PromotionActionID] = make(map[string]model.PromotionActionProduct)
		}
		result[item.PromotionActionID][item.SourceSKU] = item
	}
	return result
}

func actionIDsForActions(actions []model.PromotionAction) []uint {
	result := make([]uint, 0, len(actions))
	for _, action := range actions {
		result = append(result, action.ID)
	}
	return result
}

func resolveOfficialCandidateProductID(item ozon.ActionCandidate) int64 {
	if item.ProductID > 0 {
		return item.ProductID
	}
	return item.ID
}

func calculateDiscountPercent(basePrice float64, actionPrice float64) float64 {
	if basePrice <= 0 || actionPrice <= 0 || basePrice < actionPrice {
		return 0
	}
	return (basePrice - actionPrice) / basePrice * 100
}

func dedupeCandidates(items []model.PromotionActionCandidate) []model.PromotionActionCandidate {
	seen := make(map[string]struct{}, len(items))
	result := make([]model.PromotionActionCandidate, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.SourceSKU) == "" {
			continue
		}
		key := strings.TrimSpace(item.SourceSKU)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}

func normalizeCandidateStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case model.PromotionActionCandidateStatusActive:
		return model.PromotionActionCandidateStatusActive
	case model.PromotionActionCandidateStatusInactive:
		return model.PromotionActionCandidateStatusInactive
	default:
		return model.PromotionActionCandidateStatusCandidate
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func chooseOfficialActionPrice(currentPrice float64, candidateActionPrice float64, maxActionPrice float64) float64 {
	if candidateActionPrice > 0 {
		return candidateActionPrice
	}
	switch {
	case currentPrice > 0 && maxActionPrice > 0:
		if currentPrice <= maxActionPrice {
			return currentPrice
		}
		return maxActionPrice
	case currentPrice > 0:
		return currentPrice
	case maxActionPrice > 0:
		return maxActionPrice
	default:
		return 0
	}
}

func findActionResultByID(results []dto.AutoPromotionActionResult, promotionActionID uint) *dto.AutoPromotionActionResult {
	for index := range results {
		if results[index].PromotionActionID == promotionActionID {
			return &results[index]
		}
	}
	return nil
}

func findActionResultBySourceActionID(results []dto.AutoPromotionActionResult, promotionActionID uint, sourceActionID string) *dto.AutoPromotionActionResult {
	for index := range results {
		if results[index].PromotionActionID == promotionActionID {
			return &results[index]
		}
		if results[index].SourceActionID != "" && results[index].SourceActionID == sourceActionID {
			return &results[index]
		}
	}
	return nil
}

func markActionSkipped(state *autoPromotionItemState, action *model.PromotionAction, source string) {
	if state == nil || action == nil {
		return
	}

	if source == "official" {
		if result := findActionResultByID(state.OfficialResults, action.ID); result != nil {
			if result.Status == model.PromotionActionCandidateStatusAlreadyActive {
				return
			}
			if result.Status == model.PromotionActionCandidateStatusCandidate || result.Status == model.AutoPromotionItemStatusPending {
				result.Status = model.AutoPromotionItemStatusSkipped
				result.Error = "前置活动失败，已跳过"
			}
		}
		return
	}

	if result := findActionResultBySourceActionID(state.ShopResults, action.ID, action.SourceActionID); result != nil {
		if result.Status == model.PromotionActionCandidateStatusAlreadyActive {
			return
		}
		if result.Status == model.PromotionActionCandidateStatusCandidate || result.Status == model.AutoPromotionItemStatusPending {
			result.Status = model.AutoPromotionItemStatusSkipped
			result.Error = "前置活动失败，已跳过"
		}
	}
}

func summarizeItemStatuses(state *autoPromotionItemState) (string, string, string) {
	officialStatus := summarizeActionResults(state.OfficialResults)
	shopStatus := summarizeActionResults(state.ShopResults)

	if officialStatus == model.AutoPromotionItemStatusFailed || shopStatus == model.AutoPromotionItemStatusFailed {
		return model.AutoPromotionItemStatusFailed, officialStatus, shopStatus
	}
	if officialStatus == model.AutoPromotionItemStatusSkipped && shopStatus == model.AutoPromotionItemStatusSkipped {
		return model.AutoPromotionItemStatusSkipped, officialStatus, shopStatus
	}
	return model.AutoPromotionItemStatusSuccess, officialStatus, shopStatus
}

func summarizeActionResults(results []dto.AutoPromotionActionResult) string {
	if len(results) == 0 {
		return model.AutoPromotionItemStatusSkipped
	}

	failed := false
	success := false
	skippedOnly := true
	for _, result := range results {
		switch result.Status {
		case model.AutoPromotionItemStatusFailed:
			failed = true
		case model.AutoPromotionItemStatusSuccess, model.PromotionActionCandidateStatusAlreadyActive:
			success = true
			skippedOnly = false
		case model.AutoPromotionItemStatusSkipped:
		default:
			skippedOnly = false
		}
	}

	if failed {
		return model.AutoPromotionItemStatusFailed
	}
	if skippedOnly && !success {
		return model.AutoPromotionItemStatusSkipped
	}
	if success {
		return model.AutoPromotionItemStatusSuccess
	}
	return model.AutoPromotionItemStatusSkipped
}

func summarizeRunStatus(successCount, failedCount, skippedCount int) string {
	if failedCount == 0 {
		return model.AutoPromotionRunStatusSuccess
	}
	if successCount == 0 && skippedCount == 0 {
		return model.AutoPromotionRunStatusFailed
	}
	return model.AutoPromotionRunStatusPartialSuccess
}

func sortedStateKeys(states map[string]*autoPromotionItemState) []string {
	keys := make([]string, 0, len(states))
	for sku := range states {
		keys = append(keys, sku)
	}
	sort.Strings(keys)
	return keys
}
