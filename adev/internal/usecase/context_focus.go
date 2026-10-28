package usecase

import (
	"encoding/json"

	"github.com/Reddetk/Adev/internal/domain"
)

// focusContext returns the part of the stage context that one entry needs. The
// HLD is filled container by container and the LLD entry by entry: a call reads
// the branch of its container or contract, not the whole design. Every other
// stage and the skeleton call keep the whole context.
func focusContext(output domain.FormType, stageContext json.RawMessage, labels []entryLabel, entry json.RawMessage) json.RawMessage {
	if (output != domain.TypeLLD && output != domain.TypeHLD) || len(labels) == 0 || len(stageContext) == 0 {
		return stageContext
	}
	var forms map[string]json.RawMessage
	if err := json.Unmarshal(stageContext, &forms); err != nil {
		return stageContext
	}
	kind, id := labels[len(labels)-1].plan.kind, labels[len(labels)-1].idValue
	containerID := ""
	if labels[0].plan.kind == "containers" {
		containerID = labels[0].idValue
	}
	containers, contexts := map[string]bool{}, map[string]bool{}
	var nfrRefs []string
	if output == domain.TypeHLD {
		// The container entry of the skeleton names its bounded context and the
		// NFRs that drive its technology. A v2 call reads the same values from the
		// input HLD when the skeleton left them empty.
		boundedContextID, refs := entryBranch(entry)
		nfrRefs = refs
		for _, form := range forms {
			_, payload, formType := decodeForm(form)
			if formType != string(domain.TypeHLD) {
				continue
			}
			bounded, refs := containerBranch(payload, containerID)
			if boundedContextID == "" {
				boundedContextID = bounded
			}
			if len(nfrRefs) == 0 {
				nfrRefs = refs
			}
		}
		if containerID != "" {
			containers[containerID] = true
		}
		if boundedContextID != "" {
			contexts[boundedContextID] = true
		}
	} else {
		for _, form := range forms {
			_, payload, formType := decodeForm(form)
			if formType != string(domain.TypeHLD) {
				continue
			}
			branchOf(payload, kind, id, containerID, containers, contexts)
		}
	}
	focused := make(map[string]json.RawMessage, len(forms))
	for key, form := range forms {
		envelope, payload, formType := decodeForm(form)
		switch formType {
		case string(domain.TypeHLD):
			payload = focusHLD(payload, contractOf(kind, id), containers, contexts)
		case string(domain.TypeADR):
			payload = focusADR(payload, containers)
		case string(domain.TypeSRS):
			if output != domain.TypeHLD {
				focused[key] = form
				continue
			}
			payload = focusSRS(payload, nfrRefs)
		default:
			focused[key] = form
			continue
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			focused[key] = form
			continue
		}
		envelope = trimEnvelope(envelope, encoded)
		form, err = json.Marshal(envelope)
		if err != nil {
			focused[key] = form
			continue
		}
		focused[key] = form
	}
	out, err := json.Marshal(focused)
	if err != nil {
		return stageContext
	}
	return out
}

// decodeForm takes the payload and the type out of one projected form.
func decodeForm(form json.RawMessage) (map[string]json.RawMessage, map[string]json.RawMessage, string) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(form, &envelope); err != nil {
		return nil, nil, ""
	}
	var formType string
	_ = json.Unmarshal(envelope["formType"], &formType)
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(envelope["payload"], &payload); err != nil {
		return envelope, nil, formType
	}
	return envelope, payload, formType
}

// branchOf fills the container and context sets of one entry: a contract names
// its relation, a container names its bounded context.
func branchOf(payload map[string]json.RawMessage, kind, id, containerID string, containers, contexts map[string]bool) {
	switch {
	case kind == "contracts" && id != "":
		for _, relation := range jsonRows(payload, "contextRelations") {
			if fieldOf(relation, "contractId") != id {
				continue
			}
			contexts[fieldOf(relation, "from")] = true
			contexts[fieldOf(relation, "to")] = true
		}
	case containerID != "":
		containers[containerID] = true
		for _, container := range jsonRows(payload, "containers") {
			if fieldOf(container, "objectId") == containerID {
				contexts[fieldOf(container, "boundedContextId")] = true
			}
		}
	}
	// A context names its containers: the branch follows the context.
	for _, context := range jsonRows(payload, "boundedContexts") {
		if !contexts[fieldOf(context, "id")] {
			continue
		}
		for _, containerID := range jsonStrings(context, "containerIds") {
			containers[containerID] = true
		}
	}
	delete(containers, "")
	delete(contexts, "")
}

// entryBranch reads the branch a container entry names: the bounded context it
// implements and the NFRs that drive its technology.
func entryBranch(entry json.RawMessage) (string, []string) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(entry, &object); err != nil {
		return "", nil
	}
	return fieldOf(entry, "boundedContextId"), jsonStrings(entry, "nfrRefs")
}

// containerBranch finds one container in an input HLD and returns its bounded
// context and NFR refs: a v2 call reads what v1 already decided.
func containerBranch(payload map[string]json.RawMessage, containerID string) (string, []string) {
	if containerID == "" {
		return "", nil
	}
	for _, container := range jsonRows(payload, "containers") {
		if fieldOf(container, "objectId") != containerID {
			continue
		}
		return fieldOf(container, "boundedContextId"), jsonStrings(container, "nfrRefs")
	}
	return "", nil
}

