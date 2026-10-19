package domain

type TargetSystem struct {
	ID          uint   `json:"id"`
	Name        string `json:"name" validate:"required"`
	RepoURL     string `json:"repourl" validate:"required"`
	Description string `json:"description" validate:"required"`
}

func (t *TargetSystem) Validate() error {
	if t.Name == "" {
		return ErrInvalidTargetSystemName
	}
	if t.RepoURL == "" {
		return ErrInvalidTargetSystemRepoURL
	}
	if t.Description == "" {
		return ErrInvalidTargetSystemDescription
	}
	return nil
}
