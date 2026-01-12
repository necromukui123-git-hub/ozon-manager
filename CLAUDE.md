# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 推荐模型

本项目推荐使用 **Claude Opus 4.5** (model ID: `claude-opus-4-5-20251101`) 进行开发协助。

## 项目概述

Ozon店铺管理系统 - 用于管理Ozon电商平台的商品促销活动。主要功能：
- 批量将商品报名促销活动（弹性折扣、28折扣）
- 处理亏损商品：退出促销、改价、重新报名
- 导出可推广商品列表
- 三层角色权限管理（系统管理员 / 店铺管理员 / 员工）

## 用户角色

系统采用三层角色架构，实现完全隔离：

| 角色 | 用户名示例 | 密码 | 权限 |
|------|-----------|------|------|
| 系统管理员 (super_admin) | `super_admin` | `admin123` | 管理店铺管理员，查看系统概览（只读） |
| 店铺管理员 (shop_admin) | `admin` | `admin123` | 管理自己的店铺和员工，执行业务操作 |
| 员工 (staff) | - | - | 操作被分配的店铺 |

**隔离原则**：
- 一个店铺只属于一个店铺管理员
- 员工只属于创建他的店铺管理员
- 不同店铺管理员之间完全隔离

## 技术栈

**后端**: Go 1.21 + Gin + GORM + PostgreSQL
**前端**: Vue 3 + Vite + Element Plus + Pinia

## 常用命令

### 后端

```bash
cd backend

# 运行后端服务（需先配置 config/config.yaml）
go run cmd/server/main.go

# 构建
go build -o server cmd/server/main.go

# 重置密码工具
go run cmd/reset-password/main.go
```

### 前端

```bash
cd frontend

# 安装依赖
npm install

# 开发模式运行（端口5173，代理到后端8080）
npm run dev

# 构建生产版本
npm run build
```

### 数据库

使用 PostgreSQL，迁移脚本位于 `backend/migrations/`：
- `001_create_tables.up.sql` - 初始表结构
- `002_refactor_user_roles.up.sql` - 三层角色重构
- `002_refactor_user_roles.down.sql` - 回滚脚本

## 项目架构

### 后端结构 (backend/)

```
cmd/
  server/main.go          # 入口文件，路由注册
  reset-password/main.go  # 密码重置工具
internal/
  config/
    config.go             # 配置加载（Server、Database、JWT、Log）
  model/
    user.go               # User 模型（三层角色）
    shop.go               # Shop、UserShop 模型
    product.go            # Product、LossProduct、PromotedProduct、PromotionAction、OperationLog 模型
  dto/
    request.go            # 请求 DTO
    response.go           # 响应 DTO
  repository/
    db.go                 # 数据库初始化
    user_repo.go          # 用户数据访问
    shop_repo.go          # 店铺数据访问
    product_repo.go       # 商品数据访问
    promotion_repo.go     # 促销数据访问
    operation_log_repo.go # 操作日志数据访问
  service/
    auth_service.go       # 认证业务逻辑
    user_service.go       # 用户管理
    shop_service.go       # 店铺管理
    product_service.go    # 商品管理
    promotion_service.go  # 促销活动
  handler/
    auth_handler.go       # 认证 HTTP 处理器
    user_handler.go       # 用户管理处理器
    shop_handler.go       # 店铺管理处理器
    product_handler.go    # 商品管理处理器
    promotion_handler.go  # 促销活动处理器
    operation_log_handler.go # 操作日志处理器
  middleware/
    auth.go               # JWT 认证中间件
    role.go               # 角色权限中间件
    operation_log.go      # 操作日志中间件
pkg/
  jwt/jwt.go              # JWT 工具
  ozon/
    client.go             # Ozon API HTTP 客户端
    actions.go            # 促销活动 API
    prices.go             # 价格管理 API
  excel/
    importer.go           # Excel 导入
    exporter.go           # Excel 导出
migrations/               # 数据库迁移脚本
config/config.yaml        # 运行时配置文件
```

### 前端结构 (frontend/src/)

```
views/
  auth/Login.vue          # 登录页
  Dashboard.vue           # 仪表盘
  Layout.vue              # 主布局（导航、侧边栏）
  products/
    ProductList.vue       # 商品列表管理
  promotions/
    ActionList.vue        # 促销活动列表
    BatchEnroll.vue       # 批量报名促销
    LossProcess.vue       # 亏损处理
    Reprice.vue           # 改价推广
  admin/
    OperationLogs.vue     # 操作日志查看
  shop-admin/
    MyShops.vue           # 店铺管理员的店铺管理
    MyStaff.vue           # 店铺管理员的员工管理
  super-admin/
    ShopAdminList.vue     # 系统管理员的店铺管理员列表
    SystemOverview.vue    # 系统概览
api/
  auth.js                 # 认证 API
  admin.js                # 系统管理员 API
  shopAdmin.js            # 店铺管理员 API
  product.js              # 商品 API
  promotion.js            # 促销 API（含 V1/V2）
  shop.js                 # 店铺 API
  user.js                 # 用户 API
  log.js                  # 操作日志 API
stores/
  user.js                 # Pinia 用户状态管理
router/
  index.js                # Vue Router 路由配置
utils/
  request.js              # Axios 封装
styles/
  main.scss               # 全局样式
```

