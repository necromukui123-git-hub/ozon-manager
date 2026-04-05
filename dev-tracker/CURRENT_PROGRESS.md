# Ozon Manager 当前进度

最后更新时间：2026-04-04
状态：进行中（Search CPO 单一自动化流程已完成开发与构建/测试校验；业务主线继续推进真实环境回归、Chrome 商店上架准备与执行引擎监控）

## 本次交付单元
本次目标：完成 Search CPO 从“双标签（商品池与手动报名 + 状态迁移自动化）”到“单一自动化页面”的整体收口；同时落地退出活动配置、状态1/2/3/4新口径、统一执行历史与退出失败后的跳过规则。

## 已完成（含关键文件）
0. Search CPO 单一自动化流程已落地：
   - 后端配置与测试：`backend/internal/model/search_cpo.go`、`backend/internal/dto/search_cpo.go`、`backend/internal/repository/search_cpo_repo.go`、`backend/internal/service/search_cpo_service.go`、`backend/internal/service/search_cpo_service_test.go` 已支持 `exit_official_action_ids/exit_shop_action_ids`；补齐了真实 `GetConfig/UpdateConfig` 读写链路测试与 source mismatch 校验覆盖。对应提交：`24294ae`、`29615c7`、`929d63d`。
   - 后端状态与自动化链路：`backend/internal/model/search_cpo.go`、`backend/internal/dto/search_cpo_automation.go`、`backend/internal/service/search_cpo_automation.go`、`backend/internal/service/search_cpo_automation_test.go` 已把状态口径收口为状态1/2/3/4，兼容旧 `state3_trigger/morkovsk_joined` 值，并将自动化历史统计改为 `total_state3/total_state4`。对应提交：`261ca30`。
   - 退出逻辑：`processMigrationItems()` 已改为只处理用户显式配置的退出活动；状态2/3/4 若退出失败，会显式把后续 `enable/Morkovsk` 标记为 `skipped`，并在详情消息写入“退出促销活动失败，跳过后续动作”；当次 run 配置快照会一并保存默认活动与退出活动。对应提交：`1f5a36e`。
   - 前端单页收口：`frontend/src/views/promotions/SearchCPO.vue`、`frontend/src/views/promotions/search-cpo/SearchCPOAutomationTab.vue`、`frontend/src/views/promotions/search-cpo/SearchCPOAutomationDetailDialog.vue`、`frontend/src/views/promotions/search-cpo/ui.js` 已改为单一自动化工作面；`SearchCPOManualTab.vue` 与 `SearchCPORunDetailDialog.vue` 已删除，不再展示商品池筛选、手动报名按钮或旧手动报名历史。
   - 数据库脚本：`backend/migrations/upgrade_20260404_search_cpo_automation_single_flow.sql` 已正式创建并回写 `backend/migrations/init_database.sql`。用途：为 `search_cpo_configs` 增加退出活动字段、把 `rule_state` 收口到 `state3/state4`、将自动化汇总字段收口到 `total_state3/total_state4`。执行条件：旧库需要从已包含 Search CPO 基础表结构的版本升级到本轮单一自动化流版本。执行结果：脚本已入库并通过代码侧回归验证，本次未直接在现有业务库执行。
   - 验证：
     - `cd backend && $env:GOCACHE="$env:TEMP\\ozon-manager-gocache"; go test ./internal/service -count=1` 通过。
     - `cd frontend && cmd /c npm run build` 通过。
0. Search CPO 单一自动化流程的设计与实现计划已确认：
   - `docs/superpowers/specs/2026-04-04-search-cpo-automation-single-flow-design.md`：已确认 Search CPO 将从“商品池与手动报名 + 状态迁移自动化”双标签收口为单一自动化页面；保留“默认活动 + 退出活动”两组固定配置；状态定义重收口为状态1/2/3/4；退出失败时该商品后续动作必须显式跳过并在详情展示“退出促销活动失败，跳过后续动作”。
   - `docs/superpowers/plans/2026-04-04-search-cpo-automation-single-flow.md`：已拆出后端配置扩展、状态重命名、退出逻辑改造、前端单页重构、统一历史展示、`dev-tracker` 与验证收尾六个实施任务；本轮已按计划完成核心代码实现与验证。
   - 相关数据库脚本：`backend/migrations/upgrade_20260404_search_cpo_automation_single_flow.sql` 已在本轮创建并纳入执行计划，详见上方“Search CPO 单一自动化流程已落地”条目。
0. 开发启动命令收敛修复：
   - `start-dev.bat` 已将后端启动命令从 `go run cmd/server/main.go` 改为 `go run ./cmd/server`，`frontend_static.go` 等同包文件会随包一起编译，不再出现 `isAllowedOrigin`、`detectFrontendWebRoot`、`configureFrontendStatic` 未定义。
   - `AGENTS.md`、`CLAUDE.md` 已同步修正后端运行/构建示例，避免后续协作和手工命令继续按单文件方式启动或构建 `cmd/server`。
   - 新增 `scripts/check-go-package-entry.ps1` 作为轻量回归校验，约束 `start-dev.bat`、`AGENTS.md`、`CLAUDE.md` 必须使用 `./cmd/server` 包路径。
0. 开发启动端口守卫修复：
   - `start-dev.bat` 已新增 8080/5173 监听检测；若后端或前端已在运行，会直接提示 `already listening ... skipping start`，不再重复拉起新窗口并触发 Vite `Port 5173 is already in use`。
   - 新增 `scripts/check-start-dev-port-guards.ps1` 作为轻量回归校验，约束 `start-dev.bat` 必须保留端口守卫逻辑与提示文案。
   - 已用单进程临时 `TcpListener` 占住 8080/5173 现场验证 `start-dev.bat`，确认脚本会跳过重复启动而不是继续撞端口。
0. 协作文档与仓库现状对齐：
   - `AGENTS.md` 已补充当前有效的插件打包入口 `package-browser-extension.ps1`、Windows 发布脚本 `build-windows-release.ps1`、插件 service worker 实际入口 `background_search_cpo_bootstrap.js` 与 patch 加载链说明。
   - `AGENTS.md` 已同步补充 access token + refresh token 会话约束、`user_refresh_tokens` 基线说明、店铺 `execution_engine_mode` 约束，以及当前阶段重点从“待实现的插件接入能力”改为“商店上架、真实环境回归、执行引擎监控与 Search CPO live 收敛”。
   - `dev-tracker/CURRENT_PROGRESS.md`、`dev-tracker/CHANGELOG.md` 已同步记录本次文档收敛；本次无数据库结构变更，无新增 migration 脚本。
