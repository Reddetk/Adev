-- The greenfield route takes 04 back: the two statements below only moved data,
-- and a route with the stage again finds its own cursor.
UPDATE pipelines SET current_stage = '04' WHERE current_stage = '05' AND route = 'greenfield' AND status <> 'completed';
