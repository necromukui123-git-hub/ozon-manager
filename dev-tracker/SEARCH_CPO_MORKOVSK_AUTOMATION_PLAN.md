# Search CPO Morkovsk 自动化规则计划

## Summary
- 当前系统已实现 CPO 隐藏商品抓取、Search CPO 页面手动筛选与报名、普通官方/店铺活动报名、运行历史与页签上下文复用。
- 当前系统未实现 `search_promo_availability`、`product/enable`、`carrots/batch_enable`，也未实现“状态1 -> 状态2 -> 状态3”的规则自动化。
- 目标是基于现有 Search CPO 页面，新增一套既支持手动触发、又支持定时自动触发的规则引擎：
  - 状态1商品：加入初始活动，并显式调用 `enable`
  - 状态2商品：持续观察状态变化
  - 状态3触发商品：先退出其它促销活动，再调用 `carrots/batch_enable` 加入 `Morkovsk`

## 已确认的产品决策
- 第一阶段继续沿用现有 Search CPO 页面里的默认活动配置作为“初始活动”集合，不单独新建一套初始活动配置。
- `Morkovsk` 不走普通活动 ID 配置；它没有活动 ID，加入方式固定为调用 `carrots/batch_enable`。
- 自动化任务既支持手动执行一次，也支持用户开启定时任务并设置时间点自动触发。
- 第一步需要显式调用 `/performance-api/seller-api/search-performance-cpo/mainpage/v1/product/enable`。
- 第三步的退出范围不是“初始活动集合”，而是“除 Morkovsk 外的其它促销活动”；实际执行时需要精确识别商品当前命中的官方/店铺活动后再退出。

## 当前实现盘点
- 已实现：
  - 插件通过 Seller 私有 `list` 接口抓取 CPO 隐藏商品。
  - 插件已支持真实请求头捕获与现有 CPO 页签复用。
  - 后端已支持 Search CPO 配置保存、商品缓存、手动报名 run、历史查询。
  - 前端已有 `/promotions/search-cpo` 页面，支持刷新、筛选、保存默认活动、报名当前筛选结果、查看历史。
  - 后端已有统一报名 / 统一退出 / 自动调度基础设施可复用。
- 未实现：
  - `search_promo_availability` 未接入。
  - `carrotsStatus` 未结构化保存。
  - `product/enable` 未接入。
  - `carrots/batch_enable` 未接入。
  - Search CPO 规则自动化调度与专用运行历史未实现。
  - 店铺活动 remove 逻辑仍按 candidate 匹配，不适合“退出所有其它活动”的场景。

## 目标状态定义
- `state1`
  - `searchPromoStatus = SEARCH_PROMO_STATUS_DISABLED`
  - `carrotsStatus = CARROTS_STATUS_DISABLED`
  - `availability_promo = false`
- `state2`
  - `searchPromoStatus = SEARCH_PROMO_STATUS_DISABLED`
  - `carrotsStatus = CARROTS_STATUS_DISABLED`
  - `availability_promo = true`
- `state3_trigger`
  - 商品曾进入过 `state2`
  - 当前刷新时 `searchPromoStatus = SEARCH_PROMO_STATUS_ENABLED`

## 实现方案

### 1. 后端数据结构扩展
- 扩展 `search_cpo_configs`
  - 新增 `auto_enabled`
  - 新增 `schedule_time`
  - 新增 `enable_step`
- 扩展 `search_cpo_products`
  - 新增 `carrots_status`
  - 新增 `availability_promo`
  - 新增 `availability_checked_at`
  - 新增 `rule_state`
  - 新增 `state2_detected_at`
  - 新增 `morkovsk_joined_at`
  - 继续保留原始 payload
  - 新增 availability 原始 payload 字段
- 新增运行表
  - `search_cpo_auto_runs`
  - `search_cpo_auto_run_items`
- 数据库改动要求
  - 新增增量脚本 `backend/migrations/upgrade_YYYYMMDD_search_cpo_morkovsk_automation.sql`
  - 同步回写 `backend/migrations/init_database.sql`

