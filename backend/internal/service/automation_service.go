package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
	"ozon-manager/pkg/ozon"
)

type AutomationService struct {
	automationRepo *repository.AutomationRepository
	productRepo    *repository.ProductRepository
	shopRepo       *repository.ShopRepository
}

const extensionPollIntervalMS = 5000

func NewAutomationService(
	automationRepo *repository.AutomationRepository,
	productRepo *repository.ProductRepository,
	shopRepo *repository.ShopRepository,
) *AutomationService {
	return &AutomationService{
		automationRepo: automationRepo,
		productRepo:    productRepo,
		shopRepo:       shopRepo,
	}
}

func (s *AutomationService) CreateJob(userID uint, req *dto.CreateAutomationJobRequest) (*model.AutomationJob, error) {
	if _, err := s.shopRepo.FindByID(req.ShopID); err != nil {
		return nil, fmt.Errorf("shop not found: %w", err)
	}

	rateLimit := req.RateLimit
	if rateLimit <= 0 {
		rateLimit = 30
	}

	jobStatus := model.AutomationJobStatusPending
	if req.RequiresConfirmation {
		jobStatus = model.AutomationJobStatusAwaitConfirm
	}
	if req.DryRun {
		jobStatus = model.AutomationJobStatusDryRunCompleted
	}

	job := &model.AutomationJob{
		ShopID:               req.ShopID,
		CreatedBy:            userID,
		JobType:              req.JobType,
		Status:               jobStatus,
		DryRun:               req.DryRun,
		RequiresConfirmation: req.RequiresConfirmation,
		RateLimit:            rateLimit,
		TotalItems:           len(req.Items),
	}
	if req.DryRun {
		now := time.Now()
		job.CompletedAt = &now
	}

	items := make([]model.AutomationJobItem, 0, len(req.Items))
	for _, reqItem := range req.Items {
		item := model.AutomationJobItem{
			SourceSKU:   reqItem.SourceSKU,
			TargetPrice: reqItem.TargetPrice,

			OverallStatus:     model.AutomationStepStatusPending,
			StepExitStatus:    model.AutomationStepStatusPending,
			StepRepriceStatus: model.AutomationStepStatusPending,
			StepReaddStatus:   model.AutomationStepStatusPending,
		}

		product, err := s.productRepo.FindBySourceSKU(req.ShopID, reqItem.SourceSKU)
		if err == nil {
			item.ProductID = &product.ID
		}

		if req.DryRun {
			item.OverallStatus = model.AutomationStepStatusSkipped
			item.StepExitStatus = model.AutomationStepStatusSkipped
			item.StepRepriceStatus = model.AutomationStepStatusSkipped
			item.StepReaddStatus = model.AutomationStepStatusSkipped
		}

		items = append(items, item)
	}

	if req.DryRun {
		job.SuccessItems = len(items)
	}

	if err := s.automationRepo.CreateJobWithItems(job, items); err != nil {
		return nil, fmt.Errorf("failed to create automation job: %w", err)
	}

	payload := map[string]interface{}{
		"dry_run":    req.DryRun,
		"item_count": len(req.Items),
		"rate_limit": rateLimit,
		"job_type":   req.JobType,
	}
	payloadBytes, _ := json.Marshal(payload)

	eventType := "job_created"
	eventMessage := "automation job created"
	if req.DryRun {
		eventType = "job_dry_run_completed"
		eventMessage = "dry-run completed, no real execution"
	}

	event := &model.AutomationJobEvent{
		JobID:     job.ID,
		EventType: eventType,
		Message:   eventMessage,
		Payload:   payloadBytes,
		CreatedBy: &userID,
	}
	if err := s.automationRepo.CreateJobEvent(event); err != nil {
		return nil, fmt.Errorf("failed to create automation job event: %w", err)
	}

	return s.automationRepo.FindJobByIDAndShop(job.ID, req.ShopID)
}

