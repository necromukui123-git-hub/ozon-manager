# 自动加促销上架时间规则改为日期段设计

## 背景

当前“自动加促销”页面的上架时间规则支持三种模式：

1. 昨天
2. 今天
3. 自定义日期

这套设计只能表达“单日”筛选，无法覆盖“按一段上架日期批量处理商品”的实际使用场景。用户当前明确要求保留“昨天 / 今天”两个快捷规则，同时把“自定义日期”升级为“自定义日期段”。

现有实现中：

1. 配置层只保存 `target_date_mode + target_date`
2. 运行层只记录单个实际执行日期
3. 目录筛选只支持“上架日期等于某一天”

因此本轮需要把自动加促销从“单日规则”升级为“日期段规则”，并保持现有页面、接口、历史记录与定时调度的语义可理解、可兼容、可迁移。

## 目标

1. 自动加促销页面保留三种规则：昨天、今天、自定义日期段。
2. 自定义模式支持开始日期和结束日期，且按闭区间筛选。
3. 定时执行与手动执行都统一解析出“本次实际日期段”并写入历史。
4. 历史列表和详情能清晰展示规则与实际日期段。
5. 尽量最小化改动范围，避免对自动加促销其它执行逻辑做无关重构。

## 非目标

1. 本轮不引入“最近 N 天”“本周”“上周”等新的相对日期模式。
2. 本轮不重做自动加促销整体交互结构，只调整日期规则相关表单与展示。
3. 本轮不清理旧自动加促销历史数据，只做兼容迁移。
4. 本轮不改动官方活动 / 店铺活动的执行顺序与候选刷新策略。

## 备选方案对比

### 方案 A：保留昨天 / 今天 / 自定义，并把自定义升级为日期段

- 优点：与用户要求完全一致；改动面集中在日期字段、筛选查询和历史展示。
- 缺点：需要补充一个结束日期字段，并同步调整数据库、DTO、前端表单和历史展示。
- 结论：本轮采用。

### 方案 B：完全改成纯日期段，不再保留昨天 / 今天

- 优点：模型最统一，所有执行都只处理日期段。
- 缺点：丢失高频快捷规则，不符合用户明确要求。
- 结论：不采用。

### 方案 C：继续保留单个 `target_date`，把结束日期塞进快照或额外 JSON

- 优点：表结构变化最少。
- 缺点：接口和数据库语义会变脏，查询和历史回放都不清晰。
- 结论：不采用。

## 选定方案

### 总体思路

保留 `target_date_mode` 枚举 `yesterday / today / custom`，但把 `custom` 的语义从“自定义单日”升级为“自定义日期段”。

后端统一把每次执行解析为：

1. `target_date_start`
2. `target_date_end`

其中：

1. `yesterday` 解析为“昨天到昨天”
2. `today` 解析为“今天到今天”
3. `custom` 解析为“用户选择的开始日期到结束日期”

商品筛选改为闭区间：

1. `listing_date >= target_date_start`
2. `listing_date <= target_date_end`

## 数据模型设计

### 配置表

继续复用 `auto_promotion_configs.target_date` 作为开始日期，并新增：

1. `target_date_end date`

字段语义：

1. `target_date_mode`
   - `yesterday`
   - `today`
   - `custom`
2. `target_date`
   - 表示自定义日期段开始日期
3. `target_date_end`
   - 表示自定义日期段结束日期

保存策略：

1. `yesterday / today` 模式下，`target_date` 与 `target_date_end` 都不保存固定值。
2. `custom` 模式下，必须同时保存 `target_date` 与 `target_date_end`。

### 运行表

继续复用 `auto_promotion_runs.target_date` 作为实际开始日期，并新增：

1. `target_date_end date not null`

运行表始终记录“本次实际日期段”，不区分是否来自快捷模式：

1. `yesterday` 运行时写入“昨天到昨天”
2. `today` 运行时写入“今天到今天”
3. `custom` 运行时写入“开始日期到结束日期”

这样历史记录不需要二次推导，能直接准确回放。

### 配置快照

`config_snapshot` 中同步改为保存：

1. `target_date_mode`
2. `target_date_start`
3. `target_date_end`

快照用途不变，仍用于查看当次执行的配置现场。

## 接口设计

### 配置接口

`AutoPromotionConfigRequest` / `AutoPromotionConfigResponse` 调整为：

1. 保留 `target_date_mode`
2. 把单个 `target_date` 改为：
   - `target_date_start`
   - `target_date_end`

兼容策略：

1. 服务端短期保留对旧 `target_date` 入参的兜底兼容。
2. 若收到旧 `target_date` 且未收到新字段，则按“开始=结束=该日期”处理。

### 手动执行接口

`AutoPromotionRunRequest` 调整为：

1. 保留 `target_date_mode`
2. 改为接收：
   - `target_date_start`
   - `target_date_end`

校验规则：

1. `yesterday / today` 模式下允许不传日期段。
2. `custom` 模式下必须同时传开始和结束日期。
3. 开始日期不能晚于结束日期。

### 历史接口

运行历史和详情接口统一返回：

