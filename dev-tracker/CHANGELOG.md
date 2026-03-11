# Ozon Manager 变更日志

## 2026-03-11（补充）
### 主题
新增“自动加促销”完整链路：配置、调度、候选刷新、执行历史与逐商品失败明细。

### 关键变更
1. 后端新增自动加促销数据模型与迁移：
   - 新增 `promotion_action_candidates`、`auto_promotion_configs`、`auto_promotion_runs`、`auto_promotion_run_items`。
   - 新增升级脚本 `backend/migrations/upgrade_20260311_auto_promotion_add.sql`。
   - `backend/migrations/init_database.sql` 已同步回写到最新结构。
2. 后端新增 `AutoPromotionService` / `AutoPromotionHandler`：
   - 提供配置读取、保存、手动触发、历史列表、详情查询接口。
   - 服务启动后按分钟扫描启用配置，按保存的绝对日期执行。
   - 执行前先刷新 Ozon 目录，再刷新所选活动候选商品缓存。
3. 官方促销执行链增强：
   - `/v1/actions/candidates` 改为 `last_id` 分页。
   - `/v1/actions/products/activate` 解析 `result.rejected[]`，按商品记录失败原因。
4. 插件新增 `sync_action_candidates` 任务：
   - 复用 Seller 候选商品接口同步店铺活动候选商品。
   - 产物类型新增 `action_candidates_snapshot`，供后端导入候选缓存。
5. 前端新增页面：`/promotions/auto-add`。
   - 支持保存“启用状态 + 执行时间 + 绝对日期 + 官方/店铺活动”配置。
   - 支持手动执行、运行中轮询、历史记录和逐商品详情。

### 影响范围
1. 用户可以在独立页面完成自动加促销配置与手动触发，不再依赖“批量报名”页面人工重复操作。
2. 官方与店铺活动都纳入同一条自动执行链，且失败结果可按商品追踪。
3. 本次包含数据库结构变更，需要执行 `upgrade_20260311_auto_promotion_add.sql` 升级旧库。

### 验证
1. 后端回归：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./...` 通过。
2. 插件语法检查：`node --check browser-extension/ozon-shop-bridge/background.js` 通过。
3. 前端构建：`cd frontend && cmd /c npm run build` 通过。

## 2026-03-11
### 主题
新增官方促销接口标准说明文档：`/v1/actions/candidates` 与 `/v1/actions/products/activate`。

### 关键变更
1. 新增文档：`doc/ozon-promos-candidates-activate-standard.md`。
2. 新增机读规范：`doc/ozon-promos-candidates-activate.openapi.yaml`。
3. 文档统一沉淀：
   - 通用鉴权头（`Client-Id`、`Api-Key`）。
   - `POST /v1/actions/candidates` 的请求体、游标分页、`offset` 弃用说明、候选商品响应字段与示例。
   - `POST /v1/actions/products/activate` 的请求体限制（`products <= 1000`）、成功返回、`result.rejected` 结构与示例。
   - 错误响应说明与典型调用顺序（先查促销、再查候选、最后加促销）。
4. 文档补充兼容性说明：
   - `offset` 自 2025-05-05 起不应继续使用，应切换为 `last_id`。
   - `result.rejected[]` 明确为对象数组，包含 `product_id` 与 `reason` 两个字段。
5. 根目录 `.gitignore` 增加精确白名单规则，确保新增的两份 `doc/` 文档可进入版本控制，同时不放开整个 `doc/` 目录。

### 影响范围
1. 为官方促销商品筛选与自动加促销流程提供统一的人读版和机读版接口说明。
2. 不涉及业务代码、数据库结构和接口行为变更。

### 验证
1. 文档与官方来源页面已核对：
   - `https://docs.ozon.ru/api/seller/zh/#operation/PromosCandidates`
   - `https://docs.ozon.ru/api/seller/zh/#operation/PromosProductsActivate`
2. 文档内部字段一致性已人工核对完成。

## 2026-03-05（补充四）
### 主题
修复 Ozon 商品目录刷新 `404 page not found`：库存接口从 `/v3/product/info/stocks` 切换到 `/v4/product/info/stocks`。

### 关键变更
1. `backend/pkg/ozon/catalog.go`：
   - `GetProductStocks` 主调用路径改为 `/v4/product/info/stocks`。
   - 新增兼容回退：当 `v4` 返回 404 时自动降级请求 `/v3/product/info/stocks`，兼容历史环境。
