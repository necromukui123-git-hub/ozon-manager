-- Ozon Manager 增量升级脚本
-- 文件: upgrade_20260405_auto_promotion_date_range.sql
-- 适用范围: 已存在 auto_promotion_configs / auto_promotion_runs 的历史数据库
-- 用途: 为自动加促销补充 target_date_end，使自定义规则可表达闭区间日期段
-- 执行前检查:
--   1. 确认数据库已执行过自动加促销基础迁移。
--   2. 建议在执行前备份数据库。
-- 失败处理建议:
--   1. 若脚本中断，先回滚当前事务或恢复备份后重试。
--   2. 本脚本尽量幂等，可在排障后重复执行。

BEGIN;

ALTER TABLE auto_promotion_configs
    ADD COLUMN IF NOT EXISTS target_date_end DATE;

ALTER TABLE auto_promotion_runs
    ADD COLUMN IF NOT EXISTS target_date_end DATE;

UPDATE auto_promotion_configs
SET target_date_end = target_date
WHERE target_date IS NOT NULL
  AND target_date_end IS NULL;

UPDATE auto_promotion_runs
SET target_date_end = target_date
WHERE target_date_end IS NULL;

ALTER TABLE auto_promotion_runs
    ALTER COLUMN target_date_end SET NOT NULL;

COMMIT;
