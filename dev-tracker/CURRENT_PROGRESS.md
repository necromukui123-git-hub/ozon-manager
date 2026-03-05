# Ozon Manager 当前进度

最后更新时间：2026-03-05  
状态：进行中（本迭代已交付，等待下一轮任务）

## 本次交付单元
本次目标：完成执行引擎路由防抢、插件状态可视化、非 localhost 自动同步优化、后端关键逻辑测试补齐，规范数据库迁移脚本职责，重构店铺活动商品详情页可读性，修复“商品列表-同步商品”404及失败日志可观测性，并补齐 Ozon 商品核心接口标准说明文档。

## 已完成（含关键文件）
0. 商品列表同步“成功但数据库无数据”修复（按 `/doc` 重构调用与失败语义）：
   - 根因：`/v3/product/info/list` 响应结构与文档存在差异（顶层 `items`），旧实现仅按 `result.items` 解析；且批次错误被 `continue` 吞掉，前端仍提示同步成功。
   - 处理：`ozon` 客户端改为兼容 `items`/`result.items` 双结构，`/v3/product/list` 请求体去除非标准 `current_page`，`product_id` 按文档改为字符串数组。
   - 处理：`ProductService.SyncProducts` 改为“先基础 upsert 再详情补全”；存在批次失败时返回失败，不再假成功；远端有商品但最终 0 落库直接报错。
   - 处理：前端“同步商品”失败提示改为展示后端真实错误文案。
   - 涉及：`backend/pkg/ozon/catalog.go`、`backend/internal/service/product_service.go`、`backend/internal/service/ozon_catalog_service.go`、`backend/pkg/ozon/catalog_test.go`、`frontend/src/views/products/ProductList.vue`。
1. 店铺执行引擎模式落地：`auto` / `extension` / `agent`。  
涉及：`backend/internal/model/shop.go`、`backend/internal/service/shop_service.go`、`backend/internal/handler/shop_handler.go`。
2. 后端路由防抢逻辑：agent/extension 按店铺模式领取任务，`auto` 下 extension 优先。  
涉及：`backend/internal/service/automation_service.go`、`backend/internal/repository/automation_repo.go`。
3. extension 回报归属校验：`assigned_agent_id` 与当前 extension 绑定。  
涉及：`backend/internal/service/automation_service.go`。
4. 系统管理员插件状态接口与面板。  
涉及：`backend/internal/handler/automation_handler.go`、`frontend/src/views/super-admin/SystemOverview.vue`、`frontend/src/api/admin.js`。
5. 店铺管理员可配置执行引擎模式。  
涉及：`frontend/src/views/shop-admin/MyShops.vue`、`frontend/src/api/shopAdmin.js`。
6. 插件非 localhost 自动同步升级为白名单按需授权。  
涉及：`browser-extension/ozon-shop-bridge/background.js`、`popup.html`、`popup.js`、`manifest.json`、`README.md`。
7. 数据库脚本同步：
   - 全量基线：`backend/migrations/init_database.sql` 已回写。
   - 增量升级：新增 `backend/migrations/upgrade_legacy_to_current.sql`。
   - 执行方式：统一直接执行 `upgrade_YYYYMMDD_<topic>.sql`，不再通过 standalone 入口。
8. 文档治理更新：
   - `AGENTS.md` 已补充数据库迁移与发布规则。
   - `dev-tracker/OVERALL_TASKS.md` 已重构为任务看板。
   - 新增 `dev-tracker/CHANGELOG.md`（见历史变更索引）。
9. 插件调用后端 extension 接口 `403 Forbidden` 修复：
   - 根因：后端 CORS 未放行 `chrome-extension://` 来源，导致插件请求被浏览器拦截。
   - 处理：将后端 CORS 改为 `AllowOriginFunc`，精确放行本地前端域名与开发插件 Origin（`chrome-extension://dlfkfajoedolilbndpjkhleljafedcej`）。
   - 涉及：`backend/cmd/server/main.go`。
10. 店铺活动同步抓取兼容修复：
   - 根因：插件使用旧 Seller 活动列表端点（`/api/seller-actions/list` 等）在当前页面返回 404，导致 `sync_shop_actions` 任务失败（未获取到店铺活动数据）。
   - 处理：插件优先调用 `/api/site/marketplace-seller-actions/v2/action/list`（支持分页），并补充新结构字段解析（`skuCount`、`actionParameters.dateStart/dateEnd`）。
   - 涉及：`browser-extension/ozon-shop-bridge/background.js`。
