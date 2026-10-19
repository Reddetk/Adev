DROP TABLE IF EXISTS object_entries;

CREATE TABLE object_entries (
    id               BIGSERIAL PRIMARY KEY,
    target_system_id BIGINT NOT NULL,
    kind             TEXT NOT NULL,
    name             TEXT NOT NULL,
    path             TEXT NOT NULL,
    container_id     TEXT NOT NULL,
    status           TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS object_entries_target_idx ON object_entries (target_system_id, kind, status);

CREATE TABLE form_records (
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

CREATE TABLE form_refs (
    id           BIGSERIAL PRIMARY KEY,
    form_id      TEXT NOT NULL,
    form_version INTEGER NOT NULL,
    ref_form_id  TEXT NOT NULL,
    ref_version  INTEGER NOT NULL,
    pointer      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS form_refs_form_idx ON form_refs (form_id, form_version);
