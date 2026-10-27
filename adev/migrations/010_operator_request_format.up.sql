-- The input control the operator question expects: choice, number, list or text.
ALTER TABLE operator_requests ADD COLUMN IF NOT EXISTS format TEXT NOT NULL DEFAULT '';
