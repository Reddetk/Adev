package domain

import (
	"errors"
	"fmt"
)

// ValidateFeasibility checks the FEASIBILITY gates.
// Gates that need the object registry or a stored BRD are left to the orchestrator.
func ValidateFeasibility(f Feasibility) error {
	var errs []error
	if len(f.Options) == 0 {
		errs = append(errs, invalid("options", "must contain at least one entry"))
	}
	if f.RecommendedOption == "" {
		errs = append(errs, invalid("recommendedOption", "is empty"))
	}
	switch f.Verdict {
	case VerdictGo, VerdictNoGo, VerdictRevise:
	default:
		errs = append(errs, invalid("verdict", fmt.Sprintf("unknown value %q", f.Verdict)))
	}
	if f.Verdict == "revise" && len(f.ScopeDelta) == 0 {
		errs = append(errs, invalid("scopeDelta", "must not be empty when verdict is revise"))
	}

	for i, option := range f.Options {
		if option.ID == "" {
			errs = append(errs, invalid(fmt.Sprintf("options[%d].id", i), "is empty"))
		}
		if option.Confidence != ConfidenceLow && option.Confidence != ConfidenceMedium && option.Confidence != ConfidenceHigh {
			errs = append(errs, invalid(fmt.Sprintf("options[%d].confidence", i), fmt.Sprintf("unknown value %q", option.Confidence)))
		}
	}
	for i, risk := range f.Risks {
		if risk.Likelihood < 1 || risk.Likelihood > 5 {
			errs = append(errs, invalid(fmt.Sprintf("risks[%d].likelihood", i), "must be between 1 and 5"))
		}
		if risk.Impact < 1 || risk.Impact > 5 {
			errs = append(errs, invalid(fmt.Sprintf("risks[%d].impact", i), "must be between 1 and 5"))
		}
	}
	return errors.Join(errs...)
}

// ValidateSRS checks the SRS gates.
func ValidateSRS(s SRS) error {
	var errs []error
	if len(s.FR) == 0 && len(s.NFR) == 0 {
		errs = append(errs, invalid("fr", "at least one of fr or nfr must have an entry"))
	}
	for i, fr := range s.FR {
		if fr.ID == "" {
			errs = append(errs, invalid(fmt.Sprintf("fr[%d].id", i), "is empty"))
		}
		if fr.Statement.Text == "" {
			errs = append(errs, invalid(fmt.Sprintf("fr[%d].statement", i), "is empty"))
		}
		if fr.Priority != PriorityMust && fr.Priority != PriorityHigh && fr.Priority != PriorityLow {
			errs = append(errs, invalid(fmt.Sprintf("fr[%d].priority", i), fmt.Sprintf("unknown value %q", fr.Priority)))
		}
		if len(fr.Acceptance) == 0 {
			errs = append(errs, invalid(fmt.Sprintf("fr[%d].acceptance", i), "must contain at least one scenario"))
		}
		for j, scenario := range fr.Acceptance {
			if scenario.Given == "" || scenario.When == "" || scenario.Then == "" {
				errs = append(errs, invalid(fmt.Sprintf("fr[%d].acceptance[%d]", i, j), "given, when and then must be non-empty"))
			}
		}
	}
	for i, nfr := range s.NFR {
		if nfr.ID == "" {
			errs = append(errs, invalid(fmt.Sprintf("nfr[%d].id", i), "is empty"))
		}
		if nfr.Metric == "" || nfr.Unit == "" || nfr.Threshold == "" || nfr.MeasurementPoint == "" || nfr.Window == "" {
			errs = append(errs, invalid(fmt.Sprintf("nfr[%d]", i), "metric, unit, threshold, measurementPoint and window must be non-empty"))
		}
	}
	return errors.Join(errs...)
}

