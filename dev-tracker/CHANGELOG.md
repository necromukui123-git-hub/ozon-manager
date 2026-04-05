# Ozon Manager 变更日志

## 2026-04-04（补充四）
### 主题
完成 Search CPO 单一自动化流程落地：后端补齐退出活动配置、状态1/2/3/4 与退出失败跳过规则，前端从双标签收口为单一自动化页面，并统一执行历史与详情展示。

### 关键变更
1. 后端配置与迁移：
   - `backend/internal/model/search_cpo.go`、`backend/internal/dto/search_cpo.go`、`backend/internal/repository/search_cpo_repo.go`、`backend/internal/service/search_cpo_service.go` 支持 `exit_official_action_ids/exit_shop_action_ids`。
   - `backend/migrations/upgrade_20260404_search_cpo_automation_single_flow.sql` 正式落地，并同步回写 `backend/migrations/init_database.sql`；旧 `rule_state` 会迁移为 `state3/state4`，自动化统计字段收口到 `total_state3/total_state4`。
2. 后端自动化执行：
   - `backend/internal/service/search_cpo_automation.go` 把状态口径收口为状态1/2/3/4，并兼容旧 `state3_trigger/morkovsk_joined`。
   - `processMigrationItems()` 不再按 SKU 扫全量命中活动，而是只对用户配置的退出活动执行退出；状态2/3/4 若退出失败，会显式把后续步骤记为 `skipped`，并在详情消息写入“退出促销活动失败，跳过后续动作”。
   - 自动化 run 快照与详情已携带默认活动、退出活动和触发时间。
3. 前端页面收口：
   - `frontend/src/views/promotions/SearchCPO.vue` 改为单一自动化工作面，不再使用 tabs。
   - `frontend/src/views/promotions/search-cpo/SearchCPOAutomationTab.vue` 现在同时承载自动执行设置、默认活动、退出活动、状态概览和统一执行历史。
   - `frontend/src/views/promotions/search-cpo/SearchCPOAutomationDetailDialog.vue` 已按新口径显示 `total_state3/total_state4`，并展示当次默认活动/退出活动快照。
   - 删除 `frontend/src/views/promotions/search-cpo/SearchCPOManualTab.vue` 与 `frontend/src/views/promotions/search-cpo/SearchCPORunDetailDialog.vue`，页面不再展示旧手动报名工作面与旧历史详情。

### 影响范围
1. `/promotions/search-cpo` 现在只保留一条自动化链路，手动触发一次与定时执行共用同一张历史表 `search_cpo_auto_runs`。
2. Search CPO 用户可见状态与详情文案全部改为状态1/2/3/4，不再暴露 `state3_trigger`、`morkovsk_joined` 和“退出其它活动”等旧口径。
3. 旧手动报名接口与表结构本轮未删除，但前端已不再消费。

### 验证
1. `cd backend && $env:GOCACHE="$env:TEMP\\ozon-manager-gocache"; go test ./internal/service -count=1` 通过。
2. `cd frontend && cmd /c npm run build` 通过。

## 2026-04-04（补充三）
### 主题
规划 Search CPO 的下一轮目标：取消“商品池与手动报名”工作面，收口为单一自动化流程，并在执行前补齐正式 spec 与 implementation plan。

### 关键变更
1. `docs/superpowers/specs/2026-04-04-search-cpo-automation-single-flow-design.md`：
   - 明确 Search CPO 将改为单一自动化页面，保留“默认活动 + 退出活动”两组固定配置。
   - 状态定义收口为状态1=推广已关闭、状态2=可加入推广、状态3=已加入推广未加入 Morkovsk、状态4=已加入推广且已加入 Morkovsk。
   - 退出逻辑改为只处理用户显式配置的退出活动；状态2/3/4一旦退出失败，该商品后续步骤显式跳过，并在详情记录“退出促销活动失败，跳过后续动作”。
2. `docs/superpowers/plans/2026-04-04-search-cpo-automation-single-flow.md`：
   - 已把后端配置扩展、状态常量/历史统计重命名、退出逻辑重写、前端单页化、验证与 `dev-tracker` 更新拆成六个可执行任务。
   - 当前仅完成设计与计划，尚未进入代码实现。

## 2026-04-04（补充二）
### 主题
修复 `start-dev.bat` 在前后端已运行时仍重复拉起实例的问题，避免 Vite 因 5173 端口占用直接报错退出。

### 关键变更
1. `start-dev.bat`：
   - 新增 8080 与 5173 监听检测；若端口已被监听，则输出 `already listening ... skipping start` 并跳过对应服务启动。
   - 保留原有“端口空闲时直接启动”的行为，只对重复启动场景加守卫，不改现有开发端口约定。
2. `scripts/check-start-dev-port-guards.ps1`：
   - 新增轻量回归脚本，检查 `start-dev.bat` 必须包含 8080/5173 端口守卫与跳过提示，避免后续再次回退到无条件拉起实例。

### 影响范围
1. 连续双击或重复执行 `start-dev.bat` 时，已在运行的 Vite / 后端实例不会再被重复拉起。
2. 用户在已有前端开发服务器占用 5173 的情况下，再次执行启动脚本时会得到明确提示，而不是仅看到 Vite `Port 5173 is already in use` 报错。
3. 本次不涉及业务逻辑、接口协议或数据库结构变更。

### 验证
1. `powershell -ExecutionPolicy Bypass -File .\scripts\check-start-dev-port-guards.ps1` 通过。
2. 使用单进程临时 `TcpListener` 占住 8080/5173 后执行 `cmd /c "echo.| start-dev.bat"`，输出已确认包含：
   - `Backend already listening on port 8080 ..., skipping start.`
   - `Frontend already listening on port 5173 ..., skipping start.`

## 2026-04-04（补充）
### 主题
修复 `start-dev.bat` 与协作文档中错误的 Go 单文件启动命令，避免 `cmd/server` 同包文件未参与编译导致开发环境无法启动。

### 关键变更
1. `start-dev.bat`：
   - 后端启动命令由 `go run cmd/server/main.go` 改为 `go run ./cmd/server`，`frontend_static.go` 中的 `isAllowedOrigin`、`detectFrontendWebRoot`、`configureFrontendStatic` 会随同包一起编译。
2. `AGENTS.md`、`CLAUDE.md`：
   - 后端运行与构建示例统一改为 `go run ./cmd/server`、`go build -o server ./cmd/server`，消除手工启动和代理协作时继续复用错误命令的风险。
3. `scripts/check-go-package-entry.ps1`：
   - 新增轻量回归脚本，检查 `start-dev.bat`、`AGENTS.md`、`CLAUDE.md` 不得再出现 `cmd/server/main.go` 这种单文件 Go 入口命令。

### 影响范围
1. Windows 下双击 `start-dev.bat` 时，后端可按包正常编译启动，不再因同目录辅助文件未被纳入构建而直接报错。
2. 后续手工开发、代理协作与命令复制场景会统一使用正确的 `./cmd/server` 包路径。
3. 本次不涉及业务逻辑、接口协议或数据库结构变更。

### 验证
1. `cd backend && go build cmd/server/main.go` 失败，稳定复现与用户一致的 `undefined: isAllowedOrigin/detectFrontendWebRoot/configureFrontendStatic` 报错。
2. `cd backend && go build ./cmd/server` 通过，确认根因是“按文件构建”而不是“代码缺失”。
3. `powershell -ExecutionPolicy Bypass -File .\scripts\check-go-package-entry.ps1` 已在修复前失败，修复后应通过。

## 2026-04-04
### 主题
收敛仓库协作文档，修正 AGENTS 中已经过时的插件打包、Windows 发布、认证会话与阶段状态说明，避免后续代理按旧约定执行。