2. `backend/pkg/ozon/catalog_test.go`：
   - 更新库存接口路径断言为 `v4`。
   - 新增 `v4 404 -> v3` 回退测试。

### 影响范围
1. `POST /api/v1/products/ozon-catalog/refresh` 不再因库存端点版本不匹配而报 `404 page not found`。
2. Ozon 商品目录刷新链路在库存接口版本差异场景下具备更高兼容性。
3. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 后端定向测试：`cd backend && go test ./pkg/ozon ./internal/service` 通过。
2. 后端全量测试：`cd backend && go test ./...` 通过。

## 2026-03-05（补充三）
### 主题
修复 Ozon 商品目录刷新失败：兼容 `/v3/product/info/list` 返回 `primary_image` 数组形态。

### 关键变更
1. `backend/pkg/ozon/catalog.go`：
   - `ProductInfoListItem` 增加 `primary_image` 柔性解析，兼容 `string`、`[]string`、对象结构（如 `{url: ...}`）。
   - 对无法识别的 `primary_image` 形态降级为空字符串，不再抛出反序列化错误中断整批同步。
   - 增加 `statuses.status -> status.state` 回填兼容，统一状态读取口径。
2. `backend/pkg/ozon/catalog_test.go`：
   - 新增 `primary_image` 数组、字符串、对象、异常形态兼容测试。
   - 新增 `statuses.status` 回填测试。
3. `backend/internal/service/ozon_catalog_service_test.go`：
   - 新增 `mergeCatalogInfo` 主图优先与 `images[0]` 回退测试，确保目录图像字段稳定。

### 影响范围
1. `POST /api/v1/products/ozon-catalog/refresh` 不再因 `primary_image` 为数组导致刷新任务失败。
2. `GET /api/v1/products/ozon-catalog` 的 `refresh_status.last_error` 在该场景下不再出现反序列化报错。
3. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 后端定向测试：`cd backend && go test ./pkg/ozon ./internal/service` 通过。

## 2026-03-05
### 主题
新增 Ozon 商品核心接口标准说明文档（`/v3/product/list` + `/v3/product/info/list`）。

### 关键变更
1. 新增文档：`doc/ozon-seller-product-apis-v3-list-info.md`。
2. 文档统一沉淀：
   - 通用鉴权与调用约束（`Client-Id`、`Api-Key`、后端调用、限流提示）。
   - `/v3/product/list` 请求参数、游标分页与响应核心字段。
   - `/v3/product/info/list` 批量查询参数与响应核心字段。
   - “先 list 后 info/list”标准流程、常见错误与排障建议。
   - cURL / Go 示例代码（与本仓库客户端封装对齐）。
3. 文档补充兼容性说明：
   - `/v2/product/list` 已废弃，建议统一走 `/v3/product/list`。
   - `/v3/product/info/list` 已在官方文档中转入正式方法。

### 影响范围
1. 为后端对接、联调和排障提供统一的接口说明入口。
2. 不涉及业务代码、数据库结构和接口行为变更。

### 验证
1. 文档与当前仓库实现对照核验：`backend/pkg/ozon/client.go`、`backend/pkg/ozon/catalog.go`、`backend/internal/service/product_service.go`。
2. 官方来源页面已核对：`https://docs.ozon.ru/api/seller/zh/#operation/ProductAPI_GetProductList`。

## 2026-03-05（补充）
### 主题
修复“商品列表同步成功但数据库无数据”问题（接口参数/响应结构对齐 + 同步失败语义收敛）。

### 关键变更
1. `backend/pkg/ozon/catalog.go`：
   - `/v3/product/list` 请求体移除非标准 `current_page`。
   - `/v3/product/info/list` 请求 `product_id` 改为字符串数组（对齐文档）。
   - 商品详情响应兼容顶层 `items` 与历史 `result.items` 两种结构。
2. `backend/internal/service/product_service.go`：
   - 同步流程改为“先基础 upsert（product_id/offer_id）再详情补全”。
   - 批次错误不再静默吞掉；有错误时返回失败，避免前端误提示成功。
   - 远端有商品但本地最终 0 入库时直接失败。
3. `backend/internal/service/ozon_catalog_service.go` 同步兼容 `items` 读取方式。
4. `frontend/src/views/products/ProductList.vue` 同步失败提示改为展示后端真实错误信息。
5. 新增/更新 `backend/pkg/ozon/catalog_test.go`，覆盖：
   - `GetProductListV3` 请求体不含 `current_page`；
   - `product_id` 字符串序列化；
   - `items/result.items` 双响应兼容。

