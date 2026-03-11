package service

import (
	"strings"

	"ozon-manager/internal/model"
)

func automationJobFailureMessage(job *model.AutomationJob, fallback string) string {
	if job == nil {
		return fallback
	}
	if trimmed := strings.TrimSpace(job.ErrorMessage); trimmed != "" {
		return trimmed
	}
	for _, item := range job.Items {
		if trimmed := firstNonEmptyServiceTrimmed(item.StepExitError, item.StepRepriceError, item.StepReaddError); trimmed != "" {
			return trimmed
		}
	}
	return fallback
}

func firstNonEmptyServiceTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