### 关键变更
1. `AGENTS.md`：
   - 项目结构补充 `agent/`、`docs/`、`release/`，并明确 Chrome 插件当前实际入口是 `manifest.json -> background_search_cpo_bootstrap.js -> background_search_cpo*.js/patch` 加载链。
   - 构建命令补充 `cmd/migrate-passwords`、根目录 `package-browser-extension.ps1` 与 `build-windows-release.ps1`，同时把扩展目录内旧 `scripts/package.ps1` 标记为历史脚本，不再作为当前默认入口。
   - 新增 Windows 发布约定、认证与会话约束，写明 access token + refresh token、`user_refresh_tokens` 基线、插件仍依赖管理端 `localStorage.token/currentShopId` 自动同步，以及店铺 `execution_engine_mode` 的当前约束。
   - 阶段状态更新为当前实际进度：执行引擎模式、插件状态面板、非 localhost 按需授权、Windows 单机发布链路与 Web refresh token 均已落地；后续重点改为 Chrome 商店上架、真实环境回归、执行引擎监控和 Search CPO live 收敛。
2. `dev-tracker/CURRENT_PROGRESS.md`：
   - 更新“最后更新时间”“本次目标”“下一步”，并补记本次协作文档收敛，确保与 `OVERALL_TASKS.md` 中的当前待办保持一致。

### 影响范围
1. 后续协作默认会使用当前有效的插件打包入口与 Windows 发布链路，不再误用旧脚本。
2. 后续涉及鉴权、插件登录态、执行引擎路由或发布包调整时，优先参考更新后的 AGENTS 约束，减少按过期状态实施变更的风险。
3. 本次仅涉及文档，不包含业务代码、数据库结构或接口行为变更。

### 验证
1. 已对照 `browser-extension/ozon-shop-bridge/manifest.json`、`background_search_cpo_bootstrap.js`、`package-browser-extension.ps1`、`build-windows-release.ps1`、`README-windows-deploy.md`、`backend/cmd/server/main.go`、`backend/config/config.yaml.example` 与 `dev-tracker/OVERALL_TASKS.md` 逐项核对。
2. 本次无代码变更，未执行 Go 测试、前端构建或数据库脚本。

## 2026-03-31
### 主题
完成 Web 端 access token + refresh token 认证升级，实现 refresh token 轮换、会话撤销与前端静默续期，并保持 Chrome 插件对管理端 token 同步的兼容。

### 关键变更
1. `backend/internal/config/config.go`、`backend/config/config.yaml.example`、`backend/pkg/jwt/jwt.go`、`backend/pkg/jwt/jwt_test.go`：
   - JWT 配置新增 `access_expire_minutes`、`refresh_expire_hours`、`refresh_cookie_name`、`refresh_cookie_secure`。
   - access token 现带 `token_type=access` claim，`ParseToken` 只接受 access token。
2. `backend/internal/model/refresh_token.go`、`backend/internal/repository/refresh_token_repo.go`、`backend/migrations/init_database.sql`、`backend/migrations/upgrade_20260331_refresh_tokens.sql`：
   - 新增 `user_refresh_tokens` 持久化模型、仓储和迁移脚本；refresh token 仅保存哈希，支持按 token、family、user 维度撤销与轮换。
3. `backend/internal/service/auth_service.go`、`backend/internal/handler/auth_handler.go`、`backend/cmd/server/main.go`：
   - 登录成功后同时签发 access token 与 refresh token；refresh token 通过 `HttpOnly` Cookie 下发。
   - 新增公开 `POST /api/v1/auth/refresh`，refresh 成功会轮换 refresh token 并返回新的 `token_expires_at`。
   - `POST /api/v1/auth/logout` 改为公开路由，可在 access token 过期后仍撤销当前 refresh token 并清理 cookie。
4. `backend/internal/service/user_service.go`、`backend/internal/service/user_service_test.go`：
   - 改密、重置密码、禁用店铺管理员/员工与通用用户禁用场景下，统一撤销该用户全部 refresh token。
5. `frontend/src/stores/user.js`、`frontend/src/utils/request.js`、`frontend/src/api/auth.js`、`frontend/src/main.js`：
   - 前端新增 `tokenExpiresAt` 状态、应用启动静默 refresh、401 单飞刷新与 refresh 成功后的本地 token/user 同步。
   - refresh 失败时统一清理本地登录态并跳转 `/login`；Chrome 插件继续消费管理端 `localStorage.token/currentShopId`。
6. `backend/internal/repository/user_repo.go`、相关测试：
   - `UpdateLastLogin` 改为使用 `time.Now()`，避免 sqlite 测试环境下 `NOW()` 不兼容。
   - auth / refresh token / user service 测试库已改成按测试名隔离的 sqlite 内存 DSN，避免包内串库。

### 影响范围
1. Web 管理端现在默认采用短期 access token + 长期 refresh token 自动续期，隔夜重新打开浏览器时会优先尝试静默 refresh。
2. 登出、改密、重置密码、禁用账号后，旧 refresh token 会失效，原会话无法继续刷新。
3. Chrome 插件本轮未引入独立 refresh 流程，仍依赖管理端 `localStorage.token/currentShopId`；前端 refresh 成功后会继续同步该 token。
4. 旧数据库升级需要执行 `backend/migrations/upgrade_20260331_refresh_tokens.sql`；新环境直接执行最新 `backend/migrations/init_database.sql`。

### 验证
1. `cd backend && $env:GOCACHE="$PWD\.gocache-auth"; $env:GOMODCACHE="$PWD\.gomodcache"; $env:GOPATH="$PWD\.gopath"; go test ./internal/service ./internal/handler` 通过。
2. `cd backend && $env:GOCACHE="$PWD\.gocache-user"; $env:GOMODCACHE="$PWD\.gomodcache"; $env:GOPATH="$PWD\.gopath"; go test ./internal/service` 通过。
3. `cd backend && $env:GOCACHE="$PWD\.gocache-all"; $env:GOMODCACHE="$PWD\.gomodcache"; $env:GOPATH="$PWD\.gopath"; go test ./...` 通过。
4. `cd frontend && cmd /c npm run build` 通过。

## 2026-03-25
### 主题
修复 Windows 单机发布包在目标机上的三类部署阻塞：空库默认管理员登录异常、旧库缺少 `users.owner_id` 导致无法创建店铺管理员，以及店铺 API Key 脏值/错误提示不清导致促销活动同步失败。

### 关键变更
1. `backend/migrations/init_database.sql`：
   - 默认超级管理员 `super_admin` 的基线哈希已替换为真正匹配当前登录协议 `bcrypt(SHA256(admin123))` 的值，空库初始化后可直接登录。
2. `backend/cmd/reset-password/main.go`、`backend/cmd/migrate-passwords/main.go`：
   - 密码工具统一改回与当前登录逻辑一致的 `bcrypt(SHA256(明文密码))`，不再生成与前端登录链路不兼容的哈希。
3. `frontend/src/utils/request.js`：
   - 登录接口 `POST /api/v1/auth/login` 返回 `401` 时，前端不再清空本地状态并提示“登录已过期”，而是直接显示后端返回的登录失败消息。
4. `backend/migrations/init_database_test.go`：
   - 新增回归测试，直接校验基线 SQL 中 `super_admin` 的哈希必须满足 `bcrypt(SHA256(admin123))`。
5. `README-windows-deploy.md`：
   - 补充默认管理员账号说明，并明确 2026-03-25 之前的旧发布包初始化空库后，若出现默认账号无法登录，最简单的处理方式是重新用新基线初始化。
6. `backend/migrations/upgrade_20260325_users_owner_id.sql`、`build-windows-release.ps1`：
   - 为旧库补齐 `users.owner_id` 字段、外键和索引，并让发布包把所有 `upgrade_*.sql` 一并带上；目标机遇到 `SQLSTATE 42703` 时可直接执行增量脚本修复。
7. `backend/internal/service/shop_service.go`、`backend/pkg/ozon/client.go`、`backend/internal/handler/promotion_handler.go`、`frontend/src/views/shop-admin/MyShops.vue`：
   - 店铺保存与实际出站请求都会清理 `Api-Key` 前后空白，`sync-actions` 会把官方 `Invalid Api-Key` 映射成明确中文提示；前端编辑店铺时也避免把未返回的凭证字段误写成空值或 `undefined`。

### 影响范围
1. 新生成的 Windows 发布包在空库环境下可直接使用 `super_admin / admin123` 登录。
2. 已经用旧基线初始化过的数据库不会自动修复；需要重新初始化空库，或手工更新 `super_admin` 的 `password_hash`。
3. 已按版本规则新增 `upgrade_20260325_users_owner_id.sql`，旧库可在不重建数据的前提下补齐员工归属字段并恢复店铺管理员/员工管理链路。
4. 新生成的发布包与后端代码对店铺 `Api-Key` 的空白字符更宽容，且当 Ozon 明确返回 `Invalid Api-Key` 时，页面会直接提示用户回到“我的店铺”修正凭证。