0. access token + refresh token 认证升级已完成代码实现与验证：
   - 设计与实施文档：`docs/superpowers/specs/2026-03-31-auth-refresh-token-design.md`、`docs/superpowers/plans/2026-03-31-auth-refresh-token-implementation.md` 已作为本轮实现基线保留。
   - 后端认证链路：`backend/pkg/jwt/jwt.go`、`backend/internal/service/auth_service.go`、`backend/internal/handler/auth_handler.go`、`backend/cmd/server/main.go` 已切到短期 access token + `HttpOnly` refresh cookie；新增公开 `POST /api/v1/auth/refresh` 与公开 `POST /api/v1/auth/logout`，refresh 成功会轮换 refresh token 并返回新的 `token_expires_at`。
   - refresh token 持久化：`backend/internal/model/refresh_token.go`、`backend/internal/repository/refresh_token_repo.go`、`backend/migrations/init_database.sql`、`backend/migrations/upgrade_20260331_refresh_tokens.sql` 已新增 `user_refresh_tokens` 表、唯一 `token_hash` 与 `family_id` 索引；旧库升级执行条件为“当前环境尚未存在 `user_refresh_tokens` 表或未包含本轮 refresh token 字段/索引”，本次代码交付未直接对现成业务库执行 SQL，结果为“脚本已创建并通过 `go test ./migrations ./internal/repository` 与 `go test ./...` 回归验证”。
   - 会话撤销：`backend/internal/service/user_service.go` 已在改密、重置店铺管理员/员工密码、禁用店铺管理员/员工、通用用户禁用等场景下同步撤销该用户全部 refresh token。
   - 前端续期：`frontend/src/stores/user.js`、`frontend/src/utils/request.js`、`frontend/src/api/auth.js`、`frontend/src/main.js` 已接入 `token_expires_at` 本地状态、应用启动静默 refresh、401 单飞刷新、refresh 成功后的 `localStorage.token` 更新，以及失败后的统一清理与跳转登录页。
   - 插件兼容边界：本轮未改 Chrome 插件独立 refresh；插件继续依赖管理端 `localStorage.token/currentShopId`，因此前端 refresh 成功后仍会同步更新 `localStorage.token`。
   - 本轮验证已完成：`cd backend && $env:GOCACHE="$PWD\.gocache-all"; $env:GOMODCACHE="$PWD\.gomodcache"; $env:GOPATH="$PWD\.gopath"; go test ./...` 通过；`cd frontend && cmd /c npm run build` 通过。
0. 店铺 API Key 清洗与同步活动错误前移：
   - `backend/internal/service/shop_service.go`、`backend/internal/service/shop_service_test.go`：店铺创建/更新新增 `normalizeShopAPIKey`，统一清理前后空白并拒绝空白 API Key；补了对应单测，避免脏值继续落库。
   - `backend/pkg/ozon/client.go`、`backend/pkg/ozon/actions_test.go`：Ozon client 发请求前会再次 `TrimSpace(Api-Key)`，并把官方 400 `Invalid Api-Key` 识别成 `ErrInvalidAPIKey`；补了 header 清洗和错误映射测试。
   - `backend/internal/handler/promotion_handler.go`、`backend/internal/handler/shop_handler.go`：`sync-actions` 与店铺保存接口现会把 API Key 问题明确映射成 400 与中文提示，不再把上游报错整段透给用户。
   - `frontend/src/views/shop-admin/MyShops.vue`：店铺创建时会提交 trim 后的 `client_id/api_key`；编辑时只在用户实际输入新值时才回传，避免把旧接口未返回的凭证字段误写成空值或字符串 `undefined`。
   - 验证：`cd backend && $env:GOCACHE="$PWD\.gocache-build"; go test ./internal/service ./pkg/ozon` 通过；`cd backend && $env:GOCACHE="$PWD\.gocache-build"; go test ./...` 通过；`cd frontend && cmd /c npm run build` 通过。
0. `users.owner_id` 基线缺口修复：
   - `backend/migrations/init_database.sql`：`users` 表已补齐 `owner_id` 字段和 `idx_users_owner_id` 索引，新库初始化后可直接创建店铺管理员、员工并查询上下级关系。
   - `backend/migrations/upgrade_20260325_users_owner_id.sql`：新增旧库增量脚本，可幂等补齐 `owner_id` 字段、外键和索引，直接修复目标机现有数据库的 `SQLSTATE 42703`。
   - `backend/migrations/init_database_test.go`：新增回归测试，锁定 `users` 表必须包含 `owner_id` 字段与索引，避免模型和基线再次漂移。
   - `build-windows-release.ps1`、`README-windows-deploy.md`：发布包现会一并带上 `upgrade_*.sql`，部署说明也已明确旧库遇到该报错时应执行 `server/database/upgrade_20260325_users_owner_id.sql`。
   - 验证：`cd backend && $env:GOCACHE="$PWD\.gocache-build"; go test ./migrations` 通过；`cd backend && $env:GOCACHE="$PWD\.gocache-build"; go test ./...` 通过；`cd frontend && cmd /c npm run build` 通过。
0. 默认管理员哈希与登录失败提示修复：
   - `backend/migrations/init_database.sql`：空库默认 `super_admin` 的 `password_hash` 已改成真正匹配当前登录协议 `bcrypt(SHA256(admin123))` 的基线值，重新初始化后的基线可直接登录。
   - `backend/cmd/reset-password/main.go`、`backend/cmd/migrate-passwords/main.go`：密码工具统一改回与当前登录逻辑一致的 `bcrypt(SHA256(明文密码))`，避免生成与前端登录链路不兼容的哈希。
   - `frontend/src/utils/request.js`：登录接口 `POST /auth/login` 返回 `401` 时，前端不再误提示“登录已过期”，而是直接显示后端返回的登录失败消息。
   - `backend/migrations/init_database_test.go`：新增回归测试，直接校验基线 SQL 中 `super_admin` 的哈希必须满足 `bcrypt(SHA256(admin123))`，并拒绝把原始 `admin123` 误当成存储协议。
   - 验证：`cd backend && $env:GOCACHE="$PWD\.gocache-build"; go test ./migrations` 通过；`cd backend && $env:GOCACHE="$PWD\.gocache-build"; go test ./...` 通过；`cd frontend && cmd /c npm run build` 通过。

0. Windows 单机发布包与异机部署链路：
   - `backend/cmd/server/main.go`、`backend/cmd/server/frontend_static.go`、`backend/cmd/server/frontend_static_test.go`：后端新增可选 `web/` 静态托管与 SPA fallback，`/api/*` 未命中仍保持 JSON 404；CORS 改为放行任意 `chrome-extension://` 来源，目标机手工加载插件后可直接访问本机后端。
   - `build-windows-release.ps1`、`package-browser-extension.ps1`、`start-release-server.bat`、`README-windows-deploy.md`：新增统一发布链路，可一次性构建前端 `dist`、后端 `server.exe`、插件 zip 与 `release/ozon-manager-win-x64` 目录；发布包内含 `server/start-ozon-manager.bat`、`server/database/init_database.sql`、部署说明和可直接加载的插件目录。
   - 本次无新增 migration 脚本、无表结构变更；目标机空库只需执行 `init_database.sql` 的发布副本，不迁移旧数据。
   - 验证：`cd backend && $env:GOCACHE="$PWD\.gocache-build"; go test ./cmd/server` 通过；`cd frontend && cmd /c npm run build` 通过；`powershell -ExecutionPolicy Bypass -File .\build-windows-release.ps1` 已在非沙箱环境完整跑通，成功产出 `release/ozon-manager-win-x64` 与同名 zip。

