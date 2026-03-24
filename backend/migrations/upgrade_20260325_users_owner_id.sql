-- Ozon Manager 增量升级脚本
-- 文件: upgrade_20260325_users_owner_id.sql
-- 适用范围: 已执行过旧版 init_database.sql，但 users 表尚未包含 owner_id 字段的历史数据库
-- 用途: 为 users 表补齐员工归属店铺管理员所需的 owner_id 字段与索引
-- 执行前检查:
--   1. 建议先执行: SELECT column_name FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'owner_id';
--   2. 建议在执行前备份数据库。
-- 失败处理建议:
--   1. 若脚本中断，先回滚当前事务或恢复备份后重试。
--   2. 本脚本尽量幂等，排障后可重复执行。

BEGIN;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS owner_id INTEGER;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_users_owner_id'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT fk_users_owner_id
            FOREIGN KEY (owner_id) REFERENCES users(id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_users_owner_id ON users(owner_id);

COMMIT;
