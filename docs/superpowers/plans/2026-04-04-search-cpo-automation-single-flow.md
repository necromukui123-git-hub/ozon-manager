# Search CPO Single-Flow Automation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Search CPO 收口为单一自动化页面，保留“默认活动 + 退出活动”两组固定配置，统一记录自动执行与手动触发历史，并把退出失败后的停止规则落到后端执行与详情展示中。

**Architecture:** 后端继续以 `search_cpo_configs` + `search_cpo_auto_runs` 为核心数据面，新增退出活动字段，收口状态定义为状态 1/2/3/4，并把自动化退出逻辑改为“只处理用户配置的活动”。前端删除独立手动报名工作面，重构为单页自动化配置与历史查看，所有手动操作都通过现有自动化 run 入口触发。

**Tech Stack:** Go 1.21, Gin, GORM, PostgreSQL migration SQL, Vue 3, Vite, Element Plus

---

## File Map

### Backend config and schema

- Create: `backend/migrations/upgrade_20260404_search_cpo_automation_single_flow.sql`
- Modify: `backend/migrations/init_database.sql`
- Modify: `backend/internal/model/search_cpo.go`
- Modify: `backend/internal/dto/search_cpo.go`
- Modify: `backend/internal/repository/search_cpo_repo.go`
- Modify: `backend/internal/service/search_cpo_service.go`

### Backend automation state and execution

- Modify: `backend/internal/dto/search_cpo_automation.go`
- Modify: `backend/internal/service/search_cpo_automation.go`
- Modify: `backend/internal/service/search_cpo_automation_test.go`
- Modify: `backend/internal/service/search_cpo_service_test.go`

### Frontend single-flow page

- Modify: `frontend/src/views/Layout.vue`
- Modify: `frontend/src/views/promotions/SearchCPO.vue`
- Modify: `frontend/src/views/promotions/search-cpo/SearchCPOAutomationTab.vue`
- Modify: `frontend/src/views/promotions/search-cpo/SearchCPOAutomationDetailDialog.vue`
- Modify: `frontend/src/views/promotions/search-cpo/ui.js`
- Modify: `frontend/src/api/promotion.js`
- Delete: `frontend/src/views/promotions/search-cpo/SearchCPOManualTab.vue`
- Delete: `frontend/src/views/promotions/search-cpo/SearchCPORunDetailDialog.vue`

### Route/handler compatibility

- Modify: `backend/internal/handler/search_cpo_handler.go`
- Modify: `backend/cmd/server/main.go`

### Tracking docs

- Modify: `dev-tracker/OVERALL_TASKS.md`
- Modify: `dev-tracker/CURRENT_PROGRESS.md`
- Modify: `dev-tracker/CHANGELOG.md`

## Task 1: Expand Search CPO config schema for exit actions

**Files:**
- Create: `backend/migrations/upgrade_20260404_search_cpo_automation_single_flow.sql`
- Modify: `backend/migrations/init_database.sql`
- Modify: `backend/internal/model/search_cpo.go`
- Modify: `backend/internal/dto/search_cpo.go`
- Modify: `backend/internal/repository/search_cpo_repo.go`
- Modify: `backend/internal/service/search_cpo_service.go`
- Test: `backend/internal/service/search_cpo_service_test.go`

- [ ] **Step 1: Write the failing backend config test**

Add a test proving config DTO/service conversion includes both exit-action arrays.

```go
func TestToSearchCPOConfigDTOIncludesExitActionIDs(t *testing.T) {
	config := &model.SearchCPOConfig{
		OfficialActionIDs: mustJSON([]uint{11}),
		ShopActionIDs:     mustJSON([]uint{22}),
		ExitOfficialActionIDs: mustJSON([]uint{33}),
		ExitShopActionIDs:     mustJSON([]uint{44}),
		AutoEnabled: true,
		ScheduleTime: "09:05",
	}

	dto := toSearchCPOConfigDTO(config)

	if !reflect.DeepEqual(dto.ExitOfficialActionIDs, []uint{33}) {
		t.Fatalf("ExitOfficialActionIDs = %#v", dto.ExitOfficialActionIDs)
	}
	if !reflect.DeepEqual(dto.ExitShopActionIDs, []uint{44}) {
		t.Fatalf("ExitShopActionIDs = %#v", dto.ExitShopActionIDs)
	}
}
```