### 验证
1. `cd backend && $env:GOCACHE="$PWD\.gocache-build"; go test ./migrations` 通过。
2. `cd backend && $env:GOCACHE="$PWD\.gocache-build"; go test ./...` 通过。
3. `cd frontend && cmd /c npm run build` 通过。
4. `powershell -ExecutionPolicy Bypass -File .\build-windows-release.ps1` 已重新产出包含升级脚本与 API Key 修复的 Windows 发布包。


## 2026-03-23
### 主题
新增 Windows 单机发布包链路，让另一台 Windows 电脑在不安装 Go/Node 的前提下，直接运行后端、前端与 Chrome 插件。

### 关键变更
1. `backend/cmd/server/main.go`、`backend/cmd/server/frontend_static.go`、`backend/cmd/server/frontend_static_test.go`：
   - 后端新增可选 `web/` 静态托管与 SPA fallback；命中前端路由时返回 `index.html`，`/api/*` 未命中时仍保持 JSON 404。
   - CORS 由固定单一插件 ID 放宽为允许任意 `chrome-extension://` 来源，解决目标机手工加载扩展后无法访问后端的问题。
2. `build-windows-release.ps1`、`package-browser-extension.ps1`、`start-release-server.bat`、`README-windows-deploy.md`：
   - 新增统一发布脚本，可一次性构建前端 `dist`、后端 `server.exe`、插件 zip，并组装 `release/ozon-manager-win-x64` 与同名 zip。
   - 发布包内同时包含 `server/start-ozon-manager.bat`、`server/database/init_database.sql`、部署说明和可直接加载的插件目录。
3. Chrome 插件发布形态调整：
   - 打包不再依赖旧的手工文件清单，而是按扩展运行目录递归复制真实运行文件，并排除 `dist/`、`scripts/`。
   - 若旧 zip 正被占用，打包脚本会自动回退到时间戳文件名，避免整包构建失败；发布总包内仍统一输出稳定文件名 `ozon-shop-bridge-v<version>.zip`。

### 影响范围
1. 目标机只需安装 PostgreSQL 和 Chrome，即可通过 `server/start-ozon-manager.bat` 启动系统并在 `http://127.0.0.1:8080` 打开管理端。
2. 空库初始化只需执行发布包中的 `server/database/init_database.sql`，不需要迁移旧数据库数据。
3. 本次无数据库结构变更，无新增 migration 脚本。

### 验证
1. `cd backend && $env:GOCACHE="$PWD\.gocache-build"; go test ./cmd/server` 通过。
2. `cd frontend && cmd /c npm run build` 通过。
3. `powershell -ExecutionPolicy Bypass -File .\build-windows-release.ps1` 已在非沙箱环境完整跑通，成功生成 `release/ozon-manager-win-x64` 与 `release/ozon-manager-win-x64.zip`。

## 2026-03-22
### 主题
恢复“自动加促销”的相对日期语义：支持昨天 / 今天 / 自定义日期，并让定时任务按规则动态计算本次执行日期。

### 关键变更
1. `backend/internal/service/auto_promotion_service.go`、`backend/internal/model/auto_promotion.go`、`backend/internal/dto/auto_promotion.go`、`backend/internal/repository/auto_promotion_repo.go`：
   - 自动加促销配置和运行历史新增 `target_date_mode`，枚举固定为 `yesterday / today / custom`。
   - 配置表中的 `target_date` 改为仅在 `custom` 模式下保存；手动执行和定时调度都会在运行时解析出本次实际日期并写入 run。
   - `GetConfig` 的默认语义改为“昨天”，不再靠预填一个固定绝对日期模拟默认值。
2. `frontend/src/views/promotions/AutoAdd.vue`：
   - 页面配置项由“目标日期”改成“上架时间规则”，支持昨天、今天、自定义日期三种模式。
   - 历史列表和详情弹窗新增“日期规则”，原 `target_date` 明确展示为“实际日期”。
   - 自定义日期模式下新增前端校验，避免空日期直接发请求。
3. `backend/migrations/init_database.sql`、`backend/migrations/upgrade_20260322_auto_promotion_relative_date_mode.sql`：
   - 新环境基线已同步为最新结构。
   - 旧环境可通过新增升级脚本把已有配置和历史回填为 `custom`，同时放宽配置表 `target_date` 的非空约束。
4. `backend/internal/service/auto_promotion_service_test.go`：
   - 新增日期规则解析与配置校验测试，覆盖昨天、今天、自定义日期、空模式兼容和非法输入。

### 影响范围
1. “自动加促销”重新具备“每天自动处理昨天上架商品”的核心语义。
2. 现有已保存配置不会被静默改成“昨天”；升级后默认按 `custom` 保留原行为，运营可自行切换回相对规则。
3. 本次包含数据库结构变更，旧库需要执行 `upgrade_20260322_auto_promotion_relative_date_mode.sql`。

### 验证
1. `cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./...` 通过。
2. `cd frontend && cmd /c npm run build` 通过。
## 2026-03-22
### 主题
收口搜索推广商品页面的信息架构，保留单一路由入口，但把人工批量报名和状态迁移自动化拆成同路由双标签，并同步统一用户可见文案。

### 关键变更
1. `frontend/src/views/promotions/SearchCPO.vue`：
   - 页面改成容器组件，保留共享数据加载、轮询、详情弹窗状态与 `?tab=manual|automation` 路由同步，不再继续承载全部业务块和诊断 UI。
   - 顶部概览改为缓存商品、当前筛选、默认活动、最近同步、自动化状态五个指标卡，真正展示 `last_synced`。
2. `frontend/src/views/promotions/search-cpo/SearchCPOManualTab.vue`、`SearchCPOAutomationTab.vue`、`SearchCPORunDetailDialog.vue`、`SearchCPOAutomationDetailDialog.vue`、`ui.js`：
   - 手动标签集中承载商品刷新、默认活动维护、本地筛选、商品表和手动报名历史。
   - 自动化标签集中承载调度开关、触发时间、只读默认活动摘要、规则状态概览和自动化历史。
   - 深度诊断继续只留在自动化详情弹窗，不再挤占主页面工作区。
3. `frontend/src/views/Layout.vue`：
   - 菜单入口从“CPO 商品报名”统一改为“搜索推广商品”，用户可见命名与当前业务口径对齐。

### 影响范围
1. `/promotions/search-cpo` 仍是同一个页面入口，但现在更像两个相邻工作面，而不是一个巨型总控台。
2. 默认活动配置仍然是手动报名与自动化共用的一份数据，只是编辑入口收敛到了手动标签。
3. 无数据库结构变更，无新增 migration 脚本；本次仅调整前端信息架构和用户可见文案。

### 验证
1. `cd frontend && cmd /c npm run build`
## 2026-03-22
### 主题
恢复“自动加促销”的相对日期语义：支持昨天 / 今天 / 自定义日期，并让定时任务按规则动态计算本次执行日期。

### 关键变更
1. `backend/internal/service/auto_promotion_service.go`、`backend/internal/model/auto_promotion.go`、`backend/internal/dto/auto_promotion.go`、`backend/internal/repository/auto_promotion_repo.go`：
   - 自动加促销配置和运行历史新增 `target_date_mode`，枚举固定为 `yesterday / today / custom`。
   - 配置表中的 `target_date` 改为仅在 `custom` 模式下保存；手动执行和定时调度都会在运行时解析出本次实际日期并写入 run。
   - `GetConfig` 的默认语义改为“昨天”，不再靠预填一个固定绝对日期模拟默认值。
2. `frontend/src/views/promotions/AutoAdd.vue`：
   - 页面配置项由“目标日期”改成“上架时间规则”，支持昨天、今天、自定义日期三种模式。
   - 历史列表和详情弹窗新增“日期规则”，原 `target_date` 明确展示为“实际日期”。
   - 自定义日期模式下新增前端校验，避免空日期直接发请求。
