package domain

import (
	"time"
)

type AuditEvent struct {
	ID         uint      `json:"id"`
	PipelineID int       `json:"pipelineid" validate:"required,gte=0"`
	RunID      int       `json:"runid" validate:"required,gte=0"`
	EventType  string    `json:"eventtype" validate:"required"`
	CreatedAt  time.Time `json:"createdat" validate:"required"`
}

func (a *AuditEvent) Validate() error {
	if a.PipelineID < 0 {
		return ErrInvalidAuditEventPipelineID
	}
	if a.RunID < 0 {
		return ErrInvalidAuditEventRunID
	}
	if a.EventType == "" {
		return ErrInvalidAuditEventEventType
	}
	return nil
}
