package domain

import "time"

// GetAuditEventSeeds returns sample data for auditevent
func GetAuditEventSeeds() []AuditEvent {
	return []AuditEvent{
		{
			PipelineID: 10,
			RunID:      10,
			EventType:  "Sample EventType 1",
			CreatedAt:  time.Now(),
		},
		{
			PipelineID: 20,
			RunID:      20,
			EventType:  "Sample EventType 2",
			CreatedAt:  time.Now(),
		},
		{
			PipelineID: 30,
			RunID:      30,
			EventType:  "Sample EventType 3",
			CreatedAt:  time.Now(),
		},
	}
}

// GetSQLAuditEventSeeds returns SQL INSERT statements for auditevent
func GetSQLAuditEventSeeds() string {
	return `-- Sample data for table auditevent
INSERT INTO auditevents (pipelineid, runid, eventtype, createdat) VALUES (10, 10, 'Sample EventType 1', NOW());
INSERT INTO auditevents (pipelineid, runid, eventtype, createdat) VALUES (20, 20, 'Sample EventType 2', NOW());
INSERT INTO auditevents (pipelineid, runid, eventtype, createdat) VALUES (30, 30, 'Sample EventType 3', NOW());
`
}