### 影响范围
1. `POST /api/v1/products/sync` 不再出现“失败但显示同步成功”的误导行为。
2. `products` 表在详情批次失败场景下仍可保留基础商品数据，避免整表空白。
3. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 后端测试：`cd backend && $env:GOCACHE=\"E:\\developcode\\ozon-manager\\backend\\.gocache\"; go test ./...` 通过。
2. 前端构建：`cd frontend && cmd /c npm run build` 通过（非沙箱环境执行）。

## 2026-03-05（补充二）
### 主题
修复 `/v3/product/list` 响应结构漂移（保持请求 `filter.visibility`，对齐响应 `items` 新字段）并收敛 Ozon 商品目录可见性推导。

### 关键变更
1. `backend/pkg/ozon/catalog.go`：
   - `ProductListV3Item` 新增 `has_fbo_stocks`、`has_fbs_stocks`、`archived`、`is_discounted`、`quants` 字段映射。
   - 新增 `ProductListV3Quant` 类型，并保留 `visibility` 兼容字段。
2. `backend/internal/service/ozon_catalog_service.go`：
   - 目录刷新不再依赖 list 响应 `visibility`。
   - 可见性改为优先读取 `info.visible`，缺失时回退 `archived`，最终兜底 `ALL`。
3. 测试补齐：
   - `backend/pkg/ozon/catalog_test.go` 覆盖新版响应字段解析与旧 `visibility` 兼容。
   - `backend/internal/service/ozon_catalog_service_test.go` 覆盖可见性推导优先级。

### 影响范围
1. `GetProductListV3` 可直接解析最新 Seller `/v3/product/list` 响应结构。
2. Ozon 商品目录缓存可见性在 `visibility` 缺失场景下保持稳定，不再退化为错误状态。
3. 不涉及数据库结构变更，无新增迁移脚本。

### 验证
1. 后端测试：`cd backend && go test ./pkg/ozon ./internal/service` 通过。

## 2026-03-03
### 主题
执行引擎路由、防抢任务、插件状态面板、非 localhost 同步优化、数据库迁移规则规范化。

### 关键变更
1. 新增店铺执行引擎模式：`auto` / `extension` / `agent`。
2. `AgentPoll` 与 `ExtensionPoll` 按店铺模式进行任务领取路由。
3. `ExtensionReport` 增加任务归属校验（`assigned_agent_id` 匹配）。
4. 新增 `GET /api/v1/admin/extension-status` 并在系统概览展示插件状态。
5. 插件新增管理端 Origin 配置，非 localhost 同步改为白名单按需授权。
6. 新增后端测试：`backend/internal/service/automation_service_test.go`。

### 数据库变更
1. `backend/migrations/init_database.sql` 已同步 `shops.execution_engine_mode`。
2. 新增增量脚本：`backend/migrations/upgrade_legacy_to_current.sql`。
3. 迁移策略更新：仅保留 `upgrade_YYYYMMDD_<topic>.sql`，不再维护 `upgrade_standalone.sql`。

### 验证
1. 后端：`go test ./...` 通过。
2. 前端：`npm run build` 通过。
3. 插件：`node --check background.js popup.js content-auth-sync.js` 通过。

## 2026-03-03（补充）
### 主题
修复 Chrome 插件调用 extension 接口 `403 Forbidden`（CORS 白名单缺失）。

### 关键变更
1. 后端 CORS 从固定 `AllowOrigins` 调整为 `AllowOriginFunc`。
2. 精确放行插件 Origin：`chrome-extension://dlfkfajoedolilbndpjkhleljafedcej`。
3. 保留并补充本地开发前端 Origin 白名单（含 `http://127.0.0.1:5173`）。

### 影响范围
1. 插件可正常访问：`POST /api/v1/extension/register`、`/poll`、`/report`、`/reprice`。
2. 不影响业务鉴权与店铺权限校验逻辑，仅修正跨域入口。

