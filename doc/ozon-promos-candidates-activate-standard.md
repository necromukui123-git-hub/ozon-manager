# Ozon Seller API 标准接口文档（`/v1/actions/candidates` + `/v1/actions/products/activate`）

## 1. 文档元信息
- 来源页面 1: https://docs.ozon.ru/api/seller/zh/#operation/PromosCandidates
- 来源页面 2: https://docs.ozon.ru/api/seller/zh/#operation/PromosProductsActivate
- 核验时间: 2026-03-11 +08:00
- 服务地址: `https://api-seller.ozon.ru`
- 文档范围:
  - `POST /v1/actions/candidates`
  - `POST /v1/actions/products/activate`
- 说明:
  - 本文档以 Ozon Seller API 中文官方页面为主。
  - `result.rejected[].product_id` 与 `result.rejected[].reason` 为本次补充核实后的明确字段。
  - 对官方页面未逐字段展开解释的响应字段，本文档仅按字段名和页面上下文做最小语义归纳，不扩展未公开规则。

## 2. 鉴权与公共头
两个接口均为 `POST`，统一使用以下请求头：

| Header | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `Client-Id` | `string` | 是 | 用户识别号。 |
| `Api-Key` | `string` | 是 | API 密钥。 |

## 3. 接口总览
| 接口 | OperationId | 用途 | 主要请求体字段 | 主要响应字段 | 备注 |
|---|---|---|---|---|---|
| `POST /v1/actions/candidates` | `PromosCandidates` | 查询指定促销活动下可参与促销的商品清单 | `action_id`, `limit`, `last_id`, `offset(弃用)` | `result.products`, `result.total`, `result.last_id` | 自 2025-05-05 起应使用 `last_id` 分页 |
| `POST /v1/actions/products/activate` | `PromosProductsActivate` | 向已有促销活动中添加商品 | `action_id`, `products[]` | `result.product_ids`, `result.rejected[]` | `products` 最多 1000 项 |

## 4. 接口一：`POST /v1/actions/candidates`
- OperationId: `PromosCandidates`
- Summary: 可用的促销商品清单
- 描述: 通过识别号获取可参与促销活动的商品清单的方法。

### 4.1 请求体
Schema: `seller_apiGetSellerProductV1Request`

| 字段 | 类型 | 必填 | 说明 | 备注 |
|---|---|---:|---|---|
| `action_id` | `number<double>` | 否 | 活动识别号。可通过 `POST /v1/actions` 获取。 | 官方页面未显示必填标记，但按接口语义应传。 |
| `limit` | `number<double>` | 否 | 每页返回数量。 | 默认值为 `100`。 |
| `offset` | `number<double>` | 否 | 要跳过的元素数量。 | 已弃用；自 2025-05-05 起不应继续使用。 |
| `last_id` | `number<double>` | 否 | 当前页最后一个值的 ID。 | 首次请求不传；后续请求传上一页响应中的 `result.last_id`。 |

### 4.2 200 响应体
Schema: `seller_apiGetSellerProductV1Response`

顶层结构：

| 字段 | 类型 | 说明 |
|---|---|---|
| `result` | `object` | 请求结果。 |
| `result.products` | `object[]` | 商品清单。 |
| `result.total` | `number<double>` | 可用于该活动的商品总数。 |
| `result.last_id` | `number<double>` | 下一页游标。首次请求不传，后续从上一页响应中取值。 |

`result.products[]` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | `number<double>` | 商品识别号。 |
| `price` | `number<double>` | 商品当前价格。 |
| `action_price` | `number<double>` | 商品当前活动价。 |
| `alert_max_action_price_failed` | `boolean` | 最高活动价校验失败标志。 |
| `alert_max_action_price` | `number<double>` | 最高活动价提示值。 |
| `max_action_price` | `number<double>` | 允许加入该活动的最高活动价。 |
| `add_mode` | `string` | 商品加入促销的模式。示例值：`NOT_SET`、`MANUAL`。 |
| `stock` | `number<double>` | 当前活动库存。 |
| `min_stock` | `number<double>` | 最低库存阈值。 |
| `current_boost` | `number<double>` | 当前 boost 值。 |
| `price_min_elastic` | `number<double>` | 弹性促销相关的最低价格字段。 |
| `price_max_elastic` | `number<double>` | 弹性促销相关的最高价格字段。 |
| `min_boost` | `number<double>` | 最小 boost 值。 |
| `max_boost` | `number<double>` | 最大 boost 值。 |

说明：
- 上述商品项字段来自官方页面的响应示例。
- 官方页面未对这些字段逐项展开描述，表中说明以字段名和页面上下文做最小归纳。

### 4.3 错误响应
| 响应码 | 说明 |
|---|---|
| `default` | 官方页面仅标注为“错误”，未在该接口页展开错误对象结构。 |

### 4.4 请求示例
推荐的首页查询示例（使用 `last_id` 分页）：

