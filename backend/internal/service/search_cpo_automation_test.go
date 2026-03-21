package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/datatypes"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
)

func TestDeriveSearchCPORuleState(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC)
	falseValue := false
	trueValue := true

	t.Run("state1 when disabled and availability false", func(t *testing.T) {
		t.Parallel()
		state, detectedAt := deriveSearchCPORuleState(model.SearchCPOProduct{
			SearchPromoStatus: "SEARCH_PROMO_STATUS_DISABLED",
			CarrotsStatus:     "CARROTS_STATUS_DISABLED",
			AvailabilityPromo: &falseValue,
		}, "", now)
		if state != model.SearchCPORuleStateState1 {
			t.Fatalf("deriveSearchCPORuleState() = %q, want %q", state, model.SearchCPORuleStateState1)
		}
		if detectedAt != nil {
			t.Fatalf("expected no detectedAt for state1")
		}
	})

	t.Run("state2 when disabled and availability true", func(t *testing.T) {
		t.Parallel()
		state, detectedAt := deriveSearchCPORuleState(model.SearchCPOProduct{
			SearchPromoStatus: "SEARCH_PROMO_STATUS_DISABLED",
			CarrotsStatus:     "CARROTS_STATUS_DISABLED",
			AvailabilityPromo: &trueValue,
		}, "", now)
		if state != model.SearchCPORuleStateState2 {
			t.Fatalf("deriveSearchCPORuleState() = %q, want %q", state, model.SearchCPORuleStateState2)
		}
		if detectedAt == nil || !detectedAt.Equal(now) {
			t.Fatalf("expected detectedAt to be %v, got %v", now, detectedAt)
		}
	})

	t.Run("state2 keeps existing detectedAt", func(t *testing.T) {
		t.Parallel()
		previous := now.Add(-2 * time.Hour)
		state, detectedAt := deriveSearchCPORuleState(model.SearchCPOProduct{
			SearchPromoStatus: "SEARCH_PROMO_STATUS_DISABLED",
			CarrotsStatus:     "CARROTS_STATUS_DISABLED",
			AvailabilityPromo: &trueValue,
			State2DetectedAt:  &previous,
		}, model.SearchCPORuleStateState2, now)
		if state != model.SearchCPORuleStateState2 {
			t.Fatalf("deriveSearchCPORuleState() = %q, want %q", state, model.SearchCPORuleStateState2)
		}
		if detectedAt == nil || !detectedAt.Equal(previous) {
			t.Fatalf("expected detectedAt to stay %v, got %v", previous, detectedAt)
		}
	})

	t.Run("state3 trigger when enabled after state2", func(t *testing.T) {
		t.Parallel()
		previous := now.Add(-time.Hour)
		state, detectedAt := deriveSearchCPORuleState(model.SearchCPOProduct{
			SearchPromoStatus: "SEARCH_PROMO_STATUS_ENABLED",
			State2DetectedAt:  &previous,
		}, model.SearchCPORuleStateState2, now)
		if state != model.SearchCPORuleStateState3Trigger {
			t.Fatalf("deriveSearchCPORuleState() = %q, want %q", state, model.SearchCPORuleStateState3Trigger)
		}
		if detectedAt == nil || !detectedAt.Equal(previous) {
			t.Fatalf("expected detectedAt to stay %v, got %v", previous, detectedAt)
		}
	})

	t.Run("state3 trigger when enabled carrots disabled and availability true without history", func(t *testing.T) {
		t.Parallel()
		state, detectedAt := deriveSearchCPORuleState(model.SearchCPOProduct{
			SearchPromoStatus: "SEARCH_PROMO_STATUS_ENABLED",
			CarrotsStatus:     "CARROTS_STATUS_DISABLED",
			AvailabilityPromo: &trueValue,
		}, model.SearchCPORuleStateOther, now)
		if state != model.SearchCPORuleStateState3Trigger {
			t.Fatalf("deriveSearchCPORuleState() = %q, want %q", state, model.SearchCPORuleStateState3Trigger)
		}
		if detectedAt == nil || !detectedAt.Equal(now) {
			t.Fatalf("expected detectedAt to be %v, got %v", now, detectedAt)
		}
	})

	t.Run("joined wins over other states", func(t *testing.T) {
		t.Parallel()
		joinedAt := now.Add(-30 * time.Minute)
		state, _ := deriveSearchCPORuleState(model.SearchCPOProduct{
			MorkovskJoinedAt: &joinedAt,
			State2DetectedAt: &joinedAt,
		}, model.SearchCPORuleStateState3Trigger, now)
		if state != model.SearchCPORuleStateJoined {
			t.Fatalf("deriveSearchCPORuleState() = %q, want %q", state, model.SearchCPORuleStateJoined)
		}
	})
}