### 验证
1. 页面直调 `POST /api/v1/extension/poll` 返回 `200`（`message: no job`），确认 token/shop 权限链路正常。
2. 后端测试：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./...` 通过。

## 2026-03-03（补充二）
### 主题
修复店铺活动同步抓取失败（Seller 活动列表端点升级兼容）。

### 关键变更
1. 插件 `sync_shop_actions` 优先改为调用新端点：`/api/site/marketplace-seller-actions/v2/action/list`。
2. 新端点增加分页抓取（`offset/limit`），避免仅取首批数据。
3. 兼容新响应字段解析：`skuCount`、`actionParameters.dateStart/dateEnd`。
4. 保留旧端点作为回退策略，降低页面变体风险。

### 影响范围
1. 解决“官方活动可同步、店铺活动 `shop actions sync failed`”的问题。
2. 不影响后端鉴权与任务路由逻辑，仅修复插件店铺活动数据采集层。

### 验证
1. 插件脚本语法检查：`cd browser-extension/ozon-shop-bridge && node --check background.js` 通过。

### 继续补齐（同日）
1. `sync_action_products` 同步接口对齐 `campaigns` 项目，优先走 `/api/site/own-seller-products/v1/action/{actionId}/candidate`（cursor 分页）。
2. 商品数据解析补充 `offerID/ozonSku` 字段，并支持价格对象（`units+nanos`）转数值。
3. 对齐 `campaigns` 请求头策略：店铺活动列表/候选/激活/停用统一补充 `x-o3-company-id` 与 `x-o3-language`，修复 `PermissionDenied: Failed to get company ID`。

## 2026-03-03（补充三）
### 主题
修复“同步活动返回后台进行中，但列表仍无店铺活动”的落库缺口。

### 关键变更
1. `SyncPromotionActionsV2` 抽取店铺活动快照导入逻辑为复用方法，统一处理导入与 upsert。
2. 当 `sync_shop_actions` 等待超时时，保留 `shop_sync_pending=true`，同时尝试导入“最近一次成功/部分成功任务”的快照，避免列表长期为空。
3. `AutomationRepository` 新增按店铺+任务类型+状态查询最近任务的方法，供同步兜底逻辑使用。
4. 将店铺活动同步与活动商品同步等待窗口由 `25s` 提升至 `45s`，降低首轮同步超时概率。

### 影响范围
1. 前端点击“同步活动”即使返回 pending，也能优先展示最近可用的店铺活动数据（若存在成功快照）。
2. 不改变 extension/agent 任务路由，仅补齐同步接口的结果可见性。

### 验证
1. 后端测试：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./...` 通过。

## 2026-03-03（补充四）
### 主题
修复活动列表页“更多操作”点击后直接跳转，导致无法设置活动别名。

### 关键变更
1. 在活动卡片 `...` 触发按钮添加 `@click.stop`，阻止卡片父级点击事件冒泡。
2. 确保下拉菜单可正常展开，用户可点击“设置中文名称”入口。

### 影响范围
1. 促销活动列表页支持正常设置活动别名（`display_name`）。
2. 不影响活动卡片正常点击进入商品列表的行为。

### 验证
1. 前端构建：`cd frontend && npm run build` 通过。

## 2026-03-03（补充五）
### 主题
修复设置活动别名时报 `400 缺少shop_id参数`。

### 关键变更
1. `UpdateActionDisplayNameRequest` 增加 `shop_id` 字段，支持 body 传参。
2. `UpdateActionDisplayName` 处理逻辑改为优先读取 body `shop_id`，缺失时回退读取 query `shop_id`，兼容历史调用。

### 影响范围
1. 活动别名保存接口兼容前端当前调用方式（JSON body）。
2. 不影响既有 query 传 `shop_id` 的调用。

### 验证
1. 后端测试：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./...` 通过。

## 2026-03-03（补充六）
### 主题
修复店铺活动日期显示为空，并补齐活动运营关键字段展示。

### 关键变更
1. 插件 `normalizeShopAction` 修复 `actionParameters.dateStart/dateEnd` 嵌套字段提取错误，确保活动日期可同步入库。
2. 插件补充输出店铺活动扩展字段：`minimalActionPercent`、`discountType`、`actionBudgetSpent`、`promotionCompanyStatus`、`isEditable`、`canBeUpdatable`、`isParticipated`、`isTurnOn`、`isRepricerAvailable`、`highlightUrl`、`createdAt`、`status`。
3. 后端扩展 `shopActionSnapshot` 结构，扩展字段统一落到 `promotion_actions.source_payload`，数据库结构保持不变。
4. 前端活动列表新增运营标签（最低折扣/预算消耗/可编辑能力），并新增活动详情抽屉展示完整字段。

### 影响范围
1. 活动列表卡片日期不再显示 `-`（接口返回日期有效时）。
2. 运营可直接在系统内查看店铺活动关键状态，无需回 Seller 页面二次核对。
3. 无新增数据库迁移脚本，兼容现有环境与数据。

### 验证
1. 插件语法检查：`cd browser-extension/ozon-shop-bridge && node --check background.js` 通过。
2. 后端测试：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./...` 通过。
3. 前端构建：`cd frontend && npm run build` 通过。

