# Ozon Manager 开发总任务

最后更新时间：2026-03-11  
负责人：团队 + Codex  
范围：统一官方促销与店铺促销的一套业务流程，并让店铺促销在浏览器登录态下低打扰执行。

## 项目目标
1. 用户在一个流程中完成官方促销和店铺促销操作。
2. 店铺促销优先静默执行，仅在必要时触发登录兜底。
3. 异步执行任务可追踪、可回放、可定位失败原因。

## 已锁定架构
1. 官方促销由后端直连官方 API。
2. 店铺促销由浏览器插件执行。
3. 后端负责编排、任务队列、鉴权、回报汇总。
4. 前端负责统一入口、统一状态呈现。

## 数据库迁移策略（已锁定）
1. `backend/migrations/init_database.sql` 维护全量基线，必须始终可一键初始化。
2. 增量升级使用 `backend/migrations/upgrade_YYYYMMDD_<topic>.sql`。
3. 不再维护 `backend/migrations/upgrade_standalone.sql`，避免误执行和历史分叉。

## 任务看板
| ID | 任务 | 状态 | 验收标准 | 依赖/风险 |
|---|---|---|---|---|
| T1 | Chrome 商店上架清单与隐私文案 | todo | 完成商店素材、权限说明、隐私政策、审核说明 | 需确认最终权限与数据流 |
| T2 | 真实环境联调回归（多店铺 + extension/agent 混合在线） | in_progress | 完成长时间轮询回归并输出异常清单 | 依赖测试环境和账号稳定性 |
| T3 | 执行引擎路由监控指标 | todo | 可查看领取来源、agent fallback 次数、冲突阻断次数 | 需确定监控落点与告警阈值 |
| T4 | 店铺活动字段补全与日期显示修复 | done | 同步后活动卡片显示开始/结束日期，并可查看预算/折扣/可编辑状态等关键字段 | 依赖 Seller 返回字段稳定 |
| T5 | 店铺活动商品详情页信息重构（图片+双语+价格结构） | done | 活动商品页可展示图片、双行名称、双SKU、折扣/库存结构，支持关键词与状态筛选 | 依赖 Seller `active/candidate` 字段稳定 |
| T6 | 店铺活动日期与商品详情展示缺口收敛修复 | done | 活动列表日期不再空白；活动商品页显示缩略图/卖家库存/item_type 且三类编号标签清晰 | 依赖 Seller `v2 active` 端点可用 |
| T7 | 官方活动商品查询接口 `last_id` 对齐修复 | done | 官方活动商品页可正常展示活动内商品，查询链路兼容 `id/product_id` 并带 `Language` 请求头 | 依赖 Ozon 官方 `/v1/actions/products` 返回游标稳定 |
| T8 | 插件“保存并立即同步一次”反馈语义修复 | done | 点击保存后可明确看到“同步成功/跳过/失败原因”，token 过期时有重新登录提示 | 依赖插件 popup 与 background 消息结构一致 |
| T9 | 插件“有任务但失败仍显示成功”提示修复 | done | `sync.status=failed` 时 popup 首行显示失败并附原因，不再误报成功 | 依赖插件执行结果回传字段完整 |
| T10 | 新增 Ozon 实时商品列表页面（缓存+后台刷新） | done | 新增 `/products/ozon` 页面，支持可见性/ID/上架日期筛选、日期来源标记、游标翻页与手动刷新；后端新增 `ozon-catalog` 查询与刷新接口 | 依赖 Seller v3 商品列表/详情/库存接口稳定 |
| T11 | 商品列表“同步商品”404修复与失败日志可观测性补全 | done | 点击“同步商品”可成功拉取 Ozon 商品；系统日志/操作日志可直接查看失败原因 | 依赖 Seller v3 商品接口稳定 |
| T12 | Ozon 商品接口标准说明文档沉淀（`/v3/product/list` + `/v3/product/info/list`） | done | 在 `doc/` 产出工程可用版接口文档，覆盖鉴权、请求参数、分页、响应结构、错误处理与示例 | 依赖 Ozon 官方文档持续更新，需定期回看 |
| T13 | `/v3/product/list` 响应字段对齐与目录可见性推导修复 | done | 客户端可解析 `has_fbo_stocks/has_fbs_stocks/archived/is_discounted/quants`；目录刷新不再依赖 list 响应 `visibility`，改为优先 `info.visible`，其次 `archived`，最后 `ALL` | 依赖 Seller v3 列表/详情字段稳定 |
| T14 | 官方促销接口标准说明文档沉淀（`/v1/actions/candidates` + `/v1/actions/products/activate`） | done | 在 `doc/` 产出一份人读版 Markdown 与一份机读版 YAML，覆盖鉴权、分页、请求/响应结构、示例与弃用说明 | 依赖 Ozon 官方文档持续更新，需定期回看 |
| T15 | 自动添加商品至促销活动（配置 + 调度 + 执行历史） | done | 新增“自动加促销”页面；支持保存绝对日期配置、手动执行、官方/店铺活动候选刷新、按上架日期筛商品、执行历史与逐商品失败明细 | 依赖 Ozon 目录刷新、官方候选接口与浏览器插件执行链稳定 |