11. 店铺活动商品同步端点兼容补齐（参考 `campaigns` 项目）：
   - 处理：`sync_action_products` 优先改为 `/api/site/own-seller-products/v1/action/{actionId}/candidate`（cursor 分页）。
   - 处理：补充商品价格结构解析（`units + nanos`）以及 `offerID/ozonSku` 字段兼容。
   - 涉及：`browser-extension/ozon-shop-bridge/background.js`。
12. 店铺活动接口权限头兼容（参考 `campaigns` 项目）：
   - 根因：Seller 端点返回 `PermissionDenied: Failed to get company ID`。
   - 处理：为列表/候选/激活/停用请求统一补充 `x-o3-company-id`、`x-o3-language` 请求头（从 Seller cookie 读取）。
   - 涉及：`browser-extension/ozon-shop-bridge/background.js`。
13. 店铺活动同步“后台进行中”但列表未落库的问题修复：
   - 根因：`/promotions/sync-actions` 在等待 `sync_shop_actions` 超时（25s）后直接返回 `shop_sync_pending`，未执行快照导入，导致列表仍仅有官方活动。
   - 处理：超时分支新增兜底逻辑，尝试导入“最近一次成功/部分成功”的店铺活动快照；同时抽取店铺活动快照导入为复用方法。
   - 处理：将店铺活动/活动商品同步等待窗口由 `25s` 提升到 `45s`，降低首轮同步超时概率。
   - 涉及：`backend/internal/service/promotion_service.go`、`backend/internal/service/automation_service.go`、`backend/internal/repository/automation_repo.go`。
14. 活动卡片“更多操作”点击误跳转修复（影响设置别名入口）：
   - 根因：活动卡片整体绑定点击跳转，`...` 触发按钮未拦截原生点击，导致点击菜单按钮直接进入活动商品页。
   - 处理：为更多操作触发按钮增加 `@click.stop`，确保可正常展开下拉菜单并选择“设置中文名称”。
   - 涉及：`frontend/src/views/promotions/ActionList.vue`。
15. 活动别名接口 `shop_id` 参数兼容修复：
   - 根因：前端通过 JSON body 传 `shop_id`，后端仅从 query 读取，导致返回 `400 缺少shop_id参数`。
   - 处理：`UpdateActionDisplayName` 改为优先读取 body 的 `shop_id`，缺失时回退 query，兼容两种调用方式。
   - 涉及：`backend/internal/dto/request.go`、`backend/internal/handler/promotion_handler.go`。
16. 店铺活动日期为空与字段缺失补齐：
   - 根因：插件 `normalizeShopAction` 在提取嵌套 `actionParameters.dateStart/dateEnd` 时传参错误，导致日期未进入快照并最终显示为 `-`。
   - 处理：修复嵌套日期提取逻辑，补齐并归一化 `minimalActionPercent`、`discountType`、`actionBudgetSpent`、`promotionCompanyStatus`、`isEditable/canBeUpdatable`、`highlightUrl` 等字段。
   - 处理：后端扩展 `shopActionSnapshot` 并落入 `promotion_actions.source_payload`，保持数据库结构不变。
   - 处理：前端活动列表新增运营关键标签（最低折扣/预算消耗/可编辑能力）与“活动详情”抽屉，展示完整扩展字段。
   - 涉及：`browser-extension/ozon-shop-bridge/background.js`、`backend/internal/service/promotion_service.go`、`frontend/src/views/promotions/ActionList.vue`。
17. 店铺活动商品详情页信息重构（图片 + 双语 + 双SKU + 价格结构）：
   - 根因：详情页仅展示基础字段，缺少图片、类目、中文语义、折扣/库存结构，不利于运营判断。
   - 处理：插件 `sync_action_products` 补齐 `offer_id/skus/thumbnail/item_type/base_price/action_price/discount_percent/seller_stock/ozon_stock` 等字段提取，并升级去重键策略。
   - 处理：后端扩展 `promotion_action_products` 模型与 API DTO，新增关键词和状态筛选，落库扩展字段并统一中文名/币种兜底逻辑。
   - 处理：前端 `ActionProducts` 页面重构为图片列、双行名称（中文+原文）、双SKU、价格结构、库存结构与筛选搜索。
   - 涉及：`browser-extension/ozon-shop-bridge/background.js`、`backend/internal/model/product.go`、`backend/internal/service/promotion_service.go`、`backend/internal/repository/promotion_repo.go`、`frontend/src/views/promotions/ActionProducts.vue`、`backend/migrations/init_database.sql`。
