package domain

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// shapeExamples holds one allowed value of a named enum type. The skeleton shows
// the model a real value instead of a placeholder, so the enums stay in sync with
// the constants next to their types.
var shapeExamples = map[string]string{
	"Frequency":  `"day"`,
	"Confidence": `"low"`,
	"Priority":   `"must"`,
	"Modality":   `"shall"`,
	"Source":     `"operator"`,
	"Locale":     `"en"`,
}

// shapeChoices lists every allowed value of a named enum type whose constants
// live in the same package. The model sees the full set, so it stops guessing.
var shapeChoices = map[string]string{
	"Verdict":             `"go|no_go|revise"`,
	"DecisionStatus":      `"proposed|accepted|superseded"`,
	"ContextRelationType": `"sync|async"`,
	"LayerName":           `"adapter|domain|app|infra"`,
	"PathKind":            `"happy|negative"`,
	"PlanTaskType":        `"feature|fix|refactor|config|docs|test"`,
	"PlanTaskOwner":       `"dev|devops"`,
}

// PayloadShape returns the JSON skeleton of the payload of a form type.
// The skeleton is generated from the domain type, so a new field reaches the
// model without a hand-written copy in a service.
func PayloadShape(formType FormType) (string, error) {
	spec, ok := formSpecFor(formType)
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrNoPayloadGates, formType)
	}
	raw, err := json.Marshal(shapeOf(reflect.ValueOf(spec.payload())))
	if err != nil {
		return "", fmt.Errorf("payload shape: %w", err)
	}
	return string(raw), nil
}

// FormTypesWithGates returns the form types that have payload gates, in registry order.
func FormTypesWithGates() []FormType {
	types := make([]FormType, 0, len(formSpecs))
	for _, entry := range formSpecs {
		types = append(types, entry.Type)
	}
	return types
}

func shapeOf(value reflect.Value) any {
	if value.Kind() == reflect.Pointer {
		return shapeOf(reflect.New(value.Type().Elem()).Elem())
	}

	switch value.Type() {
	case reflect.TypeOf(Text{}):
		return map[string]any{"text": "...", "lang": "STE100"}
	case reflect.TypeOf(Statement{}):
		return map[string]any{"id": "...", "text": "...", "modality": "shall", "priority": "must"}
	case reflect.TypeOf(Ref{}):
		return map[string]any{"formId": "...", "formType": "...", "formVersion": 1}
	}
	if example, ok := shapeExamples[value.Type().Name()]; ok {
		return json.RawMessage(example)
	}
	if choices, ok := shapeChoices[value.Type().Name()]; ok {
		return json.RawMessage(choices)
	}

	switch value.Kind() {
	case reflect.Struct:
		out := map[string]any{}
		for i := 0; i < value.NumField(); i++ {
			name := strings.Split(value.Type().Field(i).Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			out[name] = shapeOf(value.Field(i))
		}
		return out
	case reflect.Slice:
		return []any{shapeOf(reflect.Zero(value.Type().Elem()))}
	case reflect.Int, reflect.Int64, reflect.Float64:
		return 0
	case reflect.Bool:
		return false
	default:
		return "..."
	}
}
