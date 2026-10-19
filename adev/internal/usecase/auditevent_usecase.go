package usecase

import "github.com/Reddetk/Adev/internal/domain"

type AuditEventUseCase interface {
	CreateAuditEvent(input CreateAuditEventInput) (CreateAuditEventOutput, error)
	GetAuditEvent(id int) (*domain.AuditEvent, error)
	UpdateAuditEvent(id int, input UpdateAuditEventInput) error
	DeleteAuditEvent(id int) error
	ListAuditEvents() (ListAuditEventOutput, error)
}
