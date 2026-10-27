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