- [ ] **Step 2: Run the failing test**

Run:

```powershell
cd backend
$env:GOCACHE="$env:TEMP\ozon-manager-gocache"
go test ./internal/service -run TestToSearchCPOConfigDTOIncludesExitActionIDs -count=1
```

Expected: FAIL because the config model / DTO do not yet expose `exit_official_action_ids` and `exit_shop_action_ids`.

- [ ] **Step 3: Implement the minimal config/schema changes**

Apply only the schema and config contract changes needed to make the test pass:

```go
type SearchCPOConfig struct {
	OfficialActionIDs     datatypes.JSON `gorm:"type:jsonb;not null" json:"official_action_ids"`
	ShopActionIDs         datatypes.JSON `gorm:"type:jsonb;not null" json:"shop_action_ids"`
	ExitOfficialActionIDs datatypes.JSON `gorm:"type:jsonb;not null" json:"exit_official_action_ids"`
	ExitShopActionIDs     datatypes.JSON `gorm:"type:jsonb;not null" json:"exit_shop_action_ids"`
}
```

Also update:

1. `SearchCPOConfigRequest`
2. `SearchCPOConfigResponse`
3. `UpsertConfig()` conflict-update columns
4. `UpdateConfig()` request handling
5. `toSearchCPOConfigDTO()`
6. SQL migration and `init_database.sql`

- [ ] **Step 4: Re-run the focused backend test**

Run:

```powershell
cd backend
$env:GOCACHE="$env:TEMP\ozon-manager-gocache"
go test ./internal/service -run TestToSearchCPOConfigDTOIncludesExitActionIDs -count=1
```

Expected: PASS

- [ ] **Step 5: Commit the schema/config contract**

```bash
git add backend/migrations/init_database.sql backend/migrations/upgrade_20260404_search_cpo_automation_single_flow.sql backend/internal/model/search_cpo.go backend/internal/dto/search_cpo.go backend/internal/repository/search_cpo_repo.go backend/internal/service/search_cpo_service.go backend/internal/service/search_cpo_service_test.go
git commit -m "feat: add search cpo exit action config"
```

## Task 2: Rename Search CPO states to the new 1/2/3/4 vocabulary

**Files:**
- Modify: `backend/internal/model/search_cpo.go`
- Modify: `backend/internal/dto/search_cpo_automation.go`
- Modify: `backend/internal/service/search_cpo_automation.go`
- Modify: `backend/internal/service/search_cpo_automation_test.go`
- Modify: `backend/migrations/upgrade_20260404_search_cpo_automation_single_flow.sql`
- Modify: `backend/migrations/init_database.sql`

- [ ] **Step 1: Update the failing state-derivation tests**

Replace old expectations (`state3_trigger`, `morkovsk_joined`) with the new public vocabulary (`state3`, `state4`).

```go
func TestDeriveSearchCPORuleStateState3(t *testing.T) {
	state, _ := deriveSearchCPORuleState(model.SearchCPOProduct{
		SearchPromoStatus: "SEARCH_PROMO_STATUS_ENABLED",
		CarrotsStatus:     "CARROTS_STATUS_DISABLED",
		AvailabilityPromo: boolPtr(true),
	}, model.SearchCPORuleStateState2, time.Now())

	if state != model.SearchCPORuleStateState3 {
		t.Fatalf("state = %q", state)
	}
}
```

- [ ] **Step 2: Run the focused state test suite**

Run:

```powershell
cd backend
$env:GOCACHE="$env:TEMP\ozon-manager-gocache"
go test ./internal/service -run TestDeriveSearchCPORuleState -count=1
```

Expected: FAIL because constants, summary counters, and state mapping still use old names.

- [ ] **Step 3: Implement the minimal state-vocabulary refactor**

Update constants and summaries without changing unrelated behavior:

```go
const (
	SearchCPORuleStateState1 = "state1"
	SearchCPORuleStateState2 = "state2"
	SearchCPORuleStateState3 = "state3"
	SearchCPORuleStateState4 = "state4"
	SearchCPORuleStateOther  = "other"
)
```

