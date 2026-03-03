# Ozon Manager 当前进度

最后更新时间：2026-03-03  
状态：进行中（本迭代已交付，等待下一轮任务）

## 本次交付单元
本次目标：完成执行引擎路由防抢、插件状态可视化、非 localhost 自动同步优化、后端关键逻辑测试补齐，并规范数据库迁移脚本职责。

## 已完成（含关键文件）
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

## 验证结果
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

## 数据库执行记录
1. 本轮新增可执行升级脚本：`backend/migrations/upgrade_legacy_to_current.sql`。
2. 适用场景：已有历史数据库升级到当前结构。
3. 执行方式：由维护者复制脚本到 Navicat 或使用 `psql -f` 执行。
4. 历史策略更新：不再维护 `upgrade_standalone.sql`，仅保留版本化升级脚本作为历史记录。

## 遗留问题
1. Chrome 商店上架材料与隐私文案尚未完成。
2. 缺少真实环境下长时间混合在线回归报告。
3. 执行引擎路由监控指标尚未落地。

## 下一步（最多 3 项）
1. 产出 Chrome 商店上架清单、权限说明与隐私政策文案。
2. 执行多店铺 mixed mode 联调回归并沉淀异常处置手册。
3. 增加路由监控指标和告警（extension 在线率、fallback 次数、冲突阻断次数）。
