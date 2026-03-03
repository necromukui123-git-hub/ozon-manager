# Ozon Manager 当前进度

最后更新时间：2026-03-03  
状态：进行中

## 本轮已完成

### 文档治理

1. 已更新 `AGENTS.md`，新增“开发追踪文档维护规则（强制）”：
2. 任务目标更新时必须同步 `dev-tracker/OVERALL_TASKS.md`。
3. 每完成某项执行任务时必须同步 `dev-tracker/CURRENT_PROGRESS.md`。
4. 已按当前项目实际进度补充 `AGENTS.md`：
5. 新增插件与追踪目录说明（`browser-extension/ozon-shop-bridge`、`dev-tracker`）。
6. 新增插件打包/加载命令。
7. 新增当前执行通道约束与阶段状态摘要。

### 后端

1. 已完成官方 + 店铺混合促销的统一后端路径改造。
2. 新增 extension 任务接口：
3. `POST /api/v1/extension/register`
4. `POST /api/v1/extension/poll`
5. `POST /api/v1/extension/report`
6. `POST /api/v1/extension/reprice`
7. 在 automation repository 中新增“按 shop 领取待执行任务”能力。
8. 新增 extension 服务方法：
9. `ExtensionRegister`
10. `ExtensionPoll`
11. `ExtensionReport`
12. `ExtensionRepriceProduct`
13. extension 支持任务类型扩展为：
14. `sync_shop_actions`
15. `sync_action_products`
16. `shop_action_declare`
17. `shop_action_remove`
18. `promo_unified_enroll`
19. `promo_unified_remove`
20. `remove_reprice_readd`
21. 统一流程提示文案由“等待 Agent”改为“等待浏览器插件执行”。
22. 在 remove-reprice-readd 任务元数据中补充 `shop_actions`，用于统一亏损/改价链路。
23. 已新增店铺执行引擎模式字段：`shops.execution_engine_mode`（`auto`/`extension`/`agent`）。
24. 已新增店铺执行引擎管理接口：
25. `GET /api/v1/my/shops/:id/execution-engine`
26. `PUT /api/v1/my/shops/:id/execution-engine`
27. 已在 `AgentPoll` 引入按店铺路由判断：
28. `extension` 模式下 agent 不领取；
29. `auto` 模式下 extension 在线时 agent 不领取；
30. `auto` 模式下 extension 离线时 agent 兜底领取。
31. 已在 `ExtensionPoll` 引入按店铺路由判断：
32. `agent` 模式下 extension 不领取。
33. 已将 extension 领取写入 `assigned_agent_id`，并在 `ExtensionReport` 增加归属校验（必须与当前 extension 匹配）。
34. 已新增系统管理员插件状态接口：`GET /api/v1/admin/extension-status`。
35. 已补充 service 层测试：执行引擎路由决策 + extension 任务归属校验逻辑。

### 前端

1. 上一轮已完成统一交互接线：
2. 统一报名/退出/亏损处理/改价推广已对接。
3. 已有异步轮询与结果展示。
4. 已在“我的店铺”页增加执行引擎模式展示与编辑（自动/仅插件/仅Agent）。
5. 已在系统概览页新增“插件执行状态”面板（店铺、引擎模式、在线状态、最近任务、错误信息）。
6. 已新增前端 API：`GET /api/v1/admin/extension-status`。

### 插件（新增）

1. 新增插件目录：`browser-extension/ozon-shop-bridge`
2. 已加入 MV3 `manifest.json`。
3. 已实现后台 worker：
4. 轮询后端 extension 接口。
5. 执行支持的店铺促销任务。
6. 实现“静默优先”。
7. 实现登录兜底（仅必要时拉起 Ozon 登录页）。
8. 引入插件专用工作标签页，避免干扰用户当前浏览的 Ozon 页面。
9. 已实现 content script 自动同步 `localStorage` 中 `token/currentShopId`。
10. 已提供 popup 手动配置兜底。
11. 已增加打包脚本：`browser-extension/ozon-shop-bridge/scripts/package.ps1`。
12. 已新增“管理端 Origin（可选）”配置项，用于非 localhost 自动同步。
13. 已新增基于白名单的动态 content script 注册逻辑（按需申请域名权限）。
14. 自动同步策略升级为：`localhost 默认 + 后端同源推导 + 可选管理端 Origin`。

### 验证结果

1. 后端测试通过：`go test ./...`（使用临时 GOCACHE）。
2. 插件 JS 语法检查通过（background/popup/content）。
3. 打包脚本已验证可生成 zip。
4. 前端构建通过：`npm run build`。

## 当前代码变更范围

1. 已修改：
2. `backend/cmd/server/main.go`
3. `backend/internal/model/shop.go`
4. `backend/internal/dto/request.go`
5. `backend/internal/repository/shop_repo.go`
6. `backend/internal/service/shop_service.go`
7. `backend/internal/handler/shop_handler.go`
8. `backend/internal/handler/automation_handler.go`
9. `backend/internal/repository/automation_repo.go`
10. `backend/internal/service/automation_service.go`
11. `backend/migrations/init_database.sql`
12. `frontend/src/api/admin.js`
13. `frontend/src/api/shopAdmin.js`
14. `frontend/src/views/shop-admin/MyShops.vue`
15. `frontend/src/views/super-admin/SystemOverview.vue`
16. `browser-extension/ozon-shop-bridge/background.js`
17. `browser-extension/ozon-shop-bridge/popup.html`
18. `browser-extension/ozon-shop-bridge/popup.js`
19. `browser-extension/ozon-shop-bridge/manifest.json`
20. `browser-extension/ozon-shop-bridge/README.md`
21. 已新增：
22. `backend/internal/service/automation_service_test.go`

## 下一步优先任务

1. 补齐 Chrome 商店上架清单与隐私文案。
2. 增加 extension 领取/回报链路的集成级并发测试（含真实 DB 场景）。
3. 增加执行引擎路由监控指标与告警（extension 在线率、agent fallback 频次、冲突阻断次数）。

## 风险与注意事项

1. 当前已引入路由防抢逻辑，但仍建议在灰度阶段持续观察 mixed mode（agent+extension）下的真实领取行为。
2. 非 localhost 自动同步依赖用户授予域名权限，若拒绝授权需使用手工 token/shop_id 配置。
3. 测试分发方式已具备：解压加载 + zip 打包。

## 文档维护规则

1. `OVERALL_TASKS.md`：维护总体路线与关键决策。
2. 本文件：维护短周期执行进展与下一步动作。
3. 每次关键架构或执行变化后，必须同步更新这两个文件。
4. 任务完成后，先更新本文件，再回写总体任务文档。
