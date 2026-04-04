# Search CPO 单一自动化流程改造设计

## 背景

当前 Search CPO 页面同时承载了两条链路：

1. 商品池与手动报名
2. 状态迁移自动化

这套信息架构已经超出了当前业务诉求。用户希望把 Search CPO 收口为单一自动化流程，不再保留“商品池与手动报名”这一条独立工作面，只保留一套自动化配置，并允许用户对同一条自动化流程执行“手动触发一次”。

同时，当前自动化迁移在退出活动阶段会：

1. 先同步全店活跃活动
2. 再按 SKU 查出商品当前命中的全部活动
3. 从这些命中的活动里逐个退出

这与新的业务要求不一致。新的要求是：退出范围必须由用户显式配置，不再由系统自动扩大为“当前命中的全部活动”。

## 目标

1. Search CPO 只保留一套自动化流程，支持定时执行和手动触发一次。
2. 页面和配置收口为两组活动语义：
   - 默认活动：给“推广已关闭”的商品使用
   - 退出活动：给“可加入推广 / 已加入推广”的商品使用
3. 状态定义按新的业务语义统一为状态 1/2/3/4，并在前后端显示口径上保持一致。
4. 自动化历史统一记录所有执行，包括定时执行和手动触发执行。
5. 自动化退出步骤只处理用户显式选择的活动，不再按 SKU 扫描全量命中活动。

## 非目标

1. 本轮不清理旧的 `search_cpo_runs` / `search_cpo_run_items` 历史数据。
2. 本轮不额外增加第二套配置接口或第二套执行历史接口。
3. 本轮不改插件任务类型的大框架，只调整后端自动化链路如何组织退出范围与状态判定。
4. 本轮不做与 Search CPO 无关的页面重构或菜单重组。

## 备选方案对比

### 方案 A：保留“手动报名 + 自动化”双标签，在自动化里补退出活动配置

- 优点：前后端改动相对分散但局部，兼容现有页面结构。
- 缺点：继续保留用户已明确不需要的“商品池与手动报名”，产品心智仍然分裂。
- 结论：不采用。

### 方案 B：收口为单一自动化流程，保留默认活动和退出活动两组固定配置

- 优点：最贴合当前需求；定时执行和手动执行一次复用同一条链路；配置语义清晰。
- 缺点：需要调整配置结构、页面布局、状态文案和自动化执行逻辑。
- 结论：本轮采用。

### 方案 C：完全取消默认活动，只保留退出活动和自动推进

- 优点：配置更少。
- 缺点：状态 1 缺少明确的目标活动集合，无法满足“推广已关闭的商品需要添加促销活动”的要求。
- 结论：不采用。

## 选定方案

### 总体思路

保留现有路由 `/promotions/search-cpo`，但用户可见产品入口和页面标题统一收口为“搜索推广自动化”。页面只保留一套自动化工作面：

1. 自动执行设置
2. 默认活动配置
3. 退出活动配置
4. 状态概览
5. 执行历史

“手动执行一次”不再代表独立手动报名功能，而是代表“按当前自动化配置立即执行一次同样的自动化流程”。

## 状态定义

### 用户可见状态定义

1. 状态 1：推广已关闭
2. 状态 2：可加入推广
3. 状态 3：已加入推广，未加入 Morkovsk
4. 状态 4：已加入推广，已加入 Morkovsk

### 技术映射

为保证现有 Search CPO 数据字段可复用，状态判定沿用现有 `search_promo_status`、`carrots_status`、`availability_promo` 三组 live 字段，但映射收口为下表：

| 业务状态 | 技术判定 |
|----------|----------|
| 状态 1 | `search_promo_status = SEARCH_PROMO_STATUS_DISABLED` 且 `availability_promo = false` |
| 状态 2 | `search_promo_status = SEARCH_PROMO_STATUS_DISABLED` 且 `availability_promo = true` |
| 状态 3 | `search_promo_status = SEARCH_PROMO_STATUS_ENABLED` 且 `carrots_status = CARROTS_STATUS_DISABLED` |
| 状态 4 | `search_promo_status = SEARCH_PROMO_STATUS_ENABLED` 且 `carrots_status = CARROTS_STATUS_ENABLED` |

