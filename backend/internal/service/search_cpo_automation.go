package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
)

const (
	searchCPODefaultScheduleTime     = "09:05"
	searchCPOSchedulerInterval       = time.Minute
	searchCPOAutoRunStaleAfter       = 2 * time.Hour
	searchCPOAvailabilityWaitTimeout = 90 * time.Second
	searchCPOEnableWaitTimeout       = 90 * time.Second
	searchCPOMorkovskWaitTimeout     = 90 * time.Second
)

type searchCPOAutomationConfigSnapshot struct {
	ScheduleTime      string `json:"schedule_time,omitempty"`
	EnableStep        bool   `json:"enable_step"`
	OfficialActionIDs []uint `json:"official_action_ids"`
	ShopActionIDs     []uint `json:"shop_action_ids"`
}

type searchCPOAutomationRunInput struct {
	RunID             uint
	ConfigID          *uint
	ShopID            uint
	TriggeredBy       *uint
	TriggerMode       string
	TriggerDate       time.Time
	ScheduleTime      string
	EnableStep        bool
	OfficialActionIDs []uint
	ShopActionIDs     []uint
}

type searchCPOAvailabilityArtifact struct {
	ParserRevision          string                              `json:"parser_revision,omitempty"`
	BuildRevision           string                              `json:"build_revision,omitempty"`
	ResponseRootKeys        []string                            `json:"response_root_keys,omitempty"`
	SampleResponseKeys      []string                            `json:"sample_response_keys,omitempty"`
	AvailabilityMapKeyCount int                                 `json:"availability_map_key_count,omitempty"`
	ReasonMapKeyCount       int                                 `json:"reason_map_key_count,omitempty"`
	Items                   []searchCPOAvailabilityArtifactItem `json:"items"`
}

type searchCPOAvailabilityArtifactItem struct {
	SourceSKU               string          `json:"source_sku"`
	SKU                     string          `json:"sku"`
	RequestedSKU            string          `json:"requested_sku,omitempty"`
	ParserRevision          string          `json:"parser_revision,omitempty"`
	ResponseRootKeys        []string        `json:"response_root_keys,omitempty"`
	SampleResponseKeys      []string        `json:"sample_response_keys,omitempty"`
	AvailabilityMapKeyCount int             `json:"availability_map_key_count,omitempty"`
	ReasonMapKeyCount       int             `json:"reason_map_key_count,omitempty"`
	SearchPromoStatus       string          `json:"search_promo_status"`
	CarrotsStatus           string          `json:"carrots_status"`
	AvailabilityPromo       *bool           `json:"availability_promo"`
	Error                   string          `json:"error,omitempty"`
	Payload                 json.RawMessage `json:"payload,omitempty"`
}

type searchCPOStepArtifact struct {
	Items []searchCPOStepArtifactItem `json:"items"`
}

