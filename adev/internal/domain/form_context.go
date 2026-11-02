package domain

import (
	"fmt"
	"strings"
)

// validateSystemBoundary keeps C4 levels apart: a bounded context holds containers
// of this system, never an external system. An external system is a box on the
// context diagram, not a deployable unit with a schema of its own.
func validateSystemBoundary(h HLD, containers, external map[string]bool) []error {
	var errs []error
	for i, context := range h.BoundedContexts {
		field := fmt.Sprintf("boundedContexts[%d].containerIds", i)
		for _, id := range context.ContainerIds {
			switch {
			case external[id]:
				errs = append(errs, invalid(field,
					fmt.Sprintf("the external system %q cannot sit inside a bounded context", id)))
			case !containers[id]:
				errs = append(errs, invalid(field,
					fmt.Sprintf("%q does not exist in containers", id)))
			}
		}
	}
	return errs
}

// validateActors checks the C4 people: an actor has a name and an identity, and
// an interaction reaches a container of this system. A person never talks to the
// payment provider directly; that is a relation of the container.
func validateActors(h HLD, containers, external map[string]bool) []error {
	var errs []error
	ids := map[string]bool{}
	for i, actor := range h.Actors {
		field := fmt.Sprintf("actors[%d]", i)
		if actor.ID == "" {
			errs = append(errs, invalid(field+".id", "is empty"))
		} else if ids[actor.ID] {
			errs = append(errs, invalid(field+".id", fmt.Sprintf("%q is used twice", actor.ID)))
		}
		if actor.Name == "" {
			errs = append(errs, invalid(field+".name", "is empty"))
		}
		ids[actor.ID] = true
	}

	interacted := map[string]bool{}
	for i, interaction := range h.ActorInteractions {
		field := fmt.Sprintf("actorInteractions[%d]", i)
		if interaction.ActorID == "" || !ids[interaction.ActorID] {
			errs = append(errs, invalid(field+".actorId",
				fmt.Sprintf("%q is not an actor of this design", interaction.ActorID)))
		}
		if interaction.ContainerID == "" || !containers[interaction.ContainerID] {
			errs = append(errs, invalid(field+".containerId",
				fmt.Sprintf("%q is not a container of this design", interaction.ContainerID)))
		} else if external[interaction.ContainerID] {
			errs = append(errs, invalid(field+".containerId",
				fmt.Sprintf("%q is an external system; an actor reaches a container of this system", interaction.ContainerID)))
		}
		if interaction.Description.Value == "" {
			errs = append(errs, invalid(field+".description", "must name the business action the actor performs"))
		}
		if interaction.CauseRef == "" {
			errs = append(errs, invalid(field+".causeRef", "must name the SRS requirement"))
		}
		interacted[interaction.ActorID] = true
	}
	for i, actor := range h.Actors {
		if !interacted[actor.ID] {
			errs = append(errs, invalid(fmt.Sprintf("actors[%d]", i),
				fmt.Sprintf("%q has no interaction; an actor with no arrow explains nothing", actor.Name)))
		}
	}
	return errs
}

