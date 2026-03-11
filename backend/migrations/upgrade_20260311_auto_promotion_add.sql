-- Ozon Manager 增量升级脚本
-- 文件: upgrade_20260311_auto_promotion_add.sql
-- 适用范围: 已存在 promotions / automation / ozon_catalog 基础结构的历史数据库
-- 用途: 新增自动加促销配置、运行历史、逐商品结果和活动候选商品缓存表
-- 执行前检查:
--   1. 确认数据库已包含 promotion_actions、products、shops、users 表。
--   2. 建议在执行前备份数据库。
-- 失败处理建议:
--   1. 若脚本中断，先回滚当前事务或恢复备份后重试。
--   2. 所有 CREATE/INDEX 均尽量幂等，可在排障后重复执行。

BEGIN;

CREATE TABLE IF NOT EXISTS promotion_action_candidates (
    id                  SERIAL PRIMARY KEY,
    promotion_action_id INTEGER NOT NULL REFERENCES promotion_actions(id) ON DELETE CASCADE,
    shop_id             INTEGER NOT NULL REFERENCES shops(id),
    ozon_product_id     BIGINT,
    source_sku          VARCHAR(120) NOT NULL,
    offer_id            VARCHAR(120),
    platform_sku        VARCHAR(120),
    action_price        DECIMAL(12, 2),
    max_action_price    DECIMAL(12, 2),
    discount_percent    DECIMAL(6, 2),
    stock               INTEGER DEFAULT 0,
    status              VARCHAR(30) DEFAULT 'candidate',
    payload             JSONB,
    last_synced_at      TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(promotion_action_id, source_sku)
);

CREATE TABLE IF NOT EXISTS auto_promotion_configs (
    id                  SERIAL PRIMARY KEY,
    shop_id             INTEGER NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    enabled             BOOLEAN NOT NULL DEFAULT false,
    schedule_time       VARCHAR(5) NOT NULL DEFAULT '09:05',
    target_date         DATE NOT NULL,
    official_action_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    shop_action_ids     JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shop_id)
);

CREATE TABLE IF NOT EXISTS auto_promotion_runs (
    id                  SERIAL PRIMARY KEY,
    config_id           INTEGER REFERENCES auto_promotion_configs(id) ON DELETE SET NULL,
    shop_id             INTEGER NOT NULL REFERENCES shops(id),
    triggered_by        INTEGER REFERENCES users(id),
    trigger_mode        VARCHAR(20) NOT NULL,
    trigger_date        DATE NOT NULL,
    target_date         DATE NOT NULL,
    status              VARCHAR(30) NOT NULL DEFAULT 'pending',
    total_candidates    INTEGER DEFAULT 0,
    total_selected      INTEGER DEFAULT 0,
    total_processed     INTEGER DEFAULT 0,
    success_items       INTEGER DEFAULT 0,
    failed_items        INTEGER DEFAULT 0,
    skipped_items       INTEGER DEFAULT 0,
    config_snapshot     JSONB DEFAULT '{}'::jsonb,
    error_message       TEXT,
    started_at          TIMESTAMP,
    completed_at        TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS auto_promotion_run_items (
    id                  SERIAL PRIMARY KEY,
    run_id              INTEGER NOT NULL REFERENCES auto_promotion_runs(id) ON DELETE CASCADE,
    product_id          INTEGER REFERENCES products(id) ON DELETE SET NULL,
    ozon_product_id     BIGINT,
    source_sku          VARCHAR(120) NOT NULL,
    product_name        VARCHAR(500),
    listing_date        DATE NOT NULL,
    overall_status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    official_status     VARCHAR(20) NOT NULL DEFAULT 'pending',
    shop_status         VARCHAR(20) NOT NULL DEFAULT 'pending',
    official_results    JSONB NOT NULL DEFAULT '[]'::jsonb,
    shop_results        JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(run_id, source_sku)
);

CREATE INDEX IF NOT EXISTS idx_promotion_action_candidates_action_id ON promotion_action_candidates(promotion_action_id);
CREATE INDEX IF NOT EXISTS idx_promotion_action_candidates_shop_id ON promotion_action_candidates(shop_id);
CREATE INDEX IF NOT EXISTS idx_promotion_action_candidates_ozon_product_id ON promotion_action_candidates(ozon_product_id);
CREATE INDEX IF NOT EXISTS idx_promotion_action_candidates_source_sku ON promotion_action_candidates(source_sku);
CREATE INDEX IF NOT EXISTS idx_auto_promotion_configs_shop_id ON auto_promotion_configs(shop_id);
CREATE INDEX IF NOT EXISTS idx_auto_promotion_runs_shop_id ON auto_promotion_runs(shop_id);
CREATE INDEX IF NOT EXISTS idx_auto_promotion_runs_status ON auto_promotion_runs(status);
CREATE INDEX IF NOT EXISTS idx_auto_promotion_runs_trigger_date ON auto_promotion_runs(trigger_date);
CREATE UNIQUE INDEX IF NOT EXISTS idx_auto_promotion_runs_config_trigger_scheduled ON auto_promotion_runs(config_id, trigger_date) WHERE trigger_mode = 'scheduled';
CREATE INDEX IF NOT EXISTS idx_auto_promotion_run_items_run_id ON auto_promotion_run_items(run_id);
CREATE INDEX IF NOT EXISTS idx_auto_promotion_run_items_product_id ON auto_promotion_run_items(product_id);
CREATE INDEX IF NOT EXISTS idx_auto_promotion_run_items_overall_status ON auto_promotion_run_items(overall_status);

COMMIT;
