# Automation Runbook（M4）

本手册描述自动化任务系统的日常运维与排障。

## 1. 核心接口

- 任务创建：`POST /api/v1/automation/jobs`
- 任务列表：`GET /api/v1/automation/jobs`
- 任务详情：`GET /api/v1/automation/jobs/:id`
- 任务确认：`POST /api/v1/automation/jobs/:id/confirm`
- 任务取消：`POST /api/v1/automation/jobs/:id/cancel`
- 失败重跑：`POST /api/v1/automation/jobs/:id/retry-failed`
- 事件查询：`GET /api/v1/automation/events`
- Agent 状态：`GET /api/v1/automation/agents`

Agent 通道（免登录）

- 心跳：`POST /api/v1/automation/agent/heartbeat`
- 拉任务：`POST /api/v1/automation/agent/poll`
- 回报：`POST /api/v1/automation/agent/report`

## 2. 状态机说明

任务状态：

- `pending`：等待被 Agent 拉取
- `await_confirm`：等待人工确认
- `running`：执行中
- `success`：全部成功
- `partial_success`：部分失败
- `failed`：全部失败
- `canceled`：已取消
- `dry_run_completed`：演练完成

## 3. Agent 在线判定

- Agent 每次 `heartbeat` 会刷新 `last_heartbeat_at`
- 超过 90 秒无心跳，视为 `offline`

## 4. 常见故障排查

### 4.1 Agent 一直拿不到任务

- 检查任务是否为 `pending`
- 检查是否 `dry_run=true`（dry-run 不派发）
- 检查 Agent Key 是否一致

### 4.2 任务长时间 `running`

- 检查 Agent 是否崩溃或断网
- 查看 `automation_job_events` 是否有 `job_assigned` 但无 `job_reported`
- 可人工 `cancel` 后 `retry-failed`

### 4.3 任务无法 `retry-failed`

- 仅 `failed` / `partial_success` 允许重跑
- 必须存在 `overall_status=failed` 的任务项

## 5. 安全建议

- Agent 通道建议加签名或白名单
- 不上传明文登录态到后端
- 生产环境建议将 Agent 与业务网络隔离

## 6. 升级建议

- M2 当前为最小闭环，后续可接 Playwright 真实动作
- 增加告警渠道（飞书/钉钉/邮件）
- 增加任务超时自动回收策略

## 7. Agent 模式

- `mock`：仅打通 `heartbeat/poll/report` 协议，不执行真实动作
- `playwright`：启用本机浏览器持久会话，执行网页动作模板

模板文件：`agent/flows/ozon-action-flow.js`

建议流程：

1. 先用 `mock` 跑通接口
2. 再切到 `playwright` 并手工登录
3. 逐步填充真实动作选择器并灰度上线
