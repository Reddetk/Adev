package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Reddetk/Adev/internal/domain"
)

// entryRepair is one batched entry the gates rejected: the plan that owns the list,
// the entry id the walk addresses it by, and the fields inside it that came back
// empty.
type entryRepair struct {
	plan     batchPlan
	index    int
	id       string
	fields   []string // the field names inside the entry
	problems []string // "field: reason" pairs, the way the gate stated them
}

// maxEntryRepairs caps the repair rounds of one stage attempt: a model that cannot
// fill an entry twice leaves the document to the whole-document retry.
const maxEntryRepairs = 2

// maxEntriesPerRepair caps the calls of one round, so a document with a dozen
// broken entries does not turn into a second generation.
const maxEntriesPerRepair = 5

// entryRepairs maps the gate failures onto the batched entries they belong to. A
// field inside a list entry is repaired with one call for that entry; a field
// outside every list has no entry and stays with the whole-document retry.
func entryRepairs(output domain.FormType, payload json.RawMessage, failures []domain.ValidationError) []entryRepair {
	plans := batchPlans[output]
	if len(plans) == 0 || len(payload) == 0 {
		return nil
	}
	byKey := map[string]*entryRepair{}
	var order []string
	for _, failure := range failures {
		pointer := pointerOf(failure.Field)
		segments := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
		if len(segments) < 2 {
			continue
		}
		plan, ok := planOfKind(plans, segments[0])
		if !ok {
			continue
		}
		index, err := strconv.Atoi(segments[1])
		if err != nil || index < 0 {
			continue
		}
		entries, err := listOf(payload, plan.kind)
		if err != nil || index >= len(entries) {
			continue
		}
		id := fieldOf(entries[index], plan.idFieldOf())
		if id == "" {
			continue
		}
		key := plan.kind + "/" + strconv.Itoa(index)
		repair, seen := byKey[key]
		if !seen {
			repair = &entryRepair{plan: plan, index: index, id: id}
			byKey[key] = repair
			order = append(order, key)
		}
		// The field is named relative to the entry, the way a detail answer names it.
		field := strings.Join(segments[2:], ".")
		repair.fields = append(repair.fields, field)
		repair.problems = append(repair.problems, field+": "+failure.Problem)
	}
	out := make([]entryRepair, 0, len(order))
	for _, key := range order {
		out = append(out, *byKey[key])
	}
	return out
}

// planOfKind finds the plan that owns a list, at the root or inside a root entry.
func planOfKind(plans []batchPlan, kind string) (batchPlan, bool) {
	for _, plan := range plans {
		if plan.kind == kind {
			return plan, true
		}
		if child, ok := planOfKind(plan.children, kind); ok {
			return child, true
		}
	}
	return batchPlan{}, false
}

// repairEntries asks the model once per rejected entry, with the fields that came
// back empty, and merges the answer back into the document. A failed call keeps the
// entry as it stands: the gates judge it again.
func (s *StageService) repairEntries(
	ctx context.Context, request StageRequest, client LLM, agent domain.AgentConfig,
	stage domain.Stage, output domain.FormType, shape string, stageContext json.RawMessage,
	payload json.RawMessage, repairs []entryRepair, checkpoint detailCheckpoint,
) json.RawMessage {
	if len(repairs) > maxEntriesPerRepair {
		repairs = repairs[:maxEntriesPerRepair]
	}
	for _, repair := range repairs {
		entries, err := listOf(payload, repair.plan.kind)
		if err != nil || repair.index >= len(entries) {
			continue
		}
		labels := []entryLabel{{plan: repair.plan, idValue: repair.id}}
		extra := fmt.Sprintf(" The gates rejected these fields of this entry: %s. Correct them and repeat the whole entry.",
			strings.Join(repair.problems, "; "))
		echo := s.detailCall(ctx, request, client, agent, stage, output, shape, stageContext, labels, entries[repair.index], repair.plan, payload, extra)
		if len(echo) == 0 {
			continue
		}
		echo = withEntryKeys(echo, repair.plan, labels)
		updated, ok := entryFromEcho(echo, labels)
		if !ok {
			log.Printf("stage %s: repair %s %s: the answer does not carry the entry", stage.ID, repair.plan.kind, repair.id)
			continue
		}
		entries[repair.index] = mergeEntryFields(entries[repair.index], updated)
		merged, err := setList(payload, repair.plan.kind, entries)
		if err != nil {
			log.Printf("stage %s: repair %s: store the entry: %v", stage.ID, repair.plan.kind, err)
			continue
		}
		payload = merged
		if checkpoint != nil {
			if saveErr := checkpoint(payload); saveErr != nil {
				log.Printf("stage %s: save the repaired %s: %v", stage.ID, repair.plan.kind, saveErr)
			}
		}
	}
	return payload
}
