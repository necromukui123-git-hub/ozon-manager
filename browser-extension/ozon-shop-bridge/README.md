# Ozon Manager Shop Bridge (测试版)

这是一个 Chrome 扩展（Manifest V3），用于在用户**当前浏览器登录态**下执行店铺促销任务。

## 目标行为

- 默认静默执行：用户留在你的系统页面，不跳转到 Ozon。
- 仅在未登录 Ozon Seller 时，自动打开登录标签页。
- 登录后自动继续执行任务。
- 使用插件专用工作标签页，避免打断用户正在浏览的 Ozon 页面。

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

1. 在你的管理系统中登录（扩展会尝试自动同步 token 和 `currentShopId`）。
2. 点击扩展图标，确认：
   - `后端地址`（默认 `http://127.0.0.1:8080`）
   - `店铺 ID`
   - `Token`（自动同步失败时可手填）
3. 保持“启用轮询”打开，扩展会自动处理店铺促销任务。

> 当前默认自动同步脚本仅匹配 `localhost/127.0.0.1`。
> 如果你的前端部署在正式域名，需要在 `manifest.json` 的 `content_scripts.matches` 和 `host_permissions` 补充域名后重新加载扩展。

## 当前支持任务类型

- `sync_shop_actions`
- `sync_action_products`
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
