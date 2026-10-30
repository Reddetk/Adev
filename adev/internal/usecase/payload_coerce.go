package usecase

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Reddetk/Adev/internal/domain"
)

// coercePayload bends the model answer into the shape of the payload before the
// gates judge it. The model keeps answering the meaning and losing the wrapper: a
// question arrives as a plain string instead of the Text record, a number arrives
// inside a sentence. The same repair the operator answers get, so a call that
// carried the right meaning is not thrown away over the wrapper alone.
// The gates must keep judging: a wrong value stays a rejection.
func coercePayload(formType domain.FormType, payload json.RawMessage) json.RawMessage {
	shape, err := domain.PayloadShape(formType)
	if err != nil {
		return payload
	}
	var skeleton any
	if json.Unmarshal([]byte(shape), &skeleton) != nil {
		return payload
	}
	var value any
	if json.Unmarshal(payload, &value) != nil {
		return payload
	}
	out, err := json.Marshal(coerceNode(skeleton, value))
	if err != nil {
		return payload
	}
	out = normalizePriorities(formType, out)
	if formType == domain.TypeBRD {
		out = normalizeBRDFrequency(out)
	}
	return out
}

// frequencyWord maps the words an actor answer uses onto the registry enum.
var frequencyWord = map[string]string{
	"day": "day", "daily": "day", "days": "day",
	"week": "week", "weekly": "week", "weeks": "week",
	"month": "month", "monthly": "month", "months": "month",
	"quarter": "quarter", "quarterly": "quarter", "quarters": "quarter",
	"event": "event", "events": "event", "per event": "event",
}

// normalizeBRDFrequency rewrites the frequency of every actor into the enum the// registry keeps. The operator answers in plain words ("hours", "weekly"), and a
// value the map does not know stays as it is.
func normalizeBRDFrequency(payload json.RawMessage) json.RawMessage {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(payload, &doc); err != nil {
		return payload
	}
	raw, ok := doc["actors"]
	if !ok {
		return payload
	}
	var actors []map[string]any
	if err := json.Unmarshal(raw, &actors); err != nil {
		return payload
	}
	changed := false
	for _, actor := range actors {
		value, ok := actor["frequency"].(string)
		if !ok {
			continue
		}
		mapped, ok := frequencyWord[strings.ToLower(strings.TrimSpace(value))]
		if !ok || mapped == value {
			continue
		}
		actor["frequency"] = mapped
		changed = true
	}
	if !changed {
		return payload
	}
	encoded, err := json.Marshal(actors)
	if err != nil {
		return payload
	}
	doc["actors"] = encoded
	out, err := json.Marshal(doc)
	if err != nil {
		return payload
	}
	return out
}

// moSCoWPriority maps the priority words the model answers with onto the enum the
// registry keeps. "should" and "could" are MoSCoW levels next to must/high/low:
// without the map a requirement is rejected over a synonym, and the operator gets
// a question about a value the model already decided.
var moSCoWPriority = map[string]string{
	"must": "must", "must-have": "must",
	"should": "high", "should-have": "high",
	"could": "low", "could-have": "low",
}

