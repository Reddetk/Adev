package usecase

import (
	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/messages"
)

type auditeventService struct {
	repo AuditEventRepository
}

func NewAuditEventService(repo AuditEventRepository) AuditEventUseCase {
	return &auditeventService{repo: repo}
}

func (a *auditeventService) CreateAuditEvent(input CreateAuditEventInput) (CreateAuditEventOutput, error) {
	auditevent := domain.AuditEvent{
		PipelineID: input.PipelineID,
		RunID:      input.RunID,
		EventType:  input.EventType,
		CreatedAt:  input.CreatedAt,
	}

	if err := auditevent.Validate(); err != nil {
		return CreateAuditEventOutput{}, err
	}

	if err := a.repo.Save(&auditevent); err != nil {
		return CreateAuditEventOutput{}, err
	}

	return CreateAuditEventOutput{
		ID:         auditevent.ID,
		PipelineID: auditevent.PipelineID,
		RunID:      auditevent.RunID,
		EventType:  auditevent.EventType,
		CreatedAt:  auditevent.CreatedAt,
		Message:    messages.AuditEventCreatedSuccessfully,
	}, nil
}

func (a *auditeventService) GetAuditEvent(id int) (*domain.AuditEvent, error) {
	return a.repo.FindByID(id)
}

func (a *auditeventService) UpdateAuditEvent(id int, input UpdateAuditEventInput) error {
	auditevent, err := a.repo.FindByID(id)
	if err != nil {
		return err
	}

	if input.PipelineID != nil {
		auditevent.PipelineID = *input.PipelineID
	}
	if input.RunID != nil {
		auditevent.RunID = *input.RunID
	}
	if input.EventType != nil {
		auditevent.EventType = *input.EventType
	}
	if input.CreatedAt != nil {
		auditevent.CreatedAt = *input.CreatedAt
	}

	return a.repo.Update(auditevent)
}

func (a *auditeventService) DeleteAuditEvent(id int) error {
	return a.repo.Delete(id)
}

func (a *auditeventService) ListAuditEvents() (ListAuditEventOutput, error) {
	auditevents, err := a.repo.FindAll()
	if err != nil {
		return ListAuditEventOutput{}, err
	}

	return ListAuditEventOutput{
		AuditEvents: auditevents,
		Total:       len(auditevents),
		Message:     messages.AuditEventsListedSuccessfully,
	}, nil
}
