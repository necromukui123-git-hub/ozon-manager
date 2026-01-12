-- 用户系统三层角色重构迁移脚本
-- 将现有的两层角色（admin/staff）重构为三层角色（super_admin/shop_admin/staff）

BEGIN;

-- 1. 为 users 表添加 owner_id 字段（员工所属的店铺管理员）
ALTER TABLE users ADD COLUMN IF NOT EXISTS owner_id INTEGER REFERENCES users(id);

-- 2. 为 shops 表添加 owner_id 字段（店铺所属的店铺管理员）
ALTER TABLE shops ADD COLUMN IF NOT EXISTS owner_id INTEGER REFERENCES users(id);

-- 3. 创建索引
CREATE INDEX IF NOT EXISTS idx_users_owner_id ON users(owner_id);
CREATE INDEX IF NOT EXISTS idx_shops_owner_id ON shops(owner_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- 4. 创建 super_admin 账号（密码: admin123）
-- bcrypt hash for 'admin123' (cost=10)
INSERT INTO users (username, password_hash, display_name, role, status, created_at, updated_at)
VALUES (
    'super_admin',
    '$2a$10$tgYs7ef1mKRcOw/SZN7Jr..iO/oGcMyMw55fGuvXaMEV1m0OwxrDe',
    '超级管理员',
    'super_admin',
    'active',
    NOW(),
    NOW()
)
ON CONFLICT (username) DO UPDATE SET role = 'super_admin', password_hash = '$2a$10$tgYs7ef1mKRcOw/SZN7Jr..iO/oGcMyMw55fGuvXaMEV1m0OwxrDe';

-- 5. 将现有 admin 改为 shop_admin（排除刚创建的 super_admin）
UPDATE users SET role = 'shop_admin'
WHERE role = 'admin' AND username != 'super_admin';

-- 6. 为现有店铺设置 owner（分配给第一个 shop_admin）
-- 如果没有 shop_admin，则不更新
UPDATE shops SET owner_id = (
    SELECT id FROM users WHERE role = 'shop_admin' ORDER BY id LIMIT 1
)
WHERE owner_id IS NULL
AND EXISTS (SELECT 1 FROM users WHERE role = 'shop_admin');

-- 7. 设置现有 staff 的 owner_id（根据其被分配的店铺来确定所属的 shop_admin）
UPDATE users u SET owner_id = (
    SELECT DISTINCT s.owner_id
    FROM user_shops us
    JOIN shops s ON us.shop_id = s.id
    WHERE us.user_id = u.id
    AND s.owner_id IS NOT NULL
    LIMIT 1
)
WHERE u.role = 'staff' AND u.owner_id IS NULL;

-- 8. 清理可能存在的无效数据：删除员工访问不属于其所属管理员的店铺的记录
DELETE FROM user_shops
WHERE id IN (
    SELECT us.id
    FROM user_shops us
    JOIN users u ON us.user_id = u.id
    JOIN shops s ON us.shop_id = s.id
    WHERE u.role = 'staff'
    AND u.owner_id IS NOT NULL
    AND s.owner_id IS NOT NULL
    AND u.owner_id != s.owner_id
);

COMMIT;