## 2026-03-03（补充七）
### 主题
重构店铺活动商品详情页信息结构，补齐图片/双语/双SKU/价格库存语义。

### 关键变更
1. 插件 `normalizeActionProduct` 扩展商品字段提取：
   - 新增 `offer_id`、`platform_sku`、`thumbnail_url`、`category_name`、`name_cn`、`name_origin`、`currency`。
   - 新增价格结构：`base_price`、`action_price`、`marketplace_price`、`min_seller_price`、`max_action_price`、`discount_percent`。
   - 新增库存结构：`seller_stock`、`ozon_stock`，并兼容 `is_active` 到状态映射。
2. 插件活动商品去重键升级为 `offer_id + ozon_product_id` 优先，降低错去重风险。
3. 后端扩展 `promotion_action_products` 模型与接口返回字段，支持关键词和状态筛选。
4. 后端活动商品刷新逻辑新增本地商品中文名补全、币种兜底和折扣推导。
5. 前端 `ActionProducts` 页面重构为图片列 + 双行名称 + 双SKU + 价格结构 + 库存结构，并新增筛选搜索交互。

### 数据库变更
1. `backend/migrations/init_database.sql` 已回写活动商品增强字段。
2. 新增增量脚本：`backend/migrations/upgrade_20260303_action_products_enrichment.sql`。

### 验证
1. 后端测试：`cd backend && $env:GOCACHE=\"E:\\developcode\\ozon-manager\\backend\\.gocache\"; go test ./...` 通过。
2. 前端构建：`cd frontend && npm run build` 通过。
3. 插件语法检查：`node --check browser-extension/ozon-shop-bridge/background.js` 通过。

## 2026-03-03（补充八）
### 主题
收敛活动日期与活动商品展示缺口（日期空白、缩略图/库存/item_type 缺失、编号混乱）。

### 关键变更
1. 插件 `sync_action_products` 抓取端点改为优先：
   - `/api/site/own-seller-products/v2/action/{actionId}/active`
   - `/api/site/own-seller-products/v2/action/{actionId}/active-search`
   - `/api/site/own-seller-products/v1/action/{actionId}/candidate`（最终兜底）
2. 插件 `normalizeShopAction` 增加 `action_parameters`（snake_case）解析兼容，降低活动日期漏采概率。
3. 前端 `ActionList` 增加活动日期回退解析（含 `source_payload` 嵌套参数），无日期时统一显示“日期待同步”。
4. 前端 `ActionProducts` 编号展示改为三行固定标签：
   - Offer ID
   - 平台 SKU
   - Product ID
5. 前端活动商品中文主标题改为优先显示 `category_name`（来自 `item_type`），并保持原文副标题。

### 影响范围
1. 活动列表不再出现“日期区域空白但无占位文案”的情况。
2. 活动商品详情页可稳定显示缩略图/卖家库存/中文类目（源端可用时）。
3. 商品编号语义清晰，避免 `source_sku/offer_id/product_id` 混排误读。

### 验证
1. 插件语法检查：`node --check browser-extension/ozon-shop-bridge/background.js` 通过。
2. 前端构建：`cd frontend && npm run build` 通过。

## 2026-03-03（补充九）
### 主题
修复官方活动商品无法展示（`/v1/actions/products` 分页与请求头对齐）。

### 关键变更
1. `backend/pkg/ozon/actions.go`：官方活动商品查询从 `offset` 分页切换为 `last_id` 游标分页，请求/响应结构新增 `last_id`。
2. `backend/pkg/ozon/client.go`：官方 API 请求统一补充 `Language: ZH_HANS` 请求头。
3. `backend/internal/service/promotion_service.go`：
   - 官方活动商品刷新逻辑改为 `last_id` 循环拉取；
   - 增加游标重复保护，避免异常游标导致死循环；
   - 商品主键兼容 `product_id` 与 `id`，修复新响应结构下 ID 映射失效问题；
   - 当远端返回商品但均无有效 ID 时返回错误，避免误清空本地缓存。
