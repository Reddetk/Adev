package usecase

import (
	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/messages"
)

type targetsystemService struct {
	repo TargetSystemRepository
}

func NewTargetSystemService(repo TargetSystemRepository) TargetSystemUseCase {
	return &targetsystemService{repo: repo}
}

func (t *targetsystemService) CreateTargetSystem(input CreateTargetSystemInput) (CreateTargetSystemOutput, error) {
	if err := input.Validate(); err != nil {
		return CreateTargetSystemOutput{}, err
	}

	targetsystem := domain.TargetSystem{
		Name:        input.Name,
		RepoURL:     input.RepoURL,
		Description: input.Description,
	}

	if err := targetsystem.Validate(); err != nil {
		return CreateTargetSystemOutput{}, err
	}

	if err := t.repo.Save(&targetsystem); err != nil {
		return CreateTargetSystemOutput{}, err
	}

	return CreateTargetSystemOutput{
		ID:          targetsystem.ID,
		Name:        targetsystem.Name,
		RepoURL:     targetsystem.RepoURL,
		Description: targetsystem.Description,
		Message:     messages.TargetSystemCreatedSuccessfully,
	}, nil
}

func (t *targetsystemService) GetTargetSystem(id int) (*domain.TargetSystem, error) {
	return t.repo.FindByID(id)
}

func (t *targetsystemService) UpdateTargetSystem(id int, input UpdateTargetSystemInput) error {
	targetsystem, err := t.repo.FindByID(id)
	if err != nil {
		return err
	}

	if input.Name != nil {
		targetsystem.Name = *input.Name
	}
	if input.RepoURL != nil {
		targetsystem.RepoURL = *input.RepoURL
	}
	if input.Description != nil {
		targetsystem.Description = *input.Description
	}

	return t.repo.Update(targetsystem)
}

func (t *targetsystemService) DeleteTargetSystem(id int) error {
	return t.repo.Delete(id)
}

func (t *targetsystemService) ListTargetSystems() (ListTargetSystemOutput, error) {
	targetsystems, err := t.repo.FindAll()
	if err != nil {
		return ListTargetSystemOutput{}, err
	}

	return ListTargetSystemOutput{
		TargetSystems: targetsystems,
		Total:         len(targetsystems),
		Message:       messages.TargetSystemsListedSuccessfully,
	}, nil
}
