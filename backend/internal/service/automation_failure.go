package service

import (
	"fmt"
	"strings"

	"ozon-manager/internal/model"
)

func automationJobFailureMessage(job *model.AutomationJob, fallback string) string {
	if job == nil {
		return fallback
	}
	if trimmed := strings.TrimSpace(job.ErrorMessage); trimmed != "" {
		return decorateAutomationFailureMessage(job.JobType, firstAutomationFailureSourceSKU(job.Items), trimmed)
	}
	for _, item := range job.Items {
		if trimmed := firstNonEmptyServiceTrimmed(item.StepExitError, item.StepRepriceError, item.StepReaddError); trimmed != "" {
			return decorateAutomationFailureMessage(job.JobType, item.SourceSKU, trimmed)
		}
	}
	return fallback
}

func automationJobFailureMessageForSourceSKU(job *model.AutomationJob, sourceSKU, fallback string) string {
	if job == nil {
		return fallback
	}
	normalizedSourceSKU := strings.TrimSpace(sourceSKU)
	if normalizedSourceSKU != "" {
		for _, item := range job.Items {
			if strings.TrimSpace(item.SourceSKU) != normalizedSourceSKU {
				continue
			}
			if trimmed := firstNonEmptyServiceTrimmed(item.StepExitError, item.StepRepriceError, item.StepReaddError); trimmed != "" {
				return decorateAutomationFailureMessage(job.JobType, normalizedSourceSKU, trimmed)
			}
		}
	}
	if trimmed := strings.TrimSpace(job.ErrorMessage); trimmed != "" {
		return decorateAutomationFailureMessage(job.JobType, firstNonEmptyServiceTrimmed(normalizedSourceSKU, firstAutomationFailureSourceSKU(job.Items)), trimmed)
	}
	if normalizedSourceSKU == "" {
		for _, item := range job.Items {
			if trimmed := firstNonEmptyServiceTrimmed(item.StepExitError, item.StepRepriceError, item.StepReaddError); trimmed != "" {
				return decorateAutomationFailureMessage(job.JobType, item.SourceSKU, trimmed)
			}
		}
		return fallback
	}
	return decorateAutomationFailureMessage(job.JobType, normalizedSourceSKU, fallback)
}

func decorateAutomationFailureMessage(jobType, sourceSKU, message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ""
	}
	if !isSearchCPOAutomationJobType(jobType) {
		return trimmed
	}
	normalizedSourceSKU := strings.TrimSpace(sourceSKU)
	if normalizedSourceSKU == "" || strings.Contains(trimmed, "source_sku=") {
		return trimmed
	}
	return fmt.Sprintf("source_sku=%s: %s", normalizedSourceSKU, trimmed)
}

func isSearchCPOAutomationJobType(jobType string) bool {
	switch strings.TrimSpace(jobType) {
	case model.AutomationJobTypeSyncSearchCPOAvailability,
		model.AutomationJobTypeSearchCPOEnableProducts,
		model.AutomationJobTypeSearchCPOBatchEnableMorkovsk:
		return true
	default:
		return false
	}
}

func firstAutomationFailureSourceSKU(items []model.AutomationJobItem) string {
	for _, item := range items {
		if trimmed := strings.TrimSpace(item.SourceSKU); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonEmptyServiceTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
