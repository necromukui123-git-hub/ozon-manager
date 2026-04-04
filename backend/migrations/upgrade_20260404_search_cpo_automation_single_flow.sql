-- ============================================================
-- 适用范围：为 Search CPO 配置补齐退出活动字段
-- 执行前检查：确认 search_cpo_configs 表已存在
-- 失败处理建议：修复后可重复执行；脚本按幂等方式编写
-- ============================================================

ALTER TABLE search_cpo_configs
    ADD COLUMN IF NOT EXISTS exit_official_action_ids JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE search_cpo_configs
    ADD COLUMN IF NOT EXISTS exit_shop_action_ids JSONB NOT NULL DEFAULT '[]'::jsonb;

UPDATE search_cpo_products
SET rule_state = 'state3'
WHERE rule_state = 'state3_trigger';

UPDATE search_cpo_products
SET rule_state = 'state4'
WHERE rule_state = 'morkovsk_joined';

UPDATE search_cpo_auto_run_items
SET rule_state_before = 'state3'
WHERE rule_state_before = 'state3_trigger';

UPDATE search_cpo_auto_run_items
SET rule_state_before = 'state4'
WHERE rule_state_before = 'morkovsk_joined';

UPDATE search_cpo_auto_run_items
SET rule_state_after = 'state3'
WHERE rule_state_after = 'state3_trigger';

UPDATE search_cpo_auto_run_items
SET rule_state_after = 'state4'
WHERE rule_state_after = 'morkovsk_joined';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'search_cpo_auto_runs'
          AND column_name = 'total_state3_trigger'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'search_cpo_auto_runs'
          AND column_name = 'total_state3'
    ) THEN
        EXECUTE 'ALTER TABLE search_cpo_auto_runs RENAME COLUMN total_state3_trigger TO total_state3';
    END IF;
END $$;

ALTER TABLE search_cpo_auto_runs
    ADD COLUMN IF NOT EXISTS total_state3 INTEGER NOT NULL DEFAULT 0;

ALTER TABLE search_cpo_auto_runs
    ADD COLUMN IF NOT EXISTS total_state4 INTEGER NOT NULL DEFAULT 0;

UPDATE search_cpo_auto_runs AS runs
SET total_state3 = stats.total_state3,
    total_state4 = stats.total_state4
FROM (
    SELECT
        run_id,
        COUNT(*) FILTER (WHERE rule_state_after = 'state3') AS total_state3,
        COUNT(*) FILTER (WHERE rule_state_after = 'state4') AS total_state4
    FROM search_cpo_auto_run_items
    GROUP BY run_id
) AS stats
WHERE runs.id = stats.run_id;