// ValidateHLD checks the HLD gates.
// Container and aggregate references to the object registry are left to the orchestrator.
func ValidateHLD(h HLD) error {
	var errs []error
	if len(h.BoundedContexts) == 0 {
		errs = append(errs, invalid("boundedContexts", "must contain at least one entry"))
	}
	if len(h.Containers) == 0 {
		errs = append(errs, invalid("containers", "must contain at least one entry"))
	}

	contextIDs := map[string]bool{}
	for i, bc := range h.BoundedContexts {
		if bc.ID == "" {
			errs = append(errs, invalid(fmt.Sprintf("boundedContexts[%d].id", i), "is empty"))
			continue
		}
		contextIDs[bc.ID] = true
	}
	containerIDs := map[string]bool{}
	for i, c := range h.Containers {
		if c.ObjectID == "" {
			errs = append(errs, invalid(fmt.Sprintf("containers[%d].objectId", i), "is empty"))
		}
		if !contextIDs[c.BoundedContextID] {
			errs = append(errs, invalid(fmt.Sprintf("containers[%d].boundedContextId", i), "does not exist in boundedContexts"))
		}
		containerIDs[c.ObjectID] = true
	}
	for i, a := range h.Aggregates {
		if a.ID == "" {
			errs = append(errs, invalid(fmt.Sprintf("aggregates[%d].id", i), "is empty"))
		}
		if a.Root == "" {
			errs = append(errs, invalid(fmt.Sprintf("aggregates[%d].root", i), "is empty"))
		}
		if len(a.Invariants) == 0 {
			errs = append(errs, invalid(fmt.Sprintf("aggregates[%d].invariants", i), "must contain at least one statement"))
		}
		if a.ContainerID == "" || !containerIDs[a.ContainerID] {
			errs = append(errs, invalid(fmt.Sprintf("aggregates[%d].containerId", i), "does not exist in containers"))
		}
		if a.ModuleRef == "" {
			errs = append(errs, invalid(fmt.Sprintf("aggregates[%d].moduleRef", i), "is empty"))
		}
	}
	for i, relation := range h.ContextRelations {
		if relation.From == "" || relation.To == "" {
			errs = append(errs, invalid(fmt.Sprintf("contextRelations[%d]", i), "from and to must be non-empty"))
		}
		if relation.Type != RelationSync && relation.Type != RelationAsync {
			errs = append(errs, invalid(fmt.Sprintf("contextRelations[%d].type", i), fmt.Sprintf("unknown value %q", relation.Type)))
		}
	}
	if len(h.DegradationStrategy) == 0 {
		errs = append(errs, invalid("degradationStrategy", "must contain at least one entry"))
	}
	for i, d := range h.DegradationStrategy {
		if d.TimeoutMs < 0 {
			errs = append(errs, invalid(fmt.Sprintf("degradationStrategy[%d].timeoutMs", i), "must not be negative"))
		}
	}
	return errors.Join(errs...)
}

// ValidateADR checks the ADR gates.
func ValidateADR(a ADR) error {
	var errs []error
	if len(a.Decisions) == 0 {
		errs = append(errs, invalid("decisions", "must contain at least one entry"))
	}
	ids := map[string]bool{}
	for i, d := range a.Decisions {
		if d.ID == "" {
			errs = append(errs, invalid(fmt.Sprintf("decisions[%d].id", i), "is empty"))
			continue
		}
		ids[d.ID] = true
	}
	for i, d := range a.Decisions {
		if d.Status != DecisionProposed && d.Status != DecisionAccepted && d.Status != DecisionSuperseded {
			errs = append(errs, invalid(fmt.Sprintf("decisions[%d].status", i), fmt.Sprintf("unknown value %q", d.Status)))
		}
		if len(d.Alternatives) == 0 {
			errs = append(errs, invalid(fmt.Sprintf("decisions[%d].alternatives", i), "must contain at least one rejected alternative"))
		}
		for j, alt := range d.Alternatives {
			if alt.Option.Value == "" || alt.RejectedBecause.Value == "" {
				errs = append(errs, invalid(fmt.Sprintf("decisions[%d].alternatives[%d]", i, j), "option and rejectedBecause must be non-empty"))
			}
		}
		if d.Supersedes != "" && !ids[d.Supersedes] {
			errs = append(errs, invalid(fmt.Sprintf("decisions[%d].supersedes", i), "does not exist in decisions"))
		}
	}
	return errors.Join(errs...)
}

var knownLayerNames = map[string]bool{"adapter": true, "domain": true, "app": true, "infra": true}