```json
{
  "action_id": 63337,
  "limit": 100
}
```

下一页查询示例：

```json
{
  "action_id": 63337,
  "limit": 100,
  "last_id": 1366
}
```

官方页面展示的旧分页示例：

```json
{
  "action_id": 63337,
  "limit": 10,
  "offset": 0
}
```

### 4.5 成功响应示例
```json
{
  "result": {
    "products": [
      {
        "id": 226,
        "price": 250,
        "action_price": 0,
        "alert_max_action_price_failed": true,
        "alert_max_action_price": 31,
        "max_action_price": 175,
        "add_mode": "NOT_SET",
        "stock": 0,
        "min_stock": 0,
        "current_boost": 0,
        "price_min_elastic": 0,
        "price_max_elastic": 0,
        "min_boost": 0,
        "max_boost": 0
      },
      {
        "id": 1366,
        "price": 2300,
        "action_price": 630,
        "alert_max_action_price_failed": true,
        "alert_max_action_price": 31,
        "max_action_price": 770,
        "add_mode": "MANUAL",
        "stock": 0,
        "min_stock": 0,
        "current_boost": 0,
        "price_min_elastic": 0,
        "price_max_elastic": 0,
        "min_boost": 0,
        "max_boost": 0
      }
    ],
    "total": 2,
    "last_id": 1366
  }
}
```

### 4.6 使用建议
- 新接入时不要再使用 `offset`；应统一切换到 `last_id`。
- 若用于批量同步候选商品，建议固定 `limit`，并以 `result.last_id` 做游标循环。
- `result.products[]` 中多个价格/boost 字段的业务含义未在该接口页展开，落地代码前建议按实际返回值做一次联调确认。

## 5. 接口二：`POST /v1/actions/products/activate`
- OperationId: `PromosProductsActivate`
- Summary: 在促销活动中增加一个商品
- 描述: 一种向现有促销活动添加商品的方法。

### 5.1 请求体
Schema: `seller_apiActivateProductV1Request`

| 字段 | 类型 | 必填 | 说明 | 备注 |
|---|---|---:|---|---|
| `action_id` | `number<double>` | 是 | 活动识别号。可通过 `POST /v1/actions` 获取。 | - |
| `products` | `object[]` | 是 | 待添加的商品列表。 | 最多 `1000` 项。 |
| `products[].product_id` | `number<double>` | 是 | 商品识别号。 | - |
| `products[].action_price` | `number<double>` | 是 | 商品活动期间的价格。 | - |
| `products[].stock` | `number<double>` | 否 | 《库存折扣》促销中的商品单位数量。 | 仅适用于对应促销场景。 |

### 5.2 200 响应体
Schema: `seller_apiProductV1Response`

| 字段 | 类型 | 说明 |
|---|---|---|
| `result` | `object` | 请求结果。 |
| `result.product_ids` | `number<double>[]` | 已成功添加到促销活动中的商品 ID 列表。 |
| `result.rejected` | `object[]` | 未能加入促销活动的商品列表。 |
| `result.rejected[].product_id` | `number<double>` | 商品识别号。 |
| `result.rejected[].reason` | `string` | 该商品未被加入促销活动的原因。 |

### 5.3 错误响应
| 响应码 | 说明 |
|---|---|
| `default` | 官方页面仅标注为“错误”，未在该接口页展开错误对象结构。 |

### 5.4 请求示例
```json
{
  "action_id": 60564,
  "products": [
    {
      "product_id": 1389,
      "action_price": 356,
      "stock": 10
    }
  ]
}
```

### 5.5 成功响应示例
全部成功：

```json
{
  "result": {
    "product_ids": [
      1389
    ],
    "rejected": []
  }
}
```

部分失败：

```json
{
  "result": {
    "product_ids": [
      1389
    ],
    "rejected": [
      {
        "product_id": 2001,
        "reason": "price exceeds max_action_price"
      }
    ]
  }
}
```

### 5.6 使用建议
- 调用前应先通过 `POST /v1/actions/candidates` 获取候选商品及允许的活动价格范围，再决定 `action_price`。
- 若 `result.rejected` 非空，不应简单按 HTTP 200 视为全成功，应逐项处理失败商品。
- 批量添加时建议将调用结果落日志或落任务表，便于回放哪些商品已成功加入、哪些失败以及失败原因。

## 6. 共同注意事项
1. 这两个接口均属于官方促销能力，服务地址为 `https://api-seller.ozon.ru`。
2. `candidates` 与 `activate` 的典型调用顺序为：
   - 先查促销活动列表：`POST /v1/actions`
   - 再查候选商品：`POST /v1/actions/candidates`
   - 最后添加商品：`POST /v1/actions/products/activate`
3. 若需要稳定分页，请统一使用 `last_id` 游标，而不是 `offset`。
4. 官方页面在这两个接口页仅展示 `default` 错误入口，未公开完整错误对象；接入时应保留原始错误响应文本以便排障。