func TestBuildSearchCPOActionSyncFailureMessage(t *testing.T) {
	t.Parallel()

	t.Run("empty result means no failure", func(t *testing.T) {
		t.Parallel()
		if msg := buildSearchCPOActionSyncFailureMessage(&dto.SyncActionsResult{}); msg != "" {
			t.Fatalf("buildSearchCPOActionSyncFailureMessage() = %q, want empty", msg)
		}
	})

	t.Run("pending and partial errors are joined deterministically", func(t *testing.T) {
		t.Parallel()
		msg := buildSearchCPOActionSyncFailureMessage(&dto.SyncActionsResult{
			ShopSyncPending: true,
			PartialErrors: map[string]string{
				"shop":     "shop sync failed",
				"official": "official refresh failed",
			},
		})
		want := "店铺活动同步仍在后台进行; official: official refresh failed; shop: shop sync failed"
		if msg != want {
			t.Fatalf("buildSearchCPOActionSyncFailureMessage() = %q, want %q", msg, want)
		}
	})
}

func TestBuildSearchCPOSKUMeta(t *testing.T) {
	t.Parallel()

	meta := buildSearchCPOSKUMeta([]model.SearchCPOProduct{
		{SourceSKU: "offer-1", SKU: "123456"},
		{SourceSKU: "778899", SKU: ""},
		{SourceSKU: "offer-missing", SKU: ""},
	})
	if meta == nil {
		t.Fatalf("buildSearchCPOSKUMeta() returned nil")
	}

	skuMap, ok := meta["sku_map"].(map[string]string)
	if !ok {
		t.Fatalf("sku_map type = %T, want map[string]string", meta["sku_map"])
	}
	if got := skuMap["offer-1"]; got != "123456" {
		t.Fatalf("sku_map[offer-1] = %q, want %q", got, "123456")
	}
	if got := skuMap["778899"]; got != "778899" {
		t.Fatalf("sku_map[778899] = %q, want %q", got, "778899")
	}
	if _, exists := skuMap["offer-missing"]; exists {
		t.Fatalf("unexpected mapping for offer-missing")
	}
}

func TestBuildSearchCPOSKUMetaFromStates(t *testing.T) {
	t.Parallel()

	states := map[string]*searchCPOAutomationItemState{
		"offer-1": {Product: model.SearchCPOProduct{SourceSKU: "offer-1", SKU: "123456"}},
		"445566":  {Product: model.SearchCPOProduct{SourceSKU: "445566"}},
	}

	meta := buildSearchCPOSKUMetaFromStates([]string{"offer-1", "445566", "missing"}, states)
	if meta == nil {
		t.Fatalf("buildSearchCPOSKUMetaFromStates() returned nil")
	}

	skuMap, ok := meta["sku_map"].(map[string]string)
	if !ok {
		t.Fatalf("sku_map type = %T, want map[string]string", meta["sku_map"])
	}
	if got := skuMap["offer-1"]; got != "123456" {
		t.Fatalf("sku_map[offer-1] = %q, want %q", got, "123456")
	}
	if got := skuMap["445566"]; got != "445566" {
		t.Fatalf("sku_map[445566] = %q, want %q", got, "445566")
	}
	if _, exists := skuMap["missing"]; exists {
		t.Fatalf("unexpected mapping for missing state")
	}
}

