# Ozon Automation Agent

该 Agent 用于在本机执行自动化任务，支持两种模式：

- `mock`：仅验证后端通信闭环
- `playwright`：使用持久化浏览器会话执行真实网页动作

## 1. 目录

- `agent.js`：主程序
- `executors/mock-executor.js`：模拟执行器
- `executors/playwright-executor.js`：浏览器执行器
- `flows/ozon-action-flow.js`：动作流程模板
- `flows/flow-settings.js`：流程配置加载器
- `config/ozon-flow.config.example.json`：选择器配置模板
- `.env.example`：环境变量示例

## 2. 安装

```bash
cd agent
npm init -y
npm install axios dotenv playwright
copy .env.example .env
```

## 3. 配置

关键参数：

- `BASE_URL`：后端地址
- `AGENT_KEY`：Agent 唯一标识
- `AGENT_MODE`：`mock` 或 `playwright`
- `BROWSER_USER_DATA_DIR`：持久化浏览器目录
- `OZON_FLOW_CONFIG_PATH`：动作配置 JSON 路径

## 4. 运行

```bash
node agent.js
```

## 5. Playwright 模式（新手步骤）

1. `.env` 设置 `AGENT_MODE=playwright`
2. 首次运行 `node agent.js`
3. 程序会创建持久化浏览器目录（`BROWSER_USER_DATA_DIR`）
4. 打开的浏览器中手动登录 Ozon Seller
5. 保持该 profile，后续任务自动复用登录态

## 6. 重要说明

- 当前已实现“真实动作尝试版”：按 `config/ozon-flow.config.example.json` 的选择器配置执行三步动作。
- 若页面结构与默认配置不一致，请复制该配置为本地文件并调整选择器，然后在 `.env` 里改 `OZON_FLOW_CONFIG_PATH`。
- 失败会自动截图并写入 `ARTIFACT_DIR`，错误信息里会附截图路径。

### 调整选择器建议

1. 在 Playwright 模式下先只跑 1 条商品任务
2. 失败后查看 `artifacts` 截图
3. 修改配置中的 `*Selectors` 数组（按优先级从上到下匹配）
4. 重启 Agent 再试

## 7. 安全建议

- Agent 只部署在你可控机器
- 不上传明文 cookie 到后端
- 生产环境建议给 Agent 接口加签名校验