18. 店铺活动日期与活动商品展示缺口收敛修复：
   - 根因：活动列表部分卡片日期仍空白；商品同步仍优先 `candidate` 导致缩略图/库存/item_type 缺失；编号展示混排导致识别成本高。
   - 处理：插件 `scriptFetchActionProductsPayloads` 调整为优先抓取 `/v2/action/{id}/active`（并兼容 `active-search`），`candidate` 仅作最终兜底。
   - 处理：插件 `normalizeShopAction` 增加 `action_parameters`（snake_case）兼容，避免日期字段漏采。
   - 处理：前端 `ActionList` 增加 `source_payload` 日期回退解析；无日期时显示“日期待同步”，避免空白。
   - 处理：前端 `ActionProducts` 固定三行编号（Offer ID / 平台SKU / Product ID），并优先显示 `item_type`（`category_name`）作为中文主标题。
   - 涉及：`browser-extension/ozon-shop-bridge/background.js`、`frontend/src/views/promotions/ActionList.vue`、`frontend/src/views/promotions/ActionProducts.vue`。
19. 官方活动商品查询 `last_id` 游标对齐修复：
   - 根因：官方 `/v1/actions/products` 自 2025-05-05 起关闭 `offset` 分页，后端仍使用 `offset` 拉取；同时响应商品主键在新结构中可能仅返回 `id`，导致本地 `product_id` 映射失效。
   - 处理：`ozon` 客户端活动商品请求改为 `last_id`，响应增加 `last_id` 解析，并扩展商品字段兼容结构。
   - 处理：后端 `refreshOfficialActionProducts` 改为游标循环，增加游标重复保护与 `id/product_id` 兼容解析，避免异常响应误清空缓存。
   - 处理：官方 API 请求头补充 `Language: ZH_HANS`（与文档/联调截图一致）。
   - 测试：新增 `backend/pkg/ozon/actions_test.go` 与 `backend/internal/service/promotion_service_official_products_test.go`，覆盖请求体/请求头与 ID 兼容逻辑。
   - 涉及：`backend/pkg/ozon/actions.go`、`backend/pkg/ozon/client.go`、`backend/internal/service/promotion_service.go`。
20. 插件“保存并立即同步一次”反馈语义修复（含 token 过期指引）：
   - 根因：按钮文案包含“立即同步”，但界面首行固定显示“保存成功”，用户难以快速判断同步是否失败。
   - 处理：`OZON_MANAGER_SET_CONFIG` 返回本次 `pollOnce` 同步结果（成功/跳过/失败 + 错误原因），并在 popup 首行展示“保存成功 + 同步结果”。
   - 处理：当同步错误包含“认证令牌已过期”时，popup 明确提示“请先在管理端重新登录”。
   - 涉及：`browser-extension/ozon-shop-bridge/background.js`、`browser-extension/ozon-shop-bridge/popup.js`。
21. 插件“有任务但执行失败仍提示成功”误判修复：
   - 根因：popup 仅依据 `sync.ok` 和 `hasJob` 显示“已立即同步一次（有任务）”，未识别 `sync.status=failed`。
   - 处理：`pollOnce` 在 `hasJob=true` 且任务执行失败时回传 `error`（优先提取失败条目错误）。
   - 处理：popup 新增 `sync.status=failed` 分支，首行改为“保存成功，但立即同步失败：<原因>”。
   - 涉及：`browser-extension/ozon-shop-bridge/background.js`、`browser-extension/ozon-shop-bridge/popup.js`。
22. Ozon 实时商品列表能力落地（新增页面，不替换原商品页）：
   - 后端新增商品目录缓存表 `ozon_product_catalog_items`，并新增查询/刷新接口：`GET /api/v1/products/ozon-catalog`、`POST /api/v1/products/ozon-catalog/refresh`。
   - 刷新链路采用 Seller 三接口组合：`/v3/product/list`（列表索引）+ `/v3/product/info/list`（详情）+ `/v3/product/info/stocks`（库存）。
   - 新增上架日期来源判定：优先 Ozon 时间字段，缺失时回退本地同步时间；每条记录返回 `listing_date_source=ozon|local_sync`。
   - 前端新增路由与页面：`/products/ozon`（菜单名“Ozon 商品列表”），支持可见性、OfferID/ProductID、上架日期区间、日期来源筛选及游标翻页。
   - 混合刷新策略落地：页面先读缓存，再触发后台刷新并轮询刷新状态。
   - 涉及：`backend/internal/service/ozon_catalog_service.go`、`backend/internal/repository/ozon_catalog_repo.go`、`backend/pkg/ozon/catalog.go`、`frontend/src/views/products/OzonCatalog.vue`、`frontend/src/router/index.js`、`frontend/src/views/Layout.vue`。
