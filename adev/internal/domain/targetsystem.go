package domain

type TargetSystem struct {
	ID          uint   `json:"id"`
	Name        string `json:"name" validate:"required"`
	RepoURL     string `json:"repourl,omitempty"`
	Description string `json:"description" validate:"required"`
}

func (t *TargetSystem) Validate() error {
	if t.Name == "" {
		return ErrInvalidTargetSystemName
	}
	if t.Description == "" {
		return ErrInvalidTargetSystemDescription
	}
	return nil
}
