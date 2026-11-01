package usecase

import (
	"encoding/json"

	"github.com/Reddetk/Adev/internal/domain"
)

// inheritAccepted keeps the values of the accepted version when a rewrite of the
// same document leaves them empty. Stage 04 rewrites the HLD that stage 03
// already passed: the model sees the accepted version and still returns an entry
// without its root or moduleRef, and the gate then asks the operator about a
// field the document already answered.
func inheritAccepted(accepted, next json.RawMessage) json.RawMessage {
	if len(accepted) == 0 || len(next) == 0 {
		return next
	}
	var oldValue, newValue map[string]any
	if json.Unmarshal(accepted, &oldValue) != nil || json.Unmarshal(next, &newValue) != nil {
		return next
	}
	if !fillEmpty(oldValue, newValue) {
		return next
	}
	merged, err := json.Marshal(newValue)
	if err != nil {
		return next
	}
	return merged
}

// fillEmpty copies the old value into every key the new map leaves empty and
// reports whether it changed anything. openGaps is never inherited: the old
// questions are answered, and copying them back would ask them again.
func fillEmpty(oldValue, newValue map[string]any) bool {
	changed := false
	for key, oldField := range oldValue {
		if key == "openGaps" {
			continue
		}
		newField, ok := newValue[key]
		if !ok || isEmptyValue(newField) {
			if !isEmptyValue(oldField) {
				newValue[key] = oldField
				changed = true
			}
			continue
		}
		merged, ok := fillNested(oldField, newField)
		if ok {
			newValue[key] = merged
			changed = true
		}
	}
	return changed
}

// fillNested merges one old and one new value of the same key: objects field by
// field, lists entry by entry. A list pairs its entries by id, because a
// positional pairing would copy the root of one aggregate into another.
func fillNested(oldField, newField any) (any, bool) {
	switch newValue := newField.(type) {
	case map[string]any:
		oldValue, ok := oldField.(map[string]any)
		if !ok {
			return newField, false
		}
		return newValue, fillEmpty(oldValue, newValue)
	case []any:
		oldValue, ok := oldField.([]any)
		if !ok {
			return newField, false
		}
		changed := false
		for i := range newValue {
			oldEntry, ok := counterpart(oldValue, newValue[i], i)
			if !ok {
				continue
			}
			merged, ok := fillNested(oldEntry, newValue[i])
			if ok {
				newValue[i] = merged
				changed = true
			}
		}
		return newValue, changed
	}
	return newField, false
}

// counterpart finds the old entry a new entry replaces: the entry with the same
// id, or the entry at the same index when neither side carries an id.
func counterpart(oldValue []any, newEntry any, index int) (any, bool) {
	id := entryID(newEntry)
	if id == "" {
		if index < len(oldValue) && entryID(oldValue[index]) == "" {
			return oldValue[index], true
		}
		return nil, false
	}
	for _, oldEntry := range oldValue {
		if entryID(oldEntry) == id {
			return oldEntry, true
		}
	}
	return nil, false
}

func entryID(value any) string {
	entry, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	id, _ := entry["id"].(string)
	return id
}

func isEmptyValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return typed == ""
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	}
	return false
}

// inheritPremises keeps the premises the operator already answered when a re-run
// returns the shape with zeros. A zero premise means "not given": the earlier
// value wins, so an answer survives a regeneration of the document.
func inheritPremises(previous, next json.RawMessage) json.RawMessage {
	if len(previous) == 0 || len(next) == 0 {
		return next
	}
	var oldDoc, newDoc map[string]json.RawMessage
	if json.Unmarshal(previous, &oldDoc) != nil || json.Unmarshal(next, &newDoc) != nil {
		return next
	}
	var oldPremises, newPremises domain.Premises
	if json.Unmarshal(oldDoc["premises"], &oldPremises) != nil || json.Unmarshal(newDoc["premises"], &newPremises) != nil {
		return next
	}
	keep := func(newValue *int, oldValue int) {
		if *newValue == 0 && oldValue != 0 {
			*newValue = oldValue
		}
	}
	keep(&newPremises.Traffic.PeakQps, oldPremises.Traffic.PeakQps)
	if newPremises.Traffic.ReadWriteRatio == 0 {
		newPremises.Traffic.ReadWriteRatio = oldPremises.Traffic.ReadWriteRatio
	}
	keep(&newPremises.Traffic.PayloadBytes, oldPremises.Traffic.PayloadBytes)
	if newPremises.Storage.GBPerYear == 0 {
		newPremises.Storage.GBPerYear = oldPremises.Storage.GBPerYear
	}
	if newPremises.Storage.GrowthPerYear == 0 {
		newPremises.Storage.GrowthPerYear = oldPremises.Storage.GrowthPerYear
	}
	keep(&newPremises.Team.Size, oldPremises.Team.Size)
	if len(newPremises.Team.Languages) == 0 {
		newPremises.Team.Languages = oldPremises.Team.Languages
	}
	if newPremises.Team.OpsMaturity == "" {
		newPremises.Team.OpsMaturity = oldPremises.Team.OpsMaturity
	}
	if newPremises.BudgetMonthly == nil || newPremises.BudgetMonthly.Amount <= 0 {
		newPremises.BudgetMonthly = oldPremises.BudgetMonthly
	} else if newPremises.BudgetMonthly.Currency == "" && oldPremises.BudgetMonthly != nil {
		newPremises.BudgetMonthly.Currency = oldPremises.BudgetMonthly.Currency
	}
	keep(&newPremises.DeadlineDays, oldPremises.DeadlineDays)
	if newPremises.Availability == 0 {
		newPremises.Availability = oldPremises.Availability
	}
	keep(&newPremises.RPOMinutes, oldPremises.RPOMinutes)
	keep(&newPremises.RTOMinutes, oldPremises.RTOMinutes)
	if len(newPremises.Compliance) == 0 {
		newPremises.Compliance = oldPremises.Compliance
	}
	encoded, err := json.Marshal(newPremises)
	if err != nil {
		return next
	}
	newDoc["premises"] = encoded
	out, err := json.Marshal(newDoc)
	if err != nil {
		return next
	}
	return out
}