0. 搜索推广商品页面信息架构收口：
   - `frontend/src/views/promotions/SearchCPO.vue` 现只保留共享数据加载、轮询和 `?tab=manual|automation` 标签路由；同一路由下拆成“商品池与手动报名”和“状态迁移自动化”两个工作面，并把顶部概览收口为缓存商品、当前筛选、默认活动、最近同步、自动化状态。
   - 新增 `frontend/src/views/promotions/search-cpo/SearchCPOManualTab.vue`、`SearchCPOAutomationTab.vue`、`SearchCPORunDetailDialog.vue`、`SearchCPOAutomationDetailDialog.vue` 与 `ui.js`；手动报名和自动化的主界面、历史列表、详情弹窗已从单文件拆出，默认活动配置改为只在手动标签编辑，自动化标签只展示共享配置摘要与规则状态概览。
   - `frontend/src/views/Layout.vue` 菜单入口已从“CPO 商品报名”改为“搜索推广商品”，用户不再需要在同一屏里同时处理商品筛选、活动配置、调度开关、两套历史和深度诊断。
   - 前端构建通过：`cd frontend && cmd /c npm run build`。
0. Search CPO 刷新状态落库修复：
   - `browser-extension/ozon-shop-bridge/background_search_cpo.js` 的 `normalizeSearchCPOProduct` 已补齐 `carrots_status` 映射，`list` 返回的 `carrotsStatus` 不再只留在原始 `payload` 中。
   - `frontend/src/views/promotions/SearchCPO.vue` 已把空 `search_promo_status` 改为显示“状态未知”，不再误标成“已关闭”。
1. Search CPO 自动化 SKU 语义修复：
   - 后端通过 `search_cpo_meta/search_cpo_morkovsk_meta` 下发 `source_sku -> sku` 映射；extension 执行 `sync_search_cpo_availability`、`search_cpo_enable_products`、`search_cpo_batch_enable_morkovsk` 时改为请求数值 `sku`，但回报和运行历史仍按 `source_sku` 聚合。
   - 涉及：`backend/internal/service/automation_service_search_cpo.go`、`backend/internal/handler/extension_handler.go`、`backend/internal/handler/automation_handler.go`、`backend/internal/service/search_cpo_automation.go`、`browser-extension/ozon-shop-bridge/background_search_cpo.js`。
2. Search CPO 私有接口响应结构对齐：
   - `search_promo_availability` 已按本地样本对齐 `skuToIsSearchPromoAvailable` + `skuToIsSearchPromoAvailabilityWithReason` 双 map 返回，不再把完整响应误判成“未匹配到响应”。
   - `product/enable` 已按 `bids[]` 逐 SKU 判定成功/失败，`carrots/batch_enable` 已按 `skuToInfo{}` 逐 SKU 判定成功/失败，不再只凭 HTTP 200 直接写全成功。
   - 为规避 Windows 环境下大文件直接补丁不稳定，本次新增 `background_search_cpo_bootstrap.js` + `background_search_cpo_response_patch.js` 覆盖现有 Search CPO 解析层，保留原主脚本和原有页签/调度逻辑不变。
3. Search CPO 缺响应静默兜底修复：
   - extension 侧 `normalizeSearchCPOAvailabilityItems` / `normalizeSearchCPOStepItems` 已改为“未命中 Seller 响应即显式失败”，不再默认写 `availability_promo=false` 或静默按 `success`。
   - 后端 `ApplyAvailabilityUpdates` 只在字段真实返回时回写 `search_promo_status/carrots_status/availability_promo`，避免把历史正确状态覆盖为空。
4. Search CPO 状态迁移链路回写修复：
   - 自动化 `enable` 成功后会把本地 `search_promo_status` 推进为 `SEARCH_PROMO_STATUS_ENABLED`，`Morkovsk` 成功后同步推进 `carrots_status` 与 `morkovsk_joined_at`，并把 `state2` 正确推进到 `state3_trigger/joined`。
   - 新增 `buildSearchCPOSKUMeta*` 单测，覆盖 `source_sku != sku` 场景。
5. Search CPO availability 运行时诊断补强：
   - 新增 `browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch.js`，在不改现有主链路的前提下补齐 `search_promo_availability` 的 envelope 宽松解析、`requested_sku/parser_revision/response_root_keys/sample_response_keys` 诊断字段，以及 availability / reason map key 计数。
   - `background_search_cpo_bootstrap.js` 现会在原 `background_search_cpo_response_patch.js` 之后继续加载 runtime diagnostics patch；extension register 会额外上报 `build_revision`，availability report artifact 会写入 `build_revision/parser_revision` 和逐 SKU 诊断 payload。
   - 本批继续新增 `background_search_cpo_fetch_diagnostics_patch.js`、`background_search_cpo_runtime_diagnostics_patch_v2.js` 与 `background_search_cpo_runtime_diagnostics_patch_v3.js`：`search_promo_availability` 不再在 `response.json()` 失败后静默丢失现场，而会把 `response_kind/http_status/content_type/parse_error/response_excerpt` 一起回传；扩展 build/parser revision 已提升到 `2026-03-20-d`。
6. Search CPO automation 详情取证面板：
   - 后端 `GetAutomationRunDetail` 会从 `search_cpo_products.availability_payload` 解析 availability 诊断，并随自动化详情接口返回 `availability_checked_at`、`requested_sku`、`build_revision`、`response_content_type`、`response_parse_error`、`response_excerpt` 等字段。
   - 前端 `SearchCPO.vue` 的“状态迁移详情”新增展开式诊断面板，点开单个商品即可直接看到 availability 的请求 SKU、HTTP/Content-Type、root/sample keys、解析错误和响应摘要，不用再手查数据库 artifact。
6. Search CPO 后端失败透传补强：
   - `backend/internal/dto/extension.go`、`backend/internal/service/automation_service.go` 已接收并记录 extension `build_revision`；extension report 事件会附带 `build_revision/parser_revision`。
   - `backend/internal/service/automation_failure.go` 已在 Search CPO 三类 job 失败时优先保留逐 SKU 细节，并在需要时补齐 `source_sku=` 前缀，避免运行历史和页面错误只剩泛化文案。
