-- 193_add_upstream_id_to_accounts.sql
-- accounts 表新增 upstream_id 外键（仅上游类型账号有值）
-- upstream_group 和 auto_priority 存入现有 extra JSONB，无需新列

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS upstream_id BIGINT REFERENCES upstreams(id) ON DELETE RESTRICT;

CREATE INDEX idx_accounts_upstream_id
    ON accounts(upstream_id)
    WHERE upstream_id IS NOT NULL;
