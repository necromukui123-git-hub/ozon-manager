# Ozon Manager Windows 单机部署说明

这套发布方案面向“在另一台 Windows 电脑上直接运行”，目标机不需要安装 Go 或 Node.js。

## 1. 在打包电脑上生成发布包

在仓库根目录执行：

```powershell
cd E:\developcode\ozon-manager
powershell -ExecutionPolicy Bypass -File .\build-windows-release.ps1
```

产物会输出到：

- `release/ozon-manager-win-x64/`
- `release/ozon-manager-win-x64.zip`

其中已经包含：

- 后端可执行文件 `server/server.exe`
- 前端静态文件 `server/web/`
- 数据库初始化脚本 `server/database/init_database.sql`
- 数据库增量升级脚本 `server/database/upgrade_*.sql`
- 启动脚本 `server/start-ozon-manager.bat`
- Chrome 扩展解压目录 `browser-extension/ozon-shop-bridge/`
- Chrome 扩展传输压缩包 `browser-extension/ozon-shop-bridge-v<version>.zip`

## 2. 目标机需要准备的环境

- PostgreSQL
- Google Chrome

不需要安装：

- Go
- Node.js
- npm

## 3. 初始化空数据库

1. 在 PostgreSQL 中创建空数据库，例如：`ozon_manager`
2. 执行发布包内的初始化脚本：

```text
server/database/init_database.sql
```

例如可以使用：

```powershell
psql -U postgres -d ozon_manager -f .\server\database\init_database.sql
```

这台机器是新环境时，只执行 `init_database.sql` 即可，不需要迁移旧数据库数据。

## 4. 配置后端

1. 进入发布包目录中的 `server/`
2. 复制：

```text
config/config.yaml.example -> config/config.yaml
```

3. 至少修改以下配置：

- `database.password`
- `jwt.secret`
- `server.mode`

建议在目标机上把 `server.mode` 改为 `release`。

## 5. 启动服务

双击运行：

```text
server/start-ozon-manager.bat
```

启动后在目标机本机 Chrome 中打开：

```text
http://127.0.0.1:8080
```

前端现在由后端同源托管，不需要再单独启动 Vite。

## 6. 安装 Chrome 插件

1. 打开 `chrome://extensions/`
2. 开启“开发者模式”
3. 选择“加载已解压的扩展程序”
4. 选择发布包里的目录：

```text
browser-extension/ozon-shop-bridge
```

说明：

- `browser-extension/ozon-shop-bridge-v<version>.zip` 主要用于传输和备份
- Chrome 开发者模式不能直接加载 zip，需要先解压，或者直接使用发布包内已准备好的目录

## 7. 首次使用建议

1. 先登录管理端
2. 先选择当前店铺
3. 点击插件图标，确认：
   - `后端地址` 为 `http://127.0.0.1:8080`
   - 插件已启用轮询

在这套单机场景下，插件默认可以直接连接本机管理端登录态，通常不需要手动填写 `adminOrigin`。

## 8. 默认管理员与旧包说明

- 当前基线初始化后的默认管理员账号是：
  - 用户名：`super_admin`
  - 密码：`admin123`
- 如果你之前用 2026-03-25 之前的旧发布包初始化过数据库，默认管理员哈希可能不匹配，表现为登录页提示失败或前端误显示“登录已过期”。
- 对空库最简单的处理方式是：删除旧库后，使用当前发布包里的 `server/database/init_database.sql` 重新初始化。
- 如果不想重建数据库，也可以直接执行下面这条 SQL 把 `super_admin` 修正到当前基线值：

```sql
UPDATE users
SET password_hash = '$2a$10$8G/wyJgW6W3L9SXSvnRyQ.GO4//Q1fPdzodjLNaiJprgjnygTrHu6'
WHERE username = 'super_admin';
```

## 9. 后续升级

- 新机器/空库初始化：执行最新 `init_database.sql`
- 已有旧库升级：按对应 `upgrade_YYYYMMDD_<topic>.sql` 执行
- 如果旧库是在缺少 `users.owner_id` 的老基线上初始化的，且创建店铺管理员时报 `关系 "users" 的 "owner_id" 字段不存在`，请执行：

```text
server/database/upgrade_20260325_users_owner_id.sql
```

本次部署目标是新机器空库启动，所以不涉及旧数据库数据迁移。