func TestIsSearchCPOJoinedRepairState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 22, 8, 0, 0, 0, time.UTC)

	t.Run("joined at marks repair state", func(t *testing.T) {
		t.Parallel()
		if !isSearchCPOJoinedRepairState(&searchCPOAutomationItemState{
			Product: model.SearchCPOProduct{MorkovskJoinedAt: &now},
		}) {
			t.Fatalf("expected joined item to require repair evaluation")
		}
	})

	t.Run("joined rule state also marks repair state", func(t *testing.T) {
		t.Parallel()
		if !isSearchCPOJoinedRepairState(&searchCPOAutomationItemState{
			RuleStateBefore: model.SearchCPORuleStateJoined,
		}) {
			t.Fatalf("expected joined rule state to require repair evaluation")
		}
		if !isSearchCPOJoinedRepairState(&searchCPOAutomationItemState{
			RuleStateAfter: model.SearchCPORuleStateJoined,
		}) {
			t.Fatalf("expected joined target state to require repair evaluation")
		}
	})

	t.Run("non joined state does not trigger repair", func(t *testing.T) {
		t.Parallel()
		if isSearchCPOJoinedRepairState(&searchCPOAutomationItemState{
			RuleStateBefore: model.SearchCPORuleStateState3Trigger,
			RuleStateAfter:  model.SearchCPORuleStateState3Trigger,
			Product: model.SearchCPOProduct{
				SearchPromoStatus: "SEARCH_PROMO_STATUS_ENABLED",
				CarrotsStatus:     "CARROTS_STATUS_DISABLED",
			},
		}) {
			t.Fatalf("unexpected repair state for non-joined product")
		}
	})
}

func TestMarkSearchCPOJoinedRepairSkippedSteps(t *testing.T) {
	t.Parallel()

	state := &searchCPOAutomationItemState{}
	markSearchCPOJoinedRepairSkippedSteps(state)

	if state.EnableStatus != model.SearchCPOItemStatusSkipped {
		t.Fatalf("EnableStatus = %q", state.EnableStatus)
	}
	if state.EnableResult.Message != "已加入 Morkovsk，跳过重复 enable" {
		t.Fatalf("EnableResult.Message = %q", state.EnableResult.Message)
	}
	if state.MorkovskStatus != model.SearchCPOItemStatusSkipped {
		t.Fatalf("MorkovskStatus = %q", state.MorkovskStatus)
	}
	if state.MorkovskResult.Message != "已加入 Morkovsk，跳过重复加入" {
		t.Fatalf("MorkovskResult.Message = %q", state.MorkovskResult.Message)
	}
}

