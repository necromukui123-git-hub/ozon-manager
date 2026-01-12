-- 用户系统三层角色重构回滚脚本
-- 将三层角色回滚为两层角色

BEGIN;

-- 1. 将 shop_admin 改回 admin
UPDATE users SET role = 'admin' WHERE role = 'shop_admin';

-- 2. 将 super_admin 改为 admin（或删除，根据需要选择）
UPDATE users SET role = 'admin' WHERE role = 'super_admin';

-- 3. 移除 owner_id 字段
ALTER TABLE users DROP COLUMN IF EXISTS owner_id;
ALTER TABLE shops DROP COLUMN IF EXISTS owner_id;

-- 4. 删除索引
DROP INDEX IF EXISTS idx_users_owner_id;
DROP INDEX IF EXISTS idx_shops_owner_id;
DROP INDEX IF EXISTS idx_users_role;

COMMIT;
