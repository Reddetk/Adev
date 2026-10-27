-- One feature-design pipeline per target system: the greenfield route builds
-- the baseline once, change requests walk their own routes afterwards.
CREATE UNIQUE INDEX IF NOT EXISTS pipelines_greenfield_unique_idx
    ON pipelines (target_system_id)
    WHERE route = 'greenfield';