func (s *AutomationService) CreateSyncShopActionsJob(userID uint, shopID uint) (*model.AutomationJob, error) {
	job := &model.AutomationJob{
		ShopID:     shopID,
		CreatedBy:  userID,
		JobType:    model.AutomationJobTypeSyncShopActions,
		Status:     model.AutomationJobStatusPending,
		RateLimit:  1,
		TotalItems: 1,
	}
	items := []model.AutomationJobItem{{
		SourceSKU:         "__sync_shop_actions__",
		TargetPrice:       0.01,
		OverallStatus:     model.AutomationStepStatusPending,
		StepExitStatus:    model.AutomationStepStatusPending,
		StepRepriceStatus: model.AutomationStepStatusPending,
		StepReaddStatus:   model.AutomationStepStatusPending,
	}}
	if err := s.automationRepo.CreateJobWithItems(job, items); err != nil {
		return nil, err
	}
	return s.automationRepo.FindJobByIDAndShop(job.ID, shopID)
}

func (s *AutomationService) CreateSyncActionProductsJob(userID uint, shopID uint, promotionActionID uint, sourceActionID string) (*model.AutomationJob, error) {
	job := &model.AutomationJob{
		ShopID:     shopID,
		CreatedBy:  userID,
		JobType:    model.AutomationJobTypeSyncActionProducts,
		Status:     model.AutomationJobStatusPending,
		RateLimit:  1,
		TotalItems: 1,
	}
	items := []model.AutomationJobItem{{
		SourceSKU:         fmt.Sprintf("__sync_action_products__:%d", promotionActionID),
		TargetPrice:       0.01,
		OverallStatus:     model.AutomationStepStatusPending,
		StepExitStatus:    model.AutomationStepStatusPending,
		StepRepriceStatus: model.AutomationStepStatusPending,
		StepReaddStatus:   model.AutomationStepStatusPending,
	}}
	if err := s.automationRepo.CreateJobWithItems(job, items); err != nil {
		return nil, err
	}
	meta := map[string]interface{}{
		"promotion_action_id": promotionActionID,
		"source_action_id":    sourceActionID,
	}
	if err := s.automationRepo.CreateArtifact(job.ID, "sync_action_products_meta", meta); err != nil {
		return nil, err
	}
	return s.automationRepo.FindJobByIDAndShop(job.ID, shopID)
}

func (s *AutomationService) WaitForJobCompletion(jobID uint, timeout time.Duration) (*model.AutomationJob, error) {
	deadline := time.Now().Add(timeout)
	for {
		job, err := s.automationRepo.FindJobByID(jobID)
		if err != nil {
			return nil, err
		}
		switch job.Status {
		case model.AutomationJobStatusSuccess, model.AutomationJobStatusPartialSuccess, model.AutomationJobStatusFailed, model.AutomationJobStatusCanceled:
			return job, nil
		}
		if time.Now().After(deadline) {
			return job, fmt.Errorf("job timeout")
		}
		time.Sleep(800 * time.Millisecond)
	}
}

func (s *AutomationService) GetLatestArtifact(jobID uint, artifactType string) (*model.AutomationArtifact, error) {
	return s.automationRepo.FindLatestArtifactByJob(jobID, artifactType)
}