// focusSRS keeps the quality requirements the container names. The functional
// requirements have no container link, so they all stay: the call needs the
// behavior the container must carry.
func focusSRS(payload map[string]json.RawMessage, nfrRefs []string) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(payload))
	for key, value := range payload {
		out[key] = value
	}
	if len(nfrRefs) == 0 {
		return out
	}
	keep := make(map[string]bool, len(nfrRefs))
	for _, ref := range nfrRefs {
		keep[ref] = true
	}
	out["nfr"] = keepRows(payload["nfr"], func(row json.RawMessage) bool {
		return keep[fieldOf(row, "id")]
	})
	return out
}

// focusHLD keeps the lists of the branch and passes the rest of the document
// through: the ubiquitous language and the degradation strategy are short and
// hold the terms the answer uses.
func focusHLD(payload map[string]json.RawMessage, contract string, containers, contexts map[string]bool) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(payload))
	for key, value := range payload {
		out[key] = value
	}
	out["containers"] = keepRows(payload["containers"], func(row json.RawMessage) bool {
		return containers[fieldOf(row, "objectId")]
	})
	out["aggregates"] = keepRows(payload["aggregates"], func(row json.RawMessage) bool {
		return containers[fieldOf(row, "containerId")]
	})
	out["boundedContexts"] = keepRows(payload["boundedContexts"], func(row json.RawMessage) bool {
		if contexts[fieldOf(row, "id")] {
			return true
		}
		for _, containerID := range jsonStrings(row, "containerIds") {
			if containers[containerID] {
				return true
			}
		}
		return false
	})
	out["contextRelations"] = keepRows(payload["contextRelations"], func(row json.RawMessage) bool {
		if contract != "" {
			return fieldOf(row, "contractId") == contract
		}
		return contexts[fieldOf(row, "from")] || contexts[fieldOf(row, "to")]
	})
	return out
}

// contractOf names the contract a call details; every other list has none.
func contractOf(kind, id string) string {
	if kind == "contracts" {
		return id
	}
	return ""
}

// focusADR keeps the decisions that name a container of the branch and cuts them
// to the decision itself: the rejected alternatives were weighed in the ADR stage.
// A decision without object refs constrains the whole system, so it stays.
func focusADR(payload map[string]json.RawMessage, containers map[string]bool) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(payload))
	for key, value := range payload {
		out[key] = value
	}
	kept := keepRows(payload["decisions"], func(row json.RawMessage) bool {
		refs := jsonStrings(row, "objectRefs")
		if len(refs) == 0 {
			return true
		}
		for _, objectID := range refs {
			if containers[objectID] {
				return true
			}
		}
		return false
	})
	var rows []json.RawMessage
	if err := json.Unmarshal(kept, &rows); err != nil {
		return out
	}
	trimmed := make([]json.RawMessage, 0, len(rows))
	for _, row := range rows {
		trimmed = append(trimmed, trimDecision(row))
	}
	encoded, err := json.Marshal(trimmed)
	if err != nil {
		return out
	}
	out["decisions"] = encoded
	return out
}

// trimDecision keeps the fields a downstream call reads: which decision, what it
// says, and its status.
func trimDecision(row json.RawMessage) json.RawMessage {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(row, &object); err != nil {
		return row
	}
	out := make(map[string]json.RawMessage, 4)
	for _, key := range []string{"id", "title", "decision", "status"} {
		if value, ok := object[key]; ok {
			out[key] = value
		}
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return row
	}
	return encoded
}

// trimEnvelope keeps the fields a call reads from one input form: the id, the
// type, the version and the payload. The chain and the hashes belong to the
// journal, not to the prompt.
func trimEnvelope(envelope map[string]json.RawMessage, payload json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, 4)
	for _, key := range []string{"formId", "formType", "formVersion"} {
		if value, ok := envelope[key]; ok {
			out[key] = value
		}
	}
	out["payload"] = payload
	return out
}

// keepRows filters one list of a payload. An empty result keeps the original
// list: a branch without rows tells the model less than the whole list.
func keepRows(raw json.RawMessage, keep func(json.RawMessage) bool) json.RawMessage {
	var rows []json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
		return raw
	}
	kept := make([]json.RawMessage, 0, len(rows))
	for _, row := range rows {
		if keep(row) {
			kept = append(kept, row)
		}
	}
	if len(kept) == 0 || len(kept) == len(rows) {
		return raw
	}
	encoded, err := json.Marshal(kept)
	if err != nil {
		return raw
	}
	return encoded
}

// jsonRows decodes one list of a payload; a missing list is nil.
func jsonRows(payload map[string]json.RawMessage, kind string) []json.RawMessage {
	var rows []json.RawMessage
	_ = json.Unmarshal(payload[kind], &rows)
	return rows
}

// jsonStrings decodes one list of strings of an entry.
func jsonStrings(entry json.RawMessage, field string) []string {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(entry, &object); err != nil {
		return nil
	}
	var values []string
	_ = json.Unmarshal(object[field], &values)
	return values
}