## 近期完成里程碑（已完成）
1. 按店铺执行引擎模式（`auto`/`extension`/`agent`）已落地。
2. 任务领取路由已防止 agent 与 extension 抢同类任务。
3. extension 回报已绑定 `assigned_agent_id` 做归属校验。
4. 系统概览已增加插件状态面板（在线/离线/最近任务/错误）。
5. 非 localhost 自动同步已升级为白名单与按需授权策略。
6. 后端路由与归属关键逻辑已有 service 层测试覆盖。
7. 店铺活动同步已补齐 `actionParameters` 字段映射，活动日期与运营关键指标可在系统中展示。
8. 店铺活动商品详情页已升级为信息增强版本，支持图片、双语名称、双SKU、价格/库存结构化展示与筛选查询。
9. 店铺活动商品同步已切换为优先抓取 `v2 active` 数据，活动详情页可稳定展示缩略图、卖家库存与中文类目，并固定 Offer/SKU/Product 三行编号。
10. 官方活动商品查询已切换 `last_id` 游标分页，并补齐 `Language: ZH_HANS` 请求头与 `id/product_id` 兼容映射，官方活动商品页可稳定出数。
11. 插件“保存并立即同步一次”已可回传本次同步结果，避免“仅显示保存成功”导致的状态误判。
12. 插件已修复“有任务但执行失败仍显示成功”的提示误判，失败会在首行明确展示原因。
13. 新增 Ozon 实时商品列表能力：后端新增 `ozon_product_catalog_items` 缓存与刷新链路（`/v3/product/list` + `/v3/product/info/list` + `/v3/product/info/stocks`），前端新增 `Ozon 商品列表` 菜单与页面，支持上架日期来源区分（`ozon`/`local_sync`）。
14. 商品同步链路已从 Seller 旧版 `/v2/product/list` 切换到 `/v3/product/list` + `/v3/product/info/list`，并补齐前后端失败日志字段，便于快速定位 404/5xx 根因。
15. 已新增 `doc/ozon-seller-product-apis-v3-list-info.md`，沉淀 `/v3/product/list` 与 `/v3/product/info/list` 的标准工程说明（含调用流程、示例与排障要点）。
16. 商品同步链路已补齐 `/v3/product/info/list` 响应兼容（`items/result.items`）与失败语义收敛：批次失败不再假成功，且先落基础数据避免整表为空。
17. `/v3/product/list` 客户端响应结构已对齐实测字段（含 `has_fbo_stocks/has_fbs_stocks/archived/is_discounted/quants`），目录缓存可见性改为“优先 `info.visible`，其次 `archived`，最后 `ALL`”。
18. 已新增 `doc/ozon-promos-candidates-activate-standard.md` 与 `doc/ozon-promos-candidates-activate.openapi.yaml`，沉淀官方促销候选商品查询与商品加入促销两个接口的标准说明，明确 `last_id` 分页替代 `offset`，并补齐 `result.rejected` 结构。
19. 已新增“自动加促销”完整链路：后端新增配置/运行历史/候选缓存表与调度器，插件支持 `sync_action_candidates` 任务，前端新增 `/promotions/auto-add` 页面，支持保存配置、手动执行和逐商品历史查看。

## 阶段完成标准
1. 官方与店铺促销在统一 UX 下稳定可用。
2. 常规场景店铺任务静默执行，不打断用户主流程。
3. 登录兜底仅在未登录 Seller 时触发。
4. 异步任务全链路可追踪，失败可定位。
