-- ============================================================
-- 适用范围：为 Search CPO 配置补齐退出活动字段
-- 执行前检查：确认 search_cpo_configs 表已存在
-- 失败处理建议：修复后可重复执行；脚本按幂等方式编写
-- ============================================================

ALTER TABLE search_cpo_configs
    ADD COLUMN IF NOT EXISTS exit_official_action_ids JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE search_cpo_configs
    ADD COLUMN IF NOT EXISTS exit_shop_action_ids JSONB NOT NULL DEFAULT '[]'::jsonb;
