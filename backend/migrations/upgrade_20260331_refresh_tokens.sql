-- ============================================================
-- 适用范围：为现有数据库补齐 refresh token 持久化能力
-- 执行前检查：确认 users 表已存在
-- 失败处理建议：若执行中断，可修复后重复执行；脚本按幂等方式编写
-- ============================================================

CREATE TABLE IF NOT EXISTS user_refresh_tokens (
    id                    SERIAL PRIMARY KEY,
    user_id               INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash            VARCHAR(128) NOT NULL,
    family_id             VARCHAR(64) NOT NULL,
    user_agent            TEXT,
    ip_address            VARCHAR(64),
    issued_at             TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at            TIMESTAMP NOT NULL,
    last_used_at          TIMESTAMP,
    revoked_at            TIMESTAMP,
    revoke_reason         VARCHAR(100),
    replaced_by_token_id  INTEGER REFERENCES user_refresh_tokens(id) ON DELETE SET NULL,
    created_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_refresh_tokens_token_hash ON user_refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_user_refresh_tokens_user_id ON user_refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_user_refresh_tokens_family_id ON user_refresh_tokens(family_id);
CREATE INDEX IF NOT EXISTS idx_user_refresh_tokens_expires_at ON user_refresh_tokens(expires_at);
