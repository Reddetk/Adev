package domain

import "fmt"

// Stage is one step of a pipeline.
type Stage struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Owner   Role       `json:"owner"`
	Inputs  []FormType `json:"inputs"`
	Outputs []FormType `json:"outputs"`
}

var greenfieldStages = []Stage{
	{ID: "00", Name: "BRD", Owner: RoleAnalyst,
		Inputs: []FormType{TypeUserInput}, Outputs: []FormType{TypeBRD}},
	{ID: "01", Name: "Feasibility Study", Owner: RoleAnalyst,
		// The registry names an as-is HLD input, too. Greenfield has no current
		// architecture, so the route ships without it.
		Inputs: []FormType{TypeBRD}, Outputs: []FormType{TypeFeasibility, TypeBRD}},
	{ID: "02", Name: "Software Requirements Specification", Owner: RoleAnalyst,
		Inputs: []FormType{TypeBRD, TypeFeasibility}, Outputs: []FormType{TypeSRS}},
	{ID: "03", Name: "High-Level Design", Owner: RoleArchitect,
		Inputs: []FormType{TypeSRS}, Outputs: []FormType{TypeHLD}},
	{ID: "04", Name: "Invariants and ADR", Owner: RoleArchitect,
		Inputs: []FormType{TypeHLD, TypeSRS}, Outputs: []FormType{TypeHLD, TypeADR}},
	{ID: "05", Name: "Low-Level Design", Owner: RoleAnalyst,
		Inputs: []FormType{TypeHLD, TypeADR}, Outputs: []FormType{TypeLLD}},
	{ID: "06", Name: "Task Plan", Owner: RoleAnalyst,
		Inputs: []FormType{TypeLLD, TypeSRS}, Outputs: []FormType{TypeTaskPlan}},
}

// stagesByRoute is the route registry. A route names its stages, and no agent
// ever chooses the next one.
var stagesByRoute = map[Route][]Stage{
	RouteGreenfield: greenfieldStages,
}

// Stages returns the stages of a route.
func Stages(route Route) ([]Stage, error) {
	stages, ok := stagesByRoute[route]
	if !ok {
		return nil, fmt.Errorf("route %q is not implemented yet", route)
	}
	return cloneStages(stages), nil
}

// StageByID returns one stage of a route.
func StageByID(route Route, id string) (Stage, error) {
	stages, err := Stages(route)
	if err != nil {
		return Stage{}, err
	}
	for _, stage := range stages {
		if stage.ID == id {
			return stage, nil
		}
	}
	return Stage{}, fmt.Errorf("stage %q does not exist in route %q", id, route)
}

// NextStage returns the stage that follows the given stage, or an error when the
// stage is the last one. The route belongs to the registry, never to the agent.
func NextStage(route Route, id string) (Stage, error) {
	stages, err := Stages(route)
	if err != nil {
		return Stage{}, err
	}
	for index, stage := range stages {
		if stage.ID == id {
			if index+1 >= len(stages) {
				return Stage{}, fmt.Errorf("stage %q has no next stage in route %q", id, route)
			}
			return stages[index+1], nil
		}
	}
	return Stage{}, fmt.Errorf("stage %q does not exist in route %q", id, route)
}

// Roles returns the distinct owner roles of the stages, in stage order.
func Roles(stages []Stage) []Role {
	seen := map[Role]bool{}
	var result []Role
	for _, stage := range stages {
		if seen[stage.Owner] {
			continue
		}
		seen[stage.Owner] = true
		result = append(result, stage.Owner)
	}
	return result
}

func cloneStages(stages []Stage) []Stage {
	result := make([]Stage, len(stages))
	for i, stage := range stages {
		stage.Inputs = append([]FormType(nil), stage.Inputs...)
		stage.Outputs = append([]FormType(nil), stage.Outputs...)
		result[i] = stage
	}
	return result
}
