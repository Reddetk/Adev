package domain

// GetTaskSeeds returns sample data for task
func GetTaskSeeds() []Task {
	return []Task{
		{
			PipelineID:  10,
			ParentID:    10,
			Owner:       "Sample Owner 1",
			TaskType:    "Sample TaskType 1",
			Status:      "active",
			ContractRef: "Sample ContractRef 1",
		},
		{
			PipelineID:  20,
			ParentID:    20,
			Owner:       "Sample Owner 2",
			TaskType:    "Sample TaskType 2",
			Status:      "pending",
			ContractRef: "Sample ContractRef 2",
		},
		{
			PipelineID:  30,
			ParentID:    30,
			Owner:       "Sample Owner 3",
			TaskType:    "Sample TaskType 3",
			Status:      "completed",
			ContractRef: "Sample ContractRef 3",
		},
	}
}

// GetSQLTaskSeeds returns SQL INSERT statements for task
func GetSQLTaskSeeds() string {
	return `-- Sample data for table task
INSERT INTO tasks (pipelineid, parentid, owner, tasktype, status, contractref) VALUES (10, 10, 'Sample Owner 1', 'Sample TaskType 1', 'active', 'Sample ContractRef 1');
INSERT INTO tasks (pipelineid, parentid, owner, tasktype, status, contractref) VALUES (20, 20, 'Sample Owner 2', 'Sample TaskType 2', 'pending', 'Sample ContractRef 2');
INSERT INTO tasks (pipelineid, parentid, owner, tasktype, status, contractref) VALUES (30, 30, 'Sample Owner 3', 'Sample TaskType 3', 'completed', 'Sample ContractRef 3');
`
}
