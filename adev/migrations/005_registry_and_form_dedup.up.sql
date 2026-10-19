-- Remove the duplicated control-plane tables and rebuild object_entries for domain.Object.
-- form_records is replaced by forms (append-only versions with JSONB payload).
-- form_refs is replaced by forms.inputs.

DROP TABLE IF EXISTS form_records;
DROP TABLE IF EXISTS form_refs;
DROP TABLE IF EXISTS object_entries;

CREATE TABLE object_entries (
    object_id        TEXT PRIMARY KEY,
    target_system_id BIGINT NOT NULL,
    kind             TEXT NOT NULL,
    name             TEXT NOT NULL,
    path             TEXT NOT NULL DEFAULT '',
    container_id     TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL,
    fields           JSONB NOT NULL DEFAULT '{}'::jsonb,
    introduced_by    JSONB NOT NULL,
    changed_by       JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE INDEX IF NOT EXISTS object_entries_target_idx ON object_entries (target_system_id, kind, status);
