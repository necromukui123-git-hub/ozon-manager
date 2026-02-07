# Repository Guidelines

## 项目结构与模块组织
- `backend/`：Go 1.21 后端服务，入口在 `cmd/server/main.go`。
- `backend/internal/`：分层实现，包含 `handler/`、`service/`、`repository/`、`model/`、`middleware/`、`dto/`、`config/`。
- `backend/pkg/`：通用能力（如 `ozon/`、`excel/`、`jwt/`）。
- `backend/migrations/init_database.sql`：数据库初始化与全量结构基线。
- `frontend/`：Vue 3 + Vite 前端，主代码在 `frontend/src/`（`views/`、`api/`、`stores/`、`router/`、`styles/`）。
- 根目录 `start-dev.bat`：Windows 下一键启动前后端。

## 构建、测试与开发命令
- 后端运行：`cd backend && go run cmd/server/main.go`
- 后端构建：`cd backend && go build -o server cmd/server/main.go`
- 密码重置工具：`cd backend && go run cmd/reset-password/main.go`
- 前端开发：`cd frontend && npm run dev`
- 前端构建：`cd frontend && npm run build`
- 前端预览：`cd frontend && npm run preview`
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

## 数据库变更要求（强制）
- 任何数据库相关改动（表、字段、索引、约束、初始化数据）都必须**同步更新** `backend/migrations/init_database.sql`。
- 该文件应始终保持“可在新电脑/新环境一键初始化”的最新状态，确保可快速部署本服务。
