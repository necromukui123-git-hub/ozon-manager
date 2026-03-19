-- Ozon Manager 增量升级脚本
-- 文件: upgrade_20260319_search_cpo_morkovsk_automation.sql
-- 适用范围: 已执行过 Search CPO 基础表脚本的历史数据库
-- 用途: 为 Search CPO 增加自动化配置、状态跟踪和 Morkovsk 自动化运行历史
-- 执行前检查:
--   1. 确认数据库已包含 search_cpo_configs、search_cpo_products、search_cpo_runs、search_cpo_run_items 表。
--   2. 建议在执行前备份数据库。
-- 失败处理建议:
--   1. 若脚本中断，先回滚当前事务或恢复备份后重试。
--   2. 所有 ALTER/CREATE/INDEX 均尽量幂等，可在排障后重复执行。

BEGIN;

ALTER TABLE search_cpo_configs
    ADD COLUMN IF NOT EXISTS auto_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS schedule_time VARCHAR(5) NOT NULL DEFAULT '09:05',
    ADD COLUMN IF NOT EXISTS enable_step BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE search_cpo_products
    ADD COLUMN IF NOT EXISTS carrots_status VARCHAR(80),
    ADD COLUMN IF NOT EXISTS availability_promo BOOLEAN,
    ADD COLUMN IF NOT EXISTS availability_payload JSONB,
    ADD COLUMN IF NOT EXISTS availability_checked_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS rule_state VARCHAR(40),
    ADD COLUMN IF NOT EXISTS state2_detected_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS morkovsk_joined_at TIMESTAMP;

CREATE TABLE IF NOT EXISTS search_cpo_auto_runs (
    id                    SERIAL PRIMARY KEY,
    config_id             INTEGER REFERENCES search_cpo_configs(id) ON DELETE SET NULL,
    shop_id               INTEGER NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    triggered_by          INTEGER REFERENCES users(id) ON DELETE SET NULL,
    trigger_mode          VARCHAR(20) NOT NULL,
    trigger_date          DATE NOT NULL,
    status                VARCHAR(30) NOT NULL DEFAULT 'pending',
    filter_snapshot       JSONB DEFAULT '{}'::jsonb,
    config_snapshot       JSONB DEFAULT '{}'::jsonb,
    total_fetched         INTEGER DEFAULT 0,
    total_state1          INTEGER DEFAULT 0,
    total_state2          INTEGER DEFAULT 0,
    total_state3_trigger  INTEGER DEFAULT 0,
    total_processed       INTEGER DEFAULT 0,
    success_items         INTEGER DEFAULT 0,
    failed_items          INTEGER DEFAULT 0,
    skipped_items         INTEGER DEFAULT 0,
    error_message         TEXT,
    started_at            TIMESTAMP,
    completed_at          TIMESTAMP,
    created_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS search_cpo_auto_run_items (
    id                  SERIAL PRIMARY KEY,
    run_id              INTEGER NOT NULL REFERENCES search_cpo_auto_runs(id) ON DELETE CASCADE,
    product_cache_id    INTEGER REFERENCES search_cpo_products(id) ON DELETE SET NULL,
    source_sku          VARCHAR(120) NOT NULL,
    sku                 VARCHAR(120),
    title               VARCHAR(500),
    search_promo_status VARCHAR(80),
    carrots_status      VARCHAR(80),
    availability_promo  BOOLEAN,
    rule_state_before   VARCHAR(40),
    rule_state_after    VARCHAR(40),
    overall_status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    initial_status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    enable_status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    exit_status         VARCHAR(20) NOT NULL DEFAULT 'pending',
    morkovsk_status     VARCHAR(20) NOT NULL DEFAULT 'pending',
    initial_results     JSONB NOT NULL DEFAULT '[]'::jsonb,
    enable_result       JSONB NOT NULL DEFAULT '{}'::jsonb,
    exit_results        JSONB NOT NULL DEFAULT '[]'::jsonb,
    morkovsk_result     JSONB NOT NULL DEFAULT '{}'::jsonb,
    message             TEXT,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(run_id, source_sku)
);

CREATE INDEX IF NOT EXISTS idx_search_cpo_products_rule_state ON search_cpo_products(rule_state);
CREATE INDEX IF NOT EXISTS idx_search_cpo_auto_runs_shop_id ON search_cpo_auto_runs(shop_id);
CREATE INDEX IF NOT EXISTS idx_search_cpo_auto_runs_status ON search_cpo_auto_runs(status);
CREATE INDEX IF NOT EXISTS idx_search_cpo_auto_runs_trigger_date ON search_cpo_auto_runs(trigger_date);
CREATE INDEX IF NOT EXISTS idx_search_cpo_auto_run_items_run_id ON search_cpo_auto_run_items(run_id);
CREATE INDEX IF NOT EXISTS idx_search_cpo_auto_run_items_source_sku ON search_cpo_auto_run_items(source_sku);

COMMIT;