### 2. 后端服务层
- 在 `SearchCPOService` 基础上新增自动化规则子服务，或拆出 `SearchCPORuleService`
- 新增调度器启动入口，参考 `AutoPromotionService.StartScheduler()`
- 每次执行流程：
  1. 刷新 CPO 商品列表
  2. 批量调用 `search_promo_availability`
  3. 更新商品规则状态
  4. 处理 `state1`
  5. 处理 `state3_trigger`
  6. 写入 run 与 item 明细
- `state1` 处理：
  - 用现有默认活动配置执行初始活动报名
  - 然后调用 `enable`
  - 任一步失败都写逐商品明细
- `state3_trigger` 处理：
  - 同步当前活动列表
  - 刷新活动商品成员关系
  - 找出该 SKU 当前命中的官方/店铺活动
  - 从除 `Morkovsk` 外的命中活动退出
  - 全部退出成功后调用 `carrots/batch_enable`
  - 若退出失败，不进入 `Morkovsk`

### 3. 插件侧
- 在 `background_search_cpo.js` 新增 3 个私有接口 helper
  - `search_promo_availability`
  - `product/enable`
  - `carrots/batch_enable`
- 为新链路增加专用 job type
  - `sync_search_cpo_availability`
  - `search_cpo_enable_products`
  - `search_cpo_batch_enable_morkovsk`
- 店铺 remove 逻辑改造
  - remove 场景不能再只从 candidate 查找
  - 需要优先从 active 产品接口匹配
  - SKU 不在该活动里时应返回 `skipped`，不能报 `failed`
- 响应解析采用宽松策略
  - 当前仓库只有请求抓包，没有完整响应样本
  - 所以响应字段解析要允许多路径回退，并保留原始响应以便排障

### 4. 前端页面
- 保持在现有 `SearchCPO.vue` 中扩展，不新建独立页面
- 新增“规则自动化”配置区
  - 自动化开关
  - 触发时间
  - 固定目标说明：`Morkovsk`
  - 手动执行一次按钮
- 商品列表新增字段展示
  - `carrotsStatus`
  - `availability_promo`
  - 派生规则状态
  - 最近自动化处理时间
- 历史区新增自动化运行历史
  - 展示每个商品的步骤结果：
    - 初始活动报名
    - enable
    - 状态检测
    - 退出其它活动
    - 加入 Morkovsk

## API / DTO / 接口变更
- 扩展 `GET/PUT /api/v1/promotions/search-cpo/config`
  - 返回并保存自动化配置字段
- 新增
  - `POST /api/v1/promotions/search-cpo/automation/run`
  - `GET /api/v1/promotions/search-cpo/automation/runs`
  - `GET /api/v1/promotions/search-cpo/automation/runs/:id`
- 扩展 Search CPO 商品 DTO
  - 增加 `carrots_status`
  - 增加 `availability_promo`
  - 增加 `rule_state`
  - 增加相关时间字段

## 测试要求
- 后端
  - `cd backend && go test ./...`
- 前端
  - `cd frontend && cmd /c npm run build`
- 插件
  - `node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`
- 核心用例
  - `state1` 商品成功报名并 enable
  - `state1` 报名成功但 enable 失败
  - `state2` 商品只观察，不重复 enable
  - `state3_trigger` 商品成功退出其它活动并加入 `Morkovsk`
  - 店铺 remove 遇到不在活动中的 SKU 返回 `skipped`
  - 退出任一活动失败时，不进入 `Morkovsk`
  - 已处理过的商品重复调度保持幂等

## 交付要求
- 更新 `dev-tracker/OVERALL_TASKS.md`
- 更新 `dev-tracker/CURRENT_PROGRESS.md`
- 更新 `dev-tracker/CHANGELOG.md`
- 若涉及数据库变更：
  - 回写 `init_database.sql`
  - 新增增量升级脚本
  - 在 `CURRENT_PROGRESS.md` 记录脚本名、用途、执行条件、执行结果

## 推荐执行顺序
1. 先补插件私有接口能力与后端 DTO/表结构
2. 再补规则状态刷新与手动执行链路
3. 然后补自动调度
4. 最后补 UI 展示、历史详情和交付文档
