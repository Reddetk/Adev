package domain

import (
	"fmt"
	"strings"
)

// The D7 tree and the D8 deployment pattern are closed palettes, not free text.
// A branch of the tree fires only when the design holds its premise; a decision
// then names one mechanism of that branch. The classifier of a later pass scores
// the same lists, so the prompt, the gate and the classifier cannot drift.

// DistributedPattern is one named mechanism from the reliability palette.
type DistributedPattern string

const (
	// A write and its event commit together.
	PatternTransactionalOutbox   DistributedPattern = "transactional_outbox"
	PatternTransactionLogTailing DistributedPattern = "transaction_log_tailing"
	PatternPollingPublisher      DistributedPattern = "polling_publisher"

	// A consumer sees the same message twice.
	PatternDLQ        DistributedPattern = "dlq"
	PatternParkingLot DistributedPattern = "parking_lot"

	// A producer outruns a consumer.
	PatternBoundedQueue DistributedPattern = "bounded_queue_block"
	PatternDropNew      DistributedPattern = "drop_new"
	PatternDegrade      DistributedPattern = "degrade"
	PatternScaleOut     DistributedPattern = "scale_out"

	// One operation writes in more than one service.
	PatternSagaOrchestration DistributedPattern = "saga_orchestration"
	PatternSagaChoreography  DistributedPattern = "saga_choreography"
	PatternTwoPhaseCommit    DistributedPattern = "two_phase_commit"
)

// d7Branch is the premise a pattern answers. The gate asks whether the branch
// fired, never whether the sentence sounds right.
type d7Branch string

const (
	branchOutbox       d7Branch = "outbox"
	branchDLQ          d7Branch = "dlq"
	branchBackpressure d7Branch = "backpressure"
	branchSaga         d7Branch = "saga"
)

var d7Order = []d7Branch{branchOutbox, branchDLQ, branchBackpressure, branchSaga}

// distributedPalette is the single ordered source of the D7 options. The map is
// built from it, so the prompt, the gate and the classifier list the same names.
var distributedPalette = []struct {
	pattern DistributedPattern
	branch  d7Branch
}{
	{PatternTransactionalOutbox, branchOutbox},
	{PatternTransactionLogTailing, branchOutbox},
	{PatternPollingPublisher, branchOutbox},

	{PatternDLQ, branchDLQ},
	{PatternParkingLot, branchDLQ},

	{PatternBoundedQueue, branchBackpressure},
	{PatternDropNew, branchBackpressure},
	{PatternDegrade, branchBackpressure},
	{PatternScaleOut, branchBackpressure},

	{PatternSagaOrchestration, branchSaga},
	{PatternSagaChoreography, branchSaga},
	{PatternTwoPhaseCommit, branchSaga},
}

var patternBranches = func() map[DistributedPattern]d7Branch {
	branches := make(map[DistributedPattern]d7Branch, len(distributedPalette))
	for _, entry := range distributedPalette {
		branches[entry.pattern] = entry.branch
	}
	return branches
}()

// Valid reports whether the pattern belongs to the palette.
func (p DistributedPattern) Valid() bool {
	_, ok := patternBranches[p]
	return ok
}

// DistributedPatterns lists the palette in branch order. A prompt prints it and
// the classifier scores it; the slice above stays the single source.
func DistributedPatterns() []DistributedPattern {
	out := make([]DistributedPattern, len(distributedPalette))
	for i, entry := range distributedPalette {
		out[i] = entry.pattern
	}
	return out
}

// patternsOfBranch returns the options of one branch in a stable order, so an
// error message names the same alternatives on every run.
func patternsOfBranch(branch d7Branch) []DistributedPattern {
	out := make([]DistributedPattern, 0, 3)
	for _, entry := range distributedPalette {
		if entry.branch == branch {
			out = append(out, entry.pattern)
		}
	}
	return out
}

// busTechs are the products that buffer messages. A bus is not a store: it holds
// a message until a consumer takes it.
var busTechs = map[string]bool{TechKafka: true, TechRabbit: true, TechNats: true}

// d7Premises returns the branches whose premise the design holds, with the fact
// that makes the premise true. An empty map means the tree fires nothing.
func d7Premises(h HLD) map[d7Branch]string {
	fired := map[d7Branch]string{}
	for _, relation := range h.ContextRelations {
		if relation.Type != RelationAsync {
			continue
		}
		if endpointWritesState(h, relation.From) {
			fired[branchOutbox] = fmt.Sprintf(
				"the async relation %s → %s leaves a container that writes state", relation.From, relation.To)
		}
		if relation.Guarantee == DeliveryAtLeastOnce || relation.Guarantee == DeliveryExactlyOnce {
			fired[branchDLQ] = fmt.Sprintf(
				"the async relation %s → %s promises %s, so a message can fail forever",
				relation.From, relation.To, relation.Guarantee)
		}
		if busEndpoint(h, relation.From) || busEndpoint(h, relation.To) {
			fired[branchBackpressure] = fmt.Sprintf(
				"the async relation %s → %s enters a message bus, so a producer can outrun a consumer (backpressure)",
				relation.From, relation.To)
		}
	}
	if chain, ok := crossContextChain(h); ok {
		fired[branchSaga] = fmt.Sprintf(
			"the relation chain %s crosses bounded contexts, so one operation writes in more than one service", chain)
	}
	return fired
}