Make sure to:

1. Keep backward-compatible decode paths for old stored values during rollout
2. Add DB migration SQL that rewrites existing `rule_state` values
3. Rename automation summary fields from `total_state3_trigger` to `total_state3`
4. Add `total_state4`

- [ ] **Step 4: Re-run the focused state test suite**

Run:

```powershell
cd backend
$env:GOCACHE="$env:TEMP\ozon-manager-gocache"
go test ./internal/service -run TestDeriveSearchCPORuleState -count=1
```

Expected: PASS

- [ ] **Step 5: Commit the state vocabulary refactor**

```bash
git add backend/internal/model/search_cpo.go backend/internal/dto/search_cpo_automation.go backend/internal/service/search_cpo_automation.go backend/internal/service/search_cpo_automation_test.go backend/migrations/init_database.sql backend/migrations/upgrade_20260404_search_cpo_automation_single_flow.sql
git commit -m "refactor: rename search cpo automation states"
```

## Task 3: Replace full-scan exit logic with configured exit actions only

**Files:**
- Modify: `backend/internal/service/search_cpo_automation.go`
- Modify: `backend/internal/service/search_cpo_automation_test.go`
- Modify: `backend/internal/dto/search_cpo_automation.go`

- [ ] **Step 1: Write the failing exit-selection tests**

Add tests for both critical rules:

1. only configured exit actions are attempted
2. exit failure stops the rest of the item pipeline

```go
func TestProcessMigrationItemsUsesConfiguredExitActionsOnly(t *testing.T) {
	// arrange itemStates + configured exit IDs
	// assert processMigrationItems never loads all matched actions by SKU
}

func TestProcessMigrationItemsSkipsFollowupAfterExitFailure(t *testing.T) {
	state := &searchCPOAutomationItemState{}
	// arrange exit failure
	// assert EnableStatus == skipped
	// assert MorkovskStatus == skipped
	// assert Message contains "退出促销活动失败，跳过后续动作"
}
```

- [ ] **Step 2: Run the focused migration tests and confirm failure**

Run:

```powershell
cd backend
$env:GOCACHE="$env:TEMP\ozon-manager-gocache"
go test ./internal/service -run "TestProcessMigrationItemsUsesConfiguredExitActionsOnly|TestProcessMigrationItemsSkipsFollowupAfterExitFailure" -count=1
```

Expected: FAIL because the current code still scans full active actions and does not record the new skip message.

- [ ] **Step 3: Implement the minimal exit-logic rewrite**

Refactor `processMigrationItems()` to:

1. resolve configured exit actions from `exit_official_action_ids` / `exit_shop_action_ids`
2. skip the whole exit step if both arrays are empty
3. attempt removal only against those configured actions
4. preserve post-exit verification, but only for configured actions
5. when exit fails, set:

```go
state.ExitStatus = model.SearchCPOItemStatusFailed
state.EnableStatus = model.SearchCPOItemStatusSkipped
state.MorkovskStatus = model.SearchCPOItemStatusSkipped
state.Message = appendSearchCPOItemMessage(state.Message, "退出促销活动失败，跳过后续动作")
```

6. for state 4 items, keep `enable` / `Morkovsk` permanently skipped even when exit succeeds

- [ ] **Step 4: Run the focused migration tests again**

Run:

```powershell
cd backend
$env:GOCACHE="$env:TEMP\ozon-manager-gocache"
go test ./internal/service -run "TestProcessMigrationItemsUsesConfiguredExitActionsOnly|TestProcessMigrationItemsSkipsFollowupAfterExitFailure" -count=1
```

Expected: PASS

- [ ] **Step 5: Commit the exit-logic refactor**

```bash
git add backend/internal/service/search_cpo_automation.go backend/internal/service/search_cpo_automation_test.go backend/internal/dto/search_cpo_automation.go
git commit -m "feat: use configured search cpo exit actions"
```

## Task 4: Unify automation history/detail payload for the new flow

