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

## 验证结果
1. 后端测试通过：`cd backend && go test ./...`（使用临时 `GOCACHE`）。
2. 前端构建通过：`cd frontend && npm run build`。
3. 插件脚本语法检查通过：`node --check background.js`、`popup.js`、`content-auth-sync.js`。

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