// busEndpoint reports whether a relation endpoint is a message bus. A bus that no
// relation reaches does not buffer anything, so it is no backpressure premise.
func busEndpoint(h HLD, endpoint string) bool {
	for _, container := range h.Containers {
		if container.ObjectID == endpoint {
			return busTechs[strings.ToLower(container.Tech)]
		}
	}
	return false
}

// endpointWritesState reports whether a relation endpoint is a container with an
// aggregate, or a context that holds one. A gateway, a bus and a store hold no
// aggregate, so an event they forward from their own state has no outbox premise.
func endpointWritesState(h HLD, endpoint string) bool {
	for _, aggregate := range h.Aggregates {
		if aggregate.ContainerID == endpoint {
			return true
		}
	}
	for _, container := range h.Containers {
		if container.BoundedContextID != endpoint {
			continue
		}
		for _, aggregate := range h.Aggregates {
			if aggregate.ContainerID == container.ObjectID {
				return true
			}
		}
	}
	return false
}

// crossContextChain finds A → B → C with three distinct endpoints and at least two
// bounded contexts among them. The HLD names no operations, so a chain across
// contexts is the strongest premise for a saga that the document holds.
// shortcut: a chain is not a command that must write both aggregates; refine when
// an operation names the aggregates it changes.
func crossContextChain(h HLD) (string, bool) {
	for _, first := range h.ContextRelations {
		for _, second := range h.ContextRelations {
			if first.To != second.From || first.From == second.To {
				continue
			}
			contexts := map[string]bool{}
			named := true
			for _, endpoint := range []string{first.From, first.To, second.To} {
				contextID, ok := endpointContext(h, endpoint)
				if !ok {
					named = false
					break
				}
				contexts[contextID] = true
			}
			if named && len(contexts) > 1 {
				return fmt.Sprintf("%s → %s → %s", first.From, first.To, second.To), true
			}
		}
	}
	return "", false
}

// endpointContext resolves a relation endpoint to its bounded context. A relation
// may name a container or a context; both are legal in the form.
func endpointContext(h HLD, endpoint string) (string, bool) {
	for _, container := range h.Containers {
		if container.ObjectID == endpoint {
			return container.BoundedContextID, container.BoundedContextID != ""
		}
	}
	for _, context := range h.BoundedContexts {
		if context.ID == endpoint {
			return context.ID, true
		}
	}
	return "", false
}

// validateDistributedData checks the D7 tree: every fired branch has a decision,
// and no decision answers a branch whose premise is absent.
func validateDistributedData(h HLD) []error {
	fired := d7Premises(h)
	answered := map[d7Branch]bool{}
	var errs []error

	for i, decision := range h.Decisions {
		if decision.Pattern == "" {
			continue
		}
		if !decision.Pattern.Valid() {
			errs = append(errs, invalid(fmt.Sprintf("decisions[%d].pattern", i),
				fmt.Sprintf("unknown value %q; name one of %s", decision.Pattern, joinPatterns(DistributedPatterns()))))
			continue
		}
		branch := patternBranches[decision.Pattern]
		if _, ok := fired[branch]; !ok {
			errs = append(errs, invalid(fmt.Sprintf("decisions[%d].pattern", i),
				fmt.Sprintf("%q answers the %s premise, but the design has no such premise", decision.Pattern, branch)))
			continue
		}
		answered[branch] = true
	}

	for _, branch := range d7Order {
		premise, ok := fired[branch]
		if !ok || answered[branch] {
			continue
		}
		errs = append(errs, invalid("decisions",
			fmt.Sprintf("%s; add one decision with pattern %s", premise, joinPatterns(patternsOfBranch(branch)))))
	}
	return errs
}

func joinPatterns(patterns []DistributedPattern) string {
	names := make([]string, len(patterns))
	for i, pattern := range patterns {
		names[i] = string(pattern)
	}
	return strings.Join(names, ", ")
}

// DeploymentPattern is where the containers run: one of the catalogue's
// deployment patterns. The pattern decides the checklist a later stage reads.
type DeploymentPattern string

const (
	DeploymentMultipleServicesPerHost DeploymentPattern = "multiple_services_per_host"
	DeploymentServicePerHost          DeploymentPattern = "service_per_host"
	DeploymentServicePerVM            DeploymentPattern = "service_per_vm"
	DeploymentServicePerContainer     DeploymentPattern = "service_per_container"
	DeploymentServerless              DeploymentPattern = "serverless"
	DeploymentServicePlatform         DeploymentPattern = "service_deployment_platform"
)

