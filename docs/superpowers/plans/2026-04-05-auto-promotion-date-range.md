# Auto Promotion Date Range Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把“自动加促销”的上架时间规则从“昨天 / 今天 / 自定义单日”改成“昨天 / 今天 / 自定义日期段”，并让后端筛选、运行历史、前端展示与数据库结构保持一致。

**Architecture:** 后端继续保留 `target_date_mode` 的三种枚举值，但把 `custom` 改造成开始日期和结束日期的闭区间；配置表复用 `target_date` 作为开始日期并新增 `target_date_end`，运行表同样记录实际开始和结束日期。前端只调整 `/promotions/auto-add` 页面和对应 DTO 展示，不改自动加促销其它执行链路。

**Tech Stack:** Go 1.21, Gin, GORM, PostgreSQL migration SQL, Vue 3, Vite, Element Plus

---

## File Map

### Backend

- Modify: `backend/internal/model/auto_promotion.go`
- Modify: `backend/internal/dto/auto_promotion.go`
- Modify: `backend/internal/service/auto_promotion_service.go`
- Modify: `backend/internal/service/auto_promotion_service_test.go`
- Modify: `backend/internal/repository/ozon_catalog_repo.go`

### Database

- Modify: `backend/migrations/init_database.sql`
- Create: `backend/migrations/upgrade_20260405_auto_promotion_date_range.sql`

### Frontend

- Modify: `frontend/src/views/promotions/AutoAdd.vue`

### Tracking Docs

- Modify: `dev-tracker/OVERALL_TASKS.md`
- Modify: `dev-tracker/CURRENT_PROGRESS.md`
- Modify: `dev-tracker/CHANGELOG.md`

## Task 1: Add failing backend tests for date range rules

**Files:**
- Modify: `backend/internal/service/auto_promotion_service_test.go`

- [ ] **Step 1: Write the failing tests**

Add tests that prove:

1. `resolveAutoPromotionTargetDateRange()` can resolve `yesterday / today / custom range`
2. `validateAutoPromotionConfigTargetDateRange()` rejects missing end date and inverted range
3. 同一天的自定义日期段会保留为开始=结束

- [ ] **Step 2: Run the focused tests to verify failure**

Run:

```powershell
cd backend
$env:GOCACHE="$env:TEMP\ozon-manager-gocache"
go test ./internal/service -run "TestResolveAutoPromotionTargetDateRange|TestResolveAutoPromotionTargetDateRangeErrors|TestValidateAutoPromotionConfigTargetDateRange" -count=1
```

Expected: FAIL because current code still only supports single-date `target_date`.

- [ ] **Step 3: Implement the minimal backend rule parsing**

Update `backend/internal/service/auto_promotion_service.go` with:

1. new start/end date parse helpers
2. compatibility fallback from legacy `target_date`
3. normalized custom-range validation

- [ ] **Step 4: Re-run the focused tests**

Run the same command as Step 2.

Expected: PASS

## Task 2: Add failing backend tests for range-based filtering and DTO mapping

**Files:**
- Modify: `backend/internal/service/auto_promotion_service_test.go`
- Modify: `backend/internal/model/auto_promotion.go`
- Modify: `backend/internal/dto/auto_promotion.go`
- Modify: `backend/internal/repository/ozon_catalog_repo.go`
- Modify: `backend/internal/service/auto_promotion_service.go`

- [ ] **Step 1: Write the failing tests**

Add tests that prove:

1. config/run DTOs expose `target_date_start` and `target_date_end`
2. custom config stores both dates
3. auto-promotion selection still includes both boundaries of a range

- [ ] **Step 2: Run the focused tests to verify failure**

Run:

```powershell
cd backend
$env:GOCACHE="$env:TEMP\ozon-manager-gocache"
go test ./internal/service -run "TestValidateAutoPromotionConfigTargetDateRange|TestToAutoPromotionConfigDTOUsesDateRange|TestToAutoPromotionRunSummaryDTOUsesDateRange|TestSelectEligibleItems" -count=1
```

Expected: FAIL because model/DTO/repository still expose only a single target date.

- [ ] **Step 3: Implement the minimal model/repository/service changes**

Apply only the changes needed to make those tests pass:

1. add `TargetDateEnd` to config/run models
2. add `target_date_start/target_date_end` to DTOs
3. update snapshots and run creation
4. replace single-date catalog query with range query

- [ ] **Step 4: Re-run the focused tests**

Run the same command as Step 2.

Expected: PASS

## Task 3: Update SQL migrations and keep baseline consistent

**Files:**
- Modify: `backend/migrations/init_database.sql`
- Create: `backend/migrations/upgrade_20260405_auto_promotion_date_range.sql`

- [ ] **Step 1: Write or extend the failing migration expectation**

If migration baseline tests need explicit assertions, add them so the schema must include:

1. `auto_promotion_configs.target_date_end`
2. `auto_promotion_runs.target_date_end`

- [ ] **Step 2: Run the migration tests to verify failure**

Run:

```powershell
cd backend
$env:GOCACHE="$env:TEMP\ozon-manager-gocache"
go test ./migrations -count=1
```

Expected: FAIL if the baseline assertion is extended before SQL is updated; otherwise use this step to confirm the old baseline still lacks the new columns.

- [ ] **Step 3: Implement the SQL changes**

Update `init_database.sql` and create `upgrade_20260405_auto_promotion_date_range.sql` with:

1. new config/run end-date columns
2. legacy data backfill `target_date_end = target_date`
3. notes about execution scope and idempotence

- [ ] **Step 4: Re-run migration tests**

Run the same command as Step 2.

Expected: PASS

## Task 4: Update AutoAdd page for custom date range

**Files:**
- Modify: `frontend/src/views/promotions/AutoAdd.vue`

- [ ] **Step 1: Write the minimal UI-facing failing assertion mentally from the spec**

The page must:

1. show `昨天 / 今天 / 自定义日期段`
2. submit `target_date_start/target_date_end`
3. render history as date range text

- [ ] **Step 2: Implement the frontend changes**

Change only the date-rule related UI:

1. replace single-date picker with range picker in `custom` mode
2. update validation and payload shape
3. update history/detail labels to “实际日期段”
4. keep yesterday/today behavior unchanged

- [ ] **Step 3: Run the frontend build**

Run:

```powershell
cd frontend
cmd /c npm run build
```

Expected: PASS

## Task 5: Update tracking docs and run final verification

**Files:**
- Modify: `dev-tracker/OVERALL_TASKS.md`
- Modify: `dev-tracker/CURRENT_PROGRESS.md`
- Modify: `dev-tracker/CHANGELOG.md`

- [ ] **Step 1: Update the tracking docs**

Document:

1. auto-promotion now supports custom date range
2. new migration script name, purpose, execution condition, execution result
3. backend/frontend verification commands and results

- [ ] **Step 2: Run final backend verification**

Run:

```powershell
cd backend
$env:GOCACHE="$env:TEMP\ozon-manager-gocache-final"
go test ./...
```

Expected: PASS

- [ ] **Step 3: Run final frontend verification**

Run:

```powershell
cd frontend
cmd /c npm run build
```

Expected: PASS

- [ ] **Step 4: Review git diff**

Run:

```powershell
git status --short
git diff --stat
```

Expected: only the intended auto-promotion date-range files and tracking docs are modified.