补充约束：

1. `carrots_status` 为空或 `availability_promo` 未检测到时，商品仍归入“其它 / 未识别”，不强行映射到状态 1-4。
2. 状态 4 仍使用现有 `morkovsk_joined_at` 作为本地修复和诊断辅助，但页面和历史文案不再展示 `morkovsk_joined` 旧术语。

## 配置模型

### 默认活动

“默认活动”表示“推广已关闭”的商品需要添加的促销活动。

适用范围：

1. 仅对状态 1 生效
2. 支持官方活动
3. 支持店铺活动

### 退出活动

“退出活动”表示所有“可加入推广”、“已加入推广”的商品需要退出的促销活动。

适用范围：

1. 对状态 2 / 状态 3 / 状态 4 生效
2. 支持官方活动
3. 支持店铺活动

空配置策略：

1. 若未配置任何退出活动，自动化仍可执行
2. 退出步骤整体记为 `skipped`
3. 状态 2 / 状态 3 仍继续后续 `enable` / `Morkovsk` 步骤
4. 状态 4 仍跳过重复 `enable` 和重复加入 `Morkovsk`

## 页面设计

### 路由与命名

1. 路由保持 `/promotions/search-cpo`，避免新增导航路径和兼容成本。
2. 菜单名改为“搜索推广自动化”。
3. 页面标题改为“搜索推广自动化”。

### 页面结构

页面从上到下收口为以下模块：

1. 自动执行设置
   - 自动触发开关
   - 触发时间
   - 保存设置
   - 手动执行一次
2. 默认活动
   - 官方活动选择
   - 店铺活动选择
   - 文案：推广已关闭的商品需要添加的促销活动
3. 退出活动
   - 官方活动选择
   - 店铺活动选择
   - 文案：所有可加入推广、已加入推广的商品需要退出的促销活动
   - 空配置提示：未配置退出活动时，将跳过退出促销活动步骤
4. 状态概览
   - 状态 1 数量
   - 状态 2 数量
   - 状态 3 数量
   - 状态 4 数量
   - 其它 / 未识别数量
5. 执行历史
   - 定时执行与手动触发共用同一张历史表
   - 必须显示触发方式

### 页面移除内容

以下内容不再保留：

1. 商品池与本地筛选
2. “报名当前筛选结果”按钮
3. 手动报名历史
4. 依赖筛选结果生成 `source_skus` 的独立执行链路

## 配置接口设计

继续复用现有 Search CPO 配置接口，不新增第二套配置 API。

### 请求字段

在现有 `SearchCPOConfigRequest` 基础上新增：

1. `exit_official_action_ids`
2. `exit_shop_action_ids`

### 返回字段

在现有 `SearchCPOConfigResponse` 基础上新增：

1. `exit_official_action_ids`
2. `exit_shop_action_ids`

### 兼容性

1. `official_action_ids` / `shop_action_ids` 继续保留，但语义收口为“默认活动”。
2. `enable_step` 继续作为兼容字段保留在服务端配置模型中，但前端页面不再展示为可配置项；自动化固定按开启推广步骤执行。

## 自动化执行设计

### 统一执行入口

保留现有自动化执行入口，但用户侧只感知一套自动化流程。

触发方式仅区分：

1. `scheduled`
2. `manual`

两种触发方式共用相同的配置、相同的执行链路、相同的历史表。

### 执行流水

每次执行按以下顺序处理：

1. 刷新 Search CPO 商品缓存
2. 同步 availability / 推广状态信息
3. 按新的状态定义计算状态 1/2/3/4
4. 处理状态 1：加入默认活动
5. 处理状态 2/3/4：按退出活动配置执行退出
6. 处理状态 2：若退出步骤可继续，则执行 `enable`，成功后继续加入 `Morkovsk`
7. 处理状态 3：若退出步骤可继续，则跳过重复 `enable`，直接加入 `Morkovsk`
8. 处理状态 4：若配置了退出活动，则仅执行退出清理；跳过重复 `enable` 与重复加入 `Morkovsk`

