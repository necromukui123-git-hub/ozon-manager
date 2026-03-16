-- Ozon Manager 增量升级脚本
-- 文件: upgrade_20260313_search_cpo_bulk_enroll.sql
-- 适用范围: 已存在 promotions / automation / products 基础结构的历史数据库
-- 用途: 新增 CPO 隐藏商品缓存、默认活动配置与批量报名运行历史
-- 执行前检查:
--   1. 确认数据库已包含 promotion_actions、products、shops、users 表。
--   2. 建议在执行前备份数据库。
-- 失败处理建议:
--   1. 若脚本中断，先回滚当前事务或恢复备份后重试。
--   2. 所有 CREATE/INDEX 均尽量幂等，可在排障后重复执行。

BEGIN;

CREATE TABLE IF NOT EXISTS search_cpo_configs (
    id                  SERIAL PRIMARY KEY,
    shop_id             INTEGER NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    official_action_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    shop_action_ids     JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shop_id)
);

CREATE TABLE IF NOT EXISTS search_cpo_products (
    id                  SERIAL PRIMARY KEY,
    shop_id             INTEGER NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    sku                 VARCHAR(120),
    source_sku          VARCHAR(120) NOT NULL,
    image_url           TEXT,
    title               VARCHAR(500),
    category_name       VARCHAR(300),
    price               DECIMAL(12, 2),
    is_in_stock         BOOLEAN NOT NULL DEFAULT FALSE,
    search_promo_status VARCHAR(80),
    is_favorite         BOOLEAN NOT NULL DEFAULT FALSE,
    orders              BIGINT NOT NULL DEFAULT 0,
    spent               DECIMAL(12, 2) NOT NULL DEFAULT 0,
    clicks              BIGINT NOT NULL DEFAULT 0,
    ctr_percent         DECIMAL(8, 4) NOT NULL DEFAULT 0,
    stock_total         BIGINT NOT NULL DEFAULT 0,
    payload             JSONB,
    last_synced_at      TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shop_id, source_sku)
);

CREATE TABLE IF NOT EXISTS search_cpo_runs (
    id                  SERIAL PRIMARY KEY,
    shop_id             INTEGER NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    triggered_by        INTEGER REFERENCES users(id) ON DELETE SET NULL,
    status              VARCHAR(30) NOT NULL DEFAULT 'pending',
    filter_snapshot     JSONB DEFAULT '{}'::jsonb,
    action_snapshot     JSONB DEFAULT '{}'::jsonb,
    total_fetched       INTEGER DEFAULT 0,
    total_selected      INTEGER DEFAULT 0,
    total_processed     INTEGER DEFAULT 0,
    success_items       INTEGER DEFAULT 0,
    failed_items        INTEGER DEFAULT 0,
    skipped_items       INTEGER DEFAULT 0,
    error_message       TEXT,
    started_at          TIMESTAMP,
    completed_at        TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS search_cpo_run_items (
    id                  SERIAL PRIMARY KEY,
    run_id              INTEGER NOT NULL REFERENCES search_cpo_runs(id) ON DELETE CASCADE,
    product_cache_id    INTEGER REFERENCES search_cpo_products(id) ON DELETE SET NULL,
    source_sku          VARCHAR(120) NOT NULL,
    sku                 VARCHAR(120),
    title               VARCHAR(500),
    search_promo_status VARCHAR(80),
    overall_status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    official_status     VARCHAR(20) NOT NULL DEFAULT 'pending',
    shop_status         VARCHAR(20) NOT NULL DEFAULT 'pending',
    official_results    JSONB NOT NULL DEFAULT '[]'::jsonb,
    shop_results        JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(run_id, source_sku)
);

CREATE INDEX IF NOT EXISTS idx_search_cpo_configs_shop_id ON search_cpo_configs(shop_id);
CREATE INDEX IF NOT EXISTS idx_search_cpo_products_shop_id ON search_cpo_products(shop_id);
CREATE INDEX IF NOT EXISTS idx_search_cpo_products_source_sku ON search_cpo_products(source_sku);
CREATE INDEX IF NOT EXISTS idx_search_cpo_products_status ON search_cpo_products(search_promo_status);
CREATE INDEX IF NOT EXISTS idx_search_cpo_runs_shop_id ON search_cpo_runs(shop_id);
CREATE INDEX IF NOT EXISTS idx_search_cpo_runs_status ON search_cpo_runs(status);
CREATE INDEX IF NOT EXISTS idx_search_cpo_run_items_run_id ON search_cpo_run_items(run_id);
CREATE INDEX IF NOT EXISTS idx_search_cpo_run_items_source_sku ON search_cpo_run_items(source_sku);

COMMIT;