**Files:**
- Modify: `backend/internal/dto/search_cpo_automation.go`
- Modify: `backend/internal/service/search_cpo_automation.go`
- Modify: `backend/internal/handler/search_cpo_handler.go`
- Modify: `backend/cmd/server/main.go`
- Test: `backend/internal/service/search_cpo_automation_test.go`

- [ ] **Step 1: Write the failing history/detail tests**

Add focused tests covering:

1. config snapshot includes default + exit actions
2. summary exposes `total_state3` and `total_state4`
3. detail preserves the exit-failure skip message

```go
func TestDecodeSearchCPOAutomationConfigSnapshotIncludesExitActions(t *testing.T) {}
func TestToSearchCPOAutomationRunSummaryDTOIncludesState4(t *testing.T) {}
```

- [ ] **Step 2: Run the focused history/detail tests**

Run:

```powershell
cd backend
$env:GOCACHE="$env:TEMP\ozon-manager-gocache"
go test ./internal/service -run "TestDecodeSearchCPOAutomationConfigSnapshotIncludesExitActions|TestToSearchCPOAutomationRunSummaryDTOIncludesState4" -count=1
```

Expected: FAIL because snapshot and summary contracts do not yet include the new fields.

- [ ] **Step 3: Implement the minimal history/detail contract changes**

Update backend serialization only as needed:

```go
type searchCPOAutomationConfigSnapshot struct {
	ScheduleTime          string `json:"schedule_time,omitempty"`
	OfficialActionIDs     []uint `json:"official_action_ids"`
	ShopActionIDs         []uint `json:"shop_action_ids"`
	ExitOfficialActionIDs []uint `json:"exit_official_action_ids"`
	ExitShopActionIDs     []uint `json:"exit_shop_action_ids"`
}
```

Also:

1. update `SearchCPOAutomationRunSummaryResponse`
2. update `SearchCPOAutomationRunDetailResponse`
3. leave old `/search-cpo/runs` endpoints in place as legacy, but do not extend them
4. keep `/search-cpo/automation/*` as the only payload the new UI consumes

- [ ] **Step 4: Re-run the focused history/detail tests**

Run:

```powershell
cd backend
$env:GOCACHE="$env:TEMP\ozon-manager-gocache"
go test ./internal/service -run "TestDecodeSearchCPOAutomationConfigSnapshotIncludesExitActions|TestToSearchCPOAutomationRunSummaryDTOIncludesState4" -count=1
```

Expected: PASS

- [ ] **Step 5: Commit the history/detail contract**

```bash
git add backend/internal/dto/search_cpo_automation.go backend/internal/service/search_cpo_automation.go backend/internal/service/search_cpo_automation_test.go backend/internal/handler/search_cpo_handler.go backend/cmd/server/main.go
git commit -m "feat: update search cpo automation history contract"
```

## Task 5: Refactor the frontend into a single automation page

**Files:**
- Modify: `frontend/src/views/Layout.vue`
- Modify: `frontend/src/views/promotions/SearchCPO.vue`
- Modify: `frontend/src/views/promotions/search-cpo/SearchCPOAutomationTab.vue`
- Modify: `frontend/src/views/promotions/search-cpo/SearchCPOAutomationDetailDialog.vue`
- Modify: `frontend/src/views/promotions/search-cpo/ui.js`
- Modify: `frontend/src/api/promotion.js`
- Delete: `frontend/src/views/promotions/search-cpo/SearchCPOManualTab.vue`
- Delete: `frontend/src/views/promotions/search-cpo/SearchCPORunDetailDialog.vue`

- [ ] **Step 1: Refactor the page shell to remove manual-flow wiring**

Delete tab switching, manual run polling, and manual detail dialog plumbing from `SearchCPO.vue`.

The final page state should look like:

```js
const config = reactive({
  official_action_ids: [],
  shop_action_ids: [],
  exit_official_action_ids: [],
  exit_shop_action_ids: [],
  auto_enabled: false,
  schedule_time: '09:05'
})
```

- [ ] **Step 2: Update `SearchCPOAutomationTab.vue` for the new form structure**

Add:

1. default-actions block
2. exit-actions block
3. history table with trigger-mode labels
4. state 1/2/3/4 counters

Remove:

1. references back to “商品池与手动报名”
2. “去配置默认活动” tab-switch actions

