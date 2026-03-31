# Access Token + Refresh Token Authentication Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将当前单 JWT 登录升级为商用化取向的 access token + refresh token 模型，优先解决 Web 端隔夜掉线问题，并保持 Chrome 插件对管理端 `localStorage.token` 的兼容同步。

**Architecture:** 后端签发短期 JWT access token，并通过 `HttpOnly` Cookie 下发长期 opaque refresh token；服务端仅保存 refresh token 哈希，刷新时轮换 token 并支持撤销。前端在应用启动和 `401` 场景下走单飞 refresh，成功后重放原请求；Chrome 插件本轮继续消费管理端同步下来的 access token，不单独实现插件内 refresh。

**Tech Stack:** Go 1.21, Gin, GORM, PostgreSQL SQL migrations, Vue 3, Pinia, Axios, Chrome Extension auth sync

---

### Task 1: 扩展认证配置与 JWT Access Token 生成逻辑

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/config/config.yaml.example`
- Modify: `backend/config/config.yaml`
- Modify: `backend/pkg/jwt/jwt.go`
- Test: `backend/pkg/jwt/jwt_test.go`

- [ ] **Step 1: 写 JWT 配置与 access token claim 的失败测试**

```go
func TestGenerateAccessTokenUsesAccessTTLAndType(t *testing.T) {
    // 断言 exp 按 access 配置生成，token_type=access
}
```

- [ ] **Step 2: 先运行定向测试确认失败**

Run: `cd backend && go test ./pkg/jwt`
Expected: FAIL，提示缺少新的 access token 生成逻辑或 claim 字段

- [ ] **Step 3: 扩展配置结构并实现 access token 生成**

实现要点：
- 在 `JWTConfig` 中新增 `access_expire_minutes`、`refresh_expire_hours`、`refresh_cookie_name`、`refresh_cookie_secure`
- `GenerateToken` 重命名或拆分为 `GenerateAccessToken`
- access token claim 增加 `token_type=access`
- `ParseToken` 只接受 `token_type=access`

- [ ] **Step 4: 再跑 JWT 定向测试**

Run: `cd backend && go test ./pkg/jwt`
Expected: PASS

- [ ] **Step 5: 提交本任务**

```bash
git add backend/internal/config/config.go backend/config/config.yaml.example backend/config/config.yaml backend/pkg/jwt/jwt.go backend/pkg/jwt/jwt_test.go
git commit -m "feat: add access token config and claims"
```

### Task 2: 落地 Refresh Token 持久化模型与数据库迁移

**Files:**
- Create: `backend/internal/model/refresh_token.go`
- Create: `backend/internal/repository/refresh_token_repo.go`
- Modify: `backend/migrations/init_database.sql`
- Create: `backend/migrations/upgrade_20260331_refresh_tokens.sql`
- Modify: `backend/migrations/init_database_test.go`
- Test: `backend/internal/repository/refresh_token_repo_test.go`

- [ ] **Step 1: 为 refresh token 表结构和基线 SQL 写失败测试**

```go
func TestInitDatabaseIncludesUserRefreshTokensTable(t *testing.T) {
    // 断言 user_refresh_tokens 表、token_hash 唯一索引、family_id 索引存在
}
```

- [ ] **Step 2: 运行 migration 和仓储定向测试确认失败**

Run: `cd backend && go test ./migrations ./internal/repository`
Expected: FAIL，提示表结构或仓储尚不存在

- [ ] **Step 3: 新增模型、仓储与 SQL 迁移**

实现要点：
- 模型字段包含 `user_id`、`token_hash`、`family_id`、`expires_at`、`last_used_at`、`revoked_at`、`revoke_reason`、`replaced_by_token_id`、`user_agent`、`ip_address`
- `token_hash` 建唯一索引
- 仓储提供：
  - `Create`
  - `FindActiveByTokenHash`
  - `RevokeByID`
  - `RevokeFamily`
  - `RevokeAllByUserID`
  - `TouchLastUsedAt`

- [ ] **Step 4: 再跑定向测试**

Run: `cd backend && go test ./migrations ./internal/repository`
Expected: PASS

- [ ] **Step 5: 提交本任务**

```bash
git add backend/internal/model/refresh_token.go backend/internal/repository/refresh_token_repo.go backend/migrations/init_database.sql backend/migrations/upgrade_20260331_refresh_tokens.sql backend/migrations/init_database_test.go backend/internal/repository/refresh_token_repo_test.go
git commit -m "feat: add refresh token persistence"
```

### Task 3: 实现登录、刷新、登出三条认证主链路

**Files:**
- Modify: `backend/internal/dto/request.go`
- Modify: `backend/internal/service/auth_service.go`
- Modify: `backend/internal/handler/auth_handler.go`
- Modify: `backend/cmd/server/main.go`
- Test: `backend/internal/service/auth_service_test.go`
- Test: `backend/internal/handler/auth_handler_test.go`

- [ ] **Step 1: 写登录/刷新/登出的服务层与 Handler 层失败测试**

```go
func TestLoginIssuesAccessTokenAndRefreshCookie(t *testing.T) {}
func TestRefreshRotatesRefreshToken(t *testing.T) {}
func TestLogoutRevokesCurrentRefreshToken(t *testing.T) {}
```

- [ ] **Step 2: 跑认证相关定向测试确认失败**

Run: `cd backend && go test ./internal/service ./internal/handler`
Expected: FAIL，提示缺少 refresh 接口、cookie 设置或轮换逻辑

- [ ] **Step 3: 实现后端主链路**

实现要点：
- 登录成功时：
  - 生成 access token
  - 生成 opaque refresh token 明文
  - 存储其 SHA-256 哈希
  - 设置 `HttpOnly` refresh cookie
  - 返回 `token`、`token_expires_at`、`user`
- 新增公开接口 `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout` 改为公开路由，优先撤销当前 refresh token，再清 cookie
- refresh 成功时执行 token 轮换，旧 token 标记撤销，新 token 沿用同一 `family_id`
- 若命中已轮换 token 复用，撤销整条 family

- [ ] **Step 4: 跑定向测试**

Run: `cd backend && go test ./internal/service ./internal/handler`
Expected: PASS

- [ ] **Step 5: 提交本任务**

```bash
git add backend/internal/dto/request.go backend/internal/service/auth_service.go backend/internal/handler/auth_handler.go backend/cmd/server/main.go backend/internal/service/auth_service_test.go backend/internal/handler/auth_handler_test.go
git commit -m "feat: add auth refresh and logout rotation flow"
```

### Task 4: 把改密、重置密码、禁用账号接入会话撤销

**Files:**
- Modify: `backend/internal/service/user_service.go`
- Modify: `backend/internal/handler/user_handler.go`
- Modify: `backend/internal/service/auth_service.go`
- Test: `backend/internal/service/user_service_test.go`

- [ ] **Step 1: 写“改密/重置密码/禁用后旧 refresh token 失效”的失败测试**

```go
func TestChangePasswordRevokesAllRefreshTokens(t *testing.T) {}
func TestDisableUserRevokesAllRefreshTokens(t *testing.T) {}
func TestResetPasswordRevokesAllRefreshTokens(t *testing.T) {}
```

- [ ] **Step 2: 跑用户服务定向测试确认失败**

Run: `cd backend && go test ./internal/service`
Expected: FAIL，提示相关操作后 refresh token 仍然有效

- [ ] **Step 3: 实现会话撤销**

实现要点：
- `ChangePassword` 成功后撤销该用户全部 refresh token
- `ResetShopAdminPassword` / `ResetStaffPassword` 成功后撤销目标用户全部 refresh token
- `UpdateShopAdminStatus` / `UpdateStaffStatus` 在禁用时撤销目标用户全部 refresh token
- 撤销逻辑保持幂等，避免重复执行报错

- [ ] **Step 4: 跑用户服务测试**

Run: `cd backend && go test ./internal/service`
Expected: PASS

- [ ] **Step 5: 提交本任务**

```bash
git add backend/internal/service/user_service.go backend/internal/handler/user_handler.go backend/internal/service/auth_service.go backend/internal/service/user_service_test.go
git commit -m "feat: revoke refresh tokens on credential changes"
```

### Task 5: 前端增加启动静默续期与 401 单飞重试

**Files:**
- Modify: `frontend/src/stores/user.js`
- Modify: `frontend/src/utils/request.js`
- Modify: `frontend/src/api/auth.js`
- Modify: `frontend/src/main.js`
- Modify: `frontend/src/router/index.js`

- [ ] **Step 1: 明确前端状态模型并补本地初始化逻辑**

实现要点：
- store 新增 `tokenExpiresAt`
- 登录与 refresh 成功后统一写入：
  - `token`
  - `tokenExpiresAt`
  - `user`
- 退出时清理以上字段

- [ ] **Step 2: 先实现启动初始化**

实现要点：
- 在 `main.js` 挂载前调用 `userStore.initializeAuth()`
- `initializeAuth()` 逻辑：
  - 本地无 token：直接结束
  - 本地 token 已过期或即将过期：先调 `/auth/refresh`
  - refresh 成功则恢复用户状态
  - refresh 失败则登出

- [ ] **Step 3: 改造 Axios 401 处理为单飞 refresh**

实现要点：
- 用模块级 `refreshPromise` 做单飞
- 跳过 `/auth/login`、`/auth/refresh`、`/system/logs`
- 每个原始请求只允许自动重试一次
- refresh 成功后自动重放原请求
- refresh 失败后才真正清空本地登录态并跳转 `/login`

- [ ] **Step 4: 跑前端构建检查**

Run: `cd frontend && cmd /c npm run build`
Expected: PASS

- [ ] **Step 5: 提交本任务**

```bash
git add frontend/src/stores/user.js frontend/src/utils/request.js frontend/src/api/auth.js frontend/src/main.js frontend/src/router/index.js
git commit -m "feat: add frontend silent refresh flow"
```

### Task 6: 兼容 Chrome 插件并完成最终验证与文档收口

**Files:**
- Modify: `browser-extension/ozon-shop-bridge/README.md`
- Modify: `dev-tracker/OVERALL_TASKS.md`
- Modify: `dev-tracker/CURRENT_PROGRESS.md`
- Modify: `dev-tracker/CHANGELOG.md`

- [ ] **Step 1: 核对插件兼容边界**

核对点：
- `content-auth-sync.js` 仍监听 `localStorage.token/currentShopId`
- 前端 refresh 成功会更新 `localStorage.token`
- 不新增插件独立 refresh 流程

- [ ] **Step 2: 补 README 与交付文档**

文档需要明确：
- Web 端已支持自动续期
- 插件仍依赖管理端同步 access token
- 浏览器冷启动但未打开管理端页面时，插件不保证独立续期

- [ ] **Step 3: 跑最终回归**

Run: `cd backend && go test ./...`
Expected: PASS

Run: `cd frontend && cmd /c npm run build`
Expected: PASS

- [ ] **Step 4: 做手工 smoke**

手工验证清单：
- 登录成功后收到 access token 和 refresh cookie
- 手动缩短 access token 生命周期后，业务请求触发自动 refresh 并重放成功
- 隔夜重新打开管理端时，应用先静默 refresh，再进入首页
- 退出登录后 refresh 失效
- 改密后旧 refresh 失效
- 禁用账号后旧 refresh 失效
- 打开管理端页面后，插件能同步到最新 access token

- [ ] **Step 5: 提交本任务**

```bash
git add browser-extension/ozon-shop-bridge/README.md dev-tracker/OVERALL_TASKS.md dev-tracker/CURRENT_PROGRESS.md dev-tracker/CHANGELOG.md
git commit -m "docs: document auth refresh rollout"
```

### Task 7: 发布与回滚准备

**Files:**
- Modify: `README-windows-deploy.md`
- Modify: `build-windows-release.ps1`

- [ ] **Step 1: 确认发布包包含 refresh token 相关配置与迁移脚本**

检查点：
- `upgrade_20260331_refresh_tokens.sql` 被发布包带出
- `config.yaml.example` 包含新 JWT 配置项

- [ ] **Step 2: 更新部署说明**

部署说明需要明确：
- 新环境执行 `init_database.sql`
- 老环境执行 `upgrade_20260331_refresh_tokens.sql`
- 生产环境 refresh cookie 必须启用 `Secure`

- [ ] **Step 3: 再跑一次打包/构建链路（按需要）**

Run: `powershell -ExecutionPolicy Bypass -File .\build-windows-release.ps1`
Expected: PASS，并包含最新迁移脚本与配置模板

- [ ] **Step 4: 提交本任务**

```bash
git add README-windows-deploy.md build-windows-release.ps1
git commit -m "docs: update release guidance for refresh token auth"
```

## 备注

1. 本计划默认不引入新的前端测试框架；前端以构建校验和手工 smoke 为主。
2. 若后续要让 Chrome 插件完全脱离管理端页面独立续期，应另起一个子任务，单独设计 extension 侧 refresh 通道。
3. 实施时必须同步维护 `backend/migrations/init_database.sql`、版本化增量 SQL 和 `dev-tracker` 三份追踪文档。
