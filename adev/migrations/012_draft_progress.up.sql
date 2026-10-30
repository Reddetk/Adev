-- A draft is the document being assembled: the stage writes it entry by entry and
-- rewrites the same version until the gates accept it. Only an accepted version
-- stays append-only, so a long detail walk can save its progress after every entry
-- instead of stacking one version per model call.
CREATE OR REPLACE FUNCTION forms_block_payload_change() RETURNS trigger AS $$
BEGIN
    IF NEW.inputs IS DISTINCT FROM OLD.inputs
    OR NEW.ops    IS DISTINCT FROM OLD.ops
    OR NEW.base   IS DISTINCT FROM OLD.base THEN
        RAISE EXCEPTION 'forms are append-only: inputs, ops and base cannot change';
    END IF;
    IF OLD.status NOT IN ('draft', 'waiting_operator')
    AND (NEW.payload IS DISTINCT FROM OLD.payload OR NEW.payload_hash IS DISTINCT FROM OLD.payload_hash) THEN
        RAISE EXCEPTION 'forms are append-only: payload and payload_hash change only before the form is accepted';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