- [ ] **Step 3: Update the detail dialog labels and messages**

Make the detail dialog show:

1. 状态 1/2/3/4 summary
2. “退出活动” instead of “退出其它活动”
3. `退出促销活动失败，跳过后续动作` when present
4. config snapshot data for default actions and exit actions

- [ ] **Step 4: Update menu text and remove unused frontend API exports**

Apply the minimal cleanup:

```js
// keep
getSearchCPOConfig
updateSearchCPOConfig
startSearchCPOAutomationRun
listSearchCPOAutomationRuns
getSearchCPOAutomationRunDetail

// remove UI-only usage of
startSearchCPORun
listSearchCPORuns
getSearchCPORunDetail
```

Also rename the visible menu label to `搜索推广自动化`.

- [ ] **Step 5: Run the frontend build**

Run:

```powershell
cd frontend
cmd /c npm run build
```

Expected: PASS

- [ ] **Step 6: Commit the single-page frontend**

```bash
git add frontend/src/views/Layout.vue frontend/src/views/promotions/SearchCPO.vue frontend/src/views/promotions/search-cpo/SearchCPOAutomationTab.vue frontend/src/views/promotions/search-cpo/SearchCPOAutomationDetailDialog.vue frontend/src/views/promotions/search-cpo/ui.js frontend/src/api/promotion.js
git rm frontend/src/views/promotions/search-cpo/SearchCPOManualTab.vue frontend/src/views/promotions/search-cpo/SearchCPORunDetailDialog.vue
git commit -m "feat: simplify search cpo to single automation page"
```

## Task 6: Run integration verification and update tracking docs

**Files:**
- Modify: `dev-tracker/OVERALL_TASKS.md`
- Modify: `dev-tracker/CURRENT_PROGRESS.md`
- Modify: `dev-tracker/CHANGELOG.md`

- [ ] **Step 1: Run the focused backend test packages**

Run:

```powershell
cd backend
$env:GOCACHE="$env:TEMP\ozon-manager-gocache"
go test ./internal/service
```

Expected: PASS

- [ ] **Step 2: Re-run the frontend build for final verification**

Run:

```powershell
cd frontend
cmd /c npm run build
```

Expected: PASS

- [ ] **Step 3: Update tracking documents**

Record:

1. `OVERALL_TASKS.md`
   - T16/T18 scope now points to single-flow automation instead of dual-tab UI
2. `CURRENT_PROGRESS.md`
   - mention migration script name, new exit-action config, state 1/2/3/4 rename, unified history
3. `CHANGELOG.md`
   - summarize UI removal, exit-action config, stop-on-exit-failure behavior

- [ ] **Step 4: Capture verification notes in the docs**

Add exact commands and outcomes to `CURRENT_PROGRESS.md` and `CHANGELOG.md`, including:

```text
cd backend && go test ./internal/service
cd frontend && npm run build
```

- [ ] **Step 5: Commit verification + tracker updates**

```bash
git add dev-tracker/OVERALL_TASKS.md dev-tracker/CURRENT_PROGRESS.md dev-tracker/CHANGELOG.md
git commit -m "docs: update search cpo automation tracking"
```

## Final Verification Checklist

- [ ] `SearchCPOConfig` can persist default-action IDs and exit-action IDs
- [ ] 状态文案在前后端统一为状态 1/2/3/4
- [ ] 状态 2/3/4 不再按 SKU 扫描全量命中活动
- [ ] 未配置退出活动时，退出步骤显式记为 `skipped`
- [ ] 退出活动失败时，后续动作显式记为 `skipped`
- [ ] 详情文案包含 `退出促销活动失败，跳过后续动作`
- [ ] 手动触发和定时触发都进入统一自动化历史
- [ ] 页面只剩单一自动化工作面

## Notes for the Implementer

- Follow `@superpowers:test-driven-development` for all backend behavior changes.
- Use `@superpowers:verification-before-completion` before claiming the feature is done.
- Do not expand scope into plugin task-type changes; reuse the existing Search CPO automation jobs.
- Leave legacy manual-run backend endpoints in place unless a later cleanup task explicitly removes them.
