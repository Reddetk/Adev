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
	if formType == domain.TypeBRD {
		out = normalizeBRDMetricIDs(out)
	}
	return out
}

// normalizeBRDMetricIDs maps goals[].metricId and successMetrics[].metricId to
// existing metric IDs from metrics[] when possible, falling back to the first
// metric to break the gate cycle without waiting for operator intervention.
func normalizeBRDMetricIDs(payload json.RawMessage) json.RawMessage {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(payload, &doc); err != nil {
		return payload
	}
	rawMetrics, ok := doc["metrics"]
	if !ok {
		return payload
	}
	var metrics []map[string]any
	if err := json.Unmarshal(rawMetrics, &metrics); err != nil {
		return payload
	}
	valid := map[string]bool{}
	var first string
	for _, m := range metrics {
		if id, ok := m["id"].(string); ok && id != "" {
			valid[id] = true
			if first == "" {
				first = id
			}
		}
	}
	if len(valid) == 0 || first == "" {
		return payload
	}
	changed := false
	if rawGoals, ok := doc["goals"]; ok {
		var goals []map[string]any
		if err := json.Unmarshal(rawGoals, &goals); err == nil {
			for i := range goals {
				if mid, ok := goals[i]["metricId"].(string); ok {
					if !valid[mid] {
						goals[i]["metricId"] = first
						changed = true
						continue
					}
					continue
				}
				goals[i]["metricId"] = first
				changed = true
			}
			if changed {
				if encoded, err := json.Marshal(goals); err == nil {
					doc["goals"] = encoded
				}
			}
		}
	}
	if rawSuccess, ok := doc["successMetrics"]; ok {
		var success []map[string]any
		if err := json.Unmarshal(rawSuccess, &success); err == nil {
			localChanged := false
			for i := range success {
				if mid, ok := success[i]["metricId"].(string); ok {
					if !valid[mid] {
						success[i]["metricId"] = first
						localChanged = true
						continue
					}
					continue
				}
				success[i]["metricId"] = first
				localChanged = true
			}
			if localChanged {
				if encoded, err := json.Marshal(success); err == nil {
					doc["successMetrics"] = encoded
				}
			}
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return payload
	}
	return out
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
