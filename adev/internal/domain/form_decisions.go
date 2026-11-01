package domain

import (
	"fmt"
	"strings"
)

// DecisionArea is the part of the design a decision belongs to. The list is the
// one the registry names: a decision without an area cannot be scored or found
// later.
type DecisionArea string

const (
	AreaArchitecture  DecisionArea = "architecture"
	AreaCommunication DecisionArea = "communication"
	AreaStorage       DecisionArea = "storage"
	AreaScaling       DecisionArea = "scaling"
	AreaConsistency   DecisionArea = "consistency"
	AreaReliability   DecisionArea = "reliability"
	AreaStack         DecisionArea = "stack"
	AreaOps           DecisionArea = "ops"
	AreaProcess       DecisionArea = "process"
	AreaMetrics       DecisionArea = "metrics"
)

var knownAreas = map[DecisionArea]bool{
	AreaArchitecture: true, AreaCommunication: true, AreaStorage: true, AreaScaling: true,
	AreaConsistency: true, AreaReliability: true, AreaStack: true, AreaOps: true,
	AreaProcess: true, AreaMetrics: true,
}

// ArchitectureStyle is how the system is deployed: one unit or several.
type ArchitectureStyle string

const (
	StyleMonolith        ArchitectureStyle = "monolith"
	StyleModularMonolith ArchitectureStyle = "modular_monolith"
	StyleHexagonal       ArchitectureStyle = "hexagonal"
	StyleMicroservices   ArchitectureStyle = "microservices"
	StyleEventDriven     ArchitectureStyle = "event_driven"
	StyleCellBased       ArchitectureStyle = "cell_based"
	StyleServerless      ArchitectureStyle = "serverless"
)

// DeployedApart reports a style whose containers deploy separately.
func (s ArchitectureStyle) DeployedApart() bool {
	switch s {
	case StyleMicroservices, StyleEventDriven, StyleCellBased, StyleServerless:
		return true
	}
	return false
}

func (s ArchitectureStyle) Valid() bool {
	switch s {
	case StyleMonolith, StyleModularMonolith, StyleHexagonal,
		StyleMicroservices, StyleEventDriven, StyleCellBased, StyleServerless:
		return true
	}
	return false
}

// PathRoot is the top of the code tree a technology lays out. An infrastructure
// product holds no code, so it has no root and the layout does not apply to it.
func PathRoot(tech string) (string, bool) {
	switch strings.ToLower(tech) {
	case TechGo:
		return "internal", true
	case TechJava:
		return "src/main/java", true
	case TechNode, TechPython, TechReact, TechAngular, TechVue:
		return "src", true
	}
	return "", false
}

// PathTemplate is the layout under the root: where the layer and the module go.
func PathTemplate(tech string) (string, bool) {
	switch strings.ToLower(tech) {
	case TechGo:
		return "internal/<layer>/<module>", true
	case TechJava:
		return "src/main/java/<module>/<layer>", true
	case TechNode, TechPython, TechReact, TechAngular, TechVue:
		return "src/<layer>/<module>", true
	}
	return "", false
}

