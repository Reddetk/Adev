package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Projection maps a role to the JSON Pointers of a form that the role may see.
// A pointer segment "*" matches every member of an object or every element of an array.
type Projection map[Role][]string

// Project returns the part of a form that the role may see.
func (p Projection) Project(role Role, form any) (any, error) {
	return ProjectFields(p[role], form)
}

// ProjectionFor returns the field allowlist of a form type for a role.
// A type without a projection cannot be read by an agent: the caller gets an error
// instead of the whole form.
func ProjectionFor(formType FormType, role Role) ([]string, error) {
	var table Projection
	switch formType {
	case TypeBRD:
		table = BRDProjection
	case TypeUserInput:
		table = userInputProjection
	case TypeFeasibility, TypeSRS, TypeHLD, TypeADR, TypeLLD, TypeTaskPlan:
		// shortcut: the roles read each other's design forms in full. Field-level
		// secrecy between roles is not implemented yet.
		table = openDesignProjection
	default:
		return nil, fmt.Errorf("no projection for form type %q yet", formType)
	}
	fields, ok := table[role]
	if !ok {
		return nil, fmt.Errorf("no projection of %s for role %q", formType, role)
	}
	return fields, nil
}

// ProjectFields keeps only the JSON Pointers in the allowlist.
func ProjectFields(fields []string, form any) (any, error) {
	raw, err := json.Marshal(form)
	if err != nil {
		return nil, fmt.Errorf("project: marshal form: %w", err)
	}
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("project: decode form: %w", err)
	}

	var out any
	for _, pointer := range fields {
		parts, err := parsePointer(pointer)
		if err != nil {
			return nil, err
		}
		value, ok := matchPointer(root, parts)
		if !ok {
			continue
		}
		out = mergeValue(out, value)
	}
	return out, nil
}

func parsePointer(pointer string) ([]string, error) {
	if pointer == "" || pointer == "/" {
		return nil, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("project: pointer %q must start with /", pointer)
	}
	parts := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	for _, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("project: pointer %q has an empty segment", pointer)
		}
	}
	return parts, nil
}

func matchPointer(node any, parts []string) (any, bool) {
	if len(parts) == 0 {
		return node, true
	}
	switch typed := node.(type) {
	case map[string]any:
		if parts[0] == "*" {
			result := map[string]any{}
			found := false
			for key, value := range typed {
				if sub, ok := matchPointer(value, parts[1:]); ok {
					result[key] = sub
					found = true
				}
			}
			return result, found
		}
		value, ok := typed[parts[0]]
		if !ok {
			return nil, false
		}
		sub, ok := matchPointer(value, parts[1:])
		if !ok {
			return nil, false
		}
		return map[string]any{parts[0]: sub}, true
	case []any:
		if parts[0] != "*" {
			return nil, false
		}
		result := make([]any, len(typed))
		found := false
		for i, value := range typed {
			if sub, ok := matchPointer(value, parts[1:]); ok {
				result[i] = sub
				found = true
			}
		}
		return result, found
	default:
		return nil, false
	}
}

// mergeValue joins two projections of the same document. Objects merge by key.
// Arrays merge by index, because both sides come from the same source array.
func mergeValue(left, right any) any {
	if left == nil {
		return right
	}
	leftMap, leftIsMap := left.(map[string]any)
	rightMap, rightIsMap := right.(map[string]any)
	if leftIsMap && rightIsMap {
		for key, value := range rightMap {
			if existing, ok := leftMap[key]; ok {
				leftMap[key] = mergeValue(existing, value)
				continue
			}
			leftMap[key] = value
		}
		return leftMap
	}
	leftSlice, leftIsSlice := left.([]any)
	rightSlice, rightIsSlice := right.([]any)
	if leftIsSlice && rightIsSlice {
		size := len(leftSlice)
		if len(rightSlice) > size {
			size = len(rightSlice)
		}
		result := make([]any, size)
		for i := range result {
			var a, b any
			if i < len(leftSlice) {
				a = leftSlice[i]
			}
			if i < len(rightSlice) {
				b = rightSlice[i]
			}
			result[i] = mergeValue(a, b)
		}
		return result
	}
	return right
}

// userInputProjection is the role projection for the USER_INPUT form.
// The analyst sees the request itself; the operator sees everything.
var userInputProjection = Projection{
	RoleOperator: {
		"/*",
	},
	RoleAnalyst: {
		"/formId", "/formType", "/formVersion", "/status", "/payload",
	},
}

// BRDProjection is the role projection for the BRD form.
// A developer sees the envelope only. An analyst sees the full payload.
var BRDProjection = Projection{
	RoleOperator: {
		"/*",
	},
	RoleAnalyst: {
		"/formId", "/formType", "/formVersion", "/status", "/stage", "/payloadHash", "/payload",
	},
	RoleArchitect: {
		"/formId", "/formType", "/formVersion", "/status", "/stage", "/payloadHash",
		"/payload/goals", "/payload/metrics", "/payload/scope", "/payload/actors",
	},
	RoleQA: {
		"/formId", "/formType", "/formVersion", "/status", "/payload/metrics",
	},
	RoleDev: {
		"/formId", "/formType", "/formVersion", "/status",
	},
	RoleDevOps: {
		"/formId", "/formType",
	},
}

// openDesignProjection lets every role read the full design form.
// shortcut: no field secrecy between roles yet, owner-role allowlists come later.
var openDesignProjection = Projection{
	RoleOperator:  {"/*"},
	RoleAnalyst:   {"/*"},
	RoleArchitect: {"/*"},
	RoleQA:        {"/*"},
	RoleDev:       {"/*"},
	RoleDevOps:    {"/*"},
}
