package domain

import (
	"fmt"
	"strings"
)

// ClassifyDeployment chooses the D8 pattern from the architecture style. The
// classifier is a first, deterministic step: the model explains the choice, the
// system does not let the deployment field land on the architecture decision.
func ClassifyDeployment(style ArchitectureStyle) (DeploymentPattern, bool) {
	switch style {
	case StyleServerless:
		return DeploymentServerless, true
	case StyleMicroservices, StyleEventDriven, StyleCellBased:
		return DeploymentServicePerContainer, true
	case StyleMonolith, StyleModularMonolith, StyleHexagonal:
		return DeploymentServicePerHost, true
	}
	return "", false
}

// NormalizeHLD applies the structure the design must hold after the model answer:
// one architecture decision with its style, one ops decision with its deployment,
// and the store container the storage decision names. The model writes meaning;
// the system keeps the fork in the field that owns it.
func NormalizeHLD(h *HLD) {
	if h == nil {
		return
	}
	normalizeArchitectureDecision(h)
	normalizeDeployment(h)
	normalizeStorageContainer(h)
}

// normalizeArchitectureDecision keeps the style on the architecture decision. A
// model that moves the architecture decision to area "ops" to satisfy the
// deployment rule loses the style; the style is the stronger fact.
func normalizeArchitectureDecision(h *HLD) {
	for i := range h.Decisions {
		decision := &h.Decisions[i]
		if decision.Style.Valid() {
			decision.Area = AreaArchitecture
			return
		}
	}
}

func normalizeDeployment(h *HLD) {
	style, hasStyle := architectureStyleOf(h.Decisions)
	chosen := DeploymentPattern("")
	for i := range h.Decisions {
		decision := &h.Decisions[i]
		if decision.Deployment == "" {
			continue
		}
		if decision.Area == AreaOps && chosen == "" && decision.Deployment.Valid() {
			chosen = decision.Deployment
		}
		decision.Deployment = ""
	}
	if chosen == "" && hasStyle {
		chosen, _ = ClassifyDeployment(style)
	}
	if chosen == "" {
		return
	}

	var ops *Decision
	for i := range h.Decisions {
		if h.Decisions[i].Area == AreaOps {
			ops = &h.Decisions[i]
			break
		}
	}
	if ops == nil {
		h.Decisions = append(h.Decisions, Decision{
			ID:     nextDecisionID(h.Decisions, "d-deployment"),
			Area:   AreaOps,
			Title:  Text{Value: "Deployment"},
			Status: DecisionProposed,
			Alternatives: []Alternative{{
				Option:          Text{Value: "Another deployment pattern"},
				RejectedBecause: Text{Value: "The architecture style and the operations maturity decide."},
			}},
		})
		ops = &h.Decisions[len(h.Decisions)-1]
	}
	ops.Deployment = chosen
	ops.Checklist = DeploymentChecklist(chosen)
	if ops.Title.Value == "" {
		ops.Title = Text{Value: "Deployment"}
	}
	if ops.Decision.Value == "" {
		ops.Decision = Text{Value: fmt.Sprintf("The containers deploy with %s.", chosen)}
	}
	if len(ops.Alternatives) == 0 {
		ops.Alternatives = []Alternative{{
			Option:          Text{Value: "Another deployment pattern"},
			RejectedBecause: Text{Value: "The architecture style and the operations maturity decide."},
		}}
	}
}

// AddExternalSystems creates the C4 external systems the BRD assumptions name and
// the relations from the internal container that uses them. The assumption is
// free text today, so the match is a keyword scan; the SRS FR supplies the cause
// and the delivery premise.
// shortcut: keyword detection from free-text assumptions; replace with a
// structured BRD externalSystems[] field when the BRD form grows one.
func AddExternalSystems(h *HLD, brd BRD, srs SRS) {
	if h == nil {
		return
	}
	assumptions := strings.ToLower(textsOf(brd.Assumptions))
	payment := containsAny(assumptions, "payment", "card", "payout", "bank", "terminal")
	notification := containsAny(assumptions, "email", "push", "sms", "notification")
	if !payment && !notification {
		return
	}

	if payment {
		if fr, ok := firstRequirement(srs, "payment", "card", "payout", "bank"); ok {
			if source, ok := firstSource(*h, "payment", "payout", "card"); ok {
				addExternal(h, "c-payment-gateway", "Process card payments and settle payouts.", source, fr)
			}
		}
	}
	if notification {
		// "email" alone also appears in the sign-in flow; prefer the wording of a
		// notification, then fall back to the channel word.
		fr, ok := firstRequirement(srs, "notification", "notify", "learns about", "deadline", "push", "sms")
		if !ok {
			fr, ok = firstRequirement(srs, "email")
		}
		if ok {
			source, ok := interactionSource(*h, fr.ID)
			if !ok {
				source, ok = firstSource(*h, "notification", "progress", "dashboard")
			}
			if ok {
				addExternal(h, "c-notification-provider", "Send email and push notifications.", source, fr)
			}
		}
	}
}