// ValidateLLD checks the LLD gates.
func ValidateLLD(l LLD) error {
	var errs []error
	for ci, c := range l.Containers {
		if c.ContainerID == "" {
			errs = append(errs, invalid(fmt.Sprintf("containers[%d].containerId", ci), "is empty"))
		}
		for li, layer := range c.Layers {
			switch layer.Name {
			case LayerAdapter, LayerDomain, LayerApp, LayerInfra:
			default:
				errs = append(errs, invalid(fmt.Sprintf("containers[%d].layers[%d].name", ci, li), fmt.Sprintf("unknown value %q", layer.Name)))
			}
			for pi, pkg := range layer.Packages {
				for mi, module := range pkg.Modules {
					if module.ObjectID == "" || module.Path == "" {
						errs = append(errs, invalid(fmt.Sprintf("containers[%d].layers[%d].packages[%d].modules[%d]", ci, li, pi, mi), "objectId and path must be non-empty"))
					}
				}
			}
		}
	}

	contractIDs := map[string]bool{}
	for i, contract := range l.Contracts {
		if contract.ID == "" {
			errs = append(errs, invalid(fmt.Sprintf("contracts[%d].id", i), "is empty"))
			continue
		}
		contractIDs[contract.ID] = true
	}

	// Gate: each contract has at least one happy path and one negative path.
	happy := map[string]bool{}
	negative := map[string]bool{}
	errorCodes := map[string]map[string]bool{}
	for i, path := range l.Paths {
		switch path.Kind {
		case PathHappy:
			happy[path.ContractRef] = true
		case PathNegative:
			negative[path.ContractRef] = true
			if path.ExpectedErrorCode == "" {
				errs = append(errs, invalid(fmt.Sprintf("paths[%d].expectedErrorCode", i), "must be non-empty on a negative path"))
			}
			if errorCodes[path.ContractRef] == nil {
				errorCodes[path.ContractRef] = map[string]bool{}
			}
			errorCodes[path.ContractRef][path.ExpectedErrorCode] = true
		default:
			errs = append(errs, invalid(fmt.Sprintf("paths[%d].kind", i), fmt.Sprintf("unknown value %q", path.Kind)))
		}
	}
	for i, contract := range l.Contracts {
		if !happy[contract.ID] {
			errs = append(errs, invalid(fmt.Sprintf("contracts[%d]", i), "has no happy path"))
		}
		if !negative[contract.ID] {
			errs = append(errs, invalid(fmt.Sprintf("contracts[%d]", i), "has no negative path"))
		}
		for _, contractError := range contract.Errors {
			if errorCodes[contract.ID][contractError.Code] {
				continue
			}
			errs = append(errs, invalid(fmt.Sprintf("contracts[%d].errors", i), fmt.Sprintf("code %q has no negative path", contractError.Code)))
		}
	}
	for i, event := range l.EventMaps {
		if event.IdempotencyKey == "" || event.Ordering == "" {
			errs = append(errs, invalid(fmt.Sprintf("eventMaps[%d]", i), "idempotencyKey and ordering must be non-empty"))
		}
	}
	return errors.Join(errs...)
}

var (
	knownTaskTypes = map[PlanTaskType]bool{TaskFeature: true, TaskFix: true, TaskRefactor: true, TaskConfig: true, TaskDocs: true, TaskTest: true}
	knownOwners    = map[PlanTaskOwner]bool{OwnerDev: true, OwnerDevOps: true}
)

// ValidateTaskPlan checks the TASK_PLAN gates.
func ValidateTaskPlan(p TaskPlan) error {
	var errs []error
	if len(p.Tasks) == 0 {
		errs = append(errs, invalid("tasks", "must contain at least one entry"))
		return errs[0]
	}

	ids := map[string]bool{}
	for i, task := range p.Tasks {
		if task.ID == "" {
			errs = append(errs, invalid(fmt.Sprintf("tasks[%d].id", i), "is empty"))
			continue
		}
		ids[task.ID] = true
		if !knownOwners[task.Owner] {
			errs = append(errs, invalid(fmt.Sprintf("tasks[%d].owner", i), fmt.Sprintf("unknown value %q", task.Owner)))
		}
		if !knownTaskTypes[task.TaskType] {
			errs = append(errs, invalid(fmt.Sprintf("tasks[%d].taskType", i), fmt.Sprintf("unknown value %q", task.TaskType)))
		}
		if len(task.Acceptance) == 0 {
			errs = append(errs, invalid(fmt.Sprintf("tasks[%d].acceptance", i), "must contain at least one acceptance statement"))
		}
		if len(task.ContractRefs) == 0 && len(task.AcceptanceRefs) == 0 {
			errs = append(errs, invalid(fmt.Sprintf("tasks[%d]", i), "must reference at least one contract or acceptance"))
		}
	}

	// Gates: each parent exists, the parent chain has no cycle.
	for i, task := range p.Tasks {
		if task.ParentID == "" {
			continue
		}
		if !ids[task.ParentID] {
			errs = append(errs, invalid(fmt.Sprintf("tasks[%d].parentId", i), "does not exist in tasks"))
		}
	}
	for i, task := range p.Tasks {
		seen := map[string]bool{task.ID: true}
		parent := task.ParentID
		for parent != "" {
			if seen[parent] {
				errs = append(errs, invalid(fmt.Sprintf("tasks[%d]", i), "the parent chain has a cycle"))
				break
			}
			seen[parent] = true
			var next string
			for _, candidate := range p.Tasks {
				if candidate.ID == parent {
					next = candidate.ParentID
					break
				}
			}
			parent = next
		}
	}

	for i, id := range p.CriticalPath {
		if !ids[id] {
			errs = append(errs, invalid(fmt.Sprintf("criticalPath[%d]", i), "does not exist in tasks"))
		}
	}
	return errors.Join(errs...)
}
