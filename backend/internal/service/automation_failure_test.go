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

func TestAutomationJobFailureMessageDecoratesSearchCPOJobErrors(t *testing.T) {
	t.Parallel()

	job := &model.AutomationJob{
		JobType:      model.AutomationJobTypeSyncSearchCPOAvailability,
		ErrorMessage: "未匹配到 search_promo_availability 响应 (requested_sku=3323213720, parser_revision=2026-03-20-a)",
		Items: []model.AutomationJobItem{{
			SourceSKU: "offer-1",
		}},
	}

	got := automationJobFailureMessage(job, "fallback")
	want := "source_sku=offer-1: 未匹配到 search_promo_availability 响应 (requested_sku=3323213720, parser_revision=2026-03-20-a)"
	if got != want {
		t.Fatalf("automationJobFailureMessage() = %q, want %q", got, want)
	}
}

func TestAutomationJobFailureMessageAvoidsDuplicateSearchCPOSourceSKUPrefix(t *testing.T) {
	t.Parallel()

	job := &model.AutomationJob{
		JobType: model.AutomationJobTypeSyncSearchCPOAvailability,
		Items: []model.AutomationJobItem{{
			SourceSKU:     "offer-1",
			StepExitError: "source_sku=offer-1: 未匹配到 search_promo_availability 响应",
		}},
	}

	got := automationJobFailureMessage(job, "fallback")
	want := "source_sku=offer-1: 未匹配到 search_promo_availability 响应"
	if got != want {
		t.Fatalf("automationJobFailureMessage() = %q, want %q", got, want)
	}
}

func TestAutomationJobFailureMessageForSourceSKUPrefersMatchingItem(t *testing.T) {
	t.Parallel()

	job := &model.AutomationJob{
		JobType: model.AutomationJobTypeSearchCPOEnableProducts,
		Items: []model.AutomationJobItem{
			{SourceSKU: "offer-1", StepReaddError: "source_sku=offer-1: 未匹配到 Search CPO 开启响应: 3371928661"},
			{SourceSKU: "offer-2", StepReaddError: "source_sku=offer-2: 未匹配到 Search CPO 开启响应: 3352634622"},
		},
	}

	got := automationJobFailureMessageForSourceSKU(job, "offer-2", "fallback")
	want := "source_sku=offer-2: 未匹配到 Search CPO 开启响应: 3352634622"
	if got != want {
		t.Fatalf("automationJobFailureMessageForSourceSKU() = %q, want %q", got, want)
	}
}

func TestAutomationJobFailureMessageForSourceSKUDecoratesJobErrorWithRequestedSource(t *testing.T) {
	t.Parallel()

	job := &model.AutomationJob{
		JobType:      model.AutomationJobTypeSearchCPOEnableProducts,
		ErrorMessage: "未匹配到 Search CPO 开启响应: 3371928661",
		Items: []model.AutomationJobItem{{
			SourceSKU: "offer-1",
		}},
	}

	got := automationJobFailureMessageForSourceSKU(job, "offer-2", "fallback")
	want := "source_sku=offer-2: 未匹配到 Search CPO 开启响应: 3371928661"
	if got != want {
		t.Fatalf("automationJobFailureMessageForSourceSKU() = %q, want %q", got, want)
	}
}

func TestAutomationJobFailureMessageFallsBackWhenNoErrors(t *testing.T) {
	t.Parallel()

	got := automationJobFailureMessage(&model.AutomationJob{}, "fallback")
	if got != "fallback" {
		t.Fatalf("automationJobFailureMessage() = %q", got)
	}
}
