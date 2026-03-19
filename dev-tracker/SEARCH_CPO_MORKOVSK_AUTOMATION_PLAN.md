# Search CPO 状态迁移自动化计划

## Summary
- 当前 Search CPO 页面已经明确分成两条链路：`人工批量报名` 与 `状态迁移自动化`。
- 自动化规则已收口为：`state1` 初始报名，`state2` 主触发迁移，`state3_trigger` 只作为历史漏跑兜底。
- 自动化迁移后半段固定执行：全量同步活动清单、退出命中活动、`enable`、加入 `Morkovsk`。
- 当前剩余重点是继续在真实 Seller 环境验证活动同步耗时、退出结果和长时间调度表现。

## 已确认的产品决策
- 继续沿用 Search CPO 页面里保存的默认官方/店铺活动作为 `state1` 初始活动集合。
- `Morkovsk` 仍是固定目标，不走普通活动 ID 配置。
- 自动化任务同时支持手动执行一次和定时调度。
- `enable` 不再是自动化可关闭开关；接口字段保留兼容，但服务端固定按 `true` 执行。
- 文档拆成两份：人工批量报名单独说明，状态迁移自动化单独说明；页面不拆。

## 目标状态定义
- `state1`
  - `searchPromoStatus = SEARCH_PROMO_STATUS_DISABLED`
  - `carrotsStatus = CARROTS_STATUS_DISABLED`
  - `availability_promo = false`
- `state2`
  - `searchPromoStatus = SEARCH_PROMO_STATUS_DISABLED`
  - `carrotsStatus = CARROTS_STATUS_DISABLED`
  - `availability_promo = true`
- `state3_trigger`
  - 商品曾进入过 `state2`
  - 当前刷新时 `searchPromoStatus = SEARCH_PROMO_STATUS_ENABLED`
  - 仅作为历史漏跑后的兜底状态
- `morkovsk_joined`
  - 已成功加入 `Morkovsk`

## 当前实现口径
1. `state1`
   - 只执行“加入已保存默认活动”。
   - 不在该阶段执行 `enable`。
2. `state2`
   - 先执行官方+店铺活动全量同步。
   - 再刷新活动商品成员关系，按 SKU 找出当前命中的全部官方/店铺活动。
   - 退出这些命中的活动后，执行 `enable`，最后执行 `carrots/batch_enable` 加入 `Morkovsk`。
3. `state3_trigger`
   - 复用与 `state2` 相同的迁移后半段。
   - 用于修正历史中断、漏跑或延迟调度导致的已 enabled 商品。
4. 退出链路
   - 若店铺活动退出返回 `skipped`（当前商品不在活动中），不阻断后续 `enable` 和 `Morkovsk`。
   - 若活动清单同步失败、活动商品刷新失败、真实退出失败、`enable` 失败或 `batch_enable` 失败，都按逐商品结果记录。

## 测试与联调要求
- 后端：`cd backend && $env:GOCACHE="$env:TEMP\ozon-manager-gocache"; go test ./...`
- 前端：`cd frontend && cmd /c npm run build`
- 插件：`node --check browser-extension/ozon-shop-bridge/background_search_cpo.js`
- 真实联调重点：
  - `state1` 商品可进入默认活动但不会提前 `enable`
  - `state2` 商品会立即进入“同步活动 -> 退出 -> enable -> Morkovsk”链路
  - `state3_trigger` 仍可兜底收敛历史漏跑商品
  - “当前商品不在店铺活动中”的 remove 结果不会误阻断后续步骤