### 状态级别执行规则

#### 状态 1

1. 只执行默认活动报名
2. 不执行退出活动
3. 不执行 `enable`
4. 不执行 `Morkovsk`

#### 状态 2

1. 先执行退出活动
2. 若退出活动为空，则退出步骤记 `skipped`
3. 若退出步骤失败，则该商品后续 `enable` / `Morkovsk` 记 `skipped`
4. 若退出步骤成功或跳过，则执行 `enable`
5. `enable` 成功后继续执行 `Morkovsk`

#### 状态 3

1. 先执行退出活动
2. 若退出步骤为空，则记 `skipped`
3. 若退出步骤失败，则跳过 `Morkovsk`
4. 不重复执行 `enable`
5. 继续执行加入 `Morkovsk`

#### 状态 4

1. 如配置了退出活动，则执行退出清理
2. 若未配置退出活动，则退出步骤记 `skipped`
3. 跳过重复 `enable`
4. 跳过重复加入 `Morkovsk`

### 退出活动处理规则

新的退出策略必须满足以下约束：

1. 只处理用户显式配置的退出活动
2. 不再先同步全店活跃活动并按 SKU 查出命中的全部活动
3. 不再以“命中的所有活动”作为退出集合

对每个配置的退出活动：

1. 官方活动：
   - 直接按“配置的活动 + 商品 SKU”执行退出
   - 若商品实际不在该活动中，按接口结果或后置校验记为 `skipped`
2. 店铺活动：
   - 直接按“配置的活动 + 商品 SKU”执行退出
   - 若商品实际不在该活动中，记为 `skipped`

失败判定：

1. “不在活动中”不阻断后续步骤
2. 真实退出失败才把退出步骤记为 `failed`

必要的后置校验仍可保留，但校验范围只能是“用户配置的退出活动”，不能再扩展回全量扫描。

## 历史与详情设计

### 历史列表

页面只保留一张“执行历史”表，来源统一为 `search_cpo_auto_runs`。

每条历史至少展示：

1. 任务 ID
2. 触发方式：自动执行 / 手动触发
3. 执行状态
4. 状态 1/2/3/4 数量统计
5. 创建时间
6. 错误摘要

### 历史详情

每次 run 需要保存并返回：

1. 当次配置快照
   - 默认活动
   - 退出活动
   - 触发时间
2. 每个商品的步骤结果
   - 默认活动步骤
   - 退出活动步骤
   - `enable`
   - `Morkovsk`
3. 每一步的 `success / failed / skipped`
4. 已有的诊断字段
   - `requested_sku`
   - parser / build revision
   - HTTP / response root keys

### 旧历史

1. `search_cpo_runs` 和 `search_cpo_run_items` 不在本轮删除
2. 前端不再展示旧手动报名历史
3. 后端本轮不再为该页面新增手动报名记录

## 数据模型与迁移设计

### `search_cpo_configs`

新增字段：

1. `exit_official_action_ids JSONB NOT NULL DEFAULT '[]'::jsonb`
2. `exit_shop_action_ids JSONB NOT NULL DEFAULT '[]'::jsonb`

### `search_cpo_products`

状态值收口：

1. `state3_trigger` 迁移为 `state3`
2. `morkovsk_joined` 迁移为 `state4`

### `search_cpo_auto_runs`

统计字段收口：

1. 将 `total_state3_trigger` 收口为 `total_state3`
2. 新增 `total_state4`

如果数据库层直接重命名风险较高，则允许通过新增列 + 数据回填实现，最终 API 与前端只暴露 `total_state3` / `total_state4` 新口径。

### 增量脚本

新增增量脚本：

`backend/migrations/upgrade_20260404_search_cpo_automation_single_flow.sql`

脚本职责：

