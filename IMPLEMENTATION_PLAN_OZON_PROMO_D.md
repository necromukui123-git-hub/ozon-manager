# Ozon 店铺促销自动化实施方案（路线 D）

## 1. 方案概览

目标是在现有 Web 系统上实现 `退出促销 -> 改价 -> 重加促销` 的批量编排能力。

采用架构：`本机保会话 + 后端派单`。

- 本机 Agent 持有浏览器登录态（人工登录）
- 后端负责任务下发、状态机、审计和结果归档
- 优先使用官方 API，可用能力不足时由网页动作补位

## 2. 边界与约束

- 不实现绕过验证码/风控/安全校验的机制
- 不在服务端托管完整浏览器会话明文
- 不保证私有页面接口长期稳定，通过动作适配层降低维护成本

## 3. 总体架构

- `frontend`：创建任务、查询进度、人工确认、失败重跑
- `backend`：任务模型、状态机、调度、审计
- `agent`：拉取任务、执行网页动作、回传分步结果

## 4. 后端 API 规划

- `POST /api/v1/automation/jobs`：创建任务
- `GET /api/v1/automation/jobs`：任务列表
- `GET /api/v1/automation/jobs/:id`：任务详情
- `POST /api/v1/automation/jobs/:id/confirm`：人工确认继续
- `POST /api/v1/automation/jobs/:id/cancel`：取消任务
- `POST /api/v1/automation/agent/heartbeat`：Agent 心跳
- `POST /api/v1/automation/agent/poll`：Agent 拉取任务
- `POST /api/v1/automation/agent/report`：Agent 回传结果

## 5. 数据模型规划

- `automation_jobs`：任务主表
- `automation_job_items`：商品粒度步骤状态
- `automation_agents`：执行节点与心跳
- `automation_job_events`：任务事件流
- `automation_artifacts`：截图/HAR 摘要索引

## 6. 执行链路

1. 创建任务并预检（去重、参数校验、风险分级）
2. 进入 `pending/await_confirm`
3. Agent 拉取并按限速执行三步动作
4. 实时回传步骤结果并推进状态机
5. 失败项可导出并支持按步骤重跑

## 7. 风控与可观测

- 灰度执行（小批量先跑）
- 店铺级限速（动作维度节流）
- 关键阈值触发人工确认
- 全链路审计（发起人、节点、错误码、证据摘要）

## 8. 测试与验收

- 单元测试：状态流转、幂等、重试分类
- 集成测试：`create -> poll -> report -> done`
- UAT：50/200/500 商品分档验证

## 9. 实施迭代

- **M1**：任务模型 + API + dry-run 骨架
- **M2**：Agent 心跳/拉取/回传 + 三步动作骨架
- **M3**：人工确认 + 失败重跑 + Excel 批次打通
- **M4**：审计完善 + 告警 + 运维文档

## 10. M1 范围（本次开始实现）

- 新增任务相关数据库表（最小闭环）
- 新增创建/查询任务 API
- 支持 `dry_run` 路径（不触发真实动作，仅落库与状态演进）
- 保证按店铺权限隔离访问
