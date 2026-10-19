package domain

// GetModelCallSeeds returns sample data for modelcall
func GetModelCallSeeds() []ModelCall {
	return []ModelCall{
		{
			RunID:     10,
			Profile:   "Sample Profile 1",
			Model:     "Sample Model 1",
			TokensIn:  10,
			TokensOut: 10,
			Cost:      10.50,
			LatencyMs: 10,
		},
		{
			RunID:     20,
			Profile:   "Sample Profile 2",
			Model:     "Sample Model 2",
			TokensIn:  20,
			TokensOut: 20,
			Cost:      21.00,
			LatencyMs: 20,
		},
		{
			RunID:     30,
			Profile:   "Sample Profile 3",
			Model:     "Sample Model 3",
			TokensIn:  30,
			TokensOut: 30,
			Cost:      31.50,
			LatencyMs: 30,
		},
	}
}

// GetSQLModelCallSeeds returns SQL INSERT statements for modelcall
func GetSQLModelCallSeeds() string {
	return `-- Sample data for table modelcall
INSERT INTO modelcalls (runid, profile, model, tokensin, tokensout, cost, latencyms) VALUES (10, 'Sample Profile 1', 'Sample Model 1', 10, 10, 10.50, 10);
INSERT INTO modelcalls (runid, profile, model, tokensin, tokensout, cost, latencyms) VALUES (20, 'Sample Profile 2', 'Sample Model 2', 20, 20, 21.00, 20);
INSERT INTO modelcalls (runid, profile, model, tokensin, tokensout, cost, latencyms) VALUES (30, 'Sample Profile 3', 'Sample Model 3', 30, 30, 31.50, 30);
`
}