type searchCPOStepArtifactItem struct {
	SourceSKU string `json:"source_sku"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	Message   string `json:"message,omitempty"`
}

type searchCPOAutomationItemState struct {
	Product         model.SearchCPOProduct
	RuleStateBefore string
	RuleStateAfter  string
	InitialStatus   string
	EnableStatus    string
	ExitStatus      string
	MorkovskStatus  string
	InitialResults  []dto.SearchCPORunActionResult
	ExitResults     []dto.SearchCPORunActionResult
	EnableResult    dto.SearchCPOAutomationStepResult
	MorkovskResult  dto.SearchCPOAutomationStepResult
	Message         string
}

func (s *SearchCPOService) StartAutomationScheduler() {
	if s.repo == nil {
		return
	}
	_ = s.repo.MarkStaleRunningAutoRunsFailed(time.Now().Add(-searchCPOAutoRunStaleAfter))
	go func() {
		s.scanDueAutomationConfigs(time.Now())
		ticker := time.NewTicker(searchCPOSchedulerInterval)
		defer ticker.Stop()
		for now := range ticker.C {
			s.scanDueAutomationConfigs(now)
		}
	}()
}

func (s *SearchCPOService) StartAutomationRun(userID uint, req *dto.SearchCPOAutomationRunRequest) (*dto.SearchCPOAutomationRunSummaryResponse, error) {
	if activeRun, err := s.repo.FindActiveAutoRunByShop(req.ShopID); err == nil && activeRun != nil {
		return nil, fmt.Errorf("已有 CPO 自动化任务正在执行中")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	cfg, err := s.GetConfig(req.ShopID)
	if err != nil {
		return nil, err
	}
	if len(cfg.OfficialActionIDs)+len(cfg.ShopActionIDs) == 0 {
		return nil, fmt.Errorf("请先保存默认活动配置")
	}

	now := time.Now()
	input := searchCPOAutomationRunInput{
		ShopID:            req.ShopID,
		TriggeredBy:       &userID,
		TriggerMode:       model.SearchCPOAutoTriggerModeManual,
		TriggerDate:       dateOnlyValue(now),
		ScheduleTime:      cfg.ScheduleTime,
		EnableStep:        true,
		OfficialActionIDs: uniqueUints(cfg.OfficialActionIDs),
		ShopActionIDs:     uniqueUints(cfg.ShopActionIDs),
	}
	if strings.TrimSpace(input.ScheduleTime) == "" {
		input.ScheduleTime = searchCPODefaultScheduleTime
	}

	run, err := s.createAutomationRun(input)
	if err != nil {
		return nil, err
	}
	input.RunID = run.ID
	go s.executeAutomationRun(input)
	return toSearchCPOAutomationRunSummaryDTO(run), nil
}

func (s *SearchCPOService) ListAutomationRuns(req *dto.SearchCPOAutomationRunListRequest) (*dto.SearchCPOAutomationRunListResponse, error) {
	runs, total, err := s.repo.ListAutoRunsByShop(req.ShopID, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	items := make([]dto.SearchCPOAutomationRunSummaryResponse, 0, len(runs))
	for _, run := range runs {
		runCopy := run
		items = append(items, *toSearchCPOAutomationRunSummaryDTO(&runCopy))
	}
	return &dto.SearchCPOAutomationRunListResponse{Total: total, Items: items}, nil
}

func (s *SearchCPOService) GetAutomationRunDetail(shopID, runID uint) (*dto.SearchCPOAutomationRunDetailResponse, error) {
	run, err := s.repo.FindAutoRunByIDAndShop(runID, shopID)
	if err != nil {
		return nil, err
	}
	snapshot := decodeSearchCPOAutomationConfigSnapshot(run.ConfigSnapshot)
	items := make([]dto.SearchCPOAutomationRunItemResponse, 0, len(run.RunItems))
	for _, item := range run.RunItems {
		items = append(items, dto.SearchCPOAutomationRunItemResponse{
			ID:                item.ID,
			ProductCacheID:    item.ProductCacheID,
			SourceSKU:         item.SourceSKU,
			SKU:               item.SKU,
			Title:             item.Title,
			SearchPromoStatus: item.SearchPromoStatus,
			CarrotsStatus:     item.CarrotsStatus,
			AvailabilityPromo: item.AvailabilityPromo,
			RuleStateBefore:   item.RuleStateBefore,
			RuleStateAfter:    item.RuleStateAfter,
			OverallStatus:     item.OverallStatus,
			InitialStatus:     item.InitialStatus,
			EnableStatus:      item.EnableStatus,
			ExitStatus:        item.ExitStatus,
			MorkovskStatus:    item.MorkovskStatus,
			InitialResults:    decodeSearchCPORunActionResults(item.InitialResults),
			ExitResults:       decodeSearchCPORunActionResults(item.ExitResults),
			EnableResult:      decodeSearchCPOAutomationStepResult(item.EnableResult),
			MorkovskResult:    decodeSearchCPOAutomationStepResult(item.MorkovskResult),
			Message:           strings.TrimSpace(item.Message),
		})
	}
	return &dto.SearchCPOAutomationRunDetailResponse{
		SearchCPOAutomationRunSummaryResponse: *toSearchCPOAutomationRunSummaryDTO(run),
		ShopID:                                run.ShopID,
		ConfigID:                              run.ConfigID,
		TriggeredBy:                           run.TriggeredBy,
		ScheduleTime:                          snapshot.ScheduleTime,
		EnableStep:                            snapshot.EnableStep,
		OfficialActionIDs:                     snapshot.OfficialActionIDs,
		ShopActionIDs:                         snapshot.ShopActionIDs,
		Items:                                 items,
	}, nil
}

func (s *SearchCPOService) scanDueAutomationConfigs(now time.Time) {
	configs, err := s.repo.ListAutoEnabledConfigs()
	if err != nil {
		return
	}
	currentMinute := now.Format("15:04")
	for _, config := range configs {
		scheduleTime, normErr := normalizeScheduleTime(config.ScheduleTime)
		if normErr != nil || scheduleTime != currentMinute {
			continue
		}
		if existing, err := s.repo.FindScheduledAutoRunByConfigAndDate(config.ID, dateOnlyValue(now)); err == nil && existing != nil {
			continue
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if activeRun, err := s.repo.FindActiveAutoRunByShop(config.ShopID); err == nil && activeRun != nil {
			continue
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		triggeredBy := s.resolveSearchCPOAutomationOwner(config.ShopID)
		input := searchCPOAutomationRunInput{
			ConfigID:          &config.ID,
			ShopID:            config.ShopID,
			TriggeredBy:       triggeredBy,
			TriggerMode:       model.SearchCPOAutoTriggerModeScheduled,
			TriggerDate:       dateOnlyValue(now),
			ScheduleTime:      scheduleTime,
			EnableStep:        true,
			OfficialActionIDs: decodeUintSlice(config.OfficialActionIDs),
			ShopActionIDs:     decodeUintSlice(config.ShopActionIDs),
		}
		run, createErr := s.createAutomationRun(input)
		if createErr != nil {
			continue
		}
		input.RunID = run.ID
		go s.executeAutomationRun(input)
	}
}

func (s *SearchCPOService) resolveSearchCPOAutomationOwner(shopID uint) *uint {
	shop, err := s.shopRepo.FindByID(shopID)
	if err != nil {
		return nil
	}
	ownerID := shop.OwnerID
	return &ownerID
}

func (s *SearchCPOService) createAutomationRun(input searchCPOAutomationRunInput) (*model.SearchCPOAutoRun, error) {
	scheduleTime := strings.TrimSpace(input.ScheduleTime)
	if scheduleTime == "" {
		scheduleTime = searchCPODefaultScheduleTime
	}
	snapshotBytes, _ := json.Marshal(searchCPOAutomationConfigSnapshot{
		ScheduleTime:      scheduleTime,
		EnableStep:        input.EnableStep,
		OfficialActionIDs: input.OfficialActionIDs,
		ShopActionIDs:     input.ShopActionIDs,
	})
	run := &model.SearchCPOAutoRun{
		ConfigID:       input.ConfigID,
		ShopID:         input.ShopID,
		TriggeredBy:    input.TriggeredBy,
		TriggerMode:    input.TriggerMode,
		TriggerDate:    input.TriggerDate,
		Status:         model.SearchCPORunStatusPending,
		FilterSnapshot: datatypes.JSON([]byte(`{}`)),
		ConfigSnapshot: snapshotBytes,
	}
	if err := s.repo.CreateAutoRun(run); err != nil {
		return nil, err
	}
	return run, nil
}
func (s *SearchCPOService) executeAutomationRun(input searchCPOAutomationRunInput) {
	run, err := s.repo.FindAutoRunByIDAndShop(input.RunID, input.ShopID)
	if err != nil {
		return
	}
	now := time.Now()
	run.Status = model.SearchCPORunStatusRunning
	run.StartedAt = &now
	run.ErrorMessage = ""
	_ = s.repo.UpdateAutoRun(run)

	if execErr := s.runAutomationExecution(run, input); execErr != nil {
		finishedAt := time.Now()
		run.Status = model.SearchCPORunStatusFailed
		run.ErrorMessage = execErr.Error()
		run.CompletedAt = &finishedAt
		_ = s.repo.UpdateAutoRun(run)
	}
}

func (s *SearchCPOService) runAutomationExecution(run *model.SearchCPOAutoRun, input searchCPOAutomationRunInput) error {
	triggerUserID := resolveSearchCPOTriggerUserID(input.TriggeredBy)
	if triggerUserID == 0 {
		if owner := s.resolveSearchCPOAutomationOwner(input.ShopID); owner != nil {
			triggerUserID = *owner
		}
	}

	if _, err := s.RefreshProducts(triggerUserID, input.ShopID); err != nil {
		return err
	}
	products, err := s.repo.ListProducts(input.ShopID)
	if err != nil {
		return err
	}
	availabilityMap, err := s.syncAvailability(triggerUserID, input.ShopID, products)
	if err != nil {
		return err
	}
	products, err = s.repo.ListProducts(input.ShopID)
	if err != nil {
		return err
	}

	itemStates := make(map[string]*searchCPOAutomationItemState)
	state1SKUs := make([]string, 0)
	state2SKUs := make([]string, 0)
	state3SKUs := make([]string, 0)
	now := time.Now()
	for _, product := range products {
		before := strings.TrimSpace(product.RuleState)
		after, detectedAt := deriveSearchCPORuleState(product, before, now)
		updateFields := map[string]interface{}{"rule_state": after}
		if detectedAt != nil {
			updateFields["state2_detected_at"] = detectedAt
		}
		if err := s.repo.UpdateProductFields(input.ShopID, product.SourceSKU, updateFields); err != nil {
			return err
		}
		state := &searchCPOAutomationItemState{
			Product:         product,
			RuleStateBefore: before,
			RuleStateAfter:  after,
			InitialStatus:   model.SearchCPOItemStatusSkipped,
			EnableStatus:    model.SearchCPOItemStatusSkipped,
			ExitStatus:      model.SearchCPOItemStatusSkipped,
			MorkovskStatus:  model.SearchCPOItemStatusSkipped,
			EnableResult:    dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusSkipped},
			MorkovskResult:  dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusSkipped},
		}
		if latest, ok := availabilityMap[product.SourceSKU]; ok {
			if trimmed := strings.TrimSpace(latest.CarrotsStatus); trimmed != "" {
				state.Product.CarrotsStatus = trimmed
			}
			if trimmed := firstNonEmptyServiceTrimmed(latest.SearchPromoStatus, state.Product.SearchPromoStatus); trimmed != "" {
				state.Product.SearchPromoStatus = trimmed
			}
			if latest.AvailabilityPromo != nil {
				state.Product.AvailabilityPromo = latest.AvailabilityPromo
			}
			if strings.TrimSpace(latest.Error) != "" && strings.TrimSpace(state.Message) == "" {
				state.Message = strings.TrimSpace(latest.Error)
			}
		}
		itemStates[product.SourceSKU] = state
		switch after {
		case model.SearchCPORuleStateState1:
			state1SKUs = append(state1SKUs, product.SourceSKU)
		case model.SearchCPORuleStateState2:
			state2SKUs = append(state2SKUs, product.SourceSKU)
		case model.SearchCPORuleStateState3Trigger:
			state3SKUs = append(state3SKUs, product.SourceSKU)
		}
	}

	run.TotalFetched = len(products)
	run.TotalState1 = len(state1SKUs)
	run.TotalState2 = len(state2SKUs)
	run.TotalState3Trigger = len(state3SKUs)
	run.TotalProcessed = len(state1SKUs) + len(state2SKUs) + len(state3SKUs)
	if err := s.repo.UpdateAutoRun(run); err != nil {
		return err
	}

	if len(state1SKUs) > 0 {
		if err := s.processState1Items(input, state1SKUs, itemStates); err != nil {
			return err
		}
	}
	migrationSKUs := uniqueSourceSKUs(append(append([]string{}, state2SKUs...), state3SKUs...))
	if len(migrationSKUs) > 0 {
		if err := s.processMigrationItems(input, triggerUserID, migrationSKUs, itemStates); err != nil {
			return err
		}
	}

	runItems := make([]model.SearchCPOAutoRunItem, 0, len(itemStates))
	successCount := 0
	failedCount := 0
	skippedCount := 0
	for _, sku := range sortedSearchCPOAutomationKeys(itemStates) {
		state := itemStates[sku]
		overall := summarizeSearchCPOAutomationItemStatus(state)
		switch overall {
		case model.SearchCPOItemStatusFailed:
			failedCount++
		case model.SearchCPOItemStatusSkipped:
			skippedCount++
		default:
			successCount++
		}
		initialBytes, _ := json.Marshal(state.InitialResults)
		exitBytes, _ := json.Marshal(state.ExitResults)
		enableBytes, _ := json.Marshal(state.EnableResult)
		morkovskBytes, _ := json.Marshal(state.MorkovskResult)
		item := model.SearchCPOAutoRunItem{
			SourceSKU:         state.Product.SourceSKU,
			SKU:               state.Product.SKU,
			Title:             state.Product.Title,
			SearchPromoStatus: state.Product.SearchPromoStatus,
			CarrotsStatus:     state.Product.CarrotsStatus,
			AvailabilityPromo: state.Product.AvailabilityPromo,
			RuleStateBefore:   state.RuleStateBefore,
			RuleStateAfter:    state.RuleStateAfter,
			OverallStatus:     overall,
			InitialStatus:     state.InitialStatus,
			EnableStatus:      state.EnableStatus,
			ExitStatus:        state.ExitStatus,
			MorkovskStatus:    state.MorkovskStatus,
			InitialResults:    initialBytes,
			EnableResult:      enableBytes,
			ExitResults:       exitBytes,
			MorkovskResult:    morkovskBytes,
			Message:           strings.TrimSpace(state.Message),
		}
		if state.Product.ID > 0 {
			item.ProductCacheID = &state.Product.ID
		}
		runItems = append(runItems, item)
	}
	if err := s.repo.ReplaceAutoRunItems(run.ID, runItems); err != nil {
		return err
	}

	run.SuccessItems = successCount
	run.FailedItems = failedCount
	run.SkippedItems = skippedCount
	run.Status = summarizeSearchCPOAutomationRunStatus(successCount, failedCount, skippedCount)
	finishedAt := time.Now()
	run.CompletedAt = &finishedAt
	return s.repo.UpdateAutoRun(run)
}

func (s *SearchCPOService) syncAvailability(userID, shopID uint, products []model.SearchCPOProduct) (map[string]searchCPOAvailabilityArtifactItem, error) {
	result := make(map[string]searchCPOAvailabilityArtifactItem)
	sourceSKUs := make([]string, 0, len(products))
	for _, product := range products {
		sourceSKUs = append(sourceSKUs, product.SourceSKU)
	}
	sourceSKUs = uniqueSourceSKUs(sourceSKUs)
	if len(sourceSKUs) == 0 {
		return result, nil
	}
	jobMeta := buildSearchCPOSKUMeta(products)
	job, err := s.automationService.CreateSyncSearchCPOAvailabilityJob(userID, shopID, sourceSKUs, jobMeta)
	if err != nil {
		return nil, err
	}
	waitedJob, waitErr := s.automationService.WaitForJobCompletion(job.ID, searchCPOAvailabilityWaitTimeout)
	if waitErr != nil {
		return nil, fmt.Errorf("同步 CPO 可推广状态失败: %w", waitErr)
	}
	if waitedJob.Status != model.AutomationJobStatusSuccess && waitedJob.Status != model.AutomationJobStatusPartialSuccess {
		return nil, fmt.Errorf("同步 CPO 可推广状态失败: %s", automationJobFailureMessage(waitedJob, "extension sync failed"))
	}
	artifact, err := s.automationService.GetLatestArtifact(waitedJob.ID, "search_cpo_availability_snapshot")
	if err != nil {
		return nil, err
	}
	snapshot := searchCPOAvailabilityArtifact{}
	if err := json.Unmarshal(artifact.Meta, &snapshot); err != nil {
		return nil, err
	}
	updates := make([]repository.SearchCPOProductAvailabilityUpdate, 0, len(snapshot.Items))
	checkedAt := time.Now()
	for _, item := range snapshot.Items {
		sku := strings.TrimSpace(item.SourceSKU)
		if sku == "" {
			continue
		}
		payload := item.Payload
		if len(payload) == 0 {
			payload, _ = json.Marshal(item)
		}
		updates = append(updates, repository.SearchCPOProductAvailabilityUpdate{
			SourceSKU:             sku,
			SearchPromoStatus:     trimmedSearchCPOStringPtr(item.SearchPromoStatus),
			CarrotsStatus:         trimmedSearchCPOStringPtr(item.CarrotsStatus),
			AvailabilityPromo:     item.AvailabilityPromo,
			AvailabilityPayload:   datatypes.JSON(payload),
			AvailabilityCheckedAt: checkedAt,
		})
		result[sku] = item
	}
	if err := s.repo.ApplyAvailabilityUpdates(shopID, updates); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *SearchCPOService) processState1Items(input searchCPOAutomationRunInput, sourceSKUs []string, itemStates map[string]*searchCPOAutomationItemState) error {
	actions, err := s.resolveActions(input.ShopID, input.OfficialActionIDs, input.ShopActionIDs)
	if err != nil {
		for _, sku := range sourceSKUs {
			if state := itemStates[sku]; state != nil {
				state.InitialStatus = model.SearchCPOItemStatusFailed
				state.EnableStatus = model.SearchCPOItemStatusSkipped
				state.EnableResult = dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusSkipped}
				state.Message = err.Error()
			}
		}
		return nil
	}
	officialActions, shopActions := splitActionsBySource(actions)
	states, err := s.buildRunStates(input.ShopID, sourceSKUs)
	if err != nil {
		return err
	}
	if err := s.executeOfficial(input.ShopID, sourceSKUs, states, officialActions); err != nil {
		return err
	}
	triggerUserID := resolveSearchCPOTriggerUserID(input.TriggeredBy)
	if err := s.executeShop(input.ShopID, triggerUserID, sourceSKUs, states, shopActions); err != nil {
		return err
	}

	for _, sku := range sourceSKUs {
		state := itemStates[sku]
		runState := states[sku]
		if state == nil || runState == nil {
			continue
		}
		combined := append([]dto.SearchCPORunActionResult{}, runState.OfficialResults...)
		combined = append(combined, runState.ShopResults...)
		state.InitialResults = combined
		overall, _, _ := summarizeSearchCPORowStatus(runState)
		state.InitialStatus = overall
		state.EnableStatus = model.SearchCPOItemStatusSkipped
		state.EnableResult = dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusSkipped}
	}
	return nil
}

func (s *SearchCPOService) processMigrationItems(input searchCPOAutomationRunInput, triggerUserID uint, sourceSKUs []string, itemStates map[string]*searchCPOAutomationItemState) error {
	if len(sourceSKUs) == 0 {
		return nil
	}
	if triggerUserID == 0 {
		triggerUserID = resolveSearchCPOTriggerUserID(input.TriggeredBy)
	}

	actions, err := s.syncActionsForMigration(input.ShopID, triggerUserID)
	if err != nil {
		markSearchCPOMigrationSetupFailed(sourceSKUs, itemStates, err.Error())
		return nil
	}
	grouped, err := s.loadActiveMigrationActions(input.ShopID, triggerUserID, sourceSKUs, actions)
	if err != nil {
		markSearchCPOMigrationSetupFailed(sourceSKUs, itemStates, err.Error())
		return nil
	}

	eligibleForEnable := make([]string, 0, len(sourceSKUs))
	for _, sku := range sourceSKUs {
		state := itemStates[sku]
		if state == nil {
			continue
		}
		matchedActions := grouped[sku]
		if len(matchedActions) == 0 {
			state.ExitStatus = model.SearchCPOItemStatusSkipped
			eligibleForEnable = append(eligibleForEnable, sku)
			continue
		}
		officialActions, shopActions := splitActionsBySource(matchedActions)
		exitResults := make([]dto.SearchCPORunActionResult, 0, len(matchedActions))
		exitFailed := false

		if len(officialActions) > 0 {
			for _, action := range officialActions {
				resp, removeErr := s.promotionService.removeFromOfficialActions(input.ShopID, []model.PromotionAction{action}, []string{sku})
				result := dto.SearchCPORunActionResult{
					PromotionActionID: action.ID,
					ActionID:          action.ActionID,
					Title:             displayActionName(action),
					Source:            action.Source,
					Status:            model.SearchCPOItemStatusSuccess,
				}
				if removeErr != nil || resp == nil || !resp.Success {
					result.Status = model.SearchCPOItemStatusFailed
					result.Error = firstNonEmptyServiceTrimmed(errorText(removeErr), officialRemoveError(resp), "官方活动退出失败")
					exitFailed = true
				}
				exitResults = append(exitResults, result)
			}
		}

		if len(shopActions) > 0 {
			job, createErr := s.promotionService.CreateUnifiedShopActionsJob(triggerUserID, input.ShopID, model.AutomationJobTypePromoUnifiedRemove, shopActions, []string{sku})
			if createErr != nil {
				exitFailed = true
				for _, action := range shopActions {
					exitResults = append(exitResults, dto.SearchCPORunActionResult{
						PromotionActionID: action.ID,
						SourceActionID:    action.SourceActionID,
						Title:             displayActionName(action),
						Source:            action.Source,
						Status:            model.SearchCPOItemStatusFailed,
						Error:             createErr.Error(),
					})
				}
			} else {
				waitedJob, waitErr := s.automationService.WaitForJobCompletion(job.ID, searchCPOShopWaitTimeout)
				itemBySKU := make(map[string]model.AutomationJobItem, len(waitedJob.Items))
				for _, jobItem := range waitedJob.Items {
					itemBySKU[jobItem.SourceSKU] = jobItem
				}
				for _, action := range shopActions {
					result := dto.SearchCPORunActionResult{
						PromotionActionID: action.ID,
						SourceActionID:    action.SourceActionID,
						Title:             displayActionName(action),
						Source:            action.Source,
						Status:            model.SearchCPOItemStatusSuccess,
					}
					if waitErr != nil || waitedJob.Status == model.AutomationJobStatusFailed {
						result.Status = model.SearchCPOItemStatusFailed
						result.Error = firstNonEmptyServiceTrimmed(errorText(waitErr), waitedJob.ErrorMessage, "店铺活动退出失败")
						exitFailed = true
					} else if jobItem, ok := itemBySKU[sku]; ok {
						if jobItem.OverallStatus != model.AutomationStepStatusSuccess && jobItem.OverallStatus != model.AutomationStepStatusSkipped {
							result.Status = model.SearchCPOItemStatusFailed
							result.Error = firstNonEmptyServiceTrimmed(jobItem.StepExitError, jobItem.StepReaddError, jobItem.StepRepriceError, waitedJob.ErrorMessage, "店铺活动退出失败")
							exitFailed = true
						}
					} else {
						result.Status = model.SearchCPOItemStatusFailed
						result.Error = "店铺活动未返回退出结果"
						exitFailed = true
					}
					exitResults = append(exitResults, result)
				}
			}
		}

		state.ExitResults = exitResults
		if exitFailed {
			state.ExitStatus = model.SearchCPOItemStatusFailed
			state.EnableStatus = model.SearchCPOItemStatusSkipped
			state.EnableResult = dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusSkipped, Message: "前置退出失败，未执行 enable"}
			state.MorkovskStatus = model.SearchCPOItemStatusSkipped
			state.MorkovskResult = dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusSkipped, Message: "前置退出失败，未加入 Morkovsk"}
			continue
		}
		state.ExitStatus = model.SearchCPOItemStatusSuccess
		eligibleForEnable = append(eligibleForEnable, sku)
	}

	if len(eligibleForEnable) == 0 {
		return nil
	}
	enableMeta := buildSearchCPOSKUMetaFromStates(eligibleForEnable, itemStates)
	enableMap, err := s.executeEnableStep(triggerUserID, input.ShopID, eligibleForEnable, enableMeta)
	if err != nil {
		for _, sku := range eligibleForEnable {
			if state := itemStates[sku]; state != nil {
				state.EnableStatus = model.SearchCPOItemStatusFailed
				state.EnableResult = dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusFailed, Error: err.Error()}
				state.MorkovskStatus = model.SearchCPOItemStatusSkipped
				state.MorkovskResult = dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusSkipped, Message: "enable 失败，未加入 Morkovsk"}
			}
		}
		return nil
	}

	eligibleForMorkovsk := make([]string, 0, len(eligibleForEnable))
	for _, sku := range eligibleForEnable {
		state := itemStates[sku]
		if state == nil {
			continue
		}
		step, ok := enableMap[sku]
		if !ok {
			state.EnableStatus = model.SearchCPOItemStatusFailed
			state.EnableResult = dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusFailed, Error: "未返回 enable 结果"}
			state.MorkovskStatus = model.SearchCPOItemStatusSkipped
			state.MorkovskResult = dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusSkipped, Message: "enable 失败，未加入 Morkovsk"}
			continue
		}
		state.EnableStatus = step.Status
		state.EnableResult = step
		if step.Status != model.SearchCPOItemStatusSuccess {
			state.MorkovskStatus = model.SearchCPOItemStatusSkipped
			state.MorkovskResult = dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusSkipped, Message: "enable 失败，未加入 Morkovsk"}
			continue
		}
		state.Product.SearchPromoStatus = "SEARCH_PROMO_STATUS_ENABLED"
		if state.RuleStateAfter == model.SearchCPORuleStateState2 {
			state.RuleStateAfter = model.SearchCPORuleStateState3Trigger
		}
		if err := s.repo.UpdateProductFields(input.ShopID, sku, map[string]interface{}{
			"search_promo_status": "SEARCH_PROMO_STATUS_ENABLED",
			"rule_state":          state.RuleStateAfter,
		}); err != nil {
			return err
		}
		state.MorkovskStatus = model.SearchCPOItemStatusPending
		eligibleForMorkovsk = append(eligibleForMorkovsk, sku)
	}

	if len(eligibleForMorkovsk) == 0 {
		return nil
	}
	morkovskMeta := buildSearchCPOSKUMetaFromStates(eligibleForMorkovsk, itemStates)
	morkovskMap, err := s.executeMorkovskBatchEnable(triggerUserID, input.ShopID, eligibleForMorkovsk, morkovskMeta)
	if err != nil {
		for _, sku := range eligibleForMorkovsk {
			if state := itemStates[sku]; state != nil {
				state.MorkovskStatus = model.SearchCPOItemStatusFailed
				state.MorkovskResult = dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusFailed, Error: err.Error()}
			}
		}
		return nil
	}
	for _, sku := range eligibleForMorkovsk {
		state := itemStates[sku]
		if state == nil {
			continue
		}
		if step, ok := morkovskMap[sku]; ok {
			state.MorkovskStatus = step.Status
			state.MorkovskResult = step
			if step.Status == model.SearchCPOItemStatusSuccess {
				now := time.Now()
				state.RuleStateAfter = model.SearchCPORuleStateJoined
				state.Product.SearchPromoStatus = "SEARCH_PROMO_STATUS_ENABLED"
				state.Product.CarrotsStatus = "CARROTS_STATUS_ENABLED"
				state.Product.MorkovskJoinedAt = &now
				if err := s.repo.UpdateProductFields(input.ShopID, sku, map[string]interface{}{
					"search_promo_status": "SEARCH_PROMO_STATUS_ENABLED",
					"carrots_status":      "CARROTS_STATUS_ENABLED",
					"morkovsk_joined_at":  &now,
					"rule_state":          model.SearchCPORuleStateJoined,
				}); err != nil {
					return err
				}
			}
			continue
		}
		state.MorkovskStatus = model.SearchCPOItemStatusFailed
		state.MorkovskResult = dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusFailed, Error: "未返回 Morkovsk 结果"}
	}
	return nil
}

func (s *SearchCPOService) syncActionsForMigration(shopID, triggerUserID uint) ([]model.PromotionAction, error) {
	if triggerUserID == 0 {
		return nil, fmt.Errorf("同步活动清单失败: 缺少可用的触发用户")
	}
	result, err := s.promotionService.SyncPromotionActionsV2(shopID, triggerUserID)
	if err != nil {
		return nil, fmt.Errorf("同步活动清单失败: %w", err)
	}
	if msg := buildSearchCPOActionSyncFailureMessage(result); msg != "" {
		return nil, fmt.Errorf("同步活动清单失败: %s", msg)
	}
	actions, err := s.promotionRepo.FindPromotionActionsByShopID(shopID)
	if err != nil {
		return nil, err
	}
	return actions, nil
}

func (s *SearchCPOService) loadActiveMigrationActions(shopID, triggerUserID uint, sourceSKUs []string, actions []model.PromotionAction) (map[string][]model.PromotionAction, error) {
	grouped := make(map[string][]model.PromotionAction)
	if len(actions) == 0 || len(sourceSKUs) == 0 {
		return grouped, nil
	}
	for i := range actions {
		action := actions[i]
		if action.Source == "shop" {
			if err := s.promotionService.refreshShopActionProducts(&action, triggerUserID); err != nil {
				return nil, err
			}
			continue
		}
		if err := s.promotionService.refreshOfficialActionProducts(&action); err != nil {
			return nil, err
		}
	}

	actionIDs := actionIDsForActions(actions)
	if len(actionIDs) == 0 {
		return grouped, nil
	}
	activeProducts, err := s.promotionRepo.ListActionProductsByActionIDsAndSourceSKUs(shopID, actionIDs, sourceSKUs)
	if err != nil {
		return nil, err
	}
	actionByID := make(map[uint]model.PromotionAction, len(actions))
	for _, action := range actions {
		actionByID[action.ID] = action
	}
	for _, item := range activeProducts {
		action, ok := actionByID[item.PromotionActionID]
		if !ok {
			continue
		}
		grouped[item.SourceSKU] = append(grouped[item.SourceSKU], action)
	}
	return grouped, nil
}

func markSearchCPOMigrationSetupFailed(sourceSKUs []string, itemStates map[string]*searchCPOAutomationItemState, message string) {
	message = strings.TrimSpace(message)
	for _, sku := range sourceSKUs {
		state := itemStates[sku]
		if state == nil {
			continue
		}
		state.ExitStatus = model.SearchCPOItemStatusFailed
		state.EnableStatus = model.SearchCPOItemStatusSkipped
		state.EnableResult = dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusSkipped, Message: "迁移前置失败，未执行 enable"}
		state.MorkovskStatus = model.SearchCPOItemStatusSkipped
		state.MorkovskResult = dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusSkipped, Message: "迁移前置失败，未加入 Morkovsk"}
		state.Message = message
	}
}

func buildSearchCPOActionSyncFailureMessage(result *dto.SyncActionsResult) string {
	if result == nil {
		return "未返回活动同步结果"
	}
	problems := make([]string, 0)
	if result.ShopSyncPending {
		problems = append(problems, "店铺活动同步仍在后台进行")
	}
	if len(result.PartialErrors) > 0 {
		keys := make([]string, 0, len(result.PartialErrors))
		for key := range result.PartialErrors {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			message := strings.TrimSpace(result.PartialErrors[key])
			if message == "" {
				continue
			}
			problems = append(problems, fmt.Sprintf("%s: %s", key, message))
		}
	}
	return strings.Join(problems, "; ")
}

func (s *SearchCPOService) executeEnableStep(userID, shopID uint, sourceSKUs []string, meta map[string]interface{}) (map[string]dto.SearchCPOAutomationStepResult, error) {
	results := make(map[string]dto.SearchCPOAutomationStepResult)
	job, err := s.automationService.CreateSearchCPOEnableProductsJob(userID, shopID, sourceSKUs, meta)
	if err != nil {
		return nil, err
	}
	waitedJob, waitErr := s.automationService.WaitForJobCompletion(job.ID, searchCPOEnableWaitTimeout)
	if waitErr != nil {
		return nil, waitErr
	}
	if waitedJob.Status != model.AutomationJobStatusSuccess && waitedJob.Status != model.AutomationJobStatusPartialSuccess {
		return nil, fmt.Errorf(automationJobFailureMessage(waitedJob, "extension enable failed"))
	}
	artifact, err := s.automationService.GetLatestArtifact(waitedJob.ID, "search_cpo_enable_snapshot")
	if err != nil {
		return nil, err
	}
	snapshot := searchCPOStepArtifact{}
	if err := json.Unmarshal(artifact.Meta, &snapshot); err != nil {
		return nil, err
	}
	for _, item := range snapshot.Items {
		results[strings.TrimSpace(item.SourceSKU)] = dto.SearchCPOAutomationStepResult{
			Status:  normalizeSearchCPOStepStatus(item.Status),
			Error:   strings.TrimSpace(item.Error),
			Message: strings.TrimSpace(item.Message),
		}
	}
	return results, nil
}

func (s *SearchCPOService) executeMorkovskBatchEnable(userID, shopID uint, sourceSKUs []string, meta map[string]interface{}) (map[string]dto.SearchCPOAutomationStepResult, error) {
	results := make(map[string]dto.SearchCPOAutomationStepResult)
	job, err := s.automationService.CreateSearchCPOBatchEnableMorkovskJob(userID, shopID, sourceSKUs, meta)
	if err != nil {
		return nil, err
	}
	waitedJob, waitErr := s.automationService.WaitForJobCompletion(job.ID, searchCPOMorkovskWaitTimeout)
	if waitErr != nil {
		return nil, waitErr
	}
	if waitedJob.Status != model.AutomationJobStatusSuccess && waitedJob.Status != model.AutomationJobStatusPartialSuccess {
		return nil, fmt.Errorf(automationJobFailureMessage(waitedJob, "extension batch enable failed"))
	}
	artifact, err := s.automationService.GetLatestArtifact(waitedJob.ID, "search_cpo_morkovsk_snapshot")
	if err != nil {
		return nil, err
	}
	snapshot := searchCPOStepArtifact{}
	if err := json.Unmarshal(artifact.Meta, &snapshot); err != nil {
		return nil, err
	}
	for _, item := range snapshot.Items {
		results[strings.TrimSpace(item.SourceSKU)] = dto.SearchCPOAutomationStepResult{
			Status:  normalizeSearchCPOStepStatus(item.Status),
			Error:   strings.TrimSpace(item.Error),
			Message: strings.TrimSpace(item.Message),
		}
	}
	return results, nil
}
func deriveSearchCPORuleState(product model.SearchCPOProduct, previousState string, now time.Time) (string, *time.Time) {
	if product.MorkovskJoinedAt != nil {
		return model.SearchCPORuleStateJoined, product.State2DetectedAt
	}
	searchStatus := strings.TrimSpace(product.SearchPromoStatus)
	carrotsStatus := strings.TrimSpace(product.CarrotsStatus)
	if searchStatus == "SEARCH_PROMO_STATUS_DISABLED" && carrotsStatus == "CARROTS_STATUS_DISABLED" {
		if product.AvailabilityPromo != nil && *product.AvailabilityPromo {
			if product.State2DetectedAt != nil {
				return model.SearchCPORuleStateState2, product.State2DetectedAt
			}
			detectedAt := now
			return model.SearchCPORuleStateState2, &detectedAt
		}
		if product.AvailabilityPromo != nil && !*product.AvailabilityPromo {
			return model.SearchCPORuleStateState1, product.State2DetectedAt
		}
	}
	if searchStatus == "SEARCH_PROMO_STATUS_ENABLED" && product.State2DetectedAt != nil && previousState != model.SearchCPORuleStateJoined {
		return model.SearchCPORuleStateState3Trigger, product.State2DetectedAt
	}
	return model.SearchCPORuleStateOther, product.State2DetectedAt
}

func summarizeSearchCPOAutomationItemStatus(state *searchCPOAutomationItemState) string {
	if state == nil {
		return model.SearchCPOItemStatusSkipped
	}
	statuses := []string{state.InitialStatus, state.EnableStatus, state.ExitStatus, state.MorkovskStatus}
	hasSuccess := false
	for _, status := range statuses {
		switch status {
		case model.SearchCPOItemStatusFailed:
			return model.SearchCPOItemStatusFailed
		case model.SearchCPOItemStatusSuccess, model.SearchCPOItemStatusPartialSuccess:
			hasSuccess = true
		}
	}
	if hasSuccess {
		return model.SearchCPOItemStatusSuccess
	}
	return model.SearchCPOItemStatusSkipped
}

func summarizeSearchCPOAutomationRunStatus(successCount, failedCount, skippedCount int) string {
	if failedCount == 0 {
		return model.SearchCPORunStatusSuccess
	}
	if successCount == 0 && skippedCount == 0 {
		return model.SearchCPORunStatusFailed
	}
	return model.SearchCPORunStatusPartialSuccess
}

func sortedSearchCPOAutomationKeys(states map[string]*searchCPOAutomationItemState) []string {
	keys := make([]string, 0, len(states))
	for key := range states {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func buildSearchCPOSKUMeta(products []model.SearchCPOProduct) map[string]interface{} {
	if len(products) == 0 {
		return nil
	}
	skuMap := make(map[string]string)
	for _, product := range products {
		sourceSKU := strings.TrimSpace(product.SourceSKU)
		targetSKU := normalizeSearchCPOTargetSKU(sourceSKU, product.SKU)
		if sourceSKU == "" || targetSKU == "" {
			continue
		}
		skuMap[sourceSKU] = targetSKU
	}
	if len(skuMap) == 0 {
		return nil
	}
	return map[string]interface{}{"sku_map": skuMap}
}

func buildSearchCPOSKUMetaFromStates(sourceSKUs []string, itemStates map[string]*searchCPOAutomationItemState) map[string]interface{} {
	if len(sourceSKUs) == 0 || len(itemStates) == 0 {
		return nil
	}
	skuMap := make(map[string]string)
	for _, sourceSKU := range sourceSKUs {
		normalizedSource := strings.TrimSpace(sourceSKU)
		state := itemStates[normalizedSource]
		if normalizedSource == "" || state == nil {
			continue
		}
		targetSKU := normalizeSearchCPOTargetSKU(normalizedSource, state.Product.SKU)
		if targetSKU == "" {
			continue
		}
		skuMap[normalizedSource] = targetSKU
	}
	if len(skuMap) == 0 {
		return nil
	}
	return map[string]interface{}{"sku_map": skuMap}
}

func normalizeSearchCPOTargetSKU(sourceSKU, sku string) string {
	if trimmed := strings.TrimSpace(sku); trimmed != "" {
		return trimmed
	}
	trimmedSource := strings.TrimSpace(sourceSKU)
	if trimmedSource != "" && strings.IndexFunc(trimmedSource, func(r rune) bool { return r < '0' || r > '9' }) == -1 {
		return trimmedSource
	}
	return ""
}

func trimmedSearchCPOStringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func toSearchCPOAutomationRunSummaryDTO(run *model.SearchCPOAutoRun) *dto.SearchCPOAutomationRunSummaryResponse {
	if run == nil {
		return nil
	}
	return &dto.SearchCPOAutomationRunSummaryResponse{
		ID:                 run.ID,
		TriggerMode:        run.TriggerMode,
		TriggerDate:        run.TriggerDate.Format("2006-01-02"),
		Status:             run.Status,
		TotalFetched:       run.TotalFetched,
		TotalState1:        run.TotalState1,
		TotalState2:        run.TotalState2,
		TotalState3Trigger: run.TotalState3Trigger,
		TotalProcessed:     run.TotalProcessed,
		SuccessItems:       run.SuccessItems,
		FailedItems:        run.FailedItems,
		SkippedItems:       run.SkippedItems,
		ErrorMessage:       strings.TrimSpace(run.ErrorMessage),
		StartedAt:          formatOptionalTime(run.StartedAt),
		CompletedAt:        formatOptionalTime(run.CompletedAt),
		CreatedAt:          run.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func decodeSearchCPOAutomationConfigSnapshot(raw datatypes.JSON) searchCPOAutomationConfigSnapshot {
	snapshot := searchCPOAutomationConfigSnapshot{EnableStep: true, OfficialActionIDs: []uint{}, ShopActionIDs: []uint{}}
	if len(raw) == 0 {
		return snapshot
	}
	_ = json.Unmarshal(raw, &snapshot)
	snapshot.OfficialActionIDs = uniqueUints(snapshot.OfficialActionIDs)
	snapshot.ShopActionIDs = uniqueUints(snapshot.ShopActionIDs)
	return snapshot
}

func decodeSearchCPOAutomationStepResult(raw datatypes.JSON) dto.SearchCPOAutomationStepResult {
	result := dto.SearchCPOAutomationStepResult{Status: model.SearchCPOItemStatusSkipped}
	if len(raw) == 0 {
		return result
	}
	_ = json.Unmarshal(raw, &result)
	result.Status = normalizeSearchCPOStepStatus(result.Status)
	return result
}

func normalizeSearchCPOStepStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case model.SearchCPOItemStatusSuccess, model.SearchCPOItemStatusPartialSuccess:
		return model.SearchCPOItemStatusSuccess
	case model.SearchCPOItemStatusFailed:
		return model.SearchCPOItemStatusFailed
	case model.SearchCPOItemStatusPending:
		return model.SearchCPOItemStatusPending
	default:
		return model.SearchCPOItemStatusSkipped
	}
}

func officialRemoveError(resp *dto.BatchEnrollResponse) string {
	if resp == nil || len(resp.Details) == 0 {
		return ""
	}
	for _, detail := range resp.Details {
		if strings.TrimSpace(detail.Error) != "" {
			return strings.TrimSpace(detail.Error)
		}
	}
	return ""
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func resolveSearchCPOTriggerUserID(userID *uint) uint {
	if userID == nil {
		return 0
	}
	return *userID
}
