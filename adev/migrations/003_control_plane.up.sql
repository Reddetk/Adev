-- Control plane tables for adev.
-- Replaces the removed GORM auto-migration. Apply with: make migrate-up

CREATE TABLE IF NOT EXISTS target_systems (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    repo_url    TEXT NOT NULL,
    description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pipelines (
    id               BIGSERIAL PRIMARY KEY,
    target_system_id BIGINT NOT NULL,
    route            TEXT NOT NULL,
    status           TEXT NOT NULL,
    current_stage    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS pipelines_target_system_idx ON pipelines (target_system_id);

CREATE TABLE IF NOT EXISTS form_records (
    id           BIGSERIAL PRIMARY KEY,
    form_id      TEXT NOT NULL,
    form_type    TEXT NOT NULL,
    form_version INTEGER NOT NULL,
    stage        TEXT NOT NULL,
    status       TEXT NOT NULL,
    payload_hash TEXT NOT NULL,
    produced_by  TEXT NOT NULL,
    pipeline_id  BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS form_records_pipeline_idx ON form_records (pipeline_id, form_type, form_version);

CREATE TABLE IF NOT EXISTS runs (
    id            BIGSERIAL PRIMARY KEY,
    pipeline_id   BIGINT NOT NULL,
    stage         TEXT NOT NULL,
    status        TEXT NOT NULL,
    checkpoint_id TEXT NOT NULL,
    attempts      INTEGER NOT NULL DEFAULT 0,
    last_error    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS runs_pipeline_idx ON runs (pipeline_id, stage);

CREATE TABLE IF NOT EXISTS operator_requests (
    id          BIGSERIAL PRIMARY KEY,
    pipeline_id BIGINT NOT NULL,
    pointer     TEXT NOT NULL,
    question    TEXT NOT NULL,
    blocking    BOOLEAN NOT NULL DEFAULT false,
    status      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS operator_requests_pipeline_idx ON operator_requests (pipeline_id, status);

CREATE TABLE IF NOT EXISTS form_refs (
    id           BIGSERIAL PRIMARY KEY,
    form_id      TEXT NOT NULL,
    form_version INTEGER NOT NULL,
    ref_form_id  TEXT NOT NULL,
    ref_version  INTEGER NOT NULL,
    pointer      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS form_refs_form_idx ON form_refs (form_id, form_version);

CREATE TABLE IF NOT EXISTS object_entries (
    id               BIGSERIAL PRIMARY KEY,
    target_system_id BIGINT NOT NULL,
    kind             TEXT NOT NULL,
    name             TEXT NOT NULL,
    path             TEXT NOT NULL,
    container_id     TEXT NOT NULL,
    status           TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS object_entries_target_idx ON object_entries (target_system_id, kind, status);

CREATE TABLE IF NOT EXISTS tasks (
    id           BIGSERIAL PRIMARY KEY,
    pipeline_id  BIGINT NOT NULL,
    parent_id    BIGINT NOT NULL DEFAULT 0,
    owner        TEXT NOT NULL,
    task_type    TEXT NOT NULL,
    status       TEXT NOT NULL,
    contract_ref TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS tasks_pipeline_idx ON tasks (pipeline_id, owner, status);

CREATE TABLE IF NOT EXISTS model_calls (
    id         BIGSERIAL PRIMARY KEY,
    run_id     BIGINT NOT NULL,
    profile    TEXT NOT NULL,
    model      TEXT NOT NULL,
    tokens_in  INTEGER NOT NULL DEFAULT 0,
    tokens_out INTEGER NOT NULL DEFAULT 0,
    cost       NUMERIC(12, 6) NOT NULL DEFAULT 0,
    latency_ms INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS model_calls_run_idx ON model_calls (run_id, profile);

CREATE TABLE IF NOT EXISTS audit_events (
    id          BIGSERIAL PRIMARY KEY,
    pipeline_id BIGINT NOT NULL,
    run_id      BIGINT NOT NULL,
    event_type  TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS audit_events_pipeline_idx ON audit_events (pipeline_id, created_at);
