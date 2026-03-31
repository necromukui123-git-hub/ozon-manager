# Access Token + Refresh Token 认证改造设计

## 背景

当前系统只使用单个 JWT 作为登录态凭证，实际有效期为 24 小时。前端将该 token 持久化到 `localStorage`，后端在 token 过期后直接返回 `401`，前端随即清理本地登录态并跳回登录页。  
这会导致用户在“前一天登录，第二天继续打开系统”时，经常在首次请求时被提示掉线。

同时，Chrome 插件当前默认从管理端页面同步 `localStorage.token` 与 `localStorage.currentShopId`，因此认证改造不能直接把 Web 端 access token 完全移出前端存储，否则会破坏现有插件自动连接流程。

## 目标

1. Web 端支持 access token 自动续期，尽量避免用户隔夜重新打开系统时因 access token 过期直接掉线。
2. 认证方案向商用环境靠拢，避免把长期有效凭证暴露给前端 JavaScript。
3. 支持服务端撤销 refresh token，会话在登出、改密、重置密码、账号禁用后可立即失效。
4. 尽量最小化本轮改动范围，优先完成 Web 端闭环，并保持现有 Chrome 插件 access token 同步链路兼容。

## 非目标

1. 本轮不把整个系统切换成纯 Cookie Session / BFF 架构。
2. 本轮不为 Chrome 插件单独实现“插件自身独立 refresh token 自动续期”。
3. 本轮不引入 Redis、集中式 Session 服务或多设备会话管理界面。

## 备选方案对比

### 方案 A：access token + refresh token 都放 `localStorage`

- 优点：实现最快，前后端改动最少。
- 缺点：长期凭证暴露给前端脚本，XSS 风险高，不符合偏商用的安全预期。
- 结论：不采用。

### 方案 B：access token 走 Bearer，refresh token 走 `HttpOnly` Cookie，服务端持久化 refresh token

- 优点：兼顾安全性和现有架构兼容性；Chrome 插件仍可继续读取管理端同步下来的 access token。
- 缺点：仍需暂时保留 access token 在前端可读存储中；插件本身不会自动使用 refresh cookie。
- 结论：本轮采用。

### 方案 C：纯 Cookie Session / BFF

- 优点：更接近传统企业 Web 应用的认证边界，前端几乎不接触 token。
- 缺点：需要重做插件鉴权模型、前端请求模型与开发代理，超出本轮最小改动范围。
- 结论：作为后续可选演进方向，不在本轮实施。

## 选定方案

### 总体思路

系统改为“双令牌模型”：

1. `access token`
   - 类型：JWT
   - 用途：访问业务 API
   - 有效期：默认 12 小时
   - 存储：前端 `localStorage`
   - 发送方式：`Authorization: Bearer <token>`

2. `refresh token`
   - 类型：随机不透明字符串，不使用 JWT
   - 用途：在 access token 过期或即将过期时换取新的 access token
   - 有效期：默认 30 天
   - 存储：浏览器 `HttpOnly` Cookie
   - 服务端存储：仅保存 refresh token 的 SHA-256 哈希，不保存明文

### Cookie 策略

- Cookie 名称：`ozon_refresh_token`
- `HttpOnly=true`
- `SameSite=Lax`
- `Secure`
  - 生产环境：`true`
  - 本地 HTTP 开发环境：`false`
- `Path=/api/v1/auth`
- 不主动设置 `Domain`，采用 host-only cookie，降低跨子域影响面

### Refresh Token 轮换策略

每次调用 `/api/v1/auth/refresh` 成功后：

1. 校验当前 refresh token 是否存在、未撤销、未过期、所属用户仍有效。
2. 撤销当前 refresh token 记录。
3. 生成新的 refresh token 与新的 access token。
4. 以新 refresh token 覆盖旧 cookie。
5. 将新的 access token 返回给前端。

### Refresh Token 家族与复用检测

refresh token 表保留 `family_id`，同一次登录链路轮换产生的一串 refresh token 归属同一个 family。

