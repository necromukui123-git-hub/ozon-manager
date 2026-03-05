-- ============================================================
-- 升级脚本: upgrade_20260304_ozon_catalog_cache.sql
-- 主题: 新增 Ozon 商品目录缓存表（用于实时商品列表页面）
--
-- 适用范围:
--   已存在基础业务表（users/shops/products 等）的历史环境。
--
-- 执行前检查:
--   1) 确认数据库可正常连接，并具备 CREATE TABLE / CREATE INDEX 权限。
--   2) 建议在业务低峰执行，并先做数据库备份。
--
-- 失败处理建议:
--   1) 若执行失败，先回滚当前事务。
--   2) 排查权限、锁等待和语法后重试。
-- ============================================================

BEGIN;

CREATE TABLE IF NOT EXISTS ozon_product_catalog_items (
    id                    SERIAL PRIMARY KEY,
    shop_id               INTEGER NOT NULL REFERENCES shops(id),
    ozon_product_id       BIGINT NOT NULL,
    offer_id              VARCHAR(120),
    sku                   BIGINT,
    name                  VARCHAR(500),
    primary_image_url     TEXT,
    price                 DECIMAL(12, 2),
    old_price             DECIMAL(12, 2),
    min_price             DECIMAL(12, 2),
    marketing_price       DECIMAL(12, 2),
    currency              VARCHAR(10),
    visibility            VARCHAR(30),
    status                VARCHAR(30),
    stock_total           INTEGER DEFAULT 0,
    stock_fbo             INTEGER DEFAULT 0,
    stock_fbs             INTEGER DEFAULT 0,
    listing_date          TIMESTAMP,
    listing_date_source   VARCHAR(20) NOT NULL DEFAULT 'local_sync',
    sync_token            VARCHAR(64),
    payload               JSONB,
    last_remote_synced_at TIMESTAMP,
    created_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shop_id, ozon_product_id)
);

CREATE INDEX IF NOT EXISTS idx_ozon_catalog_shop_product ON ozon_product_catalog_items(shop_id, ozon_product_id);
CREATE INDEX IF NOT EXISTS idx_ozon_catalog_shop_date ON ozon_product_catalog_items(shop_id, listing_date);
CREATE INDEX IF NOT EXISTS idx_ozon_catalog_shop_visibility ON ozon_product_catalog_items(shop_id, visibility);
CREATE INDEX IF NOT EXISTS idx_ozon_catalog_shop_offer ON ozon_product_catalog_items(shop_id, offer_id);
CREATE INDEX IF NOT EXISTS idx_ozon_catalog_sync_token ON ozon_product_catalog_items(sync_token);

COMMIT;