7. 本批验证结果：
   - runtime diagnostics 补丁脚本语法检查通过：`node --check browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch.js`。
   - Search CPO parser 样本 + envelope 回归通过：`node browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-response-parsers.mjs`。
   - Search CPO service worker 加载顺序回归通过：`node browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-service-worker-order.mjs`。
   - 后端定向测试通过：`cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./internal/service`。

7. Search CPO 迁移退出判定与 state3 修正：
   - `backend/internal/service/search_cpo_automation.go` 已放宽 `state3_trigger`：当商品 live 状态满足 `SEARCH_PROMO_STATUS_ENABLED + CARROTS_STATUS_DISABLED + availability=true` 时，即使没有历史 `state2_detected_at` 也会进入迁移后半段，并在首次命中时回填 `state2_detected_at`。
   - 店铺活动退出现在会把“商品当前不在活动中”与 `404/NotFound` 归类为可跳过结果，不再阻断后续 `enable` / `carrots/batch_enable`；逐动作明细会保留 `action_not_found/source_action_id` 诊断。
   - 官方活动退出已按 `doc\按订单付费推广商品操作\官方deactivate.txt` 对齐 `product_ids + rejected[]` 响应，逐 SKU 错误会直接透出具体 `reason`，不再只剩“部分活动退出失败”。
7.1 Search CPO 迁移前置活动 404 自动禁用与详情补强：
   - `backend/internal/service/search_cpo_automation.go` 迁移前置阶段改为仅处理 `status=active` 的活动；刷新活动商品时若命中 `404/NotFound`，会自动将对应活动标记为 `disabled` 并跳过，不再把整批 state2/state3 迁移直接打断。
   - 前置阶段若仍出现不可恢复错误，会把失败活动写入 `exit_results`，避免详情页只剩“退出其它活动失败”与备注 404，而没有具体活动行。
   - `frontend/src/views/promotions/SearchCPO.vue` 的手动/自动化详情现会在店铺活动标题后附带 `source_action_id=...`，便于直接定位失效的本地活动缓存记录。
7.2 Search CPO Enable / Morkovsk 逐 SKU 诊断与状态3收敛：
   - `browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch_v3.js` 现已把 `product/enable` 与 `carrots/batch_enable` 的 live 响应纳入与 availability 同级的包裹层扫描和运行时诊断，支持 `data/result/response` 包裹层，并在缺匹配时输出 `source_sku/requested_sku/parser_revision/HTTP/root/sample keys`。
   - `backend/internal/service/search_cpo_automation.go` 现在会优先按 `search_cpo_enable_snapshot/search_cpo_morkovsk_snapshot` 的 artifact 逐 SKU 结算步骤结果；当 job 整体失败或快照缺失时，回退到按当前 SKU 装饰错误，不再把首条 `source_sku=...` 错误复制给所有商品。
   - 状态迁移后半段里，若商品当前已是 `SEARCH_PROMO_STATUS_ENABLED`，则直接把 `Enable` 记为“跳过”，并继续进入 Morkovsk，不再因重复 enable 把状态3商品误判成失败。
   - `frontend/src/views/promotions/SearchCPO.vue` 的自动化详情已补展示 `Enable/Morkovsk` 的 `requested_sku`、`parser_revision`、HTTP、root/sample keys 和解析错误。
7.3 Search CPO 店铺活动退出按活动逐条复核：
   - `backend/internal/service/search_cpo_automation.go` 的 Search CPO 迁移后半段不再复用 `promo_unified_remove` 多活动聚合结果；店铺活动退出改为按活动逐条创建 `shop_action_remove` 任务，逐条等待并逐条落 `exit_results`，避免一个 SKU 的聚合结果被复制到所有活动行。
   - Search CPO 店铺退出请求现优先使用数值 `sku`：后端会按 `source_sku -> product.SKU` 计算目标请求 SKU，并把该值作为单活动 remove job 的执行 SKU；退出完成后会立即刷新对应活动商品并按原始 `source_sku` 复核，若商品仍留在该活动中，则把结果改判为失败并落明细 `退出后复核仍在活动中`。
   - 已经落入 `morkovsk_joined` 的商品现会被纳入自动化补救路径：若本次全量同步后仍命中官方/店铺非目标活动，则只执行“退出其它活动”清理，不再重复 `enable` 或重复加入 Morkovsk；若未命中其它活动，则本次运行仅记为跳过。
   - 由于 `dev-tracker` 文件 owner 为 `BUILTIN\\Administrators`，本次文档同步改用受控 PowerShell 文本替换完成；无新增数据库结构变更，无新增 migration 脚本。
7.4 Search CPO 历史 joined 标记回收与 live 状态优先：
   - `backend/internal/service/search_cpo_automation.go` 新增 live 状态判定：当商品当前明确满足 `state1/state2/state3/state4` 之一时，状态派生优先使用最新 `search_promo_status + carrots_status + availability`，不再让历史 `morkovsk_joined_at` 无条件压过当前 live 状态。
   - 当本地存在 `morkovsk_joined_at`、但当前 live 已明确回退到状态1/2/3 时，本次自动化运行会同步清空该商品的 `morkovsk_joined_at`，并把它重新送回正常迁移链路；其中状态2会重新执行 `退出其它活动 -> enable -> Morkovsk`，状态3会重新执行 `退出其它活动 -> 跳过重复 enable -> Morkovsk`。
   - `joined repair` 现仅对 `RuleStateAfter = morkovsk_joined` 的商品生效；availability 数据不完整、关键字段缺失或 live 状态无法确认时，不主动清空 joined 标记，避免因为 Seller challenge 或回包漂移误伤真实已迁移商品。
8. 数据库与迁移：
   - 本次无新增表结构变更，无新增 migration 脚本。
9. 本批验证结果：
   - 扩展 parser fixture 通过：`node browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-response-parsers.mjs`。
   - 扩展语法检查通过：`node --check browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch_v3.js`。
   - 后端定向测试通过：`cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./pkg/ozon ./internal/service`。
   - 前端构建通过：`cd frontend && cmd /c npm run build`。

5. 数据库与迁移：
   - 本次无新增表结构变更，无新增 migration 脚本。
6. 本批验证结果：
   - 插件主脚本语法检查通过：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`。
   - Search CPO 补丁脚本语法检查通过：`node --check browser-extension/ozon-shop-bridge/background_search_cpo_response_patch.js`。
   - bootstrap 脚本语法检查通过：`node --check browser-extension/ozon-shop-bridge/background_search_cpo_bootstrap.js`。
   - 响应样本回归通过：`node browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-response-parsers.mjs`。
   - `manifest.json` 解析校验通过：`Get-Content browser-extension/ozon-shop-bridge/manifest.json -Raw | ConvertFrom-Json | Out-Null`。
   - 后端回归测试通过：`cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./...`。
   - 前端构建通过：`cd frontend && cmd /c npm run build`。
   - 插件脚本语法检查通过：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`。
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
25. `/v3/product/list` 实测响应字段对齐与目录可见性修复：
   - 根因：Seller `/v3/product/list` 请求仍接受 `filter.visibility`，但响应 `items` 不再稳定返回 `visibility`，而是返回 `has_fbo_stocks/has_fbs_stocks/archived/is_discounted/quants`。
   - 处理：`backend/pkg/ozon/catalog.go` 扩展 `ProductListV3Item` 字段映射并新增 `ProductListV3Quant`，保留 `visibility` 兼容字段。
   - 处理：`backend/internal/service/ozon_catalog_service.go` 可见性推导改为优先读取 `info.visible`，缺失时回退 `archived`，最终兜底 `ALL`。
   - 测试：新增/更新 `backend/pkg/ozon/catalog_test.go`、`backend/internal/service/ozon_catalog_service_test.go`，覆盖新版响应字段与可见性推导优先级。