处理原则：

1. 正常刷新：撤销旧 token，签发新 token，family 不变。
2. 若发现一个已被轮换撤销的 refresh token 再次被使用，视为 refresh token 复用风险。
3. 一旦检测到复用，撤销该 family 下所有仍有效的 refresh token，并要求重新登录。

这样能覆盖 token 泄露后被重放的典型场景。

## 服务端设计

### 数据结构

新增表：`user_refresh_tokens`

建议字段：

- `id`
- `user_id`
- `token_hash`
- `family_id`
- `issued_at`
- `expires_at`
- `last_used_at`
- `revoked_at`
- `revoke_reason`
- `replaced_by_token_id`
- `user_agent`
- `ip_address`
- `created_at`
- `updated_at`

建议索引：

- `UNIQUE(token_hash)`
- `INDEX(user_id, revoked_at)`
- `INDEX(family_id)`
- `INDEX(expires_at)`

### Access Token Claims

access token JWT 增加明确类型字段，避免把其它 token 误判为 access token：

- `user_id`
- `username`
- `display_name`
- `role`
- `token_type=access`
- `sid` 或 `session_id`（可选，用于排障和日志）
- 标准 claims：`iat`、`nbf`、`exp`、`iss`

### 配置项

当前 `jwt.expire_hours` 不够表达新模型，建议扩展为：

- `jwt.secret`
- `jwt.access_expire_minutes`
- `jwt.refresh_expire_hours`
- `jwt.refresh_cookie_name`
- `jwt.refresh_cookie_secure`

如需最小改动，也可保留 `expire_hours` 作为兼容字段，但新代码只消费新的 access/refresh 配置项。

### 接口契约

#### `POST /api/v1/auth/login`

行为：

1. 校验用户名、密码、账号状态。
2. 签发 access token。
3. 创建 refresh token 记录。
4. 设置 refresh token Cookie。
5. 返回 access token 与用户信息。

响应体建议：

```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "token": "access-token",
    "token_expires_at": "2026-03-31T23:59:59+08:00",
    "user": {
      "id": 1,
      "username": "super_admin",
      "display_name": "系统管理员",
      "role": "super_admin",
      "status": "active",
      "shops": []
    }
  }
}
```

#### `POST /api/v1/auth/refresh`

行为：

1. 从 `HttpOnly` Cookie 读取 refresh token。
2. 校验 refresh token。
3. 成功则轮换 refresh token，返回新的 access token。
4. 失败则清理 refresh token Cookie，并返回 `401`。

响应体与登录接口保持同形状，便于前端复用：

```json
{
  "code": 200,
  "message": "刷新成功",
  "data": {
    "token": "new-access-token",
    "token_expires_at": "2026-04-01T11:59:59+08:00",
    "user": {
      "id": 1,
      "username": "super_admin",
      "display_name": "系统管理员",
      "role": "super_admin",
      "status": "active",
      "shops": []
    }
  }
}
```

#### `POST /api/v1/auth/logout`

行为：

1. 不再强依赖 access token 鉴权。
2. 若请求中携带 refresh token Cookie，则撤销对应 refresh token。
3. 清理 refresh token Cookie。
4. 返回幂等成功。

这样即使 access token 已过期，用户点击退出登录也能正常撤销会话。

#### `GET /api/v1/auth/me`

保持现状，继续要求有效 access token。

### 会话撤销触发点

以下场景需要撤销 refresh token：

1. 用户主动退出登录：撤销当前 refresh token。
2. 用户修改自己的密码：撤销该用户全部 refresh token。
3. 管理员重置店铺管理员/员工密码：撤销目标用户全部 refresh token。
4. 管理员禁用账号：撤销目标用户全部 refresh token。
5. 检测到 refresh token 复用：撤销对应 family 下全部 refresh token。

## 前端设计

### Token 存储策略

本轮保持：

- `localStorage.token`
- `localStorage.user`
- `localStorage.currentShopId`

新增：

- `localStorage.tokenExpiresAt`