4. 新增测试：
   - `backend/pkg/ozon/actions_test.go` 覆盖请求体去除 `offset`、携带 `last_id` 与 `Language` 请求头；
   - `backend/internal/service/promotion_service_official_products_test.go` 覆盖 `id/product_id` 兼容选择逻辑。

### 影响范围
1. 官方活动商品详情页恢复可见（数据链路：官方 API -> 后端缓存表 -> 页面查询）。
2. 店铺活动商品链路不受影响。
3. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 后端测试：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./...` 通过。

## 2026-03-04（补充）
### 主题
修复插件“有任务但执行失败仍显示成功”的提示误判。

### 关键变更
1. `background.js`：`pollOnce` 在 `hasJob=true` 且执行状态为 `failed` 时，回传 `error` 字段（优先使用失败条目错误信息）。
2. `popup.js`：`buildSaveSummary` 新增 `sync.status=failed` 分支，首行改为“保存成功，但立即同步失败：<原因>”。

### 影响范围
1. 点击“保存并立即同步一次”后，插件首行提示与任务真实执行状态一致。
2. 降低“已执行成功”误判导致的排障时间。
3. 无后端接口变更、无数据库结构变更。

### 验证
1. 插件语法检查：`node --check browser-extension/ozon-shop-bridge/background.js` 通过。
2. 插件语法检查：`node --check browser-extension/ozon-shop-bridge/popup.js` 通过。

## 2026-03-04（补充二）
### 主题
新增 Ozon 实时商品列表能力（缓存查询 + 后台刷新 + 日期来源区分）。

### 关键变更
1. 后端新增 Ozon 商品目录缓存仓储与服务：
   - 新增缓存查询接口 `GET /api/v1/products/ozon-catalog`。
   - 新增刷新触发接口 `POST /api/v1/products/ozon-catalog/refresh`。
2. Seller 接口调用链路扩展：
   - 新增 `/v3/product/list` 调用（列表索引）。
   - 新增 `/v3/product/info/list` 调用（详情补全）。
   - 新增 `/v3/product/info/stocks` 调用（库存补全）。
3. 新增上架日期来源逻辑：
   - 优先解析 Ozon 返回时间字段；
   - 缺失时回退本地同步时间；
   - 每条记录标注 `listing_date_source=ozon|local_sync`。
4. 前端新增菜单与页面：
   - 新路由 `/products/ozon`。
   - 页面支持可见性、OfferID/ProductID、上架日期区间、日期来源筛选，支持游标上一页/下一页与手动刷新。
   - 页面默认“先读缓存，再后台刷新并轮询状态”。

### 数据库变更
1. 新增增量脚本：`backend/migrations/upgrade_20260304_ozon_catalog_cache.sql`。
2. 全量基线回写：`backend/migrations/init_database.sql` 已同步新增 `ozon_product_catalog_items`。

### 验证
1. 后端测试：`cd backend && $env:GOCACHE=\"E:\\developcode\\ozon-manager\\backend\\.gocache\"; go test ./...` 通过。
2. 前端构建：`cd frontend && cmd /c npm run build` 通过。

## 2026-03-04（补充三）
### 主题
修复“商品列表-同步商品”失败（404）并增强失败日志可观测性。

### 关键变更
1. `ProductService.SyncProducts` 从 Seller 旧接口 `/v2/product/list` 切换到 `/v3/product/list`，并使用 `/v3/product/info/list` 批量拉取详情。
2. 同步详情解析增加 `product_id/id` 兼容映射，避免因字段差异导致商品记录被跳过。
3. 操作日志中间件新增响应体捕获与错误消息提取逻辑，失败记录可直接落地 `error_message`。
4. 操作日志查询接口返回新增 `error_message` 字段，前端可直接显示后端已解析的失败原因。
5. 前端请求拦截器修复错误分支变量引用问题，并补充 403/404/默认错误场景的系统日志上报（支持 `silent` 开关）。

### 影响范围
1. “商品列表 -> 同步商品”不再因旧版 Ozon 接口 404 直接失败。
2. 出现 API 失败时，运维可在系统日志/操作日志快速看到明确错误文案，缩短排障路径。
3. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 后端测试：`cd backend && $env:GOCACHE=\"E:\\developcode\\ozon-manager\\backend\\.gocache\"; go test ./...` 通过。
2. 前端构建：`cd frontend && cmd /c npm run build` 通过（非沙箱执行，规避 `spawn EPERM`）。