23. “商品列表-同步商品”404修复 + 失败日志可观测性补全：
   - 根因：商品同步仍调用 Seller 旧接口 `/v2/product/list`，在当前环境返回 404。
   - 处理：`ProductService.SyncProducts` 切换至 `/v3/product/list` + `/v3/product/info/list` 组合链路，并补齐 `product_id/id` 兼容映射。
   - 处理：操作日志中间件增加响应体捕获与错误消息提取（`message/error`），日志列表接口透出 `error_message` 字段。
   - 处理：前端请求拦截器修复 `response/config` 变量引用问题，并在 403/404/默认错误分支补充系统日志上报（可静默开关）。
   - 涉及：`backend/internal/service/product_service.go`、`backend/pkg/ozon/catalog.go`、`backend/internal/middleware/operation_log.go`、`backend/internal/handler/operation_log_handler.go`、`backend/internal/dto/response.go`、`frontend/src/utils/request.js`。
24. Ozon 商品核心接口标准说明文档沉淀（`/v3/product/list` + `/v3/product/info/list`）：
   - 处理：新增 `doc/ozon-seller-product-apis-v3-list-info.md`，统一沉淀鉴权、请求参数、分页策略、响应结构、错误处理、cURL 与 Go 示例。
   - 处理：文档明确了“先 list 后 info/list”的标准调用流程，以及游标分页与批量拉取建议。
   - 处理：文档标注官方来源链接与兼容性备注（`/v2/product/list` 废弃、`/v3/product/info/list` 正式化）。
   - 涉及：`doc/ozon-seller-product-apis-v3-list-info.md`。

## 验证结果
0. 后端回归测试通过（含本次商品同步修复）：`cd backend && $env:GOCACHE=\"E:\\developcode\\ozon-manager\\backend\\.gocache\"; go test ./...`。
0. 前端构建通过（含“同步商品”错误提示调整）：`cd frontend && cmd /c npm run build`（非沙箱环境执行）。
1. 后端测试通过：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./...`。
2. 前端构建通过：`cd frontend && npm run build`。
3. 插件脚本语法检查通过：`node --check background.js`、`popup.js`、`content-auth-sync.js`。
4. extension 通道联调验证：页面直调 `POST /api/v1/extension/poll` 返回 `200`（`message: no job`），确认 token / shop_id / 业务权限链路正常。
5. 新端点兼容语法校验通过：`cd browser-extension/ozon-shop-bridge && node --check background.js`。
6. `campaigns` 参考链路对齐：`sync_action_products` 新端点兼容语法校验通过。
7. 公司上下文头兼容语法校验通过：`cd browser-extension/ozon-shop-bridge && node --check background.js`。
8. 后端回归测试通过（含本次同步兜底修复）：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./...`。
9. 前端构建回归通过（含活动卡片更多操作修复）：`cd frontend && npm run build`。
10. 后端回归测试通过（含活动别名接口参数兼容修复）：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./...`。
11. 插件脚本语法检查通过（含店铺活动字段补齐）：`cd browser-extension/ozon-shop-bridge && node --check background.js`。
12. 后端回归测试通过（含 shopActionSnapshot 字段扩展）：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./...`。
13. 前端构建通过（含活动详情抽屉与运营字段标签）：`cd frontend && npm run build`。
14. 后端回归测试通过（含活动商品扩展字段与筛选）：`cd backend && $env:GOCACHE=\"E:\\developcode\\ozon-manager\\backend\\.gocache\"; go test ./...`。
15. 前端构建通过（含活动商品详情页重构）：`cd frontend && npm run build`。
16. 插件脚本语法检查通过（含活动商品字段扩展解析）：`node --check browser-extension/ozon-shop-bridge/background.js`。
17. 前端构建通过（含活动日期回退与三行编号布局）：`cd frontend && npm run build`。
18. 插件脚本语法检查通过（含 `v2 active` 端点优先与 `action_parameters` 兼容）：`node --check browser-extension/ozon-shop-bridge/background.js`。
19. 后端回归测试通过（含官方活动商品 `last_id` 对齐与请求头补齐）：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./...`。
20. 插件脚本语法检查通过（含“保存并立即同步一次”反馈修复）：`node --check browser-extension/ozon-shop-bridge/background.js`、`node --check browser-extension/ozon-shop-bridge/popup.js`。
21. 插件脚本语法检查通过（含 `sync.status=failed` 提示修复）：`node --check browser-extension/ozon-shop-bridge/background.js`、`node --check browser-extension/ozon-shop-bridge/popup.js`。
22. 后端回归测试通过（含 Ozon 商品目录缓存与新接口）：`cd backend && $env:GOCACHE=\"E:\\developcode\\ozon-manager\\backend\\.gocache\"; go test ./...`。
23. 前端构建通过（含 Ozon 商品列表新页面与路由）：`cd frontend && cmd /c npm run build`。
24. 后端回归测试通过（含“同步商品”v3 链路与操作日志错误提取）：`cd backend && $env:GOCACHE=\"E:\\developcode\\ozon-manager\\backend\\.gocache\"; go test ./...`。
25. 前端构建通过（含拦截器错误分支与日志上报修复）：`cd frontend && cmd /c npm run build`（非沙箱环境执行，规避 `spawn EPERM`）。
26. 文档交付核对完成：`doc/ozon-seller-product-apis-v3-list-info.md` 已按工程可用版模板落地，且已与当前仓库 Ozon 客户端调用结构对齐。