func (s *AutomationService) FindLatestCompletedSyncShopActionsJob(shopID uint) (*model.AutomationJob, error) {
	job, err := s.automationRepo.FindLatestJobByShopAndTypesAndStatuses(
		shopID,
		[]string{model.AutomationJobTypeSyncShopActions},
		[]string{model.AutomationJobStatusSuccess, model.AutomationJobStatusPartialSuccess},
	)
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (s *AutomationService) ListJobs(req *dto.AutomationJobListRequest) ([]model.AutomationJob, int64, error) {
	return s.automationRepo.FindWithFilters(req.ShopID, req.Status, req.Page, req.PageSize)
}

func (s *AutomationService) GetJobDetail(shopID, jobID uint) (*model.AutomationJob, error) {
	return s.automationRepo.FindJobByIDAndShop(jobID, shopID)
}

func (s *AutomationService) AgentHeartbeat(req *dto.AgentHeartbeatRequest) (*model.AutomationAgent, error) {
	capabilities, _ := json.Marshal(req.Capabilities)
	return s.automationRepo.UpsertAgentByKey(req.AgentKey, req.Name, req.Hostname, capabilities)
}

func (s *AutomationService) AgentPoll(req *dto.AgentPollRequest) (*model.AutomationJob, error) {
	agent, err := s.automationRepo.FindAgentByKey(req.AgentKey)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	candidates, err := s.automationRepo.ListPendingJobsByTypes(agentSupportedJobTypes(), 100)
	if err != nil {
		return nil, err
	}

	var job *model.AutomationJob
	for _, candidate := range candidates {
		allow, allowErr := s.canAgentAcquireJob(candidate.ShopID)
		if allowErr != nil {
			if allowErr == gorm.ErrRecordNotFound {
				continue
			}
			return nil, allowErr
		}
		if !allow {
			continue
		}

		claimedJob, claimErr := s.automationRepo.AcquirePendingJobByIDForAgent(candidate.ID, agent.ID)
		if claimErr != nil {
			if claimErr == gorm.ErrRecordNotFound {
				continue
			}
			return nil, claimErr
		}
		job = claimedJob
		break
	}
	if job == nil {
		return nil, nil
	}

	payload := map[string]interface{}{
		"agent_id":   agent.ID,
		"agent_key":  agent.AgentKey,
		"job_status": model.AutomationJobStatusRunning,
	}
	payloadBytes, _ := json.Marshal(payload)
	event := &model.AutomationJobEvent{
		JobID:     job.ID,
		EventType: "job_assigned",
		Message:   "job assigned to agent",
		Payload:   payloadBytes,
	}
	_ = s.automationRepo.CreateJobEvent(event)

	return job, nil
}

func (s *AutomationService) AgentReport(req *dto.AgentReportRequest) error {
	job, err := s.automationRepo.FindJobByID(req.JobID)
	if err != nil {
		return fmt.Errorf("job not found")
	}

	if job.Status != model.AutomationJobStatusRunning {
		return fmt.Errorf("job is not running")
	}

	results := make([]model.AutomationJobItem, 0, len(req.Results))
	for _, result := range req.Results {
		results = append(results, model.AutomationJobItem{
			SourceSKU:         result.SourceSKU,
			OverallStatus:     normalizeStepStatus(result.OverallStatus),
			StepExitStatus:    normalizeStepStatus(result.StepExitStatus),
			StepRepriceStatus: normalizeStepStatus(result.StepRepriceStatus),
			StepReaddStatus:   normalizeStepStatus(result.StepReaddStatus),
			StepExitError:     result.StepExitError,
			StepRepriceError:  result.StepRepriceError,
			StepReaddError:    result.StepReaddError,
		})
	}

	targetStatus := model.AutomationJobStatusSuccess
	switch req.Status {
	case model.AutomationJobStatusSuccess:
		targetStatus = model.AutomationJobStatusSuccess
	case model.AutomationJobStatusPartialSuccess:
		targetStatus = model.AutomationJobStatusPartialSuccess
	case model.AutomationJobStatusFailed:
		targetStatus = model.AutomationJobStatusFailed
	default:
		return fmt.Errorf("invalid report status")
	}

	if err := s.automationRepo.UpdateJobAndItemsByReport(req.JobID, targetStatus, results); err != nil {
		return fmt.Errorf("failed to update report: %w", err)
	}

	if len(req.Meta) > 0 {
		artifactType := artifactTypeForJob(job.JobType, "agent_payload")
		_ = s.automationRepo.CreateArtifact(req.JobID, artifactType, req.Meta)
	}

	payload := map[string]interface{}{
		"job_id": req.JobID,
		"status": req.Status,
		"count":  len(req.Results),
	}
	payloadBytes, _ := json.Marshal(payload)
	event := &model.AutomationJobEvent{
		JobID:     req.JobID,
		EventType: "job_reported",
		Message:   "agent reported execution result",
		Payload:   payloadBytes,
	}
	_ = s.automationRepo.CreateJobEvent(event)

	return nil
}

func (s *AutomationService) ExtensionRegister(userID uint, req *dto.ExtensionRegisterRequest) (*dto.ExtensionRegisterResponse, error) {
	agentKey := extensionAgentKey(userID, req.ShopID, req.ExtensionID)
	agentName := strings.TrimSpace(req.Name)
	if agentName == "" {
		agentName = "Chrome Extension"
	}

	capabilities := map[string]interface{}{
		"kind":         "chrome_extension",
		"user_id":      userID,
		"shop_id":      req.ShopID,
		"extension_id": req.ExtensionID,
		"version":      req.Version,
	}
	capabilityBytes, _ := json.Marshal(capabilities)
	hostname := fmt.Sprintf("shop-%d", req.ShopID)

	if _, err := s.automationRepo.UpsertAgentByKey(agentKey, agentName, hostname, capabilityBytes); err != nil {
		return nil, err
	}

	return &dto.ExtensionRegisterResponse{
		AgentKey:       agentKey,
		PollIntervalMS: extensionPollIntervalMS,
	}, nil
}

func (s *AutomationService) ExtensionPoll(userID uint, req *dto.ExtensionPollRequest) (*model.AutomationJob, error) {
	if _, err := s.ExtensionRegister(userID, &dto.ExtensionRegisterRequest{
		ShopID:      req.ShopID,
		ExtensionID: req.ExtensionID,
		Name:        "Chrome Extension",
	}); err != nil {
		return nil, err
	}

	mode, err := s.resolveShopExecutionEngineMode(req.ShopID)
	if err != nil {
		return nil, err
	}
	if !shouldExtensionAcquire(mode) {
		return nil, nil
	}

	agentKey := extensionAgentKey(userID, req.ShopID, req.ExtensionID)
	agent, err := s.automationRepo.FindAgentByKey(agentKey)
	if err != nil {
		return nil, fmt.Errorf("extension not registered: %w", err)
	}

	job, err := s.automationRepo.AcquirePendingJobForShop(req.ShopID, extensionSupportedJobTypes(), &agent.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	payload := map[string]interface{}{
		"user_id":      userID,
		"shop_id":      req.ShopID,
		"extension_id": req.ExtensionID,
		"job_status":   model.AutomationJobStatusRunning,
	}
	payloadBytes, _ := json.Marshal(payload)
	event := &model.AutomationJobEvent{
		JobID:     job.ID,
		EventType: "job_assigned_extension",
		Message:   "job assigned to browser extension",
		Payload:   payloadBytes,
		CreatedBy: &userID,
	}
	_ = s.automationRepo.CreateJobEvent(event)

	return job, nil
}

func (s *AutomationService) ExtensionReport(userID uint, req *dto.ExtensionReportRequest) error {
	job, err := s.automationRepo.FindJobByID(req.JobID)
	if err != nil {
		return fmt.Errorf("job not found")
	}
	if job.ShopID != req.ShopID {
		return fmt.Errorf("job does not belong to shop")
	}
	if job.Status != model.AutomationJobStatusRunning {
		return fmt.Errorf("job is not running")
	}
	agentKey := extensionAgentKey(userID, req.ShopID, req.ExtensionID)
	agent, err := s.automationRepo.FindAgentByKey(agentKey)
	if err != nil {
		return fmt.Errorf("extension not registered")
	}
	if err := validateJobAssignedAgent(job, agent.ID); err != nil {
		return err
	}

	results := make([]model.AutomationJobItem, 0, len(req.Results))
	for _, result := range req.Results {
		results = append(results, model.AutomationJobItem{
			SourceSKU:         result.SourceSKU,
			OverallStatus:     normalizeStepStatus(result.OverallStatus),
			StepExitStatus:    normalizeStepStatus(result.StepExitStatus),
			StepRepriceStatus: normalizeStepStatus(result.StepRepriceStatus),
			StepReaddStatus:   normalizeStepStatus(result.StepReaddStatus),
			StepExitError:     result.StepExitError,
			StepRepriceError:  result.StepRepriceError,
			StepReaddError:    result.StepReaddError,
		})
	}

	targetStatus := model.AutomationJobStatusSuccess
	switch req.Status {
	case model.AutomationJobStatusSuccess:
		targetStatus = model.AutomationJobStatusSuccess
	case model.AutomationJobStatusPartialSuccess:
		targetStatus = model.AutomationJobStatusPartialSuccess
	case model.AutomationJobStatusFailed:
		targetStatus = model.AutomationJobStatusFailed
	default:
		return fmt.Errorf("invalid report status")
	}

	if err := s.automationRepo.UpdateJobAndItemsByReport(req.JobID, targetStatus, results); err != nil {
		return fmt.Errorf("failed to update report: %w", err)
	}

	if len(req.Meta) > 0 {
		artifactType := artifactTypeForJob(job.JobType, "extension_payload")
		_ = s.automationRepo.CreateArtifact(req.JobID, artifactType, req.Meta)
	}

	payload := map[string]interface{}{
		"job_id":       req.JobID,
		"status":       req.Status,
		"count":        len(req.Results),
		"extension_id": req.ExtensionID,
	}
	payloadBytes, _ := json.Marshal(payload)
	event := &model.AutomationJobEvent{
		JobID:     req.JobID,
		EventType: "job_reported_extension",
		Message:   "browser extension reported execution result",
		Payload:   payloadBytes,
		CreatedBy: &userID,
	}
	_ = s.automationRepo.CreateJobEvent(event)

	return nil
}

func (s *AutomationService) ExtensionRepriceProduct(shopID uint, sourceSKU string, newPrice float64) error {
	sku := strings.TrimSpace(sourceSKU)
	if sku == "" {
		return fmt.Errorf("invalid source sku")
	}
	if newPrice <= 0 {
		return fmt.Errorf("invalid new price")
	}

	shop, err := s.shopRepo.GetWithCredentials(shopID)
	if err != nil {
		return fmt.Errorf("shop not found: %w", err)
	}
	product, err := s.productRepo.FindBySourceSKU(shopID, sku)
	if err != nil {
		return fmt.Errorf("product not found for source sku: %s", sku)
	}

	client := ozon.NewClient(shop.ClientID, shop.ApiKey)
	priceStr := fmt.Sprintf("%.2f", newPrice)
	if err := client.UpdateSinglePrice(product.OzonProductID, priceStr, "", ""); err != nil {
		return fmt.Errorf("failed to update ozon price: %w", err)
	}

	if err := s.productRepo.UpdatePrice(product.ID, newPrice); err != nil {
		return fmt.Errorf("failed to update local price: %w", err)
	}

	return nil
}

func (s *AutomationService) ConfirmJob(userID, shopID, jobID uint) error {
	job, err := s.automationRepo.FindJobByIDAndShop(jobID, shopID)
	if err != nil {
		return err
	}
	if job.Status != model.AutomationJobStatusAwaitConfirm {
		return fmt.Errorf("job is not awaiting confirmation")
	}

	if err := s.automationRepo.UpdateJobStatus(job.ID, model.AutomationJobStatusPending); err != nil {
		return err
	}

	return s.createSimpleEvent(job.ID, "job_confirmed", "job confirmed to continue", &userID)
}

func (s *AutomationService) CancelJob(userID, shopID, jobID uint) error {
	job, err := s.automationRepo.FindJobByIDAndShop(jobID, shopID)
	if err != nil {
		return err
	}
	if job.Status == model.AutomationJobStatusSuccess || job.Status == model.AutomationJobStatusFailed || job.Status == model.AutomationJobStatusCanceled {
		return fmt.Errorf("job already completed")
	}

	if err := s.automationRepo.UpdateJobStatus(job.ID, model.AutomationJobStatusCanceled); err != nil {
		return err
	}

	return s.createSimpleEvent(job.ID, "job_canceled", "job canceled by user", &userID)
}

func (s *AutomationService) RetryFailedItems(userID, shopID, jobID uint) error {
	job, err := s.automationRepo.FindJobByIDAndShop(jobID, shopID)
	if err != nil {
		return err
	}

	if job.Status != model.AutomationJobStatusFailed && job.Status != model.AutomationJobStatusPartialSuccess {
		return fmt.Errorf("job does not support retry in current status")
	}

	hasFailed := false
	for _, item := range job.Items {
		if item.OverallStatus == model.AutomationStepStatusFailed {
			hasFailed = true
			break
		}
	}
	if !hasFailed {
		return fmt.Errorf("no failed items to retry")
	}

	if err := s.automationRepo.ResetFailedItemsForRetry(job.ID); err != nil {
		return err
	}

	return s.createSimpleEvent(job.ID, "job_retry_failed_items", "retry failed items requested", &userID)
}

func (s *AutomationService) ListEvents(req *dto.AutomationEventListRequest) ([]model.AutomationJobEvent, int64, error) {
	return s.automationRepo.ListEventsByShop(req.ShopID, req.JobID, req.Page, req.PageSize)
}

func (s *AutomationService) ListAgents() ([]model.AutomationAgent, error) {
	staleBefore := time.Now().Add(-90 * time.Second)
	_ = s.automationRepo.MarkStaleAgentsOffline(staleBefore)

	agents, err := s.automationRepo.ListAgents()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	for index := range agents {
		agent := &agents[index]
		if agent.LastHeartbeatAt == nil {
			agent.Status = model.AutomationAgentStatusOffline
			continue
		}
		if now.Sub(*agent.LastHeartbeatAt) > 90*time.Second {
			agent.Status = model.AutomationAgentStatusOffline
			continue
		}
		agent.Status = model.AutomationAgentStatusOnline
	}

	return agents, nil
}

func (s *AutomationService) GetExtensionStatus() ([]dto.ExtensionStatusItem, error) {
	shops, err := s.shopRepo.FindAll()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	items := make([]dto.ExtensionStatusItem, 0, len(shops))
	for _, shop := range shops {
		mode := normalizeShopEngineMode(shop.ExecutionEngineMode)
		item := dto.ExtensionStatusItem{
			ShopID:        shop.ID,
			ShopName:      shop.Name,
			ExecutionMode: mode,
			AgentStatus:   model.AutomationAgentStatusOffline,
		}

		if agent, agentErr := s.automationRepo.FindLatestExtensionAgentByShop(shop.ID); agentErr == nil {
			item.ExtensionAgentID = &agent.ID
			item.AgentKey = agent.AgentKey
			item.LastHeartbeatAt = FormatAutomationTime(agent.LastHeartbeatAt)

			isOnline := agent.LastHeartbeatAt != nil && now.Sub(*agent.LastHeartbeatAt) <= 90*time.Second
			if isOnline {
				item.AgentStatus = model.AutomationAgentStatusOnline
			}
		}

		if latestJob, jobErr := s.automationRepo.FindLatestJobByShopAndTypes(shop.ID, extensionSupportedJobTypes()); jobErr == nil {
			item.LatestJobID = &latestJob.ID
			item.LatestJobType = latestJob.JobType
			item.LatestJobStatus = latestJob.Status
			item.LastRunAt = FormatAutomationTime(latestJob.CompletedAt)
			if item.LastRunAt == nil {
				item.LastRunAt = FormatAutomationTime(&latestJob.UpdatedAt)
			}
			if latestJob.Status == model.AutomationJobStatusFailed || latestJob.Status == model.AutomationJobStatusPartialSuccess {
				if latestJob.ErrorMessage != "" {
					item.LastError = latestJob.ErrorMessage
				} else if failedErr, failedItemErr := s.automationRepo.FindLatestFailedItemError(latestJob.ID); failedItemErr == nil {
					item.LastError = failedErr
				}
			}
		}

		items = append(items, item)
	}

	return items, nil
}

func (s *AutomationService) createSimpleEvent(jobID uint, eventType, message string, createdBy *uint) error {
	payloadBytes, _ := json.Marshal(map[string]interface{}{})
	event := &model.AutomationJobEvent{
		JobID:     jobID,
		EventType: eventType,
		Message:   message,
		Payload:   payloadBytes,
		CreatedBy: createdBy,
	}
	return s.automationRepo.CreateJobEvent(event)
}

func normalizeStepStatus(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case model.AutomationStepStatusSuccess:
		return model.AutomationStepStatusSuccess
	case model.AutomationStepStatusFailed:
		return model.AutomationStepStatusFailed
	default:
		return model.AutomationStepStatusSkipped
	}
}

func artifactTypeForJob(jobType string, fallback string) string {
	switch jobType {
	case model.AutomationJobTypeSyncShopActions:
		return "shop_actions_snapshot"
	case model.AutomationJobTypeSyncActionProducts:
		return "action_products_snapshot"
	case model.AutomationJobTypeShopActionDeclare, model.AutomationJobTypeShopActionRemove:
		return "shop_action_snapshot"
	case model.AutomationJobTypePromoUnifiedEnroll, model.AutomationJobTypePromoUnifiedRemove:
		return "promo_unified_snapshot"
	default:
		return fallback
	}
}

func extensionSupportedJobTypes() []string {
	return []string{
		model.AutomationJobTypeSyncShopActions,
		model.AutomationJobTypeSyncActionProducts,
		model.AutomationJobTypeShopActionDeclare,
		model.AutomationJobTypeShopActionRemove,
		model.AutomationJobTypePromoUnifiedEnroll,
		model.AutomationJobTypePromoUnifiedRemove,
		model.AutomationJobTypeRemoveRepriceReadd,
	}
}

func agentSupportedJobTypes() []string {
	return extensionSupportedJobTypes()
}

func (s *AutomationService) canAgentAcquireJob(shopID uint) (bool, error) {
	mode, err := s.resolveShopExecutionEngineMode(shopID)
	if err != nil {
		return false, err
	}

	if mode != model.ShopExecutionEngineAuto {
		return shouldAgentAcquire(mode, false), nil
	}

	staleAfter := time.Now().Add(-90 * time.Second)
	hasOnlineExtension, checkErr := s.automationRepo.HasOnlineExtensionForShop(shopID, staleAfter)
	if checkErr != nil {
		return false, checkErr
	}
	return shouldAgentAcquire(mode, hasOnlineExtension), nil
}

func (s *AutomationService) resolveShopExecutionEngineMode(shopID uint) (string, error) {
	mode, err := s.shopRepo.GetExecutionEngineMode(shopID)
	if err != nil {
		return "", err
	}
	return normalizeShopEngineMode(mode), nil
}

func normalizeShopEngineMode(mode string) string {
	normalized := strings.TrimSpace(strings.ToLower(mode))
	switch normalized {
	case model.ShopExecutionEngineExtension:
		return model.ShopExecutionEngineExtension
	case model.ShopExecutionEngineAgent:
		return model.ShopExecutionEngineAgent
	default:
		return model.ShopExecutionEngineAuto
	}
}

func shouldExtensionAcquire(mode string) bool {
	return mode == model.ShopExecutionEngineAuto || mode == model.ShopExecutionEngineExtension
}

func shouldAgentAcquire(mode string, hasOnlineExtension bool) bool {
	switch mode {
	case model.ShopExecutionEngineAgent:
		return true
	case model.ShopExecutionEngineExtension:
		return false
	default:
		return !hasOnlineExtension
	}
}

func validateJobAssignedAgent(job *model.AutomationJob, agentID uint) error {
	if job == nil {
		return fmt.Errorf("job not found")
	}
	if job.AssignedAgentID == nil || *job.AssignedAgentID != agentID {
		return fmt.Errorf("job is not assigned to this extension")
	}
	return nil
}

func extensionAgentKey(userID, shopID uint, extensionID string) string {
	safe := sanitizeExtensionID(extensionID)
	if safe == "" {
		safe = "default"
	}
	return fmt.Sprintf("ext:%d:%d:%s", userID, shopID, safe)
}

func sanitizeExtensionID(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(trimmed))
	for _, ch := range trimmed {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			b.WriteRune(ch)
			continue
		}
		b.WriteRune('_')
	}
	result := b.String()
	if len(result) > 60 {
		return result[:60]
	}
	return result
}

func FormatAutomationTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format("2006-01-02 15:04:05")
	return &formatted
}
