package domain

// GetRunSeeds returns sample data for run
func GetRunSeeds() []Run {
	return []Run{
		{
			PipelineID:   10,
			Stage:        "Sample Stage 1",
			Status:       "active",
			CheckpointID: "Sample CheckpointID 1",
			Attempts:     10,
			LastError:    "Sample LastError 1",
		},
		{
			PipelineID:   20,
			Stage:        "Sample Stage 2",
			Status:       "pending",
			CheckpointID: "Sample CheckpointID 2",
			Attempts:     20,
			LastError:    "Sample LastError 2",
		},
		{
			PipelineID:   30,
			Stage:        "Sample Stage 3",
			Status:       "completed",
			CheckpointID: "Sample CheckpointID 3",
			Attempts:     30,
			LastError:    "Sample LastError 3",
		},
	}
}

// GetSQLRunSeeds returns SQL INSERT statements for run
func GetSQLRunSeeds() string {
	return `-- Sample data for table run
INSERT INTO runs (pipelineid, stage, status, checkpointid, attempts, lasterror) VALUES (10, 'Sample Stage 1', 'active', 'Sample CheckpointID 1', 10, 'Sample LastError 1');
INSERT INTO runs (pipelineid, stage, status, checkpointid, attempts, lasterror) VALUES (20, 'Sample Stage 2', 'pending', 'Sample CheckpointID 2', 20, 'Sample LastError 2');
INSERT INTO runs (pipelineid, stage, status, checkpointid, attempts, lasterror) VALUES (30, 'Sample Stage 3', 'completed', 'Sample CheckpointID 3', 30, 'Sample LastError 3');
`
}
