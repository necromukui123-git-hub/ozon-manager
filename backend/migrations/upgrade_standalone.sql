-- =================================================================================
-- 这个脚本包含了最近这几周增加的所有 Automation/Promotion 相关的数据库表结构
-- 供独立运行更新。执行完毕若无报错，说明已升级完毕，可安全删除此文件。
-- =================================================================================

-- 1. 添加用户 owner_id 列
ALTER TABLE users ADD COLUMN IF NOT EXISTS owner_id INTEGER;
CREATE INDEX IF NOT EXISTS idx_users_owner_id ON users(owner_id);

-- 2. 添加店铺 owner_id 列
ALTER TABLE shops ADD COLUMN IF NOT EXISTS owner_id INTEGER;
ALTER TABLE shops ADD COLUMN IF NOT EXISTS execution_engine_mode VARCHAR(20) NOT NULL DEFAULT 'auto';
CREATE INDEX IF NOT EXISTS idx_shops_owner_id ON shops(owner_id);

-- 3. 增强 promotion_actions 表（支持官方和店铺促销）
ALTER TABLE promotion_actions ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'official';
ALTER TABLE promotion_actions ADD COLUMN IF NOT EXISTS source_action_id VARCHAR(120);

UPDATE promotion_actions SET source_action_id = CAST(action_id AS VARCHAR(120)) WHERE source_action_id IS NULL OR source_action_id = '';

ALTER TABLE promotion_actions ALTER COLUMN source_action_id SET NOT NULL;
ALTER TABLE promotion_actions ADD COLUMN IF NOT EXISTS source_payload JSONB;
ALTER TABLE promotion_actions ADD COLUMN IF NOT EXISTS last_products_synced_at TIMESTAMP;

CREATE UNIQUE INDEX IF NOT EXISTS idx_shop_source_action ON promotion_actions(shop_id, source, source_action_id);
CREATE INDEX IF NOT EXISTS idx_promotion_actions_source ON promotion_actions(source);
CREATE INDEX IF NOT EXISTS idx_promotion_actions_source_action_id ON promotion_actions(source_action_id);

-- 4. 添加 automation_agents 表
CREATE TABLE IF NOT EXISTS automation_agents (
    id SERIAL PRIMARY KEY,
    agent_key VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    hostname VARCHAR(200),
    status VARCHAR(20) NOT NULL DEFAULT 'offline',
    capabilities JSONB,
    last_heartbeat_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_automation_agents_status ON automation_agents(status);

-- 5. 添加 automation_jobs 表
CREATE TABLE IF NOT EXISTS automation_jobs (
    id SERIAL PRIMARY KEY,
    shop_id INTEGER NOT NULL REFERENCES shops(id),
    created_by INTEGER NOT NULL REFERENCES users(id),
    assigned_agent_id INTEGER REFERENCES automation_agents(id),
    job_type VARCHAR(50) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    dry_run BOOLEAN DEFAULT false,
    requires_confirmation BOOLEAN DEFAULT false,
    rate_limit INTEGER DEFAULT 30,
    total_items INTEGER DEFAULT 0,
    success_items INTEGER DEFAULT 0,
    failed_items INTEGER DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_automation_jobs_shop_id ON automation_jobs(shop_id);
CREATE INDEX IF NOT EXISTS idx_automation_jobs_status ON automation_jobs(status);
CREATE INDEX IF NOT EXISTS idx_automation_jobs_created_by ON automation_jobs(created_by);
CREATE INDEX IF NOT EXISTS idx_automation_jobs_assigned_agent_id ON automation_jobs(assigned_agent_id);

-- 6. 添加 automation_job_items 表
CREATE TABLE IF NOT EXISTS automation_job_items (
    id SERIAL PRIMARY KEY,
    job_id INTEGER NOT NULL REFERENCES automation_jobs(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    source_sku VARCHAR(100) NOT NULL,
    target_price DECIMAL(12,2) NOT NULL,
    overall_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    step_exit_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    step_reprice_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    step_readd_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    step_exit_error TEXT,
    step_reprice_error TEXT,
    step_readd_error TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(job_id, source_sku)
);
CREATE INDEX IF NOT EXISTS idx_automation_job_items_job_id ON automation_job_items(job_id);
CREATE INDEX IF NOT EXISTS idx_automation_job_items_product_id ON automation_job_items(product_id);
CREATE INDEX IF NOT EXISTS idx_automation_job_items_overall_status ON automation_job_items(overall_status);

-- 7. 添加 automation_job_events 和 artifacts 表
CREATE TABLE IF NOT EXISTS automation_job_events (
    id SERIAL PRIMARY KEY,
    job_id INTEGER NOT NULL REFERENCES automation_jobs(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    message TEXT,
    payload JSONB,
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_automation_job_events_job_id ON automation_job_events(job_id);

CREATE TABLE IF NOT EXISTS automation_artifacts (
    id SERIAL PRIMARY KEY,
    job_id INTEGER NOT NULL REFERENCES automation_jobs(id) ON DELETE CASCADE,
    job_item_id INTEGER REFERENCES automation_job_items(id) ON DELETE SET NULL,
    artifact_type VARCHAR(50) NOT NULL,
    storage_path VARCHAR(500) NOT NULL,
    checksum VARCHAR(128),
    meta JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_automation_artifacts_job_id ON automation_artifacts(job_id);

-- 8. 初始化默认 super_admin (admin123)
-- 注意: 如果您已经修改了系统管理员的密码或账号，下面的 INSERT 不会执行也不覆盖原数据
INSERT INTO users (username, password_hash, display_name, role, status)
SELECT 
    'super_admin', 
    '$2a$10$ylb8XwllNQUWAlciq5nxiev6eFJk4FqSQmU2XI04Pg9qi2rb178wq', 
    '系统管理员', 
    'super_admin', 
    'active'
WHERE NOT EXISTS (
    SELECT 1 FROM users WHERE role = 'super_admin'
);