func TestDecodeSearchCPOAvailabilityDiagnostics(t *testing.T) {
	t.Parallel()

	rawPayload := map[string]interface{}{
		"requested_sku":              "3328989428",
		"parser_revision":            "2026-03-20-b",
		"build_revision":             "2026-03-20-b",
		"response_root_keys":         []string{"data", "response_kind", "response_excerpt"},
		"sample_response_keys":       []string{"skuToIsSearchPromoAvailable"},
		"availability_map_key_count": 1,
		"reason_map_key_count":       1,
		"response_http_status":       200,
		"response_http_status_text":  "OK",
		"response_content_type":      "text/html; charset=utf-8",
		"response_parse_error":       "Unexpected token '<'",
		"response_excerpt":           "<!doctype html><html>challenge</html>",
		"response_length":            38,
		"response_kind":              "text",
		"script_result_type":         "object",
		"unavailableReason":          "PROMOTION_UNAVAILABLE_REASON_NO_SALES",
	}
	bytes, err := json.Marshal(rawPayload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	diagnostics := decodeSearchCPOAvailabilityDiagnostics(datatypes.JSON(bytes))
	if diagnostics == nil {
		t.Fatalf("decodeSearchCPOAvailabilityDiagnostics() returned nil")
	}
	if diagnostics.RequestedSKU != "3328989428" {
		t.Fatalf("RequestedSKU = %q", diagnostics.RequestedSKU)
	}
	if diagnostics.ParserRevision != "2026-03-20-b" {
		t.Fatalf("ParserRevision = %q", diagnostics.ParserRevision)
	}
	if diagnostics.BuildRevision != "2026-03-20-b" {
		t.Fatalf("BuildRevision = %q", diagnostics.BuildRevision)
	}
	if diagnostics.ResponseHTTPStatus != 200 {
		t.Fatalf("ResponseHTTPStatus = %d", diagnostics.ResponseHTTPStatus)
	}
	if diagnostics.ResponseContentType != "text/html; charset=utf-8" {
		t.Fatalf("ResponseContentType = %q", diagnostics.ResponseContentType)
	}
	if diagnostics.ResponseParseError != "Unexpected token '<'" {
		t.Fatalf("ResponseParseError = %q", diagnostics.ResponseParseError)
	}
	if diagnostics.ResponseKind != "text" {
		t.Fatalf("ResponseKind = %q", diagnostics.ResponseKind)
	}
	if diagnostics.ScriptResultType != "object" {
		t.Fatalf("ScriptResultType = %q", diagnostics.ScriptResultType)
	}
	if diagnostics.UnavailableReason != "PROMOTION_UNAVAILABLE_REASON_NO_SALES" {
		t.Fatalf("UnavailableReason = %q", diagnostics.UnavailableReason)
	}
	if len(diagnostics.ResponseRootKeys) != 3 {
		t.Fatalf("ResponseRootKeys len = %d", len(diagnostics.ResponseRootKeys))
	}
}

func TestBuildSearchCPOStepResultFromArtifactItemCarriesDiagnostics(t *testing.T) {
	t.Parallel()

	result := buildSearchCPOStepResultFromArtifactItem(searchCPOStepArtifactItem{
		SourceSKU:              "1966971285-N0wL",
		Status:                 model.SearchCPOItemStatusFailed,
		Error:                  "未匹配到 Search CPO 开启响应",
		Message:                "未匹配到 Search CPO 开启响应",
		RequestedSKU:           "3371928661",
		ParserRevision:         "2026-03-21-a",
		BuildRevision:          "2026-03-21-a",
		ResponseRootKeys:       []string{"data", "response_kind"},
		SampleResponseKeys:     []string{"result", "bids"},
		ResponseHTTPStatus:     200,
		ResponseHTTPStatusText: "OK",
		ResponseContentType:    "text/html; charset=utf-8",
		ResponseParseError:     "Unexpected token '<'",
		ResponseExcerpt:        "<!doctype html><html>challenge</html>",
		ResponseLength:         38,
		ResponseKind:           "text",
		ScriptResultType:       "object",
		ResponseItemCount:      0,
	})

	if result.Status != model.SearchCPOItemStatusFailed {
		t.Fatalf("Status = %q", result.Status)
	}
	if result.Diagnostics == nil {
		t.Fatalf("expected diagnostics to be present")
	}
	if result.Diagnostics.RequestedSKU != "3371928661" {
		t.Fatalf("RequestedSKU = %q", result.Diagnostics.RequestedSKU)
	}
	if result.Diagnostics.ParserRevision != "2026-03-21-a" {
		t.Fatalf("ParserRevision = %q", result.Diagnostics.ParserRevision)
	}
	if result.Diagnostics.ResponseHTTPStatus != 200 {
		t.Fatalf("ResponseHTTPStatus = %d", result.Diagnostics.ResponseHTTPStatus)
	}
	if result.Diagnostics.ResponseContentType != "text/html; charset=utf-8" {
		t.Fatalf("ResponseContentType = %q", result.Diagnostics.ResponseContentType)
	}
	if result.Diagnostics.ResponseItemCount != 0 {
		t.Fatalf("ResponseItemCount = %d", result.Diagnostics.ResponseItemCount)
	}
}

func TestDecodeSearchCPOAutomationStepResultNormalizesDiagnostics(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(dto.SearchCPOAutomationStepResult{
		Status:  model.SearchCPOItemStatusFailed,
		Error:   " parser failed ",
		Message: " missing match ",
		Diagnostics: &dto.SearchCPOStepDiagnostics{
			RequestedSKU:           " 3371928661 ",
			ParserRevision:         " 2026-03-21-a ",
			ResponseRootKeys:       []string{" data ", " result "},
			SampleResponseKeys:     []string{" bids "},
			ResponseHTTPStatus:     200,
			ResponseHTTPStatusText: " OK ",
			ResponseContentType:    " text/html; charset=utf-8 ",
			ResponseParseError:     " Unexpected token '<' ",
			ResponseExcerpt:        " <!doctype html> ",
			ResponseLength:         12,
			ResponseKind:           " text ",
			ScriptResultType:       " object ",
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	decoded := decodeSearchCPOAutomationStepResult(datatypes.JSON(raw))
	if decoded.Error != "parser failed" {
		t.Fatalf("Error = %q", decoded.Error)
	}
	if decoded.Message != "missing match" {
		t.Fatalf("Message = %q", decoded.Message)
	}
	if decoded.Diagnostics == nil {
		t.Fatalf("expected diagnostics to be present")
	}
	if decoded.Diagnostics.RequestedSKU != "3371928661" {
		t.Fatalf("RequestedSKU = %q", decoded.Diagnostics.RequestedSKU)
	}
	if len(decoded.Diagnostics.ResponseRootKeys) != 2 {
		t.Fatalf("ResponseRootKeys len = %d", len(decoded.Diagnostics.ResponseRootKeys))
	}
	if decoded.Diagnostics.ResponseHTTPStatusText != "OK" {
		t.Fatalf("ResponseHTTPStatusText = %q", decoded.Diagnostics.ResponseHTTPStatusText)
	}
}

func TestSummarizeSearchCPOActionResultsStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		results []dto.SearchCPORunActionResult
		want    string
	}{
		{
			name: "all skipped",
			results: []dto.SearchCPORunActionResult{
				{Status: model.SearchCPOItemStatusSkipped},
			},
			want: model.SearchCPOItemStatusSkipped,
		},
		{
			name: "mixed success and skipped becomes partial success",
			results: []dto.SearchCPORunActionResult{
				{Status: model.SearchCPOItemStatusSuccess},
				{Status: model.SearchCPOItemStatusSkipped},
			},
			want: model.SearchCPOItemStatusPartialSuccess,
		},
		{
			name: "failed wins",
			results: []dto.SearchCPORunActionResult{
				{Status: model.SearchCPOItemStatusSuccess},
				{Status: model.SearchCPOItemStatusFailed},
			},
			want: model.SearchCPOItemStatusFailed,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := summarizeSearchCPOActionResultsStatus(tc.results); got != tc.want {
				t.Fatalf("summarizeSearchCPOActionResultsStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSummarizeSearchCPOShopExitJobItem(t *testing.T) {
	t.Parallel()

	t.Run("404 not found becomes skipped", func(t *testing.T) {
		t.Parallel()
		status, message := summarizeSearchCPOShopExitJobItem(model.AutomationJobItem{
			StepExitStatus: model.AutomationStepStatusFailed,
			StepExitError:  `API error (status 404): {"code":5,"message":"rpc error: code = NotFound desc = Resource not found"}`,
		}, "")
		if status != model.SearchCPOItemStatusSkipped {
			t.Fatalf("status = %q, want %q", status, model.SearchCPOItemStatusSkipped)
		}
		if message == "" {
			t.Fatalf("expected skipped message to be preserved")
		}
	})

	t.Run("not in action stays skipped", func(t *testing.T) {
		t.Parallel()
		status, message := summarizeSearchCPOShopExitJobItem(model.AutomationJobItem{
			StepExitStatus: model.AutomationStepStatusSkipped,
			StepExitError:  "商品当前不在活动中",
		}, "")
		if status != model.SearchCPOItemStatusSkipped {
			t.Fatalf("status = %q, want %q", status, model.SearchCPOItemStatusSkipped)
		}
		if message != "商品当前不在活动中" {
			t.Fatalf("message = %q", message)
		}
	})

	t.Run("other failures remain failed", func(t *testing.T) {
		t.Parallel()
		status, message := summarizeSearchCPOShopExitJobItem(model.AutomationJobItem{
			StepExitStatus: model.AutomationStepStatusFailed,
			StepExitError:  "调用店铺活动接口失败: 500 Internal Server Error",
		}, "")
		if status != model.SearchCPOItemStatusFailed {
			t.Fatalf("status = %q, want %q", status, model.SearchCPOItemStatusFailed)
		}
		if message == "" {
			t.Fatalf("expected failure message to be preserved")
		}
	})
}

func TestBuildSearchCPOMigrationRefreshIssue(t *testing.T) {
	t.Parallel()

	t.Run("404 refresh error becomes skipped and carries action identity", func(t *testing.T) {
		t.Parallel()

		result, skippable := buildSearchCPOMigrationRefreshIssue(model.PromotionAction{
			ID:             28,
			ActionID:       28001,
			Source:         "shop",
			SourceActionID: "3231133",
			Title:          "活动 28",
		}, fmt.Errorf(`API error (status 404): {"code":5,"message":"rpc error: code = NotFound desc = Resource not found"}`))
		if !skippable {
			t.Fatalf("expected skippable issue")
		}
		if result.Status != model.SearchCPOItemStatusSkipped {
			t.Fatalf("status = %q, want %q", result.Status, model.SearchCPOItemStatusSkipped)
		}
		if result.SourceActionID != "3231133" {
			t.Fatalf("SourceActionID = %q", result.SourceActionID)
		}
		if result.PromotionActionID != 28 {
			t.Fatalf("PromotionActionID = %d", result.PromotionActionID)
		}
		if result.Error == "" || !strings.Contains(result.Error, "已自动禁用并跳过") {
			t.Fatalf("unexpected error text: %q", result.Error)
		}
	})

	t.Run("other refresh error stays failed", func(t *testing.T) {
		t.Parallel()

		result, skippable := buildSearchCPOMigrationRefreshIssue(model.PromotionAction{
			ID:       7,
			ActionID: 7001,
			Source:   "official",
			Title:    "官方活动 A",
		}, fmt.Errorf("500 Internal Server Error"))
		if skippable {
			t.Fatalf("expected non-skippable issue")
		}
		if result.Status != model.SearchCPOItemStatusFailed {
			t.Fatalf("status = %q, want %q", result.Status, model.SearchCPOItemStatusFailed)
		}
		if result.Error == "" || !strings.Contains(result.Error, "前置刷新官方活动商品失败") {
			t.Fatalf("unexpected error text: %q", result.Error)
		}
	})
}

func TestMarkSearchCPOMigrationSetupFailedCarriesExitResults(t *testing.T) {
	t.Parallel()

	itemStates := map[string]*searchCPOAutomationItemState{
		"sku-1": {},
	}
	exitResults := []dto.SearchCPORunActionResult{
		{
			PromotionActionID: 28,
			SourceActionID:    "3231133",
			Title:             "活动 28",
			Source:            "shop",
			Status:            model.SearchCPOItemStatusFailed,
			Error:             "前置刷新店铺活动商品失败: API error (status 404): resource not found",
		},
	}

	markSearchCPOMigrationSetupFailed([]string{"sku-1"}, itemStates, "迁移前置失败", exitResults)

	state := itemStates["sku-1"]
	if state == nil {
		t.Fatalf("state missing")
	}
	if state.ExitStatus != model.SearchCPOItemStatusFailed {
		t.Fatalf("ExitStatus = %q", state.ExitStatus)
	}
	if len(state.ExitResults) != 1 {
		t.Fatalf("ExitResults len = %d", len(state.ExitResults))
	}
	if state.ExitResults[0].SourceActionID != "3231133" {
		t.Fatalf("ExitResults[0].SourceActionID = %q", state.ExitResults[0].SourceActionID)
	}
	if state.Message != "迁移前置失败" {
		t.Fatalf("Message = %q", state.Message)
	}
	if state.EnableStatus != model.SearchCPOItemStatusSkipped {
		t.Fatalf("EnableStatus = %q", state.EnableStatus)
	}
	if state.MorkovskStatus != model.SearchCPOItemStatusSkipped {
		t.Fatalf("MorkovskStatus = %q", state.MorkovskStatus)
	}
}