3. `backend/migrations/init_database.sql`、`backend/migrations/upgrade_20260322_auto_promotion_relative_date_mode.sql`：
   - 新环境基线已同步为最新结构。
   - 旧环境可通过新增升级脚本把已有配置和历史回填为 `custom`，同时放宽配置表 `target_date` 的非空约束。
4. `backend/internal/service/auto_promotion_service_test.go`：
   - 新增日期规则解析与配置校验测试，覆盖昨天、今天、自定义日期、空模式兼容和非法输入。

### 影响范围
1. “自动加促销”重新具备“每天自动处理昨天上架商品”的核心语义。
2. 现有已保存配置不会被静默改成“昨天”；升级后默认按 `custom` 保留原行为，运营可自行切换回相对规则。
3. 本次包含数据库结构变更，旧库需要执行 `upgrade_20260322_auto_promotion_relative_date_mode.sql`。

### 验证
1. `cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./...` 通过。
2. `cd frontend && cmd /c npm run build` 通过。
## 2026-03-22
### 主题
修正 Search CPO 状态迁移里的两类现场误判：店铺活动退出看似成功、商品却仍留在店铺活动中，以及历史 `morkovsk_joined_at` 压过当前 live 状态导致“手动执行一次”不再把商品重新推进到状态4。

### 关键变更
1. `backend/internal/service/search_cpo_automation.go`：
   - Search CPO 店铺活动退出不再复用 `promo_unified_remove` 的多活动聚合结果，而是按活动逐条创建 `shop_action_remove` 任务、逐条等待并逐条写入 `exit_results`。
   - Search CPO 店铺退出请求现优先使用数值 `sku`；任务完成后会立即刷新对应活动商品，并按原始 `source_sku` 复核该商品是否仍在活动中，若仍存在则把退出结果改判为失败并提示 `退出后复核仍在活动中`。
   - Search CPO 状态派生新增 live 优先判定：当 `search_promo_status + carrots_status + availability` 已明确落在状态1/2/3/4 之一时，不再让历史 `morkovsk_joined_at` 无条件压过当前 live 状态；live 已回退到状态1/2/3 的商品会在本次运行里清空失效 joined 标记并重新进入正常迁移。
   - `joined repair` 现仅对 live 仍为状态4、且 `RuleStateAfter = morkovsk_joined` 的商品生效；状态2商品会重新执行 `退出其它活动 -> enable -> Morkovsk`，状态3商品会重新执行 `退出其它活动 -> 跳过重复 enable -> Morkovsk`。
2. `backend/internal/service/promotion_service.go`：
   - 新增 `CreateShopActionJobWithMeta`，为 Search CPO 专用单活动 remove 任务保留扩展元数据入口，同时兼容现有 `CreateShopActionJob` 调用方。
3. `backend/internal/service/search_cpo_automation_test.go`：
   - 新增 joined repair 辅助逻辑单测，锁定“已加入 Morkovsk 的商品跳过重复步骤”这一行为边界。

### 影响范围
1. `3371928661`、`3352634622` 这类商品即使历史上已经写过 `morkovsk_joined_at`，只要当前 Ozon live 状态被手工改回状态2/3，下一次“手动执行一次”也会重新把它们送回正常迁移，而不再被短路成 joined repair。
2. 已推进到状态4、却仍残留在店铺活动 28 中的商品，后续自动化运行会继续尝试把它们从 28 清理出去，而不会重复执行 `enable` 或重复加入 Morkovsk。
3. Search CPO 自动化详情中的店铺退出结果将按活动粒度展示真实状态；若 28 实际未删掉，会直接显示该活动失败，而不再被聚合结果掩盖。
4. 无数据库结构变更，无新增 migration 脚本。

### 验证
1. `cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./internal/service`
2. `cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./pkg/ozon`
## 2026-03-21
### 主题
修正 Search CPO 状态迁移里的两类现场误判：前置活动 404 不应整批阻断，`product/enable` / `carrots/batch_enable` 也不应因响应包裹层变化或 job 级错误扇出而误伤多个 SKU。

### 关键变更
1. `backend/internal/service/search_cpo_automation.go`：
   - 迁移前置阶段改为仅处理 `status=active` 的活动；刷新活动商品时若命中 `404/NotFound`，自动将该活动标记为 `disabled` 并跳过，不再整批阻断后续 `enable` / `Morkovsk`。
   - `enable` / `Morkovsk` 现在优先按 artifact 逐 SKU 回填步骤结果；当 job 整体失败、快照缺失或结果缺项时，会回退到“按当前 SKU 装饰错误”，不再把首条 `source_sku=...` 错误复制给所有商品。
   - 对当前已是 `SEARCH_PROMO_STATUS_ENABLED` 的商品，迁移后半段会直接跳过重复 `enable`，继续进入 Morkovsk。
2. `browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch_v3.js`：
   - `product/enable` 与 `carrots/batch_enable` 现已支持 `data/result/response` 包裹层解析，并在未匹配响应时输出 `source_sku/requested_sku/parser_revision/HTTP/root/sample keys` 诊断。
3. `frontend/src/views/promotions/SearchCPO.vue`：
   - 自动化详情里的 `Enable` / `Morkovsk` 现会补展示 step diagnostics；活动结果仍保留店铺活动 `source_action_id`，便于直接定位失效缓存记录。
4. 测试与脚本：
   - 扩展 parser fixture 已新增 `enable` / `Morkovsk` 包裹层与缺匹配诊断场景，后端也补了逐 SKU fallback / step diagnostics 单测。

### 影响范围
1. `3371928661` 这类状态2商品，若 `enable` 现场仍失败，详情会展示它自己的 `requested_sku/parser_revision/HTTP/root/sample keys`，不再被别的 SKU 错误覆盖。
2. `3352634622` 这类当前已是状态3的商品，不再因为重复 `enable` 被额外打成失败，而会直接进入 Morkovsk。
3. 当某条历史店铺/官方活动记录已失效时，自动化详情仍会明确显示对应活动与 `source_action_id`。
4. 无数据库结构变更，无新增 migration 脚本。

### 验证
1. `node browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-response-parsers.mjs`
2. `node --check browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch_v3.js`
3. `cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./pkg/ozon ./internal/service`
4. `cd frontend && cmd /c npm run build`
## 2026-03-20
### 主题
修正 Search CPO 状态迁移里的退出判定与 `state3_trigger`，让店铺活动 `404/NotFound` 不再误阻断迁移，并按官方 `deactivate` 响应解析逐商品失败原因。

### 关键变更
1. `backend/internal/service/search_cpo_automation.go`：
   - `state3_trigger` 已放宽为 live 条件触发：`SEARCH_PROMO_STATUS_ENABLED + CARROTS_STATUS_DISABLED + availability=true` 的商品，即使没有历史 `state2_detected_at` 也会进入迁移后半段，并在首次命中时回填检测时间。
   - 店铺活动退出结果新增细分：`商品当前不在活动中` 与 `404/NotFound` 会归类为 `skipped`，不再阻断后续 `enable` / `Morkovsk`；退出步骤会保留逐动作诊断信息。
2. `backend/pkg/ozon/actions.go`、`backend/internal/service/promotion_service.go`：
   - 官方活动退出已按 `doc/按订单付费推广商品操作/官方deactivate.txt` 对齐 `product_ids + rejected[]` 响应。
   - `removeFromOfficialActions` 与 `exitAllPromotions` 现在会识别 `rejected[].reason` 和“未知结果”，不再只把无 transport error 的退出一律视为成功。
3. `browser-extension/ozon-shop-bridge/background_search_cpo_remove_result_patch.js`、`background_search_cpo_bootstrap.js`：
   - extension 新增 Search CPO 店铺活动退出补丁，把 remove 阶段的 `404/NotFound` 统一归类为可跳过的 `action_not_found`，同时保留 `source_action_id` 与原始错误摘要。
   - Search CPO 扩展 build/parser revision 已提升到 `2026-03-20-d`，并同步更新回归脚本断言。
4. `backend/internal/service/search_cpo_automation_test.go`、`backend/internal/service/promotion_service_official_products_test.go`、`backend/pkg/ozon/actions_test.go`：
   - 新增/补充单测，覆盖 `state3_trigger` 放宽、店铺退出 `404` 归类、官方 `deactivate` 的 `rejected[]` 解析。

