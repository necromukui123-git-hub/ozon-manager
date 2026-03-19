package service

import (
	"fmt"

	"ozon-manager/internal/model"
)

func (s *AutomationService) CreateSyncSearchCPOAvailabilityJob(userID uint, shopID uint, sourceSKUs []string, meta map[string]interface{}) (*model.AutomationJob, error) {
	return s.createSearchCPOBulkJob(userID, shopID, model.AutomationJobTypeSyncSearchCPOAvailability, sourceSKUs, cloneSearchCPOJobMeta(meta))
}

func (s *AutomationService) CreateSearchCPOEnableProductsJob(userID uint, shopID uint, sourceSKUs []string, meta map[string]interface{}) (*model.AutomationJob, error) {
	return s.createSearchCPOBulkJob(userID, shopID, model.AutomationJobTypeSearchCPOEnableProducts, sourceSKUs, cloneSearchCPOJobMeta(meta))
}

func (s *AutomationService) CreateSearchCPOBatchEnableMorkovskJob(userID uint, shopID uint, sourceSKUs []string, meta map[string]interface{}) (*model.AutomationJob, error) {
	merged := cloneSearchCPOJobMeta(meta)
	if merged == nil {
		merged = map[string]interface{}{}
	}
	merged["target"] = "morkovsk"
	return s.createSearchCPOBulkJob(userID, shopID, model.AutomationJobTypeSearchCPOBatchEnableMorkovsk, sourceSKUs, merged)
}

func (s *AutomationService) createSearchCPOBulkJob(userID uint, shopID uint, jobType string, sourceSKUs []string, meta map[string]interface{}) (*model.AutomationJob, error) {
	skus := uniqueSKUs(sourceSKUs)
	if len(skus) == 0 {
		return nil, fmt.Errorf("没有可处理的 SKU")
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
			TargetPrice:       0.01,
			OverallStatus:     model.AutomationStepStatusPending,
			StepExitStatus:    model.AutomationStepStatusPending,
			StepRepriceStatus: model.AutomationStepStatusPending,
			StepReaddStatus:   model.AutomationStepStatusPending,
		})
	}
	if err := s.automationRepo.CreateJobWithItems(job, items); err != nil {
		return nil, err
	}
	if len(meta) > 0 {
		if err := s.automationRepo.CreateArtifact(job.ID, searchCPOJobMetaArtifact(jobType), meta); err != nil {
			return nil, err
		}
	}
	return s.automationRepo.FindJobByIDAndShop(job.ID, shopID)
}

func searchCPOJobMetaArtifact(jobType string) string {
	switch jobType {
	case model.AutomationJobTypeSearchCPOBatchEnableMorkovsk:
		return "search_cpo_morkovsk_meta"
	default:
		return "search_cpo_meta"
	}
}

func cloneSearchCPOJobMeta(meta map[string]interface{}) map[string]interface{} {
	if len(meta) == 0 {
		return nil
	}
	cloned := make(map[string]interface{}, len(meta))
	for key, value := range meta {
		cloned[key] = value
	}
	return cloned
}
