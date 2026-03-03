# Ozon Manager 变更日志

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