### 影响范围
1. `3340371883` 这类 `SEARCH_PROMO_STATUS_ENABLED + CARROTS_STATUS_DISABLED + availability=true` 的商品，不再因为缺少历史 `state2_detected_at` 被误跳过。
2. `3352634622` 这类店铺活动退出返回 `404/NotFound` 的商品，不再因为前置退出误判失败而直接跳过后续 `enable` 和 `Morkovsk`。
3. 无数据库结构变更，无新增 migration 脚本；本次仅修正 Search CPO 自动化、官方退出响应解析和 extension remove 结果归类。

### 验证
1. `node --check browser-extension/ozon-shop-bridge/background_search_cpo_remove_result_patch.js`
2. `node browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-response-parsers.mjs`
3. `node browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-service-worker-order.mjs`
4. `cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./pkg/ozon ./internal/service`
5. `cd frontend && cmd /c npm run build`
## 2026-03-20
### 主题
补齐 Search CPO availability 的原始响应摘要诊断，并把诊断字段直接展示到自动化详情页，避免排障必须手查数据库 artifact。

### 关键变更
1. `browser-extension/ozon-shop-bridge/background_search_cpo_bootstrap.js`、`background_search_cpo_fetch_diagnostics_patch.js`、`background_search_cpo_runtime_diagnostics_patch_v2.js`、`background_search_cpo_runtime_diagnostics_patch_v3.js`：
   - service worker 继续沿用补丁式加载，不重写现有大脚本；在原 response/runtime patch 之后补上一层 fetch diagnostics patch、runtime diagnostics v2 patch 和 runtime diagnostics v3 patch。
   - `search_promo_availability` 在 HTTP 200 但返回 HTML、空值或非 JSON 时，不再在 `response.json()` 失败后静默变成空对象，而会保留 `response_kind/http_status/content_type/parse_error/response_excerpt`。本轮进一步把 availability 页面执行改成自包含 inline wrapper，避免 executeScript 依赖 service worker 外部 helper。
   - extension register / poll / availability report 的 `build_revision/parser_revision` 已提升到 `2026-03-20-d`。
2. `backend/internal/dto/search_cpo_automation.go`、`backend/internal/service/search_cpo_automation.go`：
   - 自动化详情接口新增 availability 诊断结构，后端会从 `search_cpo_products.availability_payload` 解析 `requested_sku`、`build_revision`、`response_content_type`、`response_parse_error`、`response_excerpt`、root/sample keys 等字段并返回给前端。
   - 无需新增表结构；继续复用已有 `availability_payload` 落库结果。
3. `frontend/src/views/promotions/SearchCPO.vue`：
   - “状态迁移详情”表格新增展开式 availability 诊断面板，单个商品可直接查看请求 SKU、HTTP、Content-Type、解析错误、业务原因和响应摘要。
4. `browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-response-parsers.mjs`、`verify-search-cpo-service-worker-order.mjs`：
   - 回归脚本已纳入 fetch diagnostics patch / runtime v2 / runtime v3 patch，并覆盖 `2026-03-20-d` 版本断言和 HTML/parse-error 诊断场景。

### 影响范围
1. 当 CPO 自动化再次报“未匹配到 search_promo_availability 响应”时，页面上就能直接区分是请求 SKU 映射错误、旧扩展、返回 HTML/挑战页，还是脚本根本没拿到结构化对象。
2. 无数据库结构变更，无新增 migration 脚本；本次仅补充扩展诊断、后端透传和前端展示。

### 验证
1. `node --check browser-extension/ozon-shop-bridge/background_search_cpo_fetch_diagnostics_patch.js`
2. `node --check browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch_v2.js`
3. `node --check browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch_v3.js`
4. `node browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-response-parsers.mjs`
5. `node browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-service-worker-order.mjs`
6. `cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./internal/service`
7. `cd frontend && cmd /c npm run build`
## 2026-03-20
### 主题
补齐 Search CPO availability 运行时诊断，避免“未匹配响应”只留下泛化错误，并把当前扩展解析版本一起透传到后端。

### 关键变更
1. `browser-extension/ozon-shop-bridge/background_search_cpo_bootstrap.js`、`background_search_cpo_runtime_diagnostics_patch.js`：
   - 在原 `background_search_cpo_response_patch.js` 之后继续加载 runtime diagnostics patch，不改现有 CPO 页签复用、请求头捕获和 job 分发主链路。
   - availability parser 新增对 `data/result/payload/response` 包裹层的宽松搜索；未命中时错误文案会带 `source_sku`、`requested_sku`、`parser_revision`、响应 key 摘要和 map key 数量。
2. `backend/internal/dto/extension.go`、`backend/internal/service/automation_service.go`、`automation_failure.go`：
   - extension register 新增 `build_revision`，extension report 事件会附带 `build_revision/parser_revision`。
   - Search CPO 三类 job 的失败消息会优先保留逐 SKU 细节，并在需要时补齐 `source_sku=` 前缀，降低页面和运行历史排障成本。
3. `browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-response-parsers.mjs`、`verify-search-cpo-service-worker-order.mjs`：
   - parser 样本回归新增 envelope 变体与诊断字段断言。
   - 新增 service worker 加载顺序回归，验证 `background_search_cpo.js -> background_search_cpo_response_patch.js -> runtime diagnostics patch` 的真实 override 顺序和 `build_revision` 上报。

### 影响范围
1. `CPO 商品报名 -> 手动执行一次` 若 availability 仍失败，页面错误不再只有“未匹配到 search_promo_availability 响应”，而会直接给出 `requested_sku`、解析版本和响应 key 摘要。
2. 当前剩余不确定性已收敛到真实 Seller live 响应本身，不再卡在“浏览器到底跑的是哪版 parser”。
3. 无数据库结构变更，无新增 migration 脚本；本次仅调整扩展 diagnostics、后端透传与回归脚本。

### 验证
1. `node --check browser-extension/ozon-shop-bridge/background_search_cpo_runtime_diagnostics_patch.js`
2. `node browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-response-parsers.mjs`
3. `node browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-service-worker-order.mjs`
4. `cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./internal/service`
## 2026-03-19
### 主题
修复 Search CPO 自动化三接口响应解析漂移，避免 availability 误报“未匹配响应”，并补齐 enable / Morkovsk 的逐 SKU 业务失败判定。

### 关键变更
1. `browser-extension/ozon-shop-bridge/background_search_cpo_bootstrap.js`、`manifest.json`：
   - 插件 service worker 改为先加载原 `background_search_cpo.js`，再加载 `background_search_cpo_response_patch.js`，避免在 Windows 环境下直接重写大脚本时补丁工具不稳定。
   - 不改现有 Search CPO 页签复用、请求头捕获和任务分发主链路，只覆盖本次有问题的响应解析与执行结果判定函数。
2. `browser-extension/ozon-shop-bridge/background_search_cpo_response_patch.js`：
   - `search_promo_availability` 改为支持 `skuToIsSearchPromoAvailable` 与 `skuToIsSearchPromoAvailabilityWithReason` 双 map 返回，并补齐 `isAvailable` 字段解析。
   - `product/enable` 改为按 `bids[]` 逐 SKU 判定成功/失败；`carrots/batch_enable` 改为按 `skuToInfo{}` 逐 SKU 判定成功/失败。
   - enable / Morkovsk 两步不再只凭 HTTP 200 直接回报成功，缺 SKU 或业务错误会在运行历史中显式失败。
3. `browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-response-parsers.mjs`：
   - 新增基于本地抓包文档的回归脚本，直接用 `search_promo_availability.txt`、`enable.txt`、`batch_enable.txt` 样本验证 parser 行为，并覆盖业务失败场景。

### 影响范围
1. 点击 `CPO 商品报名 -> 手动执行一次` 时，availability 步骤不再把真实 map 响应误判为“未匹配到 search_promo_availability 响应”。
2. 即使 Seller 对单个 SKU 返回 enable / Morkovsk 业务失败，系统也会按逐 SKU 明细回报 `failed` 或 `partial_success`，不再静默写成全成功。
3. 无数据库结构变更，无新增 migration 脚本；本次变更仅涉及插件 service worker 加载方式与 Search CPO 解析层。