26. `.gitignore` 规则收敛（移除过宽忽略 + 去重维护）：
   - 根因：根目录 `*.ps1` 会误伤仓库内脚本资产（如插件打包脚本），且前后端子目录存在与根目录重复的 IDE/系统/环境变量规则。
   - 处理：移除根目录 `*.ps1` 全局忽略，保留临时目录与缓存忽略。
   - 处理：精简 `backend/.gitignore` 与 `frontend/.gitignore`，仅保留子项目特有规则，通用规则统一由根目录维护。
   - 涉及：`.gitignore`、`backend/.gitignore`、`frontend/.gitignore`。
27. Ozon 商品目录刷新 `primary_image` 反序列化失败修复（`string` / `array` / `object` 兼容）：
   - 根因：Seller `/v3/product/info/list` 在部分商品返回 `primary_image: []`（数组），现有 `ProductInfoListItem.PrimaryImage` 固定为 `string`，导致 `json.Unmarshal` 报错并中断整批刷新。
   - 处理：`backend/pkg/ozon/catalog.go` 为 `primary_image` 增加柔性解析（兼容 string、[]string、对象结构），解析失败时仅该字段置空，不中断整批数据解码。
   - 处理：补充 `statuses.status -> status.state` 回填兼容，避免状态字段结构变体导致目录状态退化。
   - 处理：补齐单测覆盖（数组主图、字符串主图、对象主图、异常主图形态、状态回填），并在 `mergeCatalogInfo` 增加“主图优先/图片回退”测试。
   - 涉及：`backend/pkg/ozon/catalog.go`、`backend/pkg/ozon/catalog_test.go`、`backend/internal/service/ozon_catalog_service_test.go`。
28. Ozon 商品目录刷新 `404 page not found` 修复（库存端点版本切换）：
   - 根因：目录刷新库存拉取仍调用已下线的 `/v3/product/info/stocks`，线上返回 `404 page not found` 并中断刷新流程。
   - 处理：`backend/pkg/ozon/catalog.go` 将库存查询主路径切换为 `/v4/product/info/stocks`。
   - 处理：新增兼容回退：若 `v4` 返回 404，自动回退请求 `/v3/product/info/stocks`（兼容历史环境）。
   - 处理：补齐单测：更新库存请求主路径断言为 `v4`，并新增 `v4 -> v3` 回退测试。
   - 涉及：`backend/pkg/ozon/catalog.go`、`backend/pkg/ozon/catalog_test.go`。
29. 官方促销接口标准文档沉淀（`/v1/actions/candidates` + `/v1/actions/products/activate`）：
   - 处理：新增 `doc/ozon-promos-candidates-activate-standard.md`，按工程说明模板沉淀两个接口的用途、鉴权、请求/响应结构、示例、分页与弃用说明。
   - 处理：新增 `doc/ozon-promos-candidates-activate.openapi.yaml`，以轻量 OpenAPI 3.0 子集形式输出机读结构，便于后续程序消费或继续裁剪生成代码。
   - 处理：明确 `offset` 已弃用、`last_id` 为推荐分页游标，并补齐 `result.rejected[].product_id / reason` 结构说明。
   - 处理：更新根目录 `.gitignore`，仅对白名单放行这两份新文档，避免 `doc/` 整体忽略导致交付文件无法入库。
   - 涉及：`doc/ozon-promos-candidates-activate-standard.md`、`doc/ozon-promos-candidates-activate.openapi.yaml`、`.gitignore`。
30. 自动加促销功能落地（配置 + 调度 + 候选同步 + 历史）：
   - 处理：新增 `promotion_action_candidates`、`auto_promotion_configs`、`auto_promotion_runs`、`auto_promotion_run_items` 四张表；新增 `upgrade_20260311_auto_promotion_add.sql`，并同步回写 `init_database.sql`。
   - 处理：新增 `AutoPromotionService` 与 `AutoPromotionHandler`，提供配置读写、手动触发、定时调度、执行历史与详情接口；执行前强制刷新 Ozon 目录并按 `listing_date` 过滤目标日期商品。
   - 处理：官方促销候选刷新链路落地到 `/v1/actions/candidates` `last_id` 分页，并对 `/v1/actions/products/activate` 的 `result.rejected[]` 做逐商品失败处理。
   - 处理：插件新增 `sync_action_candidates` 任务，复用 Seller 候选商品接口写入候选快照；店铺活动申报改为按单活动顺序创建 job 并等待执行结果。
   - 处理：前端新增 `/promotions/auto-add` 页面与菜单，支持保存“绝对日期 + 时间 + 官方/店铺活动”配置、手动执行、轮询历史和逐商品结果详情。
   - 涉及：`backend/internal/service/auto_promotion_service.go`、`backend/internal/handler/auto_promotion_handler.go`、`backend/internal/model/auto_promotion.go`、`backend/internal/repository/auto_promotion_repo.go`、`backend/pkg/ozon/actions.go`、`browser-extension/ozon-shop-bridge/background.js`、`frontend/src/views/promotions/AutoAdd.vue`。
31. 自动加促销官方候选刷新 `last_id` 类型漂移热修：
   - 根因：`/v1/actions/candidates` 在真实环境中对请求体 `last_id` 按 string field 校验；旧实现把纯数字游标强转成 number，导致手动执行自动加促销时第二页开始返回 `proto: ... invalid value for string field last_id`。
   - 处理：`ozon` 客户端候选接口请求改为始终以字符串发送 `last_id`，响应侧继续兼容 `string/number` 两种游标形态并统一归一为字符串。
   - 处理：补充回归测试，覆盖“请求固定 string + 响应 numeric cursor 可继续复用”为下一页字符串请求的场景。
   - 处理：同步修正文档 `doc/ozon-promos-candidates-activate-standard.md` 与 `doc/ozon-promos-candidates-activate.openapi.yaml`，避免本地说明继续误导为 `number<double>` 请求字段。
   - 涉及：`backend/pkg/ozon/actions.go`、`backend/pkg/ozon/actions_test.go`、`doc/ozon-promos-candidates-activate-standard.md`、`doc/ozon-promos-candidates-activate.openapi.yaml`。
