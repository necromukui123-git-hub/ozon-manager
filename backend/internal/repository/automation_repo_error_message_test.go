package repository

import (
	"testing"

	"ozon-manager/internal/model"
)

func TestDeriveJobErrorMessageReturnsEmptyForSuccess(t *testing.T) {
	t.Parallel()

	got := deriveJobErrorMessage(model.AutomationJobStatusSuccess, []model.AutomationJobItem{{
		StepExitError: "ignored",
	}})
	if got != "" {
		t.Fatalf("deriveJobErrorMessage() = %q, want empty", got)
	}
}

func TestDeriveJobErrorMessageReturnsFirstNonEmptyError(t *testing.T) {
	t.Parallel()

	results := []model.AutomationJobItem{
		{StepExitError: "   "},
		{StepRepriceError: "插件不支持该任务类型: sync_action_candidates"},
		{StepReaddError: "later"},
	}

	got := deriveJobErrorMessage(model.AutomationJobStatusFailed, results)
	if got != "插件不支持该任务类型: sync_action_candidates" {
		t.Fatalf("deriveJobErrorMessage() = %q", got)
	}
}

func TestDeriveJobErrorMessageSupportsPartialSuccess(t *testing.T) {
	t.Parallel()

	got := deriveJobErrorMessage(model.AutomationJobStatusPartialSuccess, []model.AutomationJobItem{{
		StepReaddError: "部分商品失败",
	}})
	if got != "部分商品失败" {
		t.Fatalf("deriveJobErrorMessage() = %q", got)
	}
}