## 数据库执行记录
0. 本次（商品同步无数据修复）无新增迁移脚本：仅修正 Ozon API 请求/响应解析、同步失败语义与前端错误提示，不涉及数据库结构变更。
1. 本轮新增可执行升级脚本：`backend/migrations/upgrade_legacy_to_current.sql`（历史总升级）。
2. 本轮新增可执行升级脚本：`backend/migrations/upgrade_20260303_action_products_enrichment.sql`（活动商品详情增强字段）。
3. 用途：为 `promotion_action_products` 增加图片、双语名称、SKU 扩展、价格结构、折扣与分层库存字段。
4. 执行条件：目标库已存在 `promotion_action_products` 表且需要升级到“活动商品增强展示”结构；脚本支持幂等重复执行。
5. 执行结果：开发环境脚本语法检查通过，`init_database.sql` 已同步回写到最新结构。
6. 本次（展示缺口收敛）无新增迁移脚本：仅调整插件采集端点优先级与前端展示回退逻辑。
7. 本次（官方活动商品 `last_id` 对齐）无新增迁移脚本：仅调整官方 API 调用参数、请求头与后端解析逻辑。
8. 本次（插件保存/立即同步反馈修复）无新增迁移脚本：仅调整插件消息返回与 popup 展示文案。
9. 本次（有任务失败提示修复）无新增迁移脚本：仅调整插件状态回传与 popup 展示分支。
10. 本次新增可执行升级脚本：`backend/migrations/upgrade_20260304_ozon_catalog_cache.sql`（Ozon 商品目录缓存表）。
11. 用途：新增 `ozon_product_catalog_items`，用于“先读缓存再后台刷新”的 Ozon 商品列表能力，并支持上架日期来源标记与库存展示。
12. 执行条件：目标库已存在基础业务表；脚本可幂等重复执行。
13. 执行结果：开发环境 SQL 已同步，`init_database.sql` 已回写至最新结构。
14. 本次（商品同步 404 修复 + 日志可观测性补全）无新增迁移脚本：仅涉及 Seller API 调用链路与日志字段透出，不涉及数据库结构变更。
15. 本次（接口文档沉淀）无新增迁移脚本：仅新增 `doc/` 说明文档与 `dev-tracker` 追踪记录，不涉及数据库结构变更。

## 遗留问题
1. Chrome 商店上架材料与隐私文案尚未完成。
2. 缺少真实环境下长时间混合在线回归报告。
3. 执行引擎路由监控指标尚未落地。

## 下一步（最多 3 项）
1. 产出 Chrome 商店上架清单、权限说明与隐私政策文案。
2. 执行多店铺 mixed mode 联调回归并沉淀异常处置手册。
3. 增加路由监控指标和告警（extension 在线率、fallback 次数、冲突阻断次数）。
