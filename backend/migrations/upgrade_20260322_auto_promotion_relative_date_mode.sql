-- Ozon Manager 增量升级脚本
-- 文件: upgrade_20260322_auto_promotion_relative_date_mode.sql
-- 适用范围: 已存在 auto_promotion_configs / auto_promotion_runs 的历史数据库
-- 用途: 为自动加促销补充相对日期规则字段，并允许配置表在非自定义模式下不保存绝对日期
-- 执行前检查:
--   1. 确认数据库已执行过自动加促销基础迁移。
--   2. 建议在执行前备份数据库。
-- 失败处理建议:
--   1. 若脚本中断，先回滚当前事务或恢复备份后重试。
--   2. 本脚本尽量幂等，可在排障后重复执行。

BEGIN;

ALTER TABLE auto_promotion_configs
    ADD COLUMN IF NOT EXISTS target_date_mode VARCHAR(20) NOT NULL DEFAULT 'custom';

ALTER TABLE auto_promotion_runs
    ADD COLUMN IF NOT EXISTS target_date_mode VARCHAR(20) NOT NULL DEFAULT 'custom';

UPDATE auto_promotion_configs
SET target_date_mode = 'custom'
WHERE COALESCE(BTRIM(target_date_mode), '') = '';

UPDATE auto_promotion_runs
SET target_date_mode = 'custom'
WHERE COALESCE(BTRIM(target_date_mode), '') = '';

ALTER TABLE auto_promotion_configs
    ALTER COLUMN target_date DROP NOT NULL;

COMMIT;