1. 为 `search_cpo_configs` 增加退出活动字段
2. 为自动化历史表补状态 4 统计字段，并收口状态 3 命名
3. 回写已有 `rule_state` 历史值

同时必须回写：

`backend/migrations/init_database.sql`

## 后端实现范围

### 需要调整的模块

1. `backend/internal/model/search_cpo.go`
2. `backend/internal/dto/search_cpo.go`
3. `backend/internal/dto/search_cpo_automation.go`
4. `backend/internal/repository/search_cpo_repo.go`
5. `backend/internal/service/search_cpo_service.go`
6. `backend/internal/service/search_cpo_automation.go`
7. `backend/internal/handler/search_cpo_handler.go`

### 主要服务端改动

1. 配置读写支持退出活动字段
2. 自动化 config snapshot 增加退出活动字段
3. 状态计算收口为状态 1/2/3/4
4. 自动化摘要和详情统计增加状态 4
5. 自动化退出逻辑改为“只处理配置活动”
6. 手动触发继续走自动化表，不再依赖手动报名 run 表

## 前端实现范围

### 需要调整的模块

1. `frontend/src/views/Layout.vue`
2. `frontend/src/views/promotions/SearchCPO.vue`
3. `frontend/src/views/promotions/search-cpo/SearchCPOAutomationTab.vue`
4. `frontend/src/views/promotions/search-cpo/SearchCPOAutomationDetailDialog.vue`
5. `frontend/src/views/promotions/search-cpo/ui.js`
6. `frontend/src/api/promotion.js`

### 主要前端改动

1. 页面标题、菜单名改为“搜索推广自动化”
2. 移除独立手动报名标签与相关组件引用
3. 自动化页面新增“退出活动”配置区
4. 状态文案改为状态 1/2/3/4 新定义
5. 历史列表明确展示自动执行 / 手动触发
6. 自动化详情展示当次默认活动和退出活动配置快照

## 测试与验证

### 后端测试

至少补充以下单测：

1. 配置保存/读取包含退出活动字段
2. 状态 1/2/3/4 判定符合新的技术映射
3. 状态 1 只执行默认活动
4. 状态 2 先退出活动，再 `enable`，再 `Morkovsk`
5. 状态 3 先退出活动，再 `Morkovsk`，不重复 `enable`
6. 状态 4 只执行退出活动清理，跳过重复 `enable` 和 `Morkovsk`
7. 未配置退出活动时，退出步骤记 `skipped`
8. 退出步骤只处理配置活动，不再扫描全量命中活动
9. 手动触发和定时触发都写入统一自动化历史

### 前端验证

至少执行一次：

`cd frontend && npm run build`

重点回归：

1. 页面只剩单一自动化工作面
2. 默认活动和退出活动保存、回显正常
3. 手动执行一次成功创建自动化 run
4. 历史列表同时展示自动执行和手动触发
5. 自动化详情能看到退出步骤结果

## 风险与缓解

1. 风险：旧的 `state3_trigger` / `morkovsk_joined` 历史值与新页面文案不一致
   - 缓解：通过迁移脚本统一回写旧值，并在服务端解码时兼容旧值
2. 风险：去掉“按 SKU 自动找命中的全部活动”后，用户若漏选退出活动，某些历史活动不会被清理
   - 缓解：页面明确提示退出范围完全由用户配置决定
3. 风险：工作区仍保留旧手动报名后端接口，后续维护时容易误用
   - 缓解：前端移除入口；实现计划中明确将该链路标记为遗留、不再扩展

## 实施边界结论

本轮按以下边界实施：

1. Search CPO 收口为单一自动化流程
2. 保留“默认活动”和“退出活动”两组固定配置
3. 自动化支持定时执行和手动触发一次
4. 历史统一记录自动执行和手动触发的所有执行
5. 退出活动严格按用户配置执行，不再按 SKU 扫描全量命中活动

这套方案在保持现有 Search CPO 数据与插件执行链复用的前提下，把页面心智、状态定义、活动配置和执行历史统一成当前业务真正需要的一条主流程。