// normalizePriorities rewrites the priority of every functional requirement into
// the registry enum. The SRS, FR and NFR documents carry the same fr list.
func normalizePriorities(formType domain.FormType, payload json.RawMessage) json.RawMessage {
	if formType != domain.TypeSRS && formType != domain.TypeFR && formType != domain.TypeNFR {
		return payload
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(payload, &doc); err != nil {
		return payload
	}
	raw, ok := doc["fr"]
	if !ok {
		return payload
	}
	var entries []map[string]any
	if err := json.Unmarshal(raw, &entries); err != nil {
		return payload
	}
	changed := false
	for _, entry := range entries {
		if normalizePriorityField(entry) {
			changed = true
		}
		if statement, ok := entry["statement"].(map[string]any); ok && normalizePriorityField(statement) {
			changed = true
		}
	}
	if !changed {
		return payload
	}
	encoded, err := json.Marshal(entries)
	if err != nil {
		return payload
	}
	doc["fr"] = encoded
	out, err := json.Marshal(doc)
	if err != nil {
		return payload
	}
	return out
}

// normalizePriorityField rewrites one priority value in place and reports a change.
func normalizePriorityField(fields map[string]any) bool {
	value, ok := fields["priority"].(string)
	if !ok {
		return false
	}
	mapped, ok := moSCoWPriority[strings.ToLower(strings.TrimSpace(value))]
	if !ok || mapped == value {
		return false
	}
	fields["priority"] = mapped
	return true
}

// normalizeLLDLayout drops the fields the LLD layout does not know: a detail
// answer sometimes nests a container in a layer or a layer in a package. The
// gates then see the real layout instead of an object the schema ignores.
func normalizeLLDLayout(formType domain.FormType, payload json.RawMessage) json.RawMessage {
	if formType != domain.TypeLLD {
		return payload
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(payload, &doc); err != nil {
		return payload
	}
	raw, ok := doc["containers"]
	if !ok {
		return payload
	}
	var containers []domain.LLDContainer
	if err := json.Unmarshal(raw, &containers); err != nil {
		return payload
	}
	encoded, err := json.Marshal(containers)
	if err != nil {
		return payload
	}
	doc["containers"] = encoded
	out, err := json.Marshal(doc)
	if err != nil {
		return payload
	}
	return out
}

// coerceNode walks one level of the payload next to its shape. A node the shape
// names no wrapper for passes through untouched.
func coerceNode(shape any, value any) any {
	if shape == nil {
		return value
	}
	switch node := shape.(type) {
	case []any:
		items, ok := value.([]any)
		if !ok || len(node) == 0 {
			return value
		}
		out := make([]any, len(items))
		for i := range items {
			out[i] = coerceNode(node[0], items[i])
		}
		return out
	case map[string]any:
		// A Text record: the words alone become the record and get the language.
		_, hasText := node["text"]
		_, hasLang := node["lang"]
		if hasText && hasLang {
			switch typed := value.(type) {
			case string:
				return map[string]any{"lang": domain.LangSTE100, "text": typed}
			case map[string]any:
				// The words are right, only the language is missing: fill it in.
				if _, has := typed["lang"]; !has {
					if _, has := typed["text"]; has {
						typed["lang"] = domain.LangSTE100
					}
				}
			}
			return value
		}
		// A record whose free text carries the sentence (the condition of an SRS
		// error entry and its kind): the other fields stay empty for the model.
		if words, ok := value.(string); ok {
			for _, key := range []string{"condition", "message", "description"} {
				if _, has := node[key]; has {
					return map[string]any{key: words}
				}
			}
		}
		record, ok := value.(map[string]any)
		if !ok {
			// A list entry the model wrote as a scalar is not a document the operator
			// can answer into. Replace it with an empty record of the shape, so the
			// later answer creates its fields instead of failing with "has no place".
			return map[string]any{}
		}
		for key, field := range record {
			if fieldShapeOf, has := node[key]; has {
				record[key] = coerceNode(fieldShapeOf, field)
			}
		}
		return record
	default:
		return coerceScalar(node, value)
	}
}

// coerceScalar lines a scalar up with its shape: a number field takes the number
// inside the sentence, a text field keeps text even for a bare number.
func coerceScalar(shape any, value any) any {
	if _, isNumberShape := shape.(float64); !isNumberShape {
		if _, isTextShape := shape.(string); isTextShape {
			if scalar, ok := value.(float64); ok {
				return fmt.Sprintf("%v", scalar)
			}
		}
		return value
	}
	if _, ok := value.(float64); ok {
		return value
	}
	if words, ok := value.(string); ok {
		match := numberInSentence.FindString(strings.ReplaceAll(words, ",", "."))
		if parsed, err := strconv.ParseFloat(match, 64); match != "" && err == nil {
			return parsed
		}
	}
	return value
}

// dropEmptyScopeDelta removes the placeholder scopeDelta records a model writes
// on every FEASIBILITY answer: an entry that adds and removes nothing is not a
// scope change.
func dropEmptyScopeDelta(payload json.RawMessage) json.RawMessage {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(payload, &doc); err != nil {
		return payload
	}
	var deltas []map[string]any
	if err := json.Unmarshal(doc["scopeDelta"], &deltas); err != nil {
		return payload
	}
	kept := make([]map[string]any, 0, len(deltas))
	for _, delta := range deltas {
		if isEmptyValue(delta["added"]) && isEmptyValue(delta["removed"]) {
			continue
		}
		kept = append(kept, delta)
	}
	if len(kept) == len(deltas) {
		return payload
	}
	encoded, err := json.Marshal(kept)
	if err != nil {
		return payload
	}
	doc["scopeDelta"] = encoded
	out, err := json.Marshal(doc)
	if err != nil {
		return payload
	}
	return out
}

// dropBRDMetricGaps removes the questions about metrics and success values: a
// greenfield system has no measured value. An old draft still carries them, and
// the model copies its openGaps into every new version, so the stage would keep
// asking a question the operator cannot answer.
func dropBRDMetricGaps(payload json.RawMessage) json.RawMessage {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(payload, &doc); err != nil {
		return payload
	}
	raw, ok := doc["openGaps"]
	if !ok {
		return payload
	}
	var gaps []map[string]any
	if err := json.Unmarshal(raw, &gaps); err != nil {
		return payload
	}
	kept := make([]map[string]any, 0, len(gaps))
	for _, gap := range gaps {
		pointer, _ := gap["pointer"].(string)
		pointer = normalizePointer(pointer)
		if strings.HasPrefix(pointer, "/metrics/") || pointer == "/metrics" ||
			strings.HasPrefix(pointer, "/successMetrics/") || pointer == "/successMetrics" {
			continue
		}
		kept = append(kept, gap)
	}
	if len(kept) == len(gaps) {
		return payload
	}
	encoded, err := json.Marshal(kept)
	if err != nil {
		return payload
	}
	doc["openGaps"] = encoded
	out, err := json.Marshal(doc)
	if err != nil {
		return payload
	}
	return out
}

// dropFilledGaps removes the questions the document already answers. The model
// copies the openGaps of the previous draft into its new answer, and a question
// about a field it just filled holds the stage for nothing.
func dropFilledGaps(payload json.RawMessage) json.RawMessage {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(payload, &doc); err != nil {
		return payload
	}
	raw, ok := doc["openGaps"]
	if !ok {
		return payload
	}
	var gaps []map[string]any
	if err := json.Unmarshal(raw, &gaps); err != nil {
		return payload
	}
	var root any
	if err := json.Unmarshal(payload, &root); err != nil {
		return payload
	}
	kept := make([]map[string]any, 0, len(gaps))
	for _, gap := range gaps {
		pointer, _ := gap["pointer"].(string)
		if gapFilled(valueAt(root, normalizePointer(pointer))) {
			continue
		}
		kept = append(kept, gap)
	}
	if len(kept) == len(gaps) {
		return payload
	}
	encoded, err := json.Marshal(kept)
	if err != nil {
		return payload
	}
	doc["openGaps"] = encoded
	out, err := json.Marshal(doc)
	if err != nil {
		return payload
	}
	return out
}

// gapFilled reports whether the field a gap asks about already holds a value.
func gapFilled(value any) bool {
	switch node := value.(type) {
	case nil:
		return false
	case string:
		text := strings.TrimSpace(node)
		return text != "" && text != "..."
	case []any:
		return len(node) > 0
	case map[string]any:
		return len(node) > 0
	}
	return true
}