var deploymentPalette = []DeploymentPattern{
	DeploymentMultipleServicesPerHost,
	DeploymentServicePerHost,
	DeploymentServicePerVM,
	DeploymentServicePerContainer,
	DeploymentServerless,
	DeploymentServicePlatform,
}

// Valid reports whether the pattern belongs to the D8 palette.
func (d DeploymentPattern) Valid() bool {
	for _, pattern := range deploymentPalette {
		if pattern == d {
			return true
		}
	}
	return false
}

// DeploymentPatterns lists the palette for the prompt and the classifier.
func DeploymentPatterns() []DeploymentPattern {
	return append([]DeploymentPattern(nil), deploymentPalette...)
}

// DeploymentChecklist is the D8 checklist of one pattern. The system writes it
// into the decision, so the operator does not read a generic "use Kubernetes".
func DeploymentChecklist(pattern DeploymentPattern) []string {
	switch pattern {
	case DeploymentMultipleServicesPerHost:
		return []string{
			"one process per service and one port map per host",
			"a host failure takes every service on that host with it",
			"one patch window and one kernel for every service",
		}
	case DeploymentServicePerHost:
		return []string{
			"one host per service, versioned as an image",
			"a health check restarts the process",
			"the host is patched without a shared blast radius",
		}
	case DeploymentServicePerVM:
		return []string{
			"a systemd unit per service",
			"log rotation and disk alerts",
			"a disk and configuration backup",
			"a rolling restart with a health check",
		}
	case DeploymentServicePerContainer:
		return []string{
			"one image per container, tagged by version",
			"a readiness probe on a real endpoint",
			"resource requests and limits",
			"externalized configuration",
			"secrets injected, not baked into the image",
		}
	case DeploymentServerless:
		return []string{
			"a cold start budget on the p99",
			"a concurrency limit per function",
			"a payload size limit",
			"state outside the function",
			"idempotency on retries",
		}
	case DeploymentServicePlatform:
		return []string{
			"one deployment per container",
			"a replica floor and ceiling",
			"a pod disruption budget",
			"readiness and liveness probes on real endpoints",
			"resource requests and limits",
			"autoscaling on a real metric",
			"externalized configuration",
			"secrets injected by the platform",
			"a network policy between bounded contexts",
		}
	}
	return nil
}

// validateDeployment checks D8: exactly one deployment pattern, from the closed
// list, and it agrees with the architecture style on the serverless case.
func validateDeployment(h HLD) []error {
	var errs []error
	chosen := DeploymentPattern("")
	count := 0
	for i, decision := range h.Decisions {
		if decision.Deployment == "" {
			continue
		}
		if !decision.Deployment.Valid() {
			errs = append(errs, invalid(fmt.Sprintf("decisions[%d].deployment", i),
				fmt.Sprintf("unknown value %q; name one of %s", decision.Deployment, strings.Join(deploymentNames(), ", "))))
			continue
		}
		if decision.Area != AreaOps {
			errs = append(errs, invalid(fmt.Sprintf("decisions[%d].area", i),
				fmt.Sprintf("a deployment decision belongs to area %q, got %q", AreaOps, decision.Area)))
		}
		count++
		chosen = decision.Deployment
	}
	if count == 0 {
		errs = append(errs, invalid("decisions",
			fmt.Sprintf("the design has no deployment pattern; add one decision with area %q and deployment from %s", AreaOps, strings.Join(deploymentNames(), ", "))))
		return errs
	}
	if count > 1 {
		errs = append(errs, invalid("decisions", fmt.Sprintf("the design has %d deployment patterns; keep one", count)))
	}

	style, hasStyle := architectureStyleOf(h.Decisions)
	if chosen == DeploymentServerless && (!hasStyle || style != StyleServerless) {
		errs = append(errs, invalid("decisions",
			fmt.Sprintf("deployment %q needs the serverless architecture style; the design chose %q", chosen, style)))
	}
	if hasStyle && style == StyleServerless && chosen != DeploymentServerless {
		errs = append(errs, invalid("decisions",
			fmt.Sprintf("the serverless style deploys as serverless, not %q", chosen)))
	}
	return errs
}

func deploymentNames() []string {
	out := make([]string, len(deploymentPalette))
	for i, pattern := range deploymentPalette {
		out[i] = string(pattern)
	}
	return out
}

// architectureStyleOf reads the style the architecture decision chose.
func architectureStyleOf(decisions []Decision) (ArchitectureStyle, bool) {
	for _, decision := range decisions {
		if decision.Area == AreaArchitecture && decision.Style.Valid() {
			return decision.Style, true
		}
	}
	return "", false
}