### 验证
1. `node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`
2. `node --check browser-extension/ozon-shop-bridge/background_search_cpo_response_patch.js`
3. `node --check browser-extension/ozon-shop-bridge/background_search_cpo_bootstrap.js`
4. `node browser-extension/ozon-shop-bridge/scripts/verify-search-cpo-response-parsers.mjs`
5. `Get-Content browser-extension/ozon-shop-bridge/manifest.json -Raw | ConvertFrom-Json | Out-Null`

## 2026-03-19
### 主题
修复 Search CPO 刷新状态缺失与自动化全跳过问题，补齐状态可见性、SKU 映射和失败语义。

### 关键变更
1. `browser-extension/ozon-shop-bridge/background_search_cpo.js`：
   - `normalizeSearchCPOProduct` 补齐 `carrots_status` 映射，`list` 返回的 `carrotsStatus` 会随刷新一起落库。
   - `sync_search_cpo_availability`、`search_cpo_enable_products`、`search_cpo_batch_enable_morkovsk` 改为读取后端下发的 `source_sku -> sku` 映射，对 Seller 私有接口统一发送数值 `sku`。
   - `normalizeSearchCPOAvailabilityItems` / `normalizeSearchCPOStepItems` 改为“未匹配到响应即显式失败”，不再静默写成 `availability_promo=false` 或默认 `success`。
2. `backend/internal/service/search_cpo_automation.go`、`automation_service_search_cpo.go`、`search_cpo_repo.go`：
   - Search CPO 三类自动化 job 新增并透传 `search_cpo_meta/search_cpo_morkovsk_meta`，供 extension 执行时恢复真实数值 `sku`。
   - availability 回写改为仅更新真实返回的 `search_promo_status/carrots_status/availability_promo`，避免缺响应时覆盖掉历史正确状态。
   - `enable` / `Morkovsk` 成功后同步推进本地 `search_promo_status`、`carrots_status`、`rule_state` 与 `morkovsk_joined_at`，避免 run 成功后列表仍停留在旧状态。
3. `frontend/src/views/promotions/SearchCPO.vue`：
   - 列表页空 `search_promo_status` 改为显示“状态未知”，不再把缺失值直接渲染成“已关闭”。
4. `dev-tracker/*`：
   - `OVERALL_TASKS.md`、`CURRENT_PROGRESS.md` 与本日志已同步补充本次 Search CPO 状态修复口径。

### 影响范围
1. 点击“刷新 CPO 商品”后，Seller `list` 已返回的 `searchPromoStatus/carrotsStatus` 现在会真实落到本地并展示。
2. 点击“手动执行一次”时，availability / enable / Morkovsk 三类 Seller job 不再把 `source_sku` 误当成请求 `sku`，不会再因为响应未命中而大面积静默 `skipped`。
3. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 后端回归测试通过：`cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./...`。
2. 前端构建通过：`cd frontend && cmd /c npm run build`。
3. 插件脚本语法检查通过：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`。
## 2026-03-19
### 主题
收口 Search CPO 状态迁移规则，按 state2 主触发改造自动化执行，并拆清人工批量报名与自动迁移文档边界。

### 关键变更
1. `backend/internal/service/search_cpo_automation.go`：
   - 自动化规则改为：`state1` 只做初始活动报名；`state2` 直接执行“全量同步活动 -> 退出命中活动 -> enable -> Morkovsk”；`state3_trigger` 只作为历史漏跑兜底。
   - 迁移前会先调用活动清单全量同步，再刷新活动商品成员关系；前置失败改为逐商品写失败详情，不再直接丢失 run 明细。
2. `backend/internal/service/search_cpo_service.go`：
   - 自动化 `enable_step` 收口为兼容字段，服务端固定按 `true` 处理并返回 `true`。
3. `frontend/src/views/promotions/SearchCPO.vue`：
   - 页面明确拆分“人工批量报名”和“状态迁移自动化”两条链路。
   - 移除自动化 `Enable 步骤` 开关，并同步调整自动化说明、筛选标签、历史统计和详情步骤顺序。
4. 文档同步：
   - `doc/按订单付费推广商品操作/推广商品添加活动规则.md` 改为只描述状态迁移自动化。
   - 新增 `doc/按订单付费推广商品操作/CPO商品批量报名.md`，单独说明人工批量报名。
   - `dev-tracker/OVERALL_TASKS.md`、`CURRENT_PROGRESS.md` 已同步更新当前任务口径。

### 影响范围
1. Search CPO 自动化现在不再在 `state1` 阶段提前执行 `enable`。
2. 检测到 `state2` 后，系统会立即尝试退出已命中的官方/店铺活动、执行 `enable` 并加入 `Morkovsk`。
3. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 后端回归测试：`cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./...`。
2. 前端构建：`cd frontend && cmd /c npm run build`。
3. 插件脚本语法检查：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`。
## 2026-03-19
### 主题
修复 Search CPO 自动化在 extension 侧的任务分发缺口，并同步补齐支持任务说明。

### 关键变更
1. `browser-extension/ozon-shop-bridge/background_search_cpo.js`：
   - `executeJob()` 不再只特判 `sync_search_cpo_products`；所有 Search CPO 专用任务统一先走 `executeSearchCPOJob()`。
   - 修复 `sync_search_cpo_availability`、`search_cpo_enable_products`、`search_cpo_batch_enable_morkovsk` 落入默认 unsupported 分支的问题。
2. 说明文档同步：
   - `browser-extension/ozon-shop-bridge/README.md` 与根目录 `AGENTS.md` 的“当前支持任务类型”补齐三类 Search CPO 自动化 job。
   - `dev-tracker/*` 与 `SEARCH_CPO_MORKOVSK_AUTOMATION_PLAN.md` 同步收敛为当前真实实现状态，避免继续把已接入接口写成“未实现”。