1. `target_date_mode`
2. `target_date_start`
3. `target_date_end`

页面展示层可以根据是否同一天决定显示为：

1. 单日：`2026-04-05`
2. 跨日：`2026-04-01 ~ 2026-04-05`

## 执行与筛选逻辑

### 规则解析

新增统一日期段解析逻辑：

1. `resolveAutoPromotionTargetDateRange(mode, start, end, reference)`
2. `validateAutoPromotionConfigTargetDateRange(mode, start, end)`

解析结果统一返回：

1. 规范化后的 `target_date_mode`
2. `target_date_start`
3. `target_date_end`

### 商品筛选

当前自动加促销通过目录缓存按单个上架日期筛商品。本轮改为按日期段查询：

1. 仓储层新增“按上架日期段查询目录商品”能力。
2. 查询语义固定为闭区间。
3. 当开始和结束是同一天时，行为等价于现有单日筛选。

### 历史统计

`TotalCandidates / TotalSelected / TotalProcessed / SuccessItems / FailedItems / SkippedItems` 的统计规则不变。

变化仅在候选商品的来源集合从“某一天”变为“某个日期段内”。

## 前端页面设计

### 配置区

页面继续保留“上架时间规则”单选：

1. 昨天
2. 今天
3. 自定义日期段

当选择“自定义日期段”时，展示日期范围选择器，或者一对开始/结束日期控件。

前端约束：

1. 自定义日期段模式下必须同时填写开始和结束日期。
2. 开始日期不能晚于结束日期。
3. 非自定义模式下不向后端传固定日期段值。

### 历史展示

列表页和详情页做两处调整：

1. “实际日期”改成“实际日期段”
2. 日期规则中的 `custom` 文案从“自定义日期”改为“自定义日期段”

展示策略：

1. 同一天显示单个日期
2. 跨天显示 `开始 ~ 结束`

## 数据迁移设计

### 基线 SQL

`backend/migrations/init_database.sql` 需要同步回写：

1. `auto_promotion_configs.target_date_end`
2. `auto_promotion_runs.target_date_end`

### 增量脚本

新增版本化迁移脚本：

1. `backend/migrations/upgrade_20260405_auto_promotion_date_range.sql`

迁移策略：

1. 为 `auto_promotion_configs` 增加 `target_date_end`
2. 为 `auto_promotion_runs` 增加 `target_date_end`
3. 旧配置数据回填：
   - 若 `target_date` 非空，则 `target_date_end = target_date`
4. 旧运行历史回填：
   - `target_date_end = target_date`

这样可以把历史单日记录自然升级为“开始=结束”的日期段，不改变旧历史的业务含义。

## 兼容性与风险

### 兼容性

1. 旧配置如果原来是 `custom + 单日`，迁移后自然变为“同一天的日期段”。
2. 旧运行历史会自动表现为“同一天的日期段”，不需要人工修复。
3. 前端发布前后短暂不一致时，后端对旧 `target_date` 请求保持兜底兼容，降低切换风险。

### 风险点

1. 若只改服务层、不改仓储层查询，会导致日期段逻辑仍按单日执行。
2. 若运行历史未落 `target_date_end`，后续详情会无法准确回放真实执行区间。
3. 若前端只校验“非空”不校验“开始 <= 结束”，会产生无效请求。

## 测试设计

本轮按 TDD 覆盖以下最小集合：

1. 日期段解析测试
   - 昨天
   - 今天
   - 自定义同一天
   - 自定义跨天
   - 空模式兼容
   - 非法模式
   - 缺失开始/结束日期
   - 开始日期晚于结束日期
2. 配置校验测试
   - `yesterday / today` 不保存固定日期
   - `custom` 正确保存开始和结束日期
3. 日期段筛选测试
   - 闭区间包含开始日期
   - 闭区间包含结束日期
   - 区间外商品不被选中
4. 前端构建验证
   - `/promotions/auto-add` 页面可正常构建
   - 历史日期段展示无语法错误

## 实施范围

预计涉及：

1. `backend/internal/model/auto_promotion.go`
2. `backend/internal/dto/auto_promotion.go`
3. `backend/internal/service/auto_promotion_service.go`
4. `backend/internal/service/auto_promotion_service_test.go`
5. `backend/internal/repository/auto_promotion_repo.go`
6. `backend/internal/repository/ozon_catalog_repo.go`
7. `backend/migrations/init_database.sql`
8. `backend/migrations/upgrade_20260405_auto_promotion_date_range.sql`
9. `frontend/src/views/promotions/AutoAdd.vue`
10. `dev-tracker/OVERALL_TASKS.md`
11. `dev-tracker/CURRENT_PROGRESS.md`
12. `dev-tracker/CHANGELOG.md`

## 验收标准

1. 页面上架时间规则显示为“昨天 / 今天 / 自定义日期段”。
2. 自定义日期段可保存、可手动执行、可参与定时执行。
3. 自动加促销按闭区间筛选目录商品，而不是只匹配单日。
4. 历史列表和详情能展示实际日期段。
5. 旧配置和旧历史升级后仍可正常读取，不丢失语义。
