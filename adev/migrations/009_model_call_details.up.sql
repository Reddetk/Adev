-- Every attempt keeps its full dialogue with the model, so the operator can see
-- what the agent sent, what the model answered, and which gates returned it.
ALTER TABLE model_calls ADD COLUMN IF NOT EXISTS stage TEXT NOT NULL DEFAULT '';
ALTER TABLE model_calls ADD COLUMN IF NOT EXISTS request TEXT NOT NULL DEFAULT '';
ALTER TABLE model_calls ADD COLUMN IF NOT EXISTS response TEXT NOT NULL DEFAULT '';
ALTER TABLE model_calls ADD COLUMN IF NOT EXISTS errors TEXT NOT NULL DEFAULT '';
