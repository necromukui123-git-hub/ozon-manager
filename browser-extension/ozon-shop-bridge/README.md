# Ozon Manager Shop Bridge (测试版)

这是一个 Chrome 扩展（Manifest V3），用于在用户**当前浏览器登录态**下执行店铺促销任务。

## 目标行为

- 默认静默执行：用户留在你的系统页面，不跳转到 Ozon。
- 仅在未登录 Ozon Seller 时，自动打开登录标签页。
- 登录后自动继续执行任务。
- 使用插件专用工作标签页，避免打断用户正在浏览的 Ozon 页面。
- 默认自动连接管理端登录态，不要求普通用户复制 token。

## 与后端接口

扩展会调用：

- `POST /api/v1/extension/register`
- `POST /api/v1/extension/poll`
- `POST /api/v1/extension/report`

鉴权方式：`Authorization: Bearer <token>`，token 默认从你的前端页面 `localStorage.token` 自动同步。

## 开发安装（测试阶段）

1. 打开 Chrome 扩展管理页：`chrome://extensions/`
2. 开启“开发者模式”
3. 选择“加载已解压的扩展程序”
4. 选择本目录 `browser-extension/ozon-shop-bridge`

## 使用说明

1. 在你的管理系统中登录，并先选择当前店铺。
2. 点击扩展图标，确认：
   - `后端地址`（默认 `http://127.0.0.1:8080`）
   - `启用轮询` 已开启
3. 插件会自动检测当前管理端页面并同步 `token/currentShopId`。
4. 如果使用的是非 localhost 域名，首次点击“重新检测”时按提示授权该管理端页面。
5. 只有自动连接失败时，才进入“高级设置 / 排障”填写 `管理端 Origin`、`店铺 ID` 或 `Token` 作为兜底。

> 自动同步策略：默认匹配 `localhost/127.0.0.1`，并从“后端地址”推导同源管理端。
> 若管理端与后端不同源，请在 popup 的“高级设置 / 排障”里填写“管理端 Origin”，然后点击“重新检测”并授权该域名。

## popup 状态说明

- `已连接管理端，可自动执行任务`：已成功读取管理端登录态与当前店铺。
- `请先在管理端登录`：当前打开的管理端页面没有登录态。
- `请先在管理端选择店铺`：已登录，但 `currentShopId` 为空。
- `需要授权访问管理端页面`：首次接入非 localhost 管理端，需要用户点击“重新检测”时授权。
- `未检测到已打开的管理端页面`：请先把管理端页面打开到当前浏览器中，再重新检测。

## 当前支持任务类型

- `sync_shop_actions`
- `sync_action_candidates`
- `sync_action_products`
- `sync_search_cpo_products`
- `sync_search_cpo_availability`
- `search_cpo_enable_products`
- `search_cpo_batch_enable_morkovsk`
- `shop_action_declare`
- `shop_action_remove`
- `promo_unified_enroll`
- `promo_unified_remove`
- `remove_reprice_readd`（店铺活动退出 -> 后端改价 -> 店铺活动重新报名）

> 注意：若同时运行旧 Agent 与本扩展，两者都可能领取同一类待执行任务。测试阶段建议优先只启用一种执行器。

## 打包（发测试安装包）

在 PowerShell 中执行：

```powershell
cd browser-extension/ozon-shop-bridge/scripts
.\package.ps1
```

默认输出到 `browser-extension/ozon-shop-bridge/dist/ozon-shop-bridge-v<version>.zip`。

