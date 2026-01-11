# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

Ozon店铺管理系统 - 用于管理Ozon电商平台的商品促销活动。主要功能：
- 批量将商品报名促销活动（弹性折扣、28折扣）
- 处理亏损商品：退出促销、改价、重新报名
- 导出可推广商品列表
- 多店铺、多用户权限管理

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

使用 PostgreSQL，初始化脚本位于 `backend/migrations/001_create_tables.up.sql`

默认管理员账号: admin / admin123

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
  middleware/           # 认证、权限、日志中间件
pkg/
  jwt/                  # JWT工具
  ozon/                 # Ozon API客户端封装
  excel/                # Excel导入导出
```

### 前端结构 (frontend/src/)

```
views/
  auth/Login.vue        # 登录页
  Dashboard.vue         # 仪表盘
  products/             # 商品管理
  promotions/           # 促销操作（批量报名、亏损处理、改价）
  admin/                # 管理员功能（店铺、用户、日志）
stores/                 # Pinia状态管理
api/                    # API调用封装
router/                 # Vue Router路由配置
```

## 核心业务逻辑

1. **批量报名促销 (BatchEnroll)**: 将非亏损、非已推广的商品批量添加到促销活动
2. **亏损处理 (LossProcess)**: 导入亏损Excel → 退出促销 → 改价 → 重新报名28折扣
3. **改价推广 (Reprice)**: 从促销中移除 → 改价 → 重新添加推广

## API路由

- `/api/v1/auth/*` - 认证相关
- `/api/v1/products/*` - 商品管理
- `/api/v1/promotions/*` - 促销操作
- `/api/v1/excel/*` - Excel导入导出
- `/api/v1/shops/*` - 店铺管理
- `/api/v1/users/*` - 用户管理（管理员）
- `/api/v1/operation-logs` - 操作日志（管理员）

## Ozon API

封装在 `pkg/ozon/`，包含：
- `client.go` - HTTP客户端
- `actions.go` - 促销活动操作
- `prices.go` - 价格操作

API基础URL: `https://api-seller.ozon.ru`