// validateDecisions checks that the design says what it promises: a consistency
// decision per context, a storage decision when it has a store, and one
// architecture decision whose style the containers can carry.
func validateDecisions(h HLD) []error {
	var errs []error
	byArea := map[DecisionArea]int{}
	contextsWithClass := map[string]bool{}
	storageDecided := false
	containers := map[string]bool{}

	for i, decision := range h.Decisions {
		field := fmt.Sprintf("decisions[%d]", i)
		if decision.Area != "" && !knownAreas[decision.Area] {
			errs = append(errs, invalid(field+".area", fmt.Sprintf("unknown value %q", decision.Area)))
		}
		byArea[decision.Area]++
		if decision.ContextRef != "" {
			found := false
			for _, context := range h.BoundedContexts {
				if context.ID == decision.ContextRef {
					found = true
					break
				}
			}
			if !found {
				errs = append(errs, invalid(field+".contextRef", fmt.Sprintf("%q is not a bounded context of this design", decision.ContextRef)))
			}
			if decision.Area == AreaConsistency {
				contextsWithClass[decision.ContextRef] = true
			}
		}
		if decision.Area == AreaStorage {
			if !decision.Ownership.Valid() {
				errs = append(errs, invalid(field+".ownership",
					fmt.Sprintf("a storage decision names database_per_service, shared_database or read_model, got %q", decision.Ownership)))
			}
			for _, tech := range decision.Techs {
				if storeTechs[strings.ToLower(tech)] {
					storageDecided = true
				}
			}
		}
		if decision.Area == AreaArchitecture {
			if !decision.Style.Valid() {
				errs = append(errs, invalid(field+".style", fmt.Sprintf("unknown value %q", decision.Style)))
			}
		}
	}

	// Every context answers its class in a decision of its own, with the rejected
	// options, so the choice has an audit trail and not only a field.
	for _, context := range h.BoundedContexts {
		if contextsWithClass[context.ID] {
			continue
		}
		errs = append(errs, invalid("decisions",
			fmt.Sprintf("bounded context %q has no consistency decision; add one with area \"consistency\" and contextRef %q", context.ID, context.ID)))
	}

	// A store container without a storage decision is a product nobody chose.
	storeContainers := 0
	codeContainers := 0
	for _, container := range h.Containers {
		containers[container.ObjectID] = true
		if storeTechs[strings.ToLower(container.Tech)] {
			storeContainers++
		}
		if _, code := PathRoot(container.Tech); code {
			codeContainers++
		}
	}
	if storeContainers > 0 && !storageDecided {
		errs = append(errs, invalid("decisions", "the design has a store container but no storage decision that names its product"))
	}

	// One style, and the containers prove it.
	styles := byArea[AreaArchitecture]
	if styles == 0 {
		errs = append(errs, invalid("decisions", "the design has no architecture decision; add one with area \"architecture\" and a style"))
	}
	if styles > 1 {
		errs = append(errs, invalid("decisions", fmt.Sprintf("the design has %d architecture decisions; keep one", styles)))
	}
	if styles == 1 {
		for _, decision := range h.Decisions {
			if decision.Area != AreaArchitecture || !decision.Style.Valid() {
				continue
			}
			if decision.Style.DeployedApart() && codeContainers < 2 {
				errs = append(errs, invalid("decisions",
					fmt.Sprintf("style %q deploys services apart but the design has %d code containers", decision.Style, codeContainers)))
			}
			if !decision.Style.DeployedApart() && codeContainers > 1 {
				errs = append(errs, invalid("decisions",
					fmt.Sprintf("style %q is one deployable unit but the design has %d code containers", decision.Style, codeContainers)))
			}
			if decision.Style.DeployedApart() {
				if ownership, ok := ownershipOf(h.Decisions); ok && ownership == OwnershipShared {
					errs = append(errs, invalid("decisions",
						fmt.Sprintf("style %q deploys services apart but the storage decision shares one database (shared_database); use database_per_service or a read model", decision.Style)))
				}
			}
		}
	}
	_ = containers
	return errs
}

// Ownership says who owns a store: one service, every service, or a read model
// built from events.
type Ownership string

const (
	OwnershipPerService Ownership = "database_per_service"
	OwnershipShared     Ownership = "shared_database"
	OwnershipReadModel  Ownership = "read_model"
)

func (o Ownership) Valid() bool {
	switch o {
	case OwnershipPerService, OwnershipShared, OwnershipReadModel:
		return true
	}
	return false
}

// validateRelations checks what a relation promises: an async edge names its
// delivery guarantee, and an at-least-once edge into a consistent context names the
// key that makes a duplicate safe.
func validateRelations(h HLD) []error {
	var errs []error
	classes := map[string]PACELC{}
	for _, context := range h.BoundedContexts {
		classes[context.ID] = context.PACELC
	}
	for i, relation := range h.ContextRelations {
		field := fmt.Sprintf("contextRelations[%d]", i)
		if relation.Type == RelationAsync {
			switch relation.Guarantee {
			case DeliveryAtMostOnce, DeliveryAtLeastOnce, DeliveryExactlyOnce:
			default:
				errs = append(errs, invalid(field+".guarantee",
					fmt.Sprintf("an async relation names at_most_once, at_least_once or exactly_once, got %q", relation.Guarantee)))
			}
			if classes[relation.To].Consistent() && relation.Guarantee != DeliveryAtMostOnce && relation.IdempotencyKey == "" {
				errs = append(errs, invalid(field+".idempotencyKey",
					fmt.Sprintf("context %q promises consistency and receives %s events; name the key that makes a duplicate safe", relation.To, relation.Guarantee)))
			}
		}
	}
	return errs
}

// ownershipOf reads the ownership of the storage decision, if the design has one.
func ownershipOf(decisions []Decision) (Ownership, bool) {
	for _, decision := range decisions {
		if decision.Area == AreaStorage {
			return decision.Ownership, true
		}
	}
	return "", false
}
