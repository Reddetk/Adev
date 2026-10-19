-- Configuration that a target system needs before a pipeline can start.

ALTER TABLE target_systems ADD COLUMN IF NOT EXISTS workspace_dir TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS agent_configs (
    id               BIGSERIAL PRIMARY KEY,
    target_system_id BIGINT NOT NULL,
    role             TEXT NOT NULL,
    provider         TEXT NOT NULL,
    base_url         TEXT NOT NULL DEFAULT '',
    model            TEXT NOT NULL,
    api_key          TEXT NOT NULL DEFAULT '',
    temperature      NUMERIC(3, 2) NOT NULL DEFAULT 0,
    max_tokens       INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (target_system_id, role)
);