32. 店铺活动候选同步失败原因透传与任务说明纠偏：
   - 根因：`sync_action_candidates` 由浏览器扩展执行时，如果扩展仍是旧代码，会直接返回 `插件不支持该任务类型: sync_action_candidates`；后端此前只把 run 级错误收敛成 `shop action candidates sync failed`，排障信息被吞掉。
   - 处理：`automation_repo.UpdateJobAndItemsByReport` 在 job 失败/部分成功时，自动把首个 item 失败原因同步写入 `automation_jobs.error_message`。
   - 处理：促销同步与自动加促销链路统一改为优先透传 job 明细错误，避免继续只显示 `shop actions/products/candidates sync failed`。
   - 处理：补充纯函数测试覆盖错误消息提取逻辑，并更新 `AGENTS.md` 的插件支持任务列表，补上 `sync_action_candidates`，避免文档继续落后于代码。
   - 涉及：`backend/internal/repository/automation_repo.go`、`backend/internal/service/promotion_service.go`、`backend/internal/service/auto_promotion_service.go`、`backend/internal/service/automation_failure.go`、`backend/internal/repository/automation_repo_error_message_test.go`、`backend/internal/service/automation_failure_test.go`、`AGENTS.md`。

33. Search CPO 批量报名能力落地（配置 + 商品同步 + 执行历史）：
   - 处理：后端新增 `search_cpo_configs`、`search_cpo_products`、`search_cpo_runs`、`search_cpo_run_items` 四张表模型、仓储、服务与 handler，新增 `search-cpo` 配置/商品/运行历史接口。
   - 处理：前端新增 `/promotions/search-cpo` 页面与菜单入口，支持默认活动配置、刷新 CPO 商品、本地筛选、按“当前筛选结果”执行报名、运行历史与详情弹窗。
   - 处理：插件主入口切换为 `manifest.json -> background_search_cpo.js`，支持 `sync_search_cpo_products` 任务并抓取 CPO 商品。
   - 涉及：`backend/internal/service/search_cpo_service.go`、`backend/internal/handler/search_cpo_handler.go`、`frontend/src/views/promotions/SearchCPO.vue`、`browser-extension/ozon-shop-bridge/background_search_cpo.js`。
34. Search CPO 已关闭商品报名可靠性补强：
   - 根因：页面将 `SEARCH_PROMO_STATUS_DISABLED` 展示为“未开启”，与运营口径不一致；官方活动报名仅依赖本地 `products` 表匹配 `source_sku -> ozon_product_id`，命不中时直接失败；同一商品官方失败后店铺活动会被自动跳过。
   - 处理：`SearchCPOService` 新增 Ozon 目录缓存兜底，按 `offer_id/sku` 回查 `ozon_product_id`，并按“本地价格 -> CPO 缓存价 -> 目录缓存价”顺序选择官方活动价。
   - 处理：店铺活动执行改为与官方结果解耦；同一商品可出现“官方失败 / 店铺成功”，逐商品总状态新增 `partial_success`。
   - 处理：前端页面将 `SEARCH_PROMO_STATUS_DISABLED` 统一展示为“已关闭”，并在执行说明中明确推荐先筛选已关闭商品再报名。
   - 测试：新增 `backend/internal/service/search_cpo_service_test.go`，覆盖目录缓存匹配、价格解析与状态汇总。
   - 涉及：`backend/internal/service/search_cpo_service.go`、`backend/internal/repository/ozon_catalog_repo.go`、`backend/internal/service/search_cpo_service_test.go`、`frontend/src/views/promotions/SearchCPO.vue`。
35. Search CPO 刷新页签复用修复：
   - 根因：插件 `sync_search_cpo_products` 复用了通用 `worker tab` 入口；当没有现成工作页时，会自动新开 Seller 页签，即使用户已经在别的窗口打开了 CPO 页面。
   - 处理：`background_search_cpo.js` 将 `sync_search_cpo_products` 从共享 `ensureSellerLoggedInTab()` 流程拆出，改为只扫描并复用当前已打开的 Seller CPO 页签。
   - 处理：刷新前先在候选页签里实时探测 `x-o3-company-id/sc_company_id` 与 `x-adv-current-organisation`；缺少现成页签或组织上下文时直接返回明确提示，不再默认新开 Ozon 页面。
   - 影响：本次改动只作用于 `sync_search_cpo_products`，不改变其它店铺活动同步/执行任务的静默工作页策略。
   - 涉及：`browser-extension/ozon-shop-bridge/background_search_cpo.js`。
36. Search CPO 页面上下文提取范围扩大：
   - 根因：真实 CPO 页面请求头里已有 `x-adv-current-organisation / x-o3-app-name / x-o3-company-id / x-o3-language`，但插件此前仅从 storage/cookie 取值，导致页面能正常发请求、插件却报“未提供组织上下文”。
   - 处理：`scriptReadSearchCPOContext` 改为同时扫描 storage/cookie、页面全局状态、请求配置对象以及常见框架根对象（如 `__INITIAL_STATE__`、`__NEXT_DATA__`、`__NUXT__`、`__PINIA__` 等），并把成功解析到的上下文缓存到当前页面。
   - 处理：`scriptFetchSearchCPOProducts` 优先复用页面探测阶段缓存下来的上下文；若缓存不存在，再回退到 cookie/storage，减少“页面请求能发、插件取不到头”的误判。
   - 涉及：`browser-extension/ozon-shop-bridge/background_search_cpo.js`。
37. Search CPO 真实请求头捕获兜底：
   - 根因：页面真实请求头里已有 `x-adv-current-organisation`，但仅靠页面对象扫描仍不一定能稳定取到，导致插件仍可能误判“未提供组织上下文”。
   - 处理：扩展新增 `webRequest` 权限，并修正 `manifest.json` 中对应权限声明格式，确保 Chrome 能正常加载请求头监听；随后监听 `https://seller.ozon.ru/performance-api/seller-api/search-performance-cpo/*` 的真实出站请求头。
   - 处理：一旦捕获到 `x-adv-current-organisation / x-o3-company-id / x-o3-language / x-o3-app-name`，就按 `tabId` 缓存在扩展存储中；`sync_search_cpo_products` 扫描到对应 CPO 页签时会优先把这份真实上下文注入页面后再执行刷新。
   - 处理：若刚重载插件导致还没捕获到任何真实请求头，错误文案会明确提示先手动刷新一次该 CPO 页面。
   - 涉及：`browser-extension/ozon-shop-bridge/manifest.json`、`browser-extension/ozon-shop-bridge/background_search_cpo.js`。
