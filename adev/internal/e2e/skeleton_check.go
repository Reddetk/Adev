//go:build e2e

package e2e

import (
	"encoding/json"

	"github.com/Reddetk/Adev/internal/domain"
)

type SkeletonCheck struct {
	OK      bool     `json:"ok"`
	Missing []string `json:"missing"`
	Extra   []string `json:"extra"`
}

func checkSkeletonBRD(payload json.RawMessage) SkeletonCheck {
	var v domain.BRD
	if err := json.Unmarshal(payload, &v); err != nil {
		return SkeletonCheck{OK: false, Missing: []string{"unmarshal"}}
	}
	missing := []string{}
	if len(v.Goals) == 0 {
		missing = append(missing, "goals")
	}
	if len(v.Metrics) == 0 {
		missing = append(missing, "metrics")
	}
	return SkeletonCheck{OK: len(missing) == 0, Missing: missing}
}

func checkSkeletonSRS(payload json.RawMessage) SkeletonCheck {
	var v domain.SRS
	if err := json.Unmarshal(payload, &v); err != nil {
		return SkeletonCheck{OK: false, Missing: []string{"unmarshal"}}
	}
	missing := []string{}
	if len(v.FR) == 0 {
		missing = append(missing, "fr")
	}
	return SkeletonCheck{OK: len(missing) == 0, Missing: missing}
}

func checkSkeletonHLD(payload json.RawMessage) SkeletonCheck {
	var v domain.HLD
	if err := json.Unmarshal(payload, &v); err != nil {
		return SkeletonCheck{OK: false, Missing: []string{"unmarshal"}}
	}
	missing := []string{}
	if len(v.BoundedContexts) == 0 {
		missing = append(missing, "boundedContexts")
	}
	if len(v.Containers) == 0 {
		missing = append(missing, "containers")
	}
	if len(v.Aggregates) == 0 {
		missing = append(missing, "aggregates")
	}
	return SkeletonCheck{OK: len(missing) == 0, Missing: missing}
}

func checkSkeletonLLD(payload json.RawMessage) SkeletonCheck {
	var v domain.LLD
	if err := json.Unmarshal(payload, &v); err != nil {
		return SkeletonCheck{OK: false, Missing: []string{"unmarshal"}}
	}
	missing := []string{}
	if len(v.Containers) == 0 {
		missing = append(missing, "containers")
	}
	if len(v.Contracts) == 0 {
		missing = append(missing, "contracts")
	}
	if len(v.Paths) == 0 {
		missing = append(missing, "paths")
	}
	return SkeletonCheck{OK: len(missing) == 0, Missing: missing}
}

func checkSkeletonTaskPlan(payload json.RawMessage) SkeletonCheck {
	var v domain.TaskPlan
	if err := json.Unmarshal(payload, &v); err != nil {
		return SkeletonCheck{OK: false, Missing: []string{"unmarshal"}}
	}
	missing := []string{}
	if len(v.Tasks) == 0 {
		missing = append(missing, "tasks")
	}
	if len(v.Roots) == 0 {
		missing = append(missing, "roots")
	}
	return SkeletonCheck{OK: len(missing) == 0, Missing: missing}
}