### 影响范围
1. `CPO 商品报名 -> 手动执行一次` 不再因为 extension 报“插件不支持该任务类型: sync_search_cpo_availability”而在 availability 同步阶段提前失败。
2. Search CPO 自动化链路中的 availability / enable / Morkovsk 三类 job 现在都会复用专用 CPO 页签上下文，不会误走通用 Seller 工作页。
3. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 插件脚本语法检查：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`。
2. 后端回归测试：`cd backend && $env:GOCACHE="$env:TEMP\\ozon-manager-gocache"; go test ./...`。
3. 前端构建：`cd frontend && cmd /c npm run build`。
## 2026-03-19
### 主题
Search CPO 刷新范围改为拉取完整 CPO 商品集合，并同步收敛页面语义。

### 关键变更
1. `browser-extension/ozon-shop-bridge/background_search_cpo.js`：
   - `scriptFetchSearchCPOProducts` 的请求体删除 `notAvailableOnly: true`，保留现有 `searchPromoStatus`、分页、页签复用与请求头逻辑不变。
   - 由于 `sync_search_cpo_products` 为共享抓取入口，本次同时影响手动刷新与自动化执行前预刷新。
2. `frontend/src/views/promotions/SearchCPO.vue`：
   - 按当前能力把“刷新隐藏商品”“隐藏 CPO 商品”等提示改为“刷新 CPO 商品”“CPO 商品”，避免继续暗示只抓隐藏集合。
   - 自动化确认文案同步更新，和新的抓取范围保持一致。
3. `dev-tracker/*`：
   - `OVERALL_TASKS.md`、`CURRENT_PROGRESS.md` 与本日志已同步更新当前能力描述，明确 Search CPO 刷新不再额外带 `notAvailableOnly` 条件。

### 影响范围
1. Search CPO 页面与自动化预刷新都会按 Seller 当前 CPO 列表返回的完整商品集合同步，不再额外限定“仅不可用商品”。
2. 当前仍保留按 `source_sku` 去重与落库的既有契约；若刷新数量仍与 Ozon 页面不同，应继续按去重逻辑排查。
3. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 插件脚本语法检查：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`。
2. 前端构建：`cd frontend && cmd /c npm run build`。

## 2026-03-19（自动化首批）
### 主题
落地 Search CPO Morkovsk 三阶段自动化第一批实现，补齐计划文件、后端自动化骨架、插件私有接口执行链路，以及现有 Search CPO 页面的自动化入口。

### 关键变更
1. `dev-tracker/SEARCH_CPO_MORKOVSK_AUTOMATION_PLAN.md`：
   - 新增跨会话可复用的完整实现计划，明确状态定义、接口边界、数据库改动、推荐执行顺序与测试要求。
2. Search CPO 自动化后端：
   - `search_cpo_configs` 新增 `auto_enabled`、`schedule_time`、`enable_step`。
   - `search_cpo_products` 新增 `carrots_status`、`availability_promo`、`availability_payload`、`availability_checked_at`、`rule_state`、`state2_detected_at`、`morkovsk_joined_at`。
   - 新增 `search_cpo_auto_runs`、`search_cpo_auto_run_items`、对应 DTO / repository / handler / service / 路由，以及 `upgrade_20260319_search_cpo_morkovsk_automation.sql`。
   - `SearchCPOService` 已具备手动触发、定时调度、availability 同步、state1 报名 + enable、state3 退出其它活动后加入 Morkovsk 的首版骨架。
3. 插件侧：
   - `background_search_cpo.js` 新增 `sync_search_cpo_availability`、`search_cpo_enable_products`、`search_cpo_batch_enable_morkovsk` 三类 job。
   - 已接入 Seller 私有 `search_promo_availability`、`product/enable`、`carrots/batch_enable`。
   - 店铺活动 remove 场景改为优先按 active 商品命中，SKU 不在活动中时返回 `skipped`。
4. 前端：
   - `SearchCPO.vue` 已新增自动化配置区、手动执行一次按钮、自动化历史与自动化详情弹窗。
   - 商品列表已展示 `carrotsStatus`、`availability_promo`、派生规则状态和关键时间点。

### 影响范围
1. Search CPO 页面现在同时支持“按当前筛选结果手动报名”和“按固定规则手动/定时自动推进 Morkovsk 迁移”。
2. 本批已具备端到端基础设施，但仍需结合真实 Seller 环境继续校准私有接口响应结构与第三步真实退出结果判定。

### 验证
1. 后端回归测试通过：`cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./...`。
2. 插件脚本语法检查通过：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`。
3. 前端构建通过：`cd frontend && cmd /c npm run build`。
## 2026-03-18
### 主题
插件改为“自动连接管理端登录态”为主流程，普通用户无需再手动复制 token。

### 关键变更
1. `browser-extension/ozon-shop-bridge/background_search_cpo.js`：
   - 新增 `OZON_MANAGER_CHECK_AUTH_SYNC` 消息，用于主动检测管理端连接状态。
   - 插件会扫描已打开的管理端页签，实时读取 `localStorage.token` 与 `currentShopId`，并返回 `connected / missing_login / missing_shop / permission_required / sync_unavailable / failed` 结构化状态。
   - `handleAuthSync` 与本地存储新增管理端连接元数据，popup 可直接展示连接来源、最近检测与失败原因。
2. `browser-extension/ozon-shop-bridge/popup.html`、`popup.js`：
   - popup 主界面改为自动连接优先，新增“重新检测”按钮。
   - `Token / 店铺 ID / 管理端 Origin` 默认收进“高级设置 / 排障”，避免普通用户被引导去手填凭证。
   - 首屏明确展示“请先登录 / 请先选择店铺 / 需要授权 / 未打开管理端页面 / 已连接”等状态。
3. `browser-extension/ozon-shop-bridge/README.md`：
   - 使用说明改写为“先登录管理端并选择店铺，非 localhost 首次授权即可”，并明确高级设置仅用于排障兜底。

### 影响范围
1. 普通用户不再需要通过 F12 手动复制 token；插件默认按“打开管理端并登录”完成接入。
2. 非 localhost 管理端首次接入时，会通过 popup 的“重新检测”触发域名授权。
3. 本次不涉及后端业务接口与数据库结构变更。

### 验证
1. 插件语法检查：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js` 通过。
2. 插件语法检查：`node --check browser-extension/ozon-shop-bridge/popup.js` 通过。
3. 插件语法检查：`node --check browser-extension/ozon-shop-bridge/content-auth-sync.js` 通过。

## 2026-03-17（补充三）
### 主题
为 Search CPO 刷新增加“真实请求头捕获”兜底，避免页面能请求成功但插件仍读不到 `x-adv-current-organisation`。

### 关键变更
1. `browser-extension/ozon-shop-bridge/manifest.json`：
   - 新增 `webRequest` 权限，并修正 `manifest.json` 中该权限的声明格式，确保扩展能正常注册请求头监听。
2. `browser-extension/ozon-shop-bridge/background_search_cpo.js`：
   - 监听 `https://seller.ozon.ru/performance-api/seller-api/search-performance-cpo/*` 的出站请求头。
   - 把捕获到的 `x-adv-current-organisation / x-o3-company-id / x-o3-language / x-o3-app-name` 按 `tabId` 缓存到扩展本地存储。
   - `sync_search_cpo_products` 复用现有 CPO 页签时，若页面探测失败，会退回使用这份真实请求头上下文并注入页面。
3. 提示语义：
   - 若刚重载插件、页面尚未产生任何真实 CPO 请求，错误文案会明确提示“先手动刷新该页面一次”。

### 影响范围
1. 首次重载扩展后，只需让目标 CPO 页面产生一次真实请求，后续插件即可复用真实请求头完成刷新。
2. 本次不涉及后端接口、前端页面和数据库结构变更。

### 验证
1. 插件语法检查：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js` 通过。
2. 插件清单解析：`Get-Content browser-extension/ozon-shop-bridge/manifest.json -Raw | ConvertFrom-Json | Out-Null` 通过。

## 2026-03-17（补充二）
### 主题
修复 Search CPO 页面明明能发请求、插件却读不到 `x-adv-current-organisation` 的问题。

### 关键变更
1. `browser-extension/ozon-shop-bridge/background_search_cpo.js`：
   - `scriptReadSearchCPOContext` 从“只读 storage/cookie”扩展为同时扫描页面全局状态、请求配置对象和常见框架根对象。
   - 对 `organization/organisation/company/language/appName` 相关字段增加对象值解析，兼容值不是纯字符串而是对象结构的场景。
   - 成功解析的上下文会缓存在当前页面，供后续 `scriptFetchSearchCPOProducts` 直接复用。
2. Search CPO 刷新链路：
   - `scriptFetchSearchCPOProducts` 改为优先复用页面探测阶段缓存下来的 `companyId/language/appName/advOrganisation`。
   - 仅在缓存不存在时才回退到 cookie/storage，减少“页面真实请求头存在但插件误判缺失”的情况。

### 影响范围
1. 已打开且已正常加载的 Seller CPO 页面，更可能被插件正确读取到请求上下文。
2. 本次不改变后端接口、前端页面和数据库结构。

### 验证
1. 插件语法检查：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js` 通过。

## 2026-03-17（补充）
### 主题
修复 Search CPO 刷新时默认新开 Ozon 页签的问题，改为优先复用已打开的 Seller CPO 页面。

### 关键变更
1. `browser-extension/ozon-shop-bridge/background_search_cpo.js`：
   - `sync_search_cpo_products` 不再走通用 `ensureSellerLoggedInTab()` 工作页流程，避免刷新时默认创建新的 Seller 页签。
   - 新增现有页签扫描逻辑，仅复用 URL 位于 `seller.ozon.ru/app/advertisement/product/cpo...` 的已打开页签。
2. Search CPO 页面上下文探测：
   - 刷新前先在候选页签中实时读取 `sc_company_id/x-o3-company-id` 与 `x-adv-current-organisation`。
   - 若未找到已打开的 CPO 页面，直接返回“请先打开 Seller 推广按订单付费页面后重试”。
   - 若页面缺少组织上下文，直接返回“请在该页面刷新或重新进入后重试”。
3. 变更边界：
   - 本次仅调整 `sync_search_cpo_products` 的页签复用策略。
   - 其它店铺活动同步、候选同步、统一报名/移除等任务继续沿用当前静默工作页机制。

### 影响范围
1. 点击“刷新 CPO 商品”时，插件会优先复用用户当前已经打开的 CPO 页面，不再默认新开 Ozon 页面打断操作。
2. Search CPO 刷新仍然通过 Seller 页面上下文直接调用 `performance-api`，不是回退成页面点击式自动化。
3. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 插件语法检查：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js` 通过。

## 2026-03-17
### 主题
补强 Search CPO 已关闭商品报名链路：官方目录缓存兜底、官方/店铺结果解耦、页面状态语义对齐。

### 关键变更
1. 后端 `SearchCPOService`：
   - 运行前按 `source_sku` 同时查询本地商品与 Ozon 目录缓存。
   - 官方活动报名新增 `offer_id/sku -> ozon_product_id` 兜底，避免仅因本地 `products` 未命中而失败。
   - 官方活动价按“本地当前价 -> CPO 缓存价 -> 目录缓存价”顺序解析。
2. 店铺活动执行链：
   - 移除“官方失败自动跳过店铺活动”的旧逻辑。
   - 逐商品总状态新增 `partial_success`，可明确显示“官方失败 / 店铺成功”这类混合结果。
3. 前端 `SearchCPO` 页面：
   - 将 `SEARCH_PROMO_STATUS_DISABLED` 明确展示为“已关闭”。
   - 执行说明改为推荐先筛选“已关闭”商品后再报名。
4. 测试：
   - 新增 `backend/internal/service/search_cpo_service_test.go`，覆盖目录缓存匹配、官方价格解析与状态汇总。

### 影响范围
1. Search CPO 商品即使未命中本地 `products` 表，也可在目录缓存存在时继续参与官方活动报名。
2. 同一商品的官方与店铺活动结果不再互相遮蔽，运行详情更接近真实执行结果。
3. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 后端：`cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./...` 通过。
2. 前端：`cd frontend && cmd /c npm run build` 通过。
## 2026-03-14
### 主题
新增 Search CPO 商品批量报名能力（页面 + 后端 + 插件任务扩展）。

### 关键变更
1. 后端新增 Search CPO 模块：
   - 新增模型、仓储、服务、handler 与路由，支持配置保存、商品刷新、任务执行、历史查询、详情查看。
   - 新增任务类型 `sync_search_cpo_products` 并接入自动化任务分发能力。
2. 前端新增页面：
   - 新增 `/promotions/search-cpo` 与菜单“CPO 商品报名”。
   - 支持本地筛选后按“当前筛选结果”执行报名，支持运行历史和逐商品详情。
3. 数据库迁移：
   - 新增增量脚本 `backend/migrations/upgrade_20260313_search_cpo_bulk_enroll.sql`。
   - `backend/migrations/init_database.sql` 已同步回写到最新结构。
4. 插件侧：
   - 插件主入口改为 `browser-extension/ozon-shop-bridge/manifest.json -> background_search_cpo.js`，支持 CPO 商品抓取任务。

### 影响范围
1. 运营可在单页完成 Search CPO 商品刷新、筛选、批量报名和历史追踪。
2. 官方活动按当前筛选商品精准报名；店铺活动复用 unified 异步任务链路。
3. 若插件主入口仍指向旧 `background.js`，会出现“插件不支持该任务类型: sync_search_cpo_products”。

### 验证
1. 后端：`go test ./...` 通过。
2. 前端：`cmd /c npm run build` 通过。
3. 插件：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js` 通过。
## 2026-03-11（补充三）
### 主题
修复店铺活动候选同步失败时错误原因被吞掉的问题，并纠正插件支持任务说明。

### 关键变更
1. `backend/internal/repository/automation_repo.go`：
   - `UpdateJobAndItemsByReport` 在 job 失败或部分成功时，将首个 item 失败明细同步写入 `automation_jobs.error_message`。
2. `backend/internal/service/promotion_service.go`、`backend/internal/service/auto_promotion_service.go`：
   - 促销同步与自动加促销在等待 extension 任务失败时，优先返回 job 明细错误，不再只报 `shop ... sync failed`。
3. 新增纯函数测试：
   - `backend/internal/repository/automation_repo_error_message_test.go`
   - `backend/internal/service/automation_failure_test.go`
4. 文档纠偏：
   - `AGENTS.md` 的“插件当前支持任务”补上 `sync_action_candidates`，与当前仓库代码保持一致。

### 影响范围
1. 下次再遇到 `sync_action_candidates`、`sync_shop_actions`、`sync_action_products` 类 extension 任务失败时，后端返回会直接带出更底层原因。
2. 自动加促销在店铺候选同步失败时，能够直接暴露如“插件不支持该任务类型”这类真实错误，缩短排障路径。
3. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 后端定向测试：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./internal/repository ./internal/service` 通过。

## 2026-03-11（补充二）
### 主题
修复自动加促销执行时官方候选分页 `last_id` 类型错误导致的 400 失败。

### 关键变更
1. `backend/pkg/ozon/actions.go`：
   - `POST /v1/actions/candidates` 的请求体 `last_id` 改为始终按字符串发送。
   - 保留响应侧对 `result.last_id` 的宽松解析，继续兼容 `string/number` 两种游标形态。
2. `backend/pkg/ozon/actions_test.go`：
   - 将候选接口单测从“断言 number 请求游标”改为“断言 string 请求游标”。
   - 新增回归用例，覆盖“首轮响应返回 numeric cursor，下一轮请求仍按 string 发送”的兼容链路。
3. 候选接口标准文档纠偏：
   - `doc/ozon-promos-candidates-activate-standard.md` 改为说明请求体 `last_id` 使用字符串。
   - `doc/ozon-promos-candidates-activate.openapi.yaml` 将请求 schema 改为 `string`，响应 schema 改为兼容 `string/number`。

### 影响范围
1. `POST /api/v1/promotions/auto-add/runs` 触发的官方候选刷新不再因第二页游标类型错误直接失败。
2. 自动加促销官方活动链路可继续刷新候选并进入后续筛选/加促销步骤。
3. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 后端定向测试：`cd backend && $env:GOCACHE=\"$env:TEMP\\ozon-manager-gocache\"; go test ./pkg/ozon ./internal/service` 通过。

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

## 2026-04-05
### 主题
修复自动加促销在真实环境中的三类稳定性问题：目录刷新并发失败、候选快照读取竞态，以及 Ozon 只读接口瞬时超时。

### 关键变更
1. `backend/internal/service/ozon_catalog_service.go`：
   - `RefreshShopCatalogSync` 遇到同店铺已有刷新运行中时，改为等待当前刷新结束并复用结果，不再直接返回 `catalog refresh already running`。
2. `backend/internal/service/automation_service.go`：
   - `GetLatestArtifact` 对刚完成的自动化任务增加短暂轮询等待，避免 job 已成功但 `action_candidates_snapshot` / 其它快照尚未落库时立即读出 `record not found`。
3. `backend/pkg/ozon/client.go`、`backend/pkg/ozon/catalog.go`、`backend/pkg/ozon/actions.go`：
   - 为商品目录和活动候选/已报名等只读 Seller 请求增加有限次瞬时超时重试，覆盖 `context deadline exceeded`、`TLS handshake timeout` 一类网络抖动。
4. `backend/internal/service/auto_promotion_service.go`：
   - 自动加促销店铺活动候选同步等待窗口从 60 秒提升到 2 分钟，降低真实浏览器链路的误报超时。
5. 新增回归测试：
   - `backend/internal/service/automation_service_artifact_test.go`
   - `backend/internal/service/ozon_catalog_service_refresh_test.go`
   - `backend/pkg/ozon/catalog_retry_test.go`
   - 并补充 `backend/internal/service/auto_promotion_service_test.go` 的候选同步等待窗口断言。

### 影响范围
1. 自动加促销与其它依赖 automation artifact 的链路，明显降低 `record not found` 竞态失败。
2. 目录刷新与官方候选/已报名同步在遭遇瞬时网络抖动时，具备有限重试能力，减少偶发 `TLS handshake timeout` / `context deadline exceeded` 直接中断。
3. 同店铺目录刷新并发时，后发请求改为等待已有刷新完成，避免自动加促销因为撞到后台刷新而直接失败。
4. 无数据库结构变更，无新增迁移脚本。

### 验证
1. 后端完整回归：`cd backend && $env:GOCACHE="$env:TEMP\\ozon-manager-gocache"; go test ./...` 通过。
