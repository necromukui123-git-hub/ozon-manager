# Ozon Manager 当前进度

最后更新时间：2026-03-02  
状态：进行中

## 本轮已完成

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

### 前端

1. 上一轮已完成统一交互接线：
2. 统一报名/退出/亏损处理/改价推广已对接。
3. 已有异步轮询与结果展示。

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

### 验证结果

1. 后端测试通过：`go test ./...`（使用临时 GOCACHE）。
2. 插件 JS 语法检查通过（background/popup/content）。
3. 打包脚本已验证可生成 zip。

## 当前代码变更范围

1. 已修改：
2. `backend/cmd/server/main.go`
3. `backend/internal/repository/automation_repo.go`
4. `backend/internal/service/automation_service.go`
5. `backend/internal/service/promotion_service.go`
6. 已新增：
7. `backend/internal/dto/extension.go`
8. `backend/internal/handler/extension_handler.go`
9. `browser-extension/ozon-shop-bridge/*`

## 下一步优先任务

1. 增加执行引擎路由开关，避免插件与旧 agent 抢任务。
2. 增加前端插件状态面板（在线/错误/最近任务）。
3. 完善非 localhost 域名场景下的 token/shop_id 自动同步策略。
4. 增加 extension 接口授权与任务锁定相关测试用例。

## 风险与注意事项

1. 若旧 agent 与插件同时运行，存在竞争领取任务风险。
2. 当前 content script 自动同步默认面向 localhost 开发环境。
3. 测试分发方式已具备：解压加载 + zip 打包。

## 文档维护规则

1. `OVERALL_TASKS.md`：维护总体路线与关键决策。
2. 本文件：维护短周期执行进展与下一步动作。
3. 每次关键架构或执行变化后，必须同步更新这两个文件。
4. 任务完成后，先更新本文件，再回写总体任务文档。