## 数据库模型

8 个核心表：

| 表名 | 说明 |
|------|------|
| users | 用户表（角色：super_admin/shop_admin/staff） |
| shops | 店铺表（含 client_id、api_key） |
| user_shops | 用户-店铺关联表（员工可访问的店铺） |
| products | 商品表（含亏损/推广状态） |
| loss_products | 亏损商品记录 |
| promoted_products | 已推广商品记录 |
| promotion_actions | 促销活动缓存 |
| operation_logs | 操作日志（JSONB 详情） |

## 核心业务逻辑

1. **批量报名促销 (BatchEnroll)**: 将非亏损、非已推广的商品批量添加到促销活动
2. **亏损处理 (LossProcess)**: 导入亏损Excel → 退出促销 → 改价 → 重新报名28折扣
3. **改价推广 (Reprice)**: 从促销中移除 → 改价 → 重新添加推广

## API 路由

### 公开路由
- `POST /api/v1/auth/login` - 登录

### 认证后路由（所有用户）
- `POST /api/v1/auth/logout` - 登出
- `GET /api/v1/auth/me` - 获取当前用户
- `PUT /api/v1/auth/password` - 修改密码
- `GET /api/v1/shops` - 获取店铺列表
- `GET /api/v1/shops/:id` - 获取店铺详情

### 系统管理员专用 (`/api/v1/admin/*`)
- `POST /admin/shop-admins` - 创建店铺管理员
- `GET /admin/shop-admins` - 店铺管理员列表
- `GET /admin/shop-admins/:id` - 店铺管理员详情
- `PUT /admin/shop-admins/:id/status` - 更新状态
- `PUT /admin/shop-admins/:id/password` - 重置密码
- `DELETE /admin/shop-admins/:id` - 删除
- `GET /admin/overview` - 系统概览

### 店铺管理员专用 (`/api/v1/my/*`)
- `POST /my/shops` - 创建店铺
- `GET /my/shops` - 我的店铺列表
- `PUT /my/shops/:id` - 更新店铺
- `DELETE /my/shops/:id` - 删除店铺
- `POST /my/staff` - 创建员工
- `GET /my/staff` - 我的员工列表
- `PUT /my/staff/:id/status` - 更新员工状态
- `PUT /my/staff/:id/password` - 重置员工密码
- `PUT /my/staff/:id/shops` - 分配店铺
- `DELETE /my/staff/:id` - 删除员工

### 业务操作（shop_admin + staff）

**商品管理**
- `GET /api/v1/products` - 商品列表
- `GET /api/v1/products/:id` - 商品详情
- `POST /api/v1/products/sync` - 同步商品

**促销活动**
- `GET /api/v1/promotions/actions` - 活动列表
- `POST /api/v1/promotions/actions/manual` - 手动添加活动
- `DELETE /api/v1/promotions/actions/:id` - 删除活动
- `POST /api/v1/promotions/sync-actions` - 同步活动

**批量操作 (V1)**
- `POST /api/v1/promotions/batch-enroll` - 批量报名
- `POST /api/v1/promotions/process-loss` - 处理亏损
- `POST /api/v1/promotions/remove-reprice-promote` - 改价推广

**批量操作 (V2 - 支持选择活动)**
- `POST /api/v1/promotions/batch-enroll-v2` - 批量报名
- `POST /api/v1/promotions/process-loss-v2` - 处理亏损
- `POST /api/v1/promotions/remove-reprice-promote-v2` - 改价推广

**Excel 操作**
- `POST /api/v1/excel/import-loss` - 导入亏损商品
- `POST /api/v1/excel/import-reprice` - 导入改价商品
- `GET /api/v1/excel/export-promotable` - 导出可推广商品
- `GET /api/v1/excel/template/loss` - 下载亏损模板

**统计与日志**
- `GET /api/v1/stats/overview` - 统计数据
- `GET /api/v1/operation-logs` - 操作日志

## 配置文件结构

```yaml
server:
  port: 8080
  mode: debug  # debug/release

database:
  host: localhost
  port: 5432
  user: postgres
  password: ***
  dbname: ozon_manager
  sslmode: disable

jwt:
  secret: your-secret-key
  expire_hours: 24

log:
  level: debug
  format: json
```

## Ozon API

封装在 `pkg/ozon/`，包含：
- `client.go` - HTTP 客户端
- `actions.go` - 促销活动操作
- `prices.go` - 价格操作

API 基础 URL: `https://api-seller.ozon.ru`
