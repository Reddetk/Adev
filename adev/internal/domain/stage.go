package domain

import "fmt"

// InputSource says where the inputs of a stage come from. The empty value means
// the documents of the same pipeline.
type InputSource string

// InputSourceSystemBaseline reads the accepted as-is documents of another pipeline,
// which is what a change impact assessment needs.
const InputSourceSystemBaseline InputSource = "system-baseline"

// Stage is one step of a pipeline.
type Stage struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Owner   Role       `json:"owner"`
	Inputs  []FormType `json:"inputs"`
	Outputs []FormType `json:"outputs"`
	// InputSource is empty for the normal flow: the stage reads its own pipeline.
	InputSource InputSource `json:"inputSource,omitempty"`
	// Conditional stages are detours: the route lists them, but a straight run
	// skips them. Stage 08.R runs only after a failed verdict.
	Conditional bool `json:"conditional,omitempty"`
}

var greenfieldStages = []Stage{
	{ID: "00", Name: "BRD", Owner: RoleAnalyst,
		Inputs: []FormType{TypeUserInput}, Outputs: []FormType{TypeBRD}},
	{ID: "01", Name: "Techno-Economic Justification", Owner: RoleAnalyst,
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
	{ID: "07", Name: "Test Contract", Owner: RoleQA,
		Inputs: []FormType{TypeLLD, TypeTaskPlan}, Outputs: []FormType{TypeTestContract}},
	{ID: "08", Name: "Implementation Spec", Owner: RoleDev,
		Inputs: []FormType{TypeTestContract, TypeLLD, TypeADR}, Outputs: []FormType{TypeImplementationSpec}},
	{ID: "08.R", Name: "Refactor Spec", Owner: RoleDev, Conditional: true,
		Inputs: []FormType{TypeVerdict, TypeTestContract}, Outputs: []FormType{TypeRefactorSpec}},
	{ID: "09", Name: "QA Verdict", Owner: RoleQA,
		Inputs: []FormType{TypeImplementationSpec, TypeTestContract}, Outputs: []FormType{TypeVerdict}},
}

// changeStages is the head of a change route: the brief, then the impact
// assessment that picks one of the two branches.
var changeStages = []Stage{
	{ID: "00", Name: "Change Request Brief", Owner: RoleAnalyst,
		Inputs: []FormType{TypeUserInput}, Outputs: []FormType{TypeCRBrief}},
	{ID: "01", Name: "Impact Assessment", Owner: RoleAnalyst,
		Inputs: []FormType{TypeCRBrief}, Outputs: []FormType{TypeImpactAssessment},
		InputSource: InputSourceSystemBaseline},
}

// stagesByRoute is the route registry. A route names its stages, and no agent
// ever chooses the next one.
var stagesByRoute = map[Route][]Stage{
	RouteGreenfield:  greenfieldStages,
	RouteChange:      changeStages,
	RouteMinorChange: changeTails("02", "06", "07", "08", "08.R", "09"),
	RouteMajorChange: changeTails("02", "03", "04", "05", "06", "07", "08", "08.R", "09"),
}

// changeTails builds a change branch from the greenfield stages it reuses.
func changeTails(ids ...string) []Stage {
	result := make([]Stage, 0, len(ids))
	for _, id := range ids {
		for _, stage := range greenfieldStages {
			if stage.ID == id {
				result = append(result, stage)
				break
			}
		}
	}
	return result
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

// IsLastStage reports whether the stage ends its route, so a successful run of it
// completes the pipeline.
func IsLastStage(route Route, id string) bool {
	stages, err := Stages(route)
	if err != nil || len(stages) == 0 {
		return false
	}
	return stages[len(stages)-1].ID == id
}

// IsAhead reports whether stage comes after cursor in the route order: the
// operator may run a later stage while the cursor still stands on a blocked one.
func IsAhead(route Route, cursor, stage string) bool {
	if cursor == stage {
		return false
	}
	stages, err := Stages(route)
	if err != nil {
		return false
	}
	cursorIdx, stageIdx := -1, -1
	for index, s := range stages {
		if s.ID == cursor {
			cursorIdx = index
		}
		if s.ID == stage {
			stageIdx = index
		}
	}
	return cursorIdx >= 0 && stageIdx > cursorIdx
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
