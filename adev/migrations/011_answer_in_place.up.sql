-- An operator answer rewrites the waiting draft in place instead of stacking a new
-- version per answer. Only a version that waits for the operator may change its
-- payload; inputs, ops and base stay append-only.
CREATE OR REPLACE FUNCTION forms_block_payload_change() RETURNS trigger AS $$
BEGIN
    IF NEW.inputs IS DISTINCT FROM OLD.inputs
    OR NEW.ops    IS DISTINCT FROM OLD.ops
    OR NEW.base   IS DISTINCT FROM OLD.base THEN
        RAISE EXCEPTION 'forms are append-only: inputs, ops and base cannot change';
    END IF;
    IF OLD.status <> 'waiting_operator'
    AND (NEW.payload IS DISTINCT FROM OLD.payload OR NEW.payload_hash IS DISTINCT FROM OLD.payload_hash) THEN
        RAISE EXCEPTION 'forms are append-only: payload and payload_hash change only while waiting for the operator';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
