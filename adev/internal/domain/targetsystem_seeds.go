package domain

// GetTargetSystemSeeds returns sample data for targetsystem
func GetTargetSystemSeeds() []TargetSystem {
	return []TargetSystem{
		{
			Name:        "John Smith",
			RepoURL:     "Sample RepoURL 1",
			Description: "Detailed description of the first item",
		},
		{
			Name:        "Jane Doe",
			RepoURL:     "Sample RepoURL 2",
			Description: "Complete information about the second item",
		},
		{
			Name:        "Bob Johnson",
			RepoURL:     "Sample RepoURL 3",
			Description: "Specific details of the third record",
		},
	}
}

// GetSQLTargetSystemSeeds returns SQL INSERT statements for targetsystem
func GetSQLTargetSystemSeeds() string {
	return `-- Sample data for table targetsystem
INSERT INTO targetsystems (name, repourl, description) VALUES ('John Smith', 'Sample RepoURL 1', 'Detailed description of the first item');
INSERT INTO targetsystems (name, repourl, description) VALUES ('Jane Doe', 'Sample RepoURL 2', 'Complete information about the second item');
INSERT INTO targetsystems (name, repourl, description) VALUES ('Bob Johnson', 'Sample RepoURL 3', 'Specific details of the third record');
`
}