38. 插件登录态自动连接主流程收敛：
   - 根因：popup 默认暴露 `token/shop_id` 手填，普通用户容易被引导去 F12 复制 token；同时插件缺少“管理端是否已登录 / 已选店铺 / 已授权”的显式状态。
   - 处理：`background_search_cpo.js` 新增 `OZON_MANAGER_CHECK_AUTH_SYNC`，主动扫描管理端页签并读取 `localStorage.token/currentShopId`，返回 `connected / missing_login / missing_shop / permission_required / sync_unavailable / failed` 结构化状态。
   - 处理：popup 重构为“自动连接为主”，新增“重新检测”；默认将 `Token / 店铺 ID / 管理端 Origin` 收进“高级设置 / 排障”，并首屏展示连接来源、管理端 Origin、最近检测和立即同步结果。
   - 处理：`README.md` 同步改写为普通用户说明：先在管理端登录并选择店铺，非 localhost 首次授权即可，只有自动连接失败时才使用高级设置兜底。
   - 涉及：`browser-extension/ozon-shop-bridge/background_search_cpo.js`、`browser-extension/ozon-shop-bridge/popup.html`、`browser-extension/ozon-shop-bridge/popup.js`、`browser-extension/ozon-shop-bridge/README.md`。
39. 自动加促销恢复相对日期规则（昨天 / 今天 / 自定义）：
   - 根因：`auto_promotion_configs.target_date` 被当成固定绝对日期保存，定时任务每天重复跑同一天，不再具备“昨天上架商品自动加促销”的原始语义。
   - 处理：后端为 `auto_promotion_configs` / `auto_promotion_runs` 新增 `target_date_mode`，配置表的 `target_date` 改为仅在 `custom` 模式下保存；手动执行与定时调度都会在运行时解析出本次实际执行日期，并把“规则 + 实际日期”一起写入 run 与快照。
   - 处理：前端 `/promotions/auto-add` 页面改为显式提供“昨天 / 今天 / 自定义日期”三种规则；历史列表与详情弹窗新增“日期规则”，原 `target_date` 明确展示为“实际日期”。
   - 处理：新增 `upgrade_20260322_auto_promotion_relative_date_mode.sql`，旧配置与历史统一回填为 `custom`，避免静默改变已在线店铺的自动任务行为；`init_database.sql` 已同步回写最新结构。
   - 测试：新增日期规则解析与配置校验单测，覆盖 `昨天/今天/自定义`、空模式兼容和非法输入分支。
   - 涉及：`backend/internal/service/auto_promotion_service.go`、`backend/internal/service/auto_promotion_service_test.go`、`backend/internal/model/auto_promotion.go`、`backend/internal/dto/auto_promotion.go`、`backend/internal/repository/auto_promotion_repo.go`、`backend/migrations/init_database.sql`、`backend/migrations/upgrade_20260322_auto_promotion_relative_date_mode.sql`、`frontend/src/views/promotions/AutoAdd.vue`。

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
27. 后端定向测试通过（含本次 `/v3/product/list` 字段对齐与可见性推导）：`cd backend && go test ./pkg/ozon ./internal/service`。
28. `.gitignore` 规则验证通过：`package.ps1` 不再被误忽略；`frontend/.env`、`frontend/.idea/*` 仍由根目录规则生效；`backend/tmp/*` 仍由后端子目录规则生效。
29. 后端定向测试通过（含 `primary_image` 数组兼容与状态回填）：`cd backend && go test ./pkg/ozon ./internal/service`。
30. 后端全量回归测试通过：`cd backend && go test ./...`。
31. 后端定向测试通过（含库存端点 `v4` 主路径与 `v3` 回退）：`cd backend && go test ./pkg/ozon ./internal/service`。
32. 后端全量回归测试通过（含库存端点版本修复）：`cd backend && go test ./...`。
33. 文档交付核对完成：`doc/ozon-promos-candidates-activate-standard.md` 与 `doc/ozon-promos-candidates-activate.openapi.yaml` 已落地；接口字段、示例与分页说明已按官方页面和补充核验结果对齐。
34. 后端回归测试通过（含自动加促销）：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./...`。
35. 插件脚本语法检查通过（含 `sync_action_candidates` 任务）：`node --check browser-extension/ozon-shop-bridge/background.js`。
36. 前端构建通过（含 `/promotions/auto-add` 页面）：`cd frontend && cmd /c npm run build`。
37. 后端定向测试通过（含官方候选 `last_id` 字符串修复）：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./pkg/ozon ./internal/service`。
38. 后端定向测试通过（含 extension 失败原因透传）：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./internal/repository ./internal/service`。

39. 后端回归测试通过（含 Search CPO 批量报名链路）：`cd backend && go test ./...`。
40. 前端构建通过（含 `/promotions/search-cpo` 页面与 UI 强化）：`cd frontend && cmd /c npm run build`。
41. 插件脚本语法检查通过（含 CPO 商品同步实现）：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`。
42. 后端回归测试通过（含 Search CPO 官方目录兜底与 partial_success 汇总）：`cd backend && $env:GOCACHE="$env:TEMP\\ozon-manager-gocache"; go test ./...`。
43. 前端构建通过（含 Search CPO “已关闭”语义调整）：`cd frontend && cmd /c npm run build`。
44. 插件脚本语法检查通过（含 Search CPO 复用已打开 CPO 页签、不再默认新开 Seller 页）：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`。
45. 插件脚本语法检查通过（含 Search CPO 页面上下文提取扩大）：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`。
46. 插件脚本语法检查通过（含 Search CPO 真实请求头捕获兜底）：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`。
47. 插件清单解析校验通过（含 `webRequest` 权限声明修正）：`Get-Content browser-extension/ozon-shop-bridge/manifest.json -Raw | ConvertFrom-Json | Out-Null`。
48. 插件脚本语法检查通过（含管理端自动连接状态收敛）：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`、`node --check browser-extension/ozon-shop-bridge/popup.js`、`node --check browser-extension/ozon-shop-bridge/content-auth-sync.js`。
49. 插件脚本语法检查通过（含 Search CPO 自动化 job 分发修复）：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`。
50. 后端回归测试通过（含 Search CPO 自动化链路回归）：`cd backend && $env:GOCACHE="$env:TEMP\\ozon-manager-gocache"; go test ./...`。
51. 前端构建通过（含 Search CPO 页面现有自动化入口回归）：`cd frontend && cmd /c npm run build`。
52. 后端回归测试通过（含自动加促销相对日期规则）：`cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./...`。
53. 前端构建通过（含 `/promotions/auto-add` 日期规则切换与历史展示调整）：`cd frontend && cmd /c npm run build`.

## 数据库执行记录
0. 本次新增可执行升级脚本：`backend/migrations/upgrade_20260319_search_cpo_morkovsk_automation.sql`（Search CPO Morkovsk 自动化第一批）。
1. 用途：扩展 `search_cpo_configs` 自动化配置字段、扩展 `search_cpo_products` 规则状态字段，并新增 `search_cpo_auto_runs`、`search_cpo_auto_run_items` 两张自动化运行表。
2. 执行条件：目标库已存在 `search_cpo_configs`、`search_cpo_products`、`promotion_actions`、`shops` 等基础表；脚本支持幂等重复执行。
3. 执行结果：脚本已编写，`init_database.sql` 已同步回写；本轮未在当前会话直接执行数据库升级。
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
16. 本次（`/v3/product/list` 响应字段对齐与目录可见性推导修复）无新增迁移脚本：仅涉及 API 响应映射与服务层推导逻辑调整，不涉及数据库结构变更。
17. 本次（`.gitignore` 收敛优化）无新增迁移脚本：仅调整版本控制忽略规则，不涉及数据库结构变更。
18. 本次（`/v3/product/info/list` `primary_image` 多形态兼容修复）无新增迁移脚本：仅涉及 Ozon 响应解析与单元测试增强，不涉及数据库结构变更。
19. 本次（`/v3/product/info/stocks` 404 修复）无新增迁移脚本：仅涉及库存接口版本切换与兼容回退逻辑，不涉及数据库结构变更。
20. 本次（官方促销接口文档沉淀）无新增迁移脚本：仅新增 `doc/` 标准接口文档与 `dev-tracker` 追踪记录，不涉及数据库结构变更。
21. 本次新增可执行升级脚本：`backend/migrations/upgrade_20260311_auto_promotion_add.sql`（自动加促销配置、候选缓存与运行历史）。
22. 用途：新增活动候选商品缓存、自动加促销配置、运行记录和逐商品结果表，为“自动加促销”页面、定时调度与失败回溯提供持久化基础。
23. 执行条件：目标库已存在 `promotion_actions`、`products`、`shops`、`users` 等基础表；脚本支持幂等重复执行。
24. 执行结果：开发环境 SQL 已同步，`init_database.sql` 已回写至最新结构。
25. 本次（自动加促销官方候选 `last_id` 类型热修）无新增迁移脚本：仅调整官方候选接口请求参数类型、单元测试与本地接口文档，不涉及数据库结构变更。
26. 本次（店铺活动候选同步失败原因透传）无新增迁移脚本：仅调整自动化任务错误消息落库与服务层错误透传，不涉及数据库结构变更。

27. 本次新增可执行升级脚本：`backend/migrations/upgrade_20260313_search_cpo_bulk_enroll.sql`（Search CPO 配置、商品缓存与运行历史）。
28. 用途：新增 CPO 商品缓存、默认活动配置、运行记录和逐商品结果表，为 `/promotions/search-cpo` 页面提供持久化支撑。
29. 执行条件：目标库已存在 `shops`、`products`、`promotion_actions` 等基础表；脚本支持幂等重复执行。
30. 执行结果：开发环境 SQL 已同步，`init_database.sql` 已回写至最新结构。
31. 本次（Search CPO 已关闭商品报名可靠性补强）无新增迁移脚本：仅调整 Search CPO 服务层匹配逻辑、运行状态汇总与前端展示文案，不涉及数据库结构变更。
32. 本次（Search CPO 刷新页签复用修复）无新增迁移脚本：仅调整插件页签选择与页面上下文探测逻辑，不涉及数据库结构变更。
33. 本次（Search CPO 页面上下文提取范围扩大）无新增迁移脚本：仅调整插件在 Seller CPO 页面中的上下文发现策略，不涉及数据库结构变更。
34. 本次（Search CPO 真实请求头捕获兜底）无新增迁移脚本：仅调整扩展权限与请求头缓存策略，不涉及数据库结构变更。
35. 本次（插件登录态自动连接主流程收敛）无新增迁移脚本：仅调整插件 popup / background 状态契约与说明文案，不涉及数据库结构变更。
36. 本次（Search CPO 刷新去掉 `notAvailableOnly`）无新增迁移脚本：仅调整插件抓取请求体、前端文案与交付文档，不涉及数据库结构变更。
37. 本次（Search CPO 自动化 extension 分发修复）无新增迁移脚本：仅调整插件任务路由与交付文档，不涉及数据库结构变更。
38. 本次（Search CPO availability 运行时诊断补强）无新增迁移脚本：仅新增扩展 runtime diagnostics patch、后端错误透传和回归脚本，不涉及数据库结构变更。
39. 本次新增可执行升级脚本：`backend/migrations/upgrade_20260322_auto_promotion_relative_date_mode.sql`（自动加促销相对日期规则）。
40. 用途：为自动加促销配置和运行历史补充 `target_date_mode`，并允许配置表在“昨天/今天”模式下不保存固定 `target_date`。
41. 执行条件：目标库已存在 `auto_promotion_configs` 与 `auto_promotion_runs`；脚本支持幂等重复执行。
42. 执行结果：脚本已编写，`init_database.sql` 已同步回写；本轮未在当前会话直接执行数据库升级。
43. 本次（开发启动脚本按包运行修复）无新增迁移脚本：仅调整 `start-dev.bat`、协作文档中的 Go 命令示例与一条轻量回归脚本，不涉及数据库结构变更。
44. 本次（开发启动脚本端口守卫修复）无新增迁移脚本：仅调整 `start-dev.bat` 与一条轻量回归脚本，不涉及数据库结构变更。
45. 本次（自动加促销稳定性补强）无新增迁移脚本：仅调整自动加促销/目录刷新/自动化产物读取的服务层时序与 Ozon 只读请求重试，不涉及数据库结构变更。
46. 处理：`OzonCatalogService.RefreshShopCatalogSync` 遇到同店铺已有刷新进行中时，改为等待当前刷新完成并复用结果，不再直接报 `catalog refresh already running`。
47. 处理：`AutomationService.GetLatestArtifact` 增加短暂轮询等待，消除 extension/agent 先报 job 成功、快照稍后落库导致的 `record not found` 竞态；同时为 `/v3/product/list`、`/v1/actions/candidates`、`/v1/actions/products` 等只读 Ozon 请求补充瞬时超时重试。
48. 处理：自动加促销店铺活动候选同步等待窗口从 60 秒提升到 2 分钟，降低真实浏览器链路中的误报超时。
49. 后端完整回归测试通过：`cd backend && $env:GOCACHE="$env:TEMP\\ozon-manager-gocache"; go test ./...`。

## 遗留问题
1. Chrome 商店上架材料与隐私文案尚未完成。
2. 缺少真实环境下长时间混合在线回归报告。
3. 执行引擎路由监控指标尚未落地。

## 下一步（最多 3 项）
1. 完成真实环境混合在线回归：重点覆盖多店铺、extension/agent 并存、隔夜 access token 静默续期与插件自动连接稳定性。
2. 补执行引擎路由监控指标，并继续收集 Search CPO live 样本与自动化详情诊断，收敛 `enable` / `carrots/batch_enable` 的剩余现场差异。
3. 继续准备 Chrome 商店上架材料：权限说明、隐私政策、审核说明与安装引导文案。

