package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/Reddetk/Adev/internal/domain"
)

// Plan is the stage sequence of one target system with its readiness.
type Plan struct {
	TargetSystemID int64            `json:"targetSystemId"`
	Route          domain.Route     `json:"route"`
	Stages         []domain.Stage   `json:"stages"`
	Readiness      domain.Readiness `json:"readiness"`
}

// Planner builds pipeline plans from the configured settings.
type Planner struct {
	settings *SettingsService
}

// NewPlanner builds a pipeline service.
func NewPlanner(settings *SettingsService) *Planner {
	return &Planner{settings: settings}
}

// Plan returns the stage sequence and the settings that the route still needs.
func (s *Planner) Plan(ctx context.Context, targetSystemID int64, route domain.Route) (Plan, error) {
	stages, err := domain.Stages(route)
	if err != nil {
		return Plan{}, err
	}

	workspace, err := s.settings.Workspace(ctx, targetSystemID)
	if err != nil && !errors.Is(err, domain.ErrSettingNotFound) {
		return Plan{}, fmt.Errorf("plan: read workspace: %w", err)
	}

	agents, err := s.settings.Agents(ctx, targetSystemID)
	if err != nil {
		return Plan{}, fmt.Errorf("plan: read agents: %w", err)
	}

	return Plan{
		TargetSystemID: targetSystemID,
		Route:          route,
		Stages:         stages,
		Readiness:      domain.Check(workspace, agents, stages),
	}, nil
}
