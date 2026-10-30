-- The greenfield route no longer has stage 04: a design keeps its decisions in the
-- HLD, so the next stage after 03 is 05. A pipeline that sat on 04 continues there,
-- and a run that still holds 04 is closed instead of resuming a stage that the
-- route cannot place.
UPDATE pipelines SET current_stage = '05' WHERE current_stage = '04' AND route = 'greenfield';

UPDATE runs
SET status = 'failed', last_error = 'stage 04 is not part of the greenfield route any more'
WHERE stage = '04' AND status = 'running' AND checkpoint_id = 'greenfield';