原因：

1. 现有路由守卫和请求拦截器已依赖 `localStorage.token`。
2. Chrome 插件现有自动连接链路也依赖它。
3. 本轮优先解决“掉线体验”和服务端安全边界，不同步做插件鉴权重构。

### 应用启动流程

前端启动时不再只靠“本地是否有 token”判断登录态，而是增加初始化流程：

1. 若本地没有 access token，按未登录处理。
2. 若本地存在 access token，但 `tokenExpiresAt` 已过期或即将过期，则优先调用 `/auth/refresh`。
3. `/auth/refresh` 成功：
   - 更新 `localStorage.token`
   - 更新 `localStorage.tokenExpiresAt`
   - 更新 `localStorage.user`
4. `/auth/refresh` 失败：
   - 清理本地登录态
   - 跳转登录页

这样可以把“第二天重新打开系统，第一下操作才提示掉线”改成“应用初始化时先静默续期”。

### 401 自动续期

Axios 响应拦截器改为：

1. 业务请求收到 `401` 时，不立刻清空本地状态。
2. 若该请求尚未重试过，走单飞 refresh 流程：
   - 若已有 refresh 中请求在执行，则等待它完成。
   - 若没有，则发起一次 `/auth/refresh`。
3. refresh 成功后，重放原始请求一次。
4. refresh 失败后，再统一清理本地登录态并跳登录页。

必须避免：

1. `/auth/login`、`/auth/refresh` 自身进入 refresh 死循环。
2. 多个并发请求同时 `401` 时并发发起多次 refresh。

### 用户信息同步

`/auth/refresh` 响应体直接回传 `user`，这样前端在 refresh 成功后即可一次性同步：

- `token`
- `token_expires_at`
- `user`

无需额外再打一次 `/auth/me` 才能恢复界面。

## Chrome 插件兼容边界

本轮保持兼容，不单独为插件实现 refresh token。

兼容方式：

1. Web 管理端 refresh 成功后仍会更新 `localStorage.token`。
2. 插件的 `content-auth-sync.js` 会继续把最新 access token 同步到插件侧。
3. 插件原有轮询、注册、上报接口继续使用 Bearer access token。

已知边界：

1. 如果浏览器重新打开后，用户未先打开管理端页面，插件可能仍持有旧 access token。
2. 这属于“插件独立续期能力”缺口，不纳入本轮。
3. 本轮先保证 Web 端主路径和“管理端打开后插件恢复同步”的兼容性。

## 安全与运维考虑

1. 服务端只保存 refresh token 哈希，避免数据库泄露时明文长期凭证被直接使用。
2. refresh token 轮换后旧 token 立即失效，降低长期重放风险。
3. 改密、重置密码、禁用用户时撤销会话，确保账号状态变化能真正落到在线会话。
4. 生产环境 refresh cookie 必须 `Secure=true`，本地开发环境允许通过配置关闭。
5. 后端日志中可记录 refresh token 的 `family_id`、`session_id`、IP、UA，但不要记录明文 token。

## 迁移与发布策略

1. 新增增量脚本：`backend/migrations/upgrade_YYYYMMDD_refresh_tokens.sql`
2. 同步回写：`backend/migrations/init_database.sql`
3. 配置文件：
   - 更新 `backend/config/config.yaml.example`
   - 本地开发环境同步更新 `backend/config/config.yaml`
4. 前端构建与后端测试通过后，再做手工回归：
   - 正常登录
   - 强制缩短 access token 生命周期验证 refresh
   - 退出登录后 refresh 失效
   - 改密后旧 refresh 失效
   - 禁用账号后旧 refresh 失效

## 本轮实施边界结论

本轮按以下范围实施：

1. Web 端完整落地 access token + refresh token 自动续期。
2. Chrome 插件保持“读取管理端同步的 access token”兼容模式。
3. 不在本轮扩展插件独立 refresh token。

这是当前仓库里最接近商用实践、同时又不把插件链路整体推翻的折中方案。
