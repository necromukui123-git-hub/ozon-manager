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