// ValidateHLDAgainstInputs checks the seam between the HLD and the forms it
// reads: every BRD actor is a person on the diagram, every interaction and edge
// names an SRS requirement, the delivery premise decides the edge type, and every
// requirement and container is carried by the flow.
func ValidateHLDAgainstInputs(h HLD, brd BRD, srs SRS) []error {
	var errs []error

	brdByID := map[string]Actor{}
	brdNames := map[string]bool{}
	for _, actor := range brd.Actors {
		brdByID[actor.ID] = actor
		brdNames[strings.ToLower(strings.TrimSpace(actor.Name))] = true
	}

	actorsByID := map[string]HLDActor{}
	actorsByBRD := map[string]bool{}
	for i, actor := range h.Actors {
		field := fmt.Sprintf("actors[%d]", i)
		if actor.BrdActorRef == "" {
			errs = append(errs, invalid(field+".brdActorRef", "must name the BRD actor this person comes from"))
		} else if _, ok := brdByID[actor.BrdActorRef]; !ok {
			errs = append(errs, invalid(field+".brdActorRef", fmt.Sprintf("%q is not a BRD actor", actor.BrdActorRef)))
		} else {
			actorsByBRD[actor.BrdActorRef] = true
			if !brdNames[strings.ToLower(strings.TrimSpace(actor.Name))] {
				errs = append(errs, invalid(field+".name",
					fmt.Sprintf("%q is not the name of that BRD actor", actor.Name)))
			}
		}
		actorsByID[actor.ID] = actor
	}
	for _, actor := range brd.Actors {
		if !actorsByBRD[actor.ID] {
			errs = append(errs, invalid("actors",
				fmt.Sprintf("the BRD actor %q (%s) has no C4 person", actor.Name, actor.ID)))
		}
	}

	frByID := map[string]FunctionalRequirement{}
	for _, fr := range srs.FR {
		frByID[fr.ID] = fr
	}
	covered := map[string]bool{}

	for i, interaction := range h.ActorInteractions {
		field := fmt.Sprintf("actorInteractions[%d]", i)
		if interaction.Description.Value == "" {
			errs = append(errs, invalid(field+".description", "must name the business action the actor performs"))
		}
		if interaction.CauseRef == "" {
			errs = append(errs, invalid(field+".causeRef", "must name the SRS requirement"))
			continue
		}
		fr, ok := frByID[interaction.CauseRef]
		if !ok {
			errs = append(errs, invalid(field+".causeRef",
				fmt.Sprintf("%q is not an SRS requirement", interaction.CauseRef)))
			continue
		}
		covered[fr.ID] = true
		actor, ok := actorsByID[interaction.ActorID]
		if !ok {
			continue
		}
		if fr.Actor != "" && !strings.EqualFold(strings.TrimSpace(actor.Name), strings.TrimSpace(fr.Actor)) {
			errs = append(errs, invalid(field+".actorId",
				fmt.Sprintf("the actor %q does not match the actor %q of requirement %q", actor.Name, fr.Actor, fr.ID)))
		}
	}

	for i, relation := range h.ContextRelations {
		field := fmt.Sprintf("contextRelations[%d]", i)
		if relation.Description.Value == "" {
			errs = append(errs, invalid(field+".description", "must name the business action the edge carries"))
		}
		if relation.CauseRef == "" {
			errs = append(errs, invalid(field+".causeRef", "must name the SRS requirement"))
			continue
		}
		fr, ok := frByID[relation.CauseRef]
		if !ok {
			errs = append(errs, invalid(field+".causeRef",
				fmt.Sprintf("%q is not an SRS requirement", relation.CauseRef)))
			continue
		}
		covered[fr.ID] = true

		switch fr.Delivery {
		case DeliverySync:
			if relation.Type != RelationSync {
				errs = append(errs, invalid(field+".type",
					fmt.Sprintf("requirement %q is sync, so the edge is sync", fr.ID)))
			}
			if relation.Guarantee != "" {
				errs = append(errs, invalid(field+".guarantee",
					fmt.Sprintf("requirement %q is sync and carries no delivery guarantee", fr.ID)))
			}
		case DeliveryAtMostOnce, DeliveryAtLeastOnce, DeliveryExactlyOnce:
			if relation.Type != RelationAsync {
				errs = append(errs, invalid(field+".type",
					fmt.Sprintf("requirement %q is async, so the edge is async", fr.ID)))
			}
			if relation.Guarantee != fr.Delivery {
				errs = append(errs, invalid(field+".guarantee",
					fmt.Sprintf("requirement %q promises %s, so the edge promises it too", fr.ID, fr.Delivery)))
			}
		}
		if fr.FailureMode == FailureRetry && relation.Type == RelationSync && !relation.Retry {
			errs = append(errs, invalid(field+".retry",
				fmt.Sprintf("requirement %q retries a failed call, so the edge names the retry", fr.ID)))
		}
	}

	for _, fr := range srs.FR {
		if !covered[fr.ID] {
			errs = append(errs, invalid("actorInteractions/contextRelations",
				fmt.Sprintf("requirement %q is not carried by any actor interaction or relation", fr.ID)))
		}
	}

	used := map[string]bool{}
	for _, relation := range h.ContextRelations {
		used[relation.From] = true
		used[relation.To] = true
	}
	for _, interaction := range h.ActorInteractions {
		used[interaction.ContainerID] = true
	}
	for i, container := range h.Containers {
		if !used[container.ObjectID] {
			errs = append(errs, invalid(fmt.Sprintf("containers[%d]", i),
				fmt.Sprintf("container %q is not reached by any relation or actor interaction", container.ObjectID)))
		}
	}
	return errs
}
