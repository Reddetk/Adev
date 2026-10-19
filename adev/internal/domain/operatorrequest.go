package domain

type OperatorRequest struct {
	ID         uint   `json:"id"`
	PipelineID int    `json:"pipelineid" validate:"required,gte=0"`
	Pointer    string `json:"pointer" validate:"required"`
	Question   string `json:"question" validate:"required"`
	Blocking   bool   `json:"blocking"`
	Status     string `json:"status" validate:"required"`
}

func (o *OperatorRequest) Validate() error {
	if o.PipelineID < 0 {
		return ErrInvalidOperatorRequestPipelineID
	}
	if o.Pointer == "" {
		return ErrInvalidOperatorRequestPointer
	}
	if o.Question == "" {
		return ErrInvalidOperatorRequestQuestion
	}
	if o.Status == "" {
		return ErrInvalidOperatorRequestStatus
	}
	return nil
}
