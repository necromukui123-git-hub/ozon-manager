-- Ozon店铺管理系统 - 数据库迁移脚本

-- 用户表：管理员和员工账号
CREATE TABLE IF NOT EXISTS users (
    id              SERIAL PRIMARY KEY,
    username        VARCHAR(50) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    display_name    VARCHAR(100) NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'staff',
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    last_login_at   TIMESTAMP,
    created_by      INTEGER REFERENCES users(id),
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 店铺表：支持多店铺管理
CREATE TABLE IF NOT EXISTS shops (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    client_id       VARCHAR(50) NOT NULL UNIQUE,
    api_key         VARCHAR(200) NOT NULL,
    is_active       BOOLEAN DEFAULT true,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 用户-店铺关联表：员工可访问的店铺（多对多）
CREATE TABLE IF NOT EXISTS user_shops (
    id              SERIAL PRIMARY KEY,
    user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shop_id         INTEGER NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, shop_id)
);

-- 商品表
CREATE TABLE IF NOT EXISTS products (
    id                  SERIAL PRIMARY KEY,
    shop_id             INTEGER NOT NULL REFERENCES shops(id),
    ozon_product_id     BIGINT NOT NULL,
    ozon_sku            BIGINT,
    source_sku          VARCHAR(100) NOT NULL,
    name                VARCHAR(500),
    current_price       DECIMAL(12, 2),
    status              VARCHAR(20) DEFAULT 'active',
    is_loss             BOOLEAN DEFAULT false,
    is_promoted         BOOLEAN DEFAULT false,
    last_synced_at      TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shop_id, ozon_product_id),
    UNIQUE(shop_id, source_sku)
);

-- 亏损商品表
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

-- 已推广商品表
CREATE TABLE IF NOT EXISTS promoted_products (
    id                  SERIAL PRIMARY KEY,
    product_id          INTEGER NOT NULL REFERENCES products(id),
    promotion_type      VARCHAR(50) NOT NULL,
    action_id           BIGINT,
    action_price        DECIMAL(12, 2),
    status              VARCHAR(20) DEFAULT 'active',
    promoted_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    exited_at           TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, promotion_type, action_id)
);

-- 促销活动缓存表
CREATE TABLE IF NOT EXISTS promotion_actions (
    id                  SERIAL PRIMARY KEY,
    shop_id             INTEGER NOT NULL REFERENCES shops(id),
    action_id           BIGINT NOT NULL,
    title               VARCHAR(200),
    action_type         VARCHAR(50),
    date_start          TIMESTAMP,
    date_end            TIMESTAMP,
    is_elastic_boost    BOOLEAN DEFAULT false,
    is_discount_28      BOOLEAN DEFAULT false,
    last_synced_at      TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shop_id, action_id)
);

-- 操作日志表
CREATE TABLE IF NOT EXISTS operation_logs (
    id                  SERIAL PRIMARY KEY,
    user_id             INTEGER NOT NULL REFERENCES users(id),
    shop_id             INTEGER REFERENCES shops(id),
    operation_type      VARCHAR(50) NOT NULL,
    operation_detail    JSONB,
    affected_count      INTEGER DEFAULT 0,
    status              VARCHAR(20) DEFAULT 'pending',
    error_message       TEXT,
    ip_address          VARCHAR(45),
    user_agent          VARCHAR(500),
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at        TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_products_shop_id ON products(shop_id);
CREATE INDEX IF NOT EXISTS idx_products_source_sku ON products(source_sku);
CREATE INDEX IF NOT EXISTS idx_products_is_loss ON products(is_loss);
CREATE INDEX IF NOT EXISTS idx_products_is_promoted ON products(is_promoted);
CREATE INDEX IF NOT EXISTS idx_loss_products_product_id ON loss_products(product_id);
CREATE INDEX IF NOT EXISTS idx_loss_products_loss_date ON loss_products(loss_date);
CREATE INDEX IF NOT EXISTS idx_promoted_products_product_id ON promoted_products(product_id);
CREATE INDEX IF NOT EXISTS idx_promoted_products_status ON promoted_products(status);
CREATE INDEX IF NOT EXISTS idx_operation_logs_user_id ON operation_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_operation_logs_created_at ON operation_logs(created_at);

-- 默认管理员账号（密码: admin123）
INSERT INTO users (username, password_hash, display_name, role, status)
VALUES ('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqBuBk0F.Gc7YMG.T9D.Z2OVOQHMu', '系统管理员', 'admin', 'active')
ON CONFLICT (username) DO NOTHING;
