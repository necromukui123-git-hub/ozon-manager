# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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
# 进入后端目录
cd backend

# 运行后端服务（需先配置 config/config.yaml）
go run cmd/server/main.go

# 构建
go build -o server cmd/server/main.go
```

### 前端

```bash
# 进入前端目录
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

## 项目架构

### 后端结构 (backend/)

```
cmd/server/main.go      # 入口文件，路由注册
internal/
  config/               # 配置加载（viper）
  model/                # GORM数据模型
  dto/                  # 请求/响应结构体
  repository/           # 数据库操作层
  service/              # 业务逻辑层
  handler/              # HTTP处理器（Gin）
  middleware/           # 中间件
    auth.go             # JWT认证
    admin_only.go       # 三层角色权限中间件
    operation_log.go    # 操作日志
pkg/
  jwt/                  # JWT工具
  ozon/                 # Ozon API客户端封装
  excel/                # Excel导入导出
migrations/             # 数据库迁移脚本
```

### 前端结构 (frontend/src/)

```
views/
  auth/Login.vue        # 登录页
  Dashboard.vue         # 仪表盘
  products/             # 商品管理
  promotions/           # 促销操作（批量报名、亏损处理、改价）
  admin/                # 旧版管理功能（店铺、用户、日志）
  super-admin/          # 系统管理员页面
    ShopAdminList.vue   # 管理店铺管理员
    SystemOverview.vue  # 系统概览
  shop-admin/           # 店铺管理员页面
    MyShops.vue         # 我的店铺
    MyStaff.vue         # 我的员工
stores/                 # Pinia状态管理
api/                    # API调用封装
  auth.js               # 认证API
  admin.js              # 系统管理员API
  shopAdmin.js          # 店铺管理员API
  product.js            # 商品API
  promotion.js          # 促销API
  shop.js               # 店铺API
  user.js               # 用户API
router/                 # Vue Router路由配置
```

## 核心业务逻辑

1. **批量报名促销 (BatchEnroll)**: 将非亏损、非已推广的商品批量添加到促销活动
2. **亏损处理 (LossProcess)**: 导入亏损Excel → 退出促销 → 改价 → 重新报名28折扣
3. **改价推广 (Reprice)**: 从促销中移除 → 改价 → 重新添加推广

## API路由

### 认证相关
- `POST /api/v1/auth/login` - 登录
- `POST /api/v1/auth/logout` - 登出
- `GET /api/v1/auth/me` - 获取当前用户
- `PUT /api/v1/auth/password` - 修改密码

### 系统管理员专用 (`/api/v1/admin/*`)
- `GET/POST /admin/shop-admins` - 店铺管理员列表/创建
- `GET /admin/shop-admins/:id` - 店铺管理员详情
- `PUT /admin/shop-admins/:id/status` - 更新状态
- `PUT /admin/shop-admins/:id/password` - 重置密码
- `DELETE /admin/shop-admins/:id` - 删除
- `GET /admin/overview` - 系统概览

### 店铺管理员专用 (`/api/v1/my/*`)
- `GET/POST /my/shops` - 我的店铺列表/创建
- `PUT/DELETE /my/shops/:id` - 更新/删除店铺
- `GET/POST /my/staff` - 我的员工列表/创建
- `PUT /my/staff/:id/status` - 更新员工状态
- `PUT /my/staff/:id/password` - 重置员工密码
- `PUT /my/staff/:id/shops` - 分配店铺
- `DELETE /my/staff/:id` - 删除员工

### 业务操作（shop_admin + staff）
- `/api/v1/products/*` - 商品管理
- `/api/v1/promotions/*` - 促销操作
- `/api/v1/excel/*` - Excel导入导出
- `/api/v1/stats/*` - 统计数据
- `/api/v1/operation-logs` - 操作日志

### 旧版管理路由（向后兼容）
- `/api/v1/shops/*` - 店铺管理
- `/api/v1/users/*` - 用户管理

## Ozon API

封装在 `pkg/ozon/`，包含：
- `client.go` - HTTP客户端
- `actions.go` - 促销活动操作
- `prices.go` - 价格操作

API基础URL: `https://api-seller.ozon.ru`