// HasExternalSystem reports whether the design names an external system.
func HasExternalSystem(h HLD) bool {
	for _, container := range h.Containers {
		if container.External() {
			return true
		}
	}
	return false
}

// ExternalSystemPremise reports whether the BRD assumptions name a system this
// design does not deploy, so the HLD has to show it.
func ExternalSystemPremise(brd BRD) bool {
	assumptions := strings.ToLower(textsOf(brd.Assumptions))
	return containsAny(assumptions, "payment", "card", "payout", "bank", "terminal",
		"email", "push", "sms", "notification")
}

func addExternal(h *HLD, id, purpose, source string, fr FunctionalRequirement) {
	for _, container := range h.Containers {
		if container.ObjectID == id {
			return
		}
	}
	h.Containers = append(h.Containers, Container{
		ObjectID: id,
		Tech:     TechExternal,
		Purpose:  Text{Value: purpose},
	})
	for _, relation := range h.ContextRelations {
		if relation.From == source && relation.To == id {
			return
		}
	}
	relation := ContextRelation{
		From: source, To: id,
		Description: Text{Value: purpose},
		CauseRef:    fr.ID,
		Type:        RelationSync,
	}
	if fr.Delivery != DeliverySync {
		relation.Type = RelationAsync
		relation.Guarantee = fr.Delivery
	}
	h.ContextRelations = append(h.ContextRelations, relation)
}

func textsOf(values []Text) string {
	var builder strings.Builder
	for _, value := range values {
		builder.WriteString(value.Value)
		builder.WriteString(" ")
	}
	return builder.String()
}

func containsAny(value string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(value, word) {
			return true
		}
	}
	return false
}

func firstRequirement(srs SRS, words ...string) (FunctionalRequirement, bool) {
	for _, fr := range srs.FR {
		text := strings.ToLower(fr.Statement.Text + " " + fr.Trigger + " " + fr.Response + " " + fr.StateChange)
		if containsAny(text, words...) {
			return fr, true
		}
	}
	return FunctionalRequirement{}, false
}

func firstSource(h HLD, words ...string) (string, bool) {
	for _, container := range h.Containers {
		if container.External() {
			continue
		}
		if containsAny(strings.ToLower(container.Purpose.Value), words...) {
			return container.ObjectID, true
		}
	}
	return "", false
}

func interactionSource(h HLD, causeRef string) (string, bool) {
	for _, interaction := range h.ActorInteractions {
		if interaction.CauseRef == causeRef {
			return interaction.ContainerID, true
		}
	}
	return "", false
}

// A decision that says "PostgreSQL" and leaves no container is a product the
// design does not have; the system creates the box the decision implies.
func normalizeStorageContainer(h *HLD) {
	for i := range h.Decisions {
		decision := &h.Decisions[i]
		if decision.Area != AreaStorage {
			continue
		}
		for _, tech := range decision.Techs {
			normalized := strings.ToLower(tech)
			if !storeTechs[normalized] || containerWithTech(h, normalized) {
				continue
			}
			contextID := firstRefContext(h, decision.ObjectRefs)
			if contextID == "" && len(h.BoundedContexts) > 0 {
				contextID = h.BoundedContexts[0].ID
			}
			if contextID == "" {
				continue
			}
			id := nextContainerID(h.Containers, "c-store")
			h.Containers = append(h.Containers, Container{
				ObjectID:         id,
				BoundedContextID: contextID,
				Tech:             normalized,
				Purpose:          Text{Value: "Store the system of record."},
			})
			for j := range h.BoundedContexts {
				if h.BoundedContexts[j].ID == contextID {
					h.BoundedContexts[j].ContainerIds = append(h.BoundedContexts[j].ContainerIds, id)
					break
				}
			}
			decision.ObjectRefs = append(decision.ObjectRefs, id)
		}
	}
}

func containerWithTech(h *HLD, tech string) bool {
	for _, container := range h.Containers {
		if strings.ToLower(container.Tech) == tech {
			return true
		}
	}
	return false
}

func firstRefContext(h *HLD, refs []string) string {
	for _, ref := range refs {
		for _, container := range h.Containers {
			if container.ObjectID == ref {
				return container.BoundedContextID
			}
		}
	}
	return ""
}

func nextDecisionID(decisions []Decision, base string) string {
	used := map[string]bool{}
	for _, decision := range decisions {
		used[decision.ID] = true
	}
	if !used[base] {
		return base
	}
	for i := 2; ; i++ {
		id := fmt.Sprintf("%s-%d", base, i)
		if !used[id] {
			return id
		}
	}
}

func nextContainerID(containers []Container, base string) string {
	used := map[string]bool{}
	for _, container := range containers {
		used[container.ObjectID] = true
	}
	if !used[base] {
		return base
	}
	for i := 2; ; i++ {
		id := fmt.Sprintf("%s-%d", base, i)
		if !used[id] {
			return id
		}
	}
}
