package domain

// GetOperatorRequestSeeds returns sample data for operatorrequest
func GetOperatorRequestSeeds() []OperatorRequest {
	return []OperatorRequest{
		{
			PipelineID: 10,
			Pointer:    "Sample Pointer 1",
			Question:   "Sample Question 1",
			Blocking:   true,
			Status:     "active",
		},
		{
			PipelineID: 20,
			Pointer:    "Sample Pointer 2",
			Question:   "Sample Question 2",
			Blocking:   false,
			Status:     "pending",
		},
		{
			PipelineID: 30,
			Pointer:    "Sample Pointer 3",
			Question:   "Sample Question 3",
			Blocking:   true,
			Status:     "completed",
		},
	}
}

// GetSQLOperatorRequestSeeds returns SQL INSERT statements for operatorrequest
func GetSQLOperatorRequestSeeds() string {
	return `-- Sample data for table operatorrequest
INSERT INTO operatorrequests (pipelineid, pointer, question, blocking, status) VALUES (10, 'Sample Pointer 1', 'Sample Question 1', true, 'active');
INSERT INTO operatorrequests (pipelineid, pointer, question, blocking, status) VALUES (20, 'Sample Pointer 2', 'Sample Question 2', false, 'pending');
INSERT INTO operatorrequests (pipelineid, pointer, question, blocking, status) VALUES (30, 'Sample Pointer 3', 'Sample Question 3', true, 'completed');
`
}
