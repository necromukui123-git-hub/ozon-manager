# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

Ozon店铺管理系统 - 用于管理Ozon电商平台的商品促销活动。主要功能：
- 批量将商品报名促销活动（弹性折扣、28折扣）
- 处理亏损商品：退出促销、改价、重新报名
- 导出可推广商品列表
- 三层角色权限管理（系统管理员 / 店铺管理员 / 员工）

## 技术栈

**后端**: Go 1.21 + Gin + GORM
**前端**: Vue 3 + Vite + Element Plus + Pinia
**数据库**: PostgreSQL

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

### 开发工作流

同时启动前后端进行开发：
1. 终端1: `cd backend && go run cmd/server/main.go`
2. 终端2: `cd frontend && npm run dev`
3. 访问 http://localhost:5173

前端开发服务器会自动将 `/api` 请求代理到后端 8080 端口。

### 数据库

使用 PostgreSQL，迁移脚本位于 `backend/migrations/`。

**重要规则**：每次进行数据库相关改动时（包括但不限于：新增/修改/删除表、字段、索引、约束等），必须同步更新 `backend/migrations/init_database.sql` 文件，确保该文件始终包含完整的最新表结构和初始数据，以便在新服务器上快速部署。

### 配置

首次运行需要创建配置文件：
```bash
cd backend/config
cp config.yaml.example config.yaml
# 编辑 config.yaml 填写数据库密码和 JWT 密钥
```

默认超级管理员账户: `super_admin` / `admin123`

## 项目架构

### 后端分层架构 (backend/)

```
cmd/server/main.go     → 入口文件，路由注册
internal/
  config/              → 配置加载
  model/               → 数据模型（User、Shop、Product、PromotionAction 等）
  dto/                 → 请求/响应 DTO
  repository/          → 数据访问层
  service/             → 业务逻辑层
  handler/             → HTTP 处理器
  middleware/          → JWT认证、角色权限、操作日志
pkg/
  jwt/                 → JWT 工具
  ozon/                → Ozon API 客户端封装
  excel/               → Excel 导入导出
```

### 前端结构 (frontend/src/)

```
views/                 → 页面组件
  auth/                → 登录
  products/            → 商品管理
  promotions/          → 促销操作（批量报名、亏损处理、改价推广）
  shop-admin/          → 店铺管理员功能
  super-admin/         → 系统管理员功能
api/                   → API 调用封装
stores/                → Pinia 状态管理
router/                → Vue Router 路由
utils/request.js       → Axios 封装
```

## 用户角色与权限

系统采用三层角色架构，实现完全隔离：

| 角色 | 权限 |
|------|------|
| 系统管理员 (super_admin) | 管理店铺管理员，查看系统概览（只读） |
| 店铺管理员 (shop_admin) | 管理自己的店铺和员工，执行业务操作 |
| 员工 (staff) | 操作被分配的店铺 |

**隔离原则**：一个店铺只属于一个店铺管理员，员工只属于创建他的店铺管理员。

## 核心业务逻辑

1. **批量报名促销 (BatchEnroll)**: 将非亏损、非已推广的商品批量添加到促销活动
2. **亏损处理 (LossProcess)**: 导入亏损Excel → 退出促销 → 改价 → 重新报名28折扣
3. **改价推广 (Reprice)**: 从促销中移除 → 改价 → 重新添加推广

## API 版本说明

- **V1 API**: 基础批量操作
- **V2 API**: 支持选择特定促销活动的批量操作

路由前缀: `/api/v1/`

### 路由权限分组

- `/api/v1/auth/*` - 公开路由（登录）
- `/api/v1/admin/*` - 系统管理员专用（管理店铺管理员）
- `/api/v1/my/*` - 店铺管理员专用（管理店铺和员工）
- `/api/v1/products/*`, `/api/v1/promotions/*`, `/api/v1/excel/*`, `/api/v1/stats/*` - 业务操作（shop_admin 和 staff）

## Ozon API

封装在 `pkg/ozon/`，API 基础 URL: `https://api-seller.ozon.ru`

## 配置文件

后端配置文件: `backend/config/config.yaml`（从 `config.yaml.example` 复制）

需要配置: server（端口、模式、TLS）、database（PostgreSQL连接）、jwt（密钥、过期时间）、log（级别、格式）
