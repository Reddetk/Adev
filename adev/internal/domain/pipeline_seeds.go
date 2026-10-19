package domain

// GetPipelineSeeds returns sample data for pipeline
func GetPipelineSeeds() []Pipeline {
	return []Pipeline{
		{
			TargetSystemID: 10,
			Route:          "Sample Route 1",
			Status:         "active",
			CurrentStage:   "Sample CurrentStage 1",
		},
		{
			TargetSystemID: 20,
			Route:          "Sample Route 2",
			Status:         "pending",
			CurrentStage:   "Sample CurrentStage 2",
		},
		{
			TargetSystemID: 30,
			Route:          "Sample Route 3",
			Status:         "completed",
			CurrentStage:   "Sample CurrentStage 3",
		},
	}
}

// GetSQLPipelineSeeds returns SQL INSERT statements for pipeline
func GetSQLPipelineSeeds() string {
	return `-- Sample data for table pipeline
INSERT INTO pipelines (targetsystemid, route, status, currentstage) VALUES (10, 'Sample Route 1', 'active', 'Sample CurrentStage 1');
INSERT INTO pipelines (targetsystemid, route, status, currentstage) VALUES (20, 'Sample Route 2', 'pending', 'Sample CurrentStage 2');
INSERT INTO pipelines (targetsystemid, route, status, currentstage) VALUES (30, 'Sample Route 3', 'completed', 'Sample CurrentStage 3');
`
}
