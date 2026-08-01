-- 192_add_upstreams_table.sql
-- 新建独立的 upstreams 表，将上游从 accounts 表中解耦

CREATE TABLE IF NOT EXISTS upstreams (
    id               BIGSERIAL PRIMARY KEY,
    name             VARCHAR(100) NOT NULL,
    platform         VARCHAR(20)  NOT NULL,           -- sub2api | newapi
    base_url         VARCHAR(500) NOT NULL,
    email            VARCHAR(200) NOT NULL,
    password         TEXT         NOT NULL,            -- 明文存储（后续可替换加密）

    -- session 状态（由 UpstreamSyncWorker 每分钟维护）
    access_token     TEXT,
    refresh_token    TEXT,
    expires_at       TIMESTAMPTZ,
    session_cookie   TEXT,
    upstream_user_id BIGINT,

    -- 同步元数据
    groups           JSONB        NOT NULL DEFAULT '[]',
    balance          DECIMAL(20,6) NOT NULL DEFAULT 0,
    health           VARCHAR(20)  NOT NULL DEFAULT 'pending',  -- pending | ok | error | syncing
    health_msg       TEXT,
    last_synced_at   TIMESTAMPTZ,

    -- 审计字段
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX idx_upstreams_platform   ON upstreams(platform);
CREATE INDEX idx_upstreams_deleted_at ON upstreams(deleted_at);
CREATE INDEX idx_upstreams_health     ON upstreams(health) WHERE deleted_at IS NULL;
