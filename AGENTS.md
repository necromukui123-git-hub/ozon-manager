# Repository Guidelines

## 项目结构与模块组织
- `backend/`：Go 1.21 后端服务，入口在 `cmd/server/main.go`。
- `backend/internal/`：分层实现，包含 `handler/`、`service/`、`repository/`、`model/`、`middleware/`、`dto/`、`config/`。
- `backend/pkg/`：通用能力（如 `ozon/`、`excel/`、`jwt/`）。
- `backend/migrations/init_database.sql`：数据库初始化与全量结构基线。
- `frontend/`：Vue 3 + Vite 前端，主代码在 `frontend/src/`（`views/`、`api/`、`stores/`、`router/`、`styles/`）。
- `browser-extension/ozon-shop-bridge/`：Chrome 插件执行通道（店铺促销任务执行，MV3）。
- `dev-tracker/`：开发追踪文档目录（`OVERALL_TASKS.md`、`CURRENT_PROGRESS.md`、`CHANGELOG.md`）。
- 根目录 `start-dev.bat`：Windows 下一键启动前后端。

## 构建、测试与开发命令
- 后端运行：`cd backend && go run cmd/server/main.go`
- 后端构建：`cd backend && go build -o server cmd/server/main.go`
- 密码重置工具：`cd backend && go run cmd/reset-password/main.go`
- 前端开发：`cd frontend && npm run dev`
- 前端构建：`cd frontend && npm run build`
- 前端预览：`cd frontend && npm run preview`
- 插件打包（测试包）：`cd browser-extension/ozon-shop-bridge/scripts && .\package.ps1`
- 插件加载（测试）：打开 `chrome://extensions/`，开启开发者模式，加载 `browser-extension/ozon-shop-bridge` 目录
- 一键启动：双击根目录 `start-dev.bat`

## 代码风格与命名规范
- Go：提交前执行 `gofmt`；包名小写；导出标识符使用 `PascalCase`；文件名建议 `snake_case`（如 `promotion_service.go`）。
- Vue/JS：2 空格缩进；组件文件使用 `PascalCase.vue`；工具与 API 模块使用语义化命名（如 `request.js`、`shopAdmin.js`）。
- 权限相关字段统一使用：`super_admin`、`shop_admin`、`staff`。

## 测试规范
- 当前仓库未内置完整自动化测试；新增功能应补充测试。
- 后端测试文件命名为 `*_test.go`，运行：`cd backend && go test ./...`
- 前端如新增测试，建议采用 `*.test.js` 或 `*.spec.js`，并在 `frontend/package.json` 补充脚本。

## 提交与合并请求规范
- 提交信息遵循 Conventional Commits：`feat:`、`refactor:`、`docs:` 等。
- 每次提交聚焦单一变更主题，避免混合重构与功能修改。
- PR 至少包含：变更说明、影响范围、关联任务（如有）、UI 变更截图（如有）。

## 沟通语言规范（强制）
- 与用户的所有对话（需求确认、进度同步、问题说明、结果交付）必须使用中文。
- 与 AI/代理协作时默认使用中文；若必须引用英文内容（接口字段、报错原文、第三方术语），需补充中文说明。
- 代码标识符、数据库字段、API 参数等技术命名遵循现有英文约定，不做强制中文化。

## 变更执行与验收约定（强制）
- 每次任务实现应尽量最小化改动范围，避免无关重构混入本次交付。
- 涉及后端改动时，至少执行受影响范围的 Go 测试（如 `cd backend && go test ./...` 或指定包）。
- 涉及前端改动时，至少执行一次构建校验（`cd frontend && npm run build`）。
- 若受环境限制无法完成测试或构建，必须在交付说明中明确“未执行项、原因、建议验证方式”。

## 开发追踪文档维护规则（强制）
- 每次任务目标发生更新（范围、优先级、阶段目标、里程碑变化），必须同步更新 `dev-tracker/OVERALL_TASKS.md`。
- 每次执行完成某项任务（功能实现、修复、验证通过、交付动作），必须同步更新 `dev-tracker/CURRENT_PROGRESS.md`。
- 每次关键交付完成后，必须补充 `dev-tracker/CHANGELOG.md`，用于保留长期可追溯记录。
- 两份文档必须保持一致性：`OVERALL_TASKS.md` 记录总体目标与路线，`CURRENT_PROGRESS.md` 记录当前进展与下一步。
- 文档更新属于交付的一部分，不可省略。
- 每次新增或调整数据库迁移脚本时，必须在 `dev-tracker/CURRENT_PROGRESS.md` 记录：脚本名、用途、执行条件、执行结果。

## 数据库迁移与发布规则（强制）
- `backend/migrations/init_database.sql` 是**全量基线**：必须始终可在任何新电脑/新服务器一键初始化数据库。
- 任何数据库改动（表、字段、索引、约束、初始化数据）都必须**同步回写**到 `init_database.sql`。
- 对已有数据库环境的升级，必须新增**版本化增量脚本**：`backend/migrations/upgrade_YYYYMMDD_<topic>.sql`。
- 增量脚本要求：
  - 仅包含“旧版本 -> 当前版本”必需 SQL。
  - 尽量幂等（优先使用 `IF NOT EXISTS`、条件更新、数据检查）。
  - 文件头部写明适用范围、执行前检查、失败处理建议。
- 不再维护 `backend/migrations/upgrade_standalone.sql`；数据库升级仅以 `upgrade_YYYYMMDD_<topic>.sql` 为执行与历史依据。

## 数据库脚本执行约定
- 新环境初始化：执行 `backend/migrations/init_database.sql`。
- 老环境升级：直接执行目标 `upgrade_YYYYMMDD_<topic>.sql`。
- 发布完成后必须确认三件事：
  - `init_database.sql` 已同步到最新结构。
  - 本次增量脚本已记录在 `dev-tracker/CURRENT_PROGRESS.md`。
  - 相关目标状态已同步到 `dev-tracker/OVERALL_TASKS.md`。

## 执行通道约束（当前架构）
- 官方促销：由后端通过官方 API 执行。
- 店铺促销：优先由浏览器插件执行（静默优先，未登录时触发登录兜底）。
- 当前旧 `agent` 与插件并存，测试阶段不建议同时启用同类任务领取，以避免任务竞争。
- 当前插件自动同步 `token/currentShopId` 默认面向 localhost 开发环境；非 localhost 域名需补充插件匹配配置。

## 阶段状态（截至当前）
- 后端 extension 接口已具备：`/api/v1/extension/register`、`/api/v1/extension/poll`、`/api/v1/extension/report`、`/api/v1/extension/reprice`。
- 插件当前支持任务：`sync_shop_actions`、`sync_action_candidates`、`sync_action_products`、`shop_action_declare`、`shop_action_remove`、`promo_unified_enroll`、`promo_unified_remove`、`remove_reprice_readd`。
- 当前后续重点：执行引擎路由开关、前端插件状态面板、非 localhost 自动同步完善、后端测试补充、Chrome 商店上架准备。
