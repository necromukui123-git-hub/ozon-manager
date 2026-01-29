-- ============================================================
-- Ozon Shop Manager - 数据库初始化脚本
-- 版本: 1.0
-- 说明: 包含所有表结构和必要的初始数据，用于快速部署
-- 使用方法: psql -U postgres -d ozon_manager -f init_database.sql
-- ============================================================

-- 创建数据库（如果需要，取消下面两行注释）
-- CREATE DATABASE ozon_manager WITH ENCODING 'UTF8';
-- \c ozon_manager

-- ============================================================
-- 1. 用户表
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id              SERIAL PRIMARY KEY,
    username        VARCHAR(50) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    display_name    VARCHAR(100) NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'staff',  -- super_admin / shop_admin / staff
    status          VARCHAR(20) NOT NULL DEFAULT 'active', -- active / disabled
    last_login_at   TIMESTAMP,
    created_by      INTEGER REFERENCES users(id),
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 2. 店铺表
-- ============================================================
CREATE TABLE IF NOT EXISTS shops (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    client_id       VARCHAR(50) NOT NULL,
    api_key         VARCHAR(200) NOT NULL,
    is_active       BOOLEAN DEFAULT true,
    owner_id        INTEGER REFERENCES users(id),
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- 3. 用户-店铺关联表（员工与店铺的多对多关系）
-- ============================================================
CREATE TABLE IF NOT EXISTS user_shops (
    id              SERIAL PRIMARY KEY,
    user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shop_id         INTEGER NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, shop_id)
);

-- ============================================================
-- 4. 商品表
-- ============================================================
CREATE TABLE IF NOT EXISTS products (
    id                  SERIAL PRIMARY KEY,
    shop_id             INTEGER NOT NULL REFERENCES shops(id),
    ozon_product_id     BIGINT NOT NULL,
    ozon_sku            BIGINT,
    source_sku          VARCHAR(100) NOT NULL,
    name                VARCHAR(500),
    current_price       DECIMAL(12, 2),
    status              VARCHAR(20) DEFAULT 'active',  -- active / inactive / archived
    is_loss             BOOLEAN DEFAULT false,
    is_promoted         BOOLEAN DEFAULT false,
    last_synced_at      TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shop_id, ozon_product_id),
    UNIQUE(shop_id, source_sku)
);

-- ============================================================
-- 5. 亏损商品表
-- ============================================================
CREATE TABLE IF NOT EXISTS loss_products (
    id                  SERIAL PRIMARY KEY,
    product_id          INTEGER NOT NULL REFERENCES products(id),
    loss_date           DATE NOT NULL,
    original_price      DECIMAL(12, 2),
    new_price           DECIMAL(12, 2) NOT NULL,
    price_updated       BOOLEAN DEFAULT false,
    promotion_exited    BOOLEAN DEFAULT false,
    promotion_rejoined  BOOLEAN DEFAULT false,
    processed_at        TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, loss_date)
);

-- ============================================================
-- 6. 已推广商品表
-- ============================================================
CREATE TABLE IF NOT EXISTS promoted_products (
    id                  SERIAL PRIMARY KEY,
    product_id          INTEGER NOT NULL REFERENCES products(id),
    promotion_type      VARCHAR(50) NOT NULL,
    action_id           BIGINT,
    action_price        DECIMAL(12, 2),
    status              VARCHAR(20) DEFAULT 'active',  -- active / exited / pending
    promoted_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    exited_at           TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, promotion_type, action_id)
);

-- ============================================================
-- 7. 促销活动缓存表
-- ============================================================
CREATE TABLE IF NOT EXISTS promotion_actions (
    id                  SERIAL PRIMARY KEY,
    shop_id             INTEGER NOT NULL REFERENCES shops(id),
    action_id           BIGINT NOT NULL,
    title               VARCHAR(200),
    display_name        VARCHAR(200),  -- 自定义中文显示名称
    action_type         VARCHAR(50),
    date_start          TIMESTAMP,
    date_end            TIMESTAMP,
    participating_count INTEGER DEFAULT 0,
    potential_count     INTEGER DEFAULT 0,
    is_manual           BOOLEAN DEFAULT false,
    status              VARCHAR(20) DEFAULT 'active',  -- active / expired / disabled
    last_synced_at      TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shop_id, action_id)
);

-- ============================================================
-- 8. 操作日志表
-- ============================================================
CREATE TABLE IF NOT EXISTS operation_logs (
    id                  SERIAL PRIMARY KEY,
    user_id             INTEGER NOT NULL REFERENCES users(id),
    shop_id             INTEGER REFERENCES shops(id),
    operation_type      VARCHAR(50) NOT NULL,
    operation_detail    JSONB,
    affected_count      INTEGER DEFAULT 0,
    status              VARCHAR(20) DEFAULT 'pending',  -- pending / success / failed
    error_message       TEXT,
    ip_address          VARCHAR(45),
    user_agent          VARCHAR(500),
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at        TIMESTAMP
);

-- ============================================================
-- 索引
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_products_shop_id ON products(shop_id);
CREATE INDEX IF NOT EXISTS idx_products_source_sku ON products(source_sku);
CREATE INDEX IF NOT EXISTS idx_products_is_loss ON products(is_loss);
CREATE INDEX IF NOT EXISTS idx_products_is_promoted ON products(is_promoted);
CREATE INDEX IF NOT EXISTS idx_shops_owner_id ON shops(owner_id);
CREATE INDEX IF NOT EXISTS idx_loss_products_product_id ON loss_products(product_id);
CREATE INDEX IF NOT EXISTS idx_loss_products_loss_date ON loss_products(loss_date);
CREATE INDEX IF NOT EXISTS idx_promoted_products_product_id ON promoted_products(product_id);
CREATE INDEX IF NOT EXISTS idx_promoted_products_status ON promoted_products(status);
CREATE INDEX IF NOT EXISTS idx_operation_logs_user_id ON operation_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_operation_logs_created_at ON operation_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_promotion_actions_shop_id ON promotion_actions(shop_id);

-- ============================================================
-- 初始数据：超级管理员账户
-- 用户名: super_admin
-- 密码: admin123
-- ============================================================
INSERT INTO users (username, password_hash, display_name, role, status)
VALUES ('super_admin', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqBuBk0F.Gc7YMG.T9D.Z2OVOQHMu', '系统管理员', 'super_admin', 'active')
ON CONFLICT (username) DO NOTHING;

-- ============================================================
-- 部署完成提示
-- ============================================================
DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Ozon Shop Manager 数据库初始化完成！';
    RAISE NOTICE '----------------------------------------';
    RAISE NOTICE '默认超级管理员账户:';
    RAISE NOTICE '  用户名: super_admin';
    RAISE NOTICE '  密码: admin123';
    RAISE NOTICE '----------------------------------------';
    RAISE NOTICE '请及时修改默认密码！';
    RAISE NOTICE '========================================';
END $$;
