-- ============================================================
-- 增量升级脚本: upgrade_20260303_action_products_enrichment.sql
-- 适用范围: 已存在 promotion_action_products 表，但缺少活动商品增强展示字段的历史环境
-- 执行前检查:
--   1) 确认目标库为 ozon-manager 业务库
--   2) 确认应用版本包含对应后端/前端字段读取逻辑
-- 失败处理建议:
--   - 若中途失败，先回滚当前事务后修正异常再重试
--   - 本脚本采用 IF NOT EXISTS，支持重复执行
-- ============================================================

BEGIN;

ALTER TABLE promotion_action_products
  ADD COLUMN IF NOT EXISTS offer_id VARCHAR(120),
  ADD COLUMN IF NOT EXISTS platform_sku VARCHAR(120),
  ADD COLUMN IF NOT EXISTS name_cn VARCHAR(500),
  ADD COLUMN IF NOT EXISTS name_origin VARCHAR(500),
  ADD COLUMN IF NOT EXISTS thumbnail_url TEXT,
  ADD COLUMN IF NOT EXISTS category_name VARCHAR(200),
  ADD COLUMN IF NOT EXISTS currency VARCHAR(10),
  ADD COLUMN IF NOT EXISTS base_price DECIMAL(12, 2),
  ADD COLUMN IF NOT EXISTS marketplace_price DECIMAL(12, 2),
  ADD COLUMN IF NOT EXISTS min_seller_price DECIMAL(12, 2),
  ADD COLUMN IF NOT EXISTS max_action_price DECIMAL(12, 2),
  ADD COLUMN IF NOT EXISTS discount_percent DECIMAL(6, 2),
  ADD COLUMN IF NOT EXISTS seller_stock INTEGER DEFAULT 0,
  ADD COLUMN IF NOT EXISTS ozon_stock INTEGER DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_promotion_action_products_offer_id ON promotion_action_products(offer_id);
CREATE INDEX IF NOT EXISTS idx_promotion_action_products_platform_sku ON promotion_action_products(platform_sku);

COMMIT;
