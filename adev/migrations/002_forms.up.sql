-- Append-only form versions.
-- A stored version is never edited. A correction is a new row with previous_form set.

CREATE TABLE IF NOT EXISTS forms (
    form_id          TEXT        NOT NULL,
    form_version     INTEGER     NOT NULL CHECK (form_version >= 1),
    form_type        TEXT        NOT NULL,
    schema_version   INTEGER     NOT NULL CHECK (schema_version >= 1),
    previous_form    JSONB,
    status           TEXT        NOT NULL,
    route            TEXT        NOT NULL,
    stage            TEXT        NOT NULL,
    mode             TEXT        NOT NULL,
    base             JSONB,
    ops              JSONB,
    pipeline_id      TEXT        NOT NULL,
    target_system_id TEXT        NOT NULL,
    produced_by      TEXT        NOT NULL,
    inputs           JSONB       NOT NULL DEFAULT '[]'::jsonb,
    payload          JSONB       NOT NULL,
    payload_hash     TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (form_id, form_version)
);

CREATE INDEX IF NOT EXISTS forms_pipeline_type_idx
    ON forms (pipeline_id, form_type, form_version);

CREATE INDEX IF NOT EXISTS forms_target_type_idx
    ON forms (target_system_id, form_type, form_version);

CREATE OR REPLACE FUNCTION forms_block_payload_change() RETURNS trigger AS $$
BEGIN
    IF NEW.payload      IS DISTINCT FROM OLD.payload
    OR NEW.payload_hash IS DISTINCT FROM OLD.payload_hash
    OR NEW.inputs       IS DISTINCT FROM OLD.inputs
    OR NEW.ops          IS DISTINCT FROM OLD.ops
    OR NEW.base         IS DISTINCT FROM OLD.base THEN
        RAISE EXCEPTION 'forms are append-only: payload, payload_hash, inputs, ops and base cannot change';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS forms_no_payload_change ON forms;
CREATE TRIGGER forms_no_payload_change
    BEFORE UPDATE ON forms
    FOR EACH ROW EXECUTE FUNCTION forms_block_payload_change();
