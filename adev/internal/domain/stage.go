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
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Owner  Role       `json:"owner"`
	Inputs []FormType `json:"inputs"`
	// OptionalInputs join the context when the route produced them. The LLD reads
	// the ADR of a change and runs without one on a new system.
	OptionalInputs []FormType `json:"optionalInputs,omitempty"`
	Outputs        []FormType `json:"outputs"`
	// InputSource is empty for the normal flow: the stage reads its own pipeline.
	InputSource InputSource `json:"inputSource,omitempty"`
	// Conditional stages are detours: the route lists them, but a straight run
	// skips them. Stage 08.R runs only after a failed verdict.
	Conditional bool `json:"conditional,omitempty"`
	// Disabled keeps a stage in the registry for a later release. The planner
	// hides it and StageByID refuses it: the operator cannot run what the project
	// does not implement yet.
	Disabled bool `json:"disabled,omitempty"`
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
		Inputs: []FormType{TypeSRS, TypeBRD, TypeFeasibility}, Outputs: []FormType{TypeHLD}},
	{ID: "05", Name: "Low-Level Design", Owner: RoleAnalyst,
		Inputs: []FormType{TypeHLD}, OptionalInputs: []FormType{TypeADR}, Outputs: []FormType{TypeLLD}},
	{ID: "06", Name: "Task Plan", Owner: RoleAnalyst, Disabled: true,
		Inputs: []FormType{TypeLLD, TypeSRS, TypeHLD}, Outputs: []FormType{TypeTaskPlan}},
	{ID: "07", Name: "Test Contract", Owner: RoleQA, Disabled: true,
		Inputs: []FormType{TypeLLD, TypeTaskPlan}, Outputs: []FormType{TypeTestContract}},
	{ID: "08", Name: "Implementation Spec", Owner: RoleDev, Disabled: true,
		Inputs: []FormType{TypeTestContract, TypeLLD}, Outputs: []FormType{TypeImplementationSpec}},
	{ID: "08.R", Name: "Refactor Spec", Owner: RoleDev, Conditional: true, Disabled: true,
		Inputs: []FormType{TypeVerdict, TypeTestContract}, Outputs: []FormType{TypeRefactorSpec}},
	{ID: "09", Name: "QA Verdict", Owner: RoleQA, Disabled: true,
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

// changeOnlyStages are the stages a change route keeps and greenfield does not.
// A decision record belongs to a system that already runs: it says why the running
// design changes, which is the history a greenfield system does not have yet. A
// greenfield design keeps its choices in the HLD.
var changeOnlyStages = []Stage{
	{ID: "04", Name: "Invariants and ADR", Owner: RoleArchitect,
		Inputs: []FormType{TypeHLD, TypeSRS}, Outputs: []FormType{TypeHLD, TypeADR}},
}

// stagesByRoute is the route registry. A route names its stages, and no agent
// ever chooses the next one.
var stagesByRoute = map[Route][]Stage{
	RouteGreenfield:  greenfieldStages,
	RouteChange:      changeStages,
	RouteMinorChange: changeTails("02", "06", "07", "08", "08.R", "09"),
	RouteMajorChange: changeTails("02", "03", "04", "05", "06", "07", "08", "08.R", "09"),
}

// changeTails builds a change branch from the stages it reuses: the greenfield
// route and the stages only a change route runs.
func changeTails(ids ...string) []Stage {
	result := make([]Stage, 0, len(ids))
	for _, id := range ids {
		for _, stage := range append(append([]Stage{}, greenfieldStages...), changeOnlyStages...) {
			if stage.ID == id {
				result = append(result, stage)
				break
			}
		}
	}
	return result
}

// Stages returns the stages of a route, including the ones kept for a later
// release. The registry names them; the operator does not run them.
func Stages(route Route) ([]Stage, error) {
	stages, ok := stagesByRoute[route]
	if !ok {
		return nil, fmt.Errorf("route %q is not implemented yet", route)
	}
	return cloneStages(stages), nil
}

// AvailableStages returns the stages an operator can run now. A disabled stage
// stays in the registry for a later release and is hidden from the plan.
func AvailableStages(route Route) ([]Stage, error) {
	stages, err := Stages(route)
	if err != nil {
		return nil, err
	}
	out := make([]Stage, 0, len(stages))
	for _, stage := range stages {
		if !stage.Disabled {
			out = append(out, stage)
		}
	}
	return out, nil
}

// StageByID returns one stage of a route. A disabled stage is not available yet:
// the operator cannot start what the project does not implement.
func StageByID(route Route, id string) (Stage, error) {
	stages, err := Stages(route)
	if err != nil {
		return Stage{}, err
	}
	for _, stage := range stages {
		if stage.ID == id {
			if stage.Disabled {
				return Stage{}, fmt.Errorf("stage %q is not available yet", id)
			}
			return stage, nil
		}
	}
	return Stage{}, fmt.Errorf("stage %q does not exist in route %q", id, route)
}

// NextStage returns the next stage an operator can run, or an error when the
// stage is the last one. Disabled stages are skipped.
func NextStage(route Route, id string) (Stage, error) {
	stages, err := Stages(route)
	if err != nil {
		return Stage{}, err
	}
	for index, stage := range stages {
		if stage.ID != id {
			continue
		}
		for next := index + 1; next < len(stages); next++ {
			if !stages[next].Disabled {
				return stages[next], nil
			}
		}
		return Stage{}, fmt.Errorf("stage %q has no next available stage in route %q", id, route)
	}
	return Stage{}, fmt.Errorf("stage %q does not exist in route %q", id, route)
}

// IsLastStage reports whether the stage ends its route among the available
// stages, so a successful run of it completes the pipeline.
func IsLastStage(route Route, id string) bool {
	stages, err := Stages(route)
	if err != nil || len(stages) == 0 {
		return false
	}
	for index, stage := range stages {
		if stage.ID != id {
			continue
		}
		for next := index + 1; next < len(stages); next++ {
			if !stages[next].Disabled {
				return false
			}
		}
		return true
	}
	return false
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

// Roles returns the distinct owner roles of the available stages, in stage order.
func Roles(stages []Stage) []Role {
	seen := map[Role]bool{}
	var result []Role
	for _, stage := range stages {
		if stage.Disabled || seen[stage.Owner] {
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
