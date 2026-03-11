package service

import (
	"testing"

	"ozon-manager/internal/model"
)

func TestAutomationJobFailureMessagePrefersJobErrorMessage(t *testing.T) {
	t.Parallel()

	job := &model.AutomationJob{
		ErrorMessage: "插件不支持该任务类型: sync_action_candidates",
		Items: []model.AutomationJobItem{{
			StepExitError: "不会被使用",
		}},
	}

	got := automationJobFailureMessage(job, "fallback")
	if got != "插件不支持该任务类型: sync_action_candidates" {
		t.Fatalf("automationJobFailureMessage() = %q", got)
	}
}

func TestAutomationJobFailureMessageFallsBackToItemErrors(t *testing.T) {
	t.Parallel()

	job := &model.AutomationJob{
		Items: []model.AutomationJobItem{{
			StepReaddError: "获取候选商品失败: 403 Forbidden",
		}},
	}

	got := automationJobFailureMessage(job, "fallback")
	if got != "获取候选商品失败: 403 Forbidden" {
		t.Fatalf("automationJobFailureMessage() = %q", got)
	}
}

func TestAutomationJobFailureMessageFallsBackWhenNoErrors(t *testing.T) {
	t.Parallel()

	got := automationJobFailureMessage(&model.AutomationJob{}, "fallback")
	if got != "fallback" {
		t.Fatalf("automationJobFailureMessage() = %q", got)
	}
}
