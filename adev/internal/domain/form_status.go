package domain

var allowedStatusChanges = map[Status]map[Status]bool{
	StatusDraft:           {StatusValid: true, StatusRejected: true, StatusWaitingOperator: true},
	StatusValid:           {StatusWaitingOperator: true, StatusApproved: true, StatusRejected: true, StatusSuperseded: true},
	StatusWaitingOperator: {StatusValid: true, StatusApproved: true, StatusRejected: true},
	StatusApproved:        {StatusSuperseded: true},
	StatusRejected:        {},
	StatusSuperseded:      {},
}

// CanChangeStatus reports whether the form lifecycle allows the status change.
func CanChangeStatus(from, to Status) bool { return allowedStatusChanges[from][to] }
