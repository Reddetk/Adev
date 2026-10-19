package domain

import (
	"fmt"
	"strings"
)

// Readiness says whether a target system can start a pipeline.
type Readiness struct {
	Ready   bool     `json:"ready"`
	Missing []string `json:"missing,omitempty"`
}

// Check reports the settings that a route still needs.
// A route needs a workspace directory and a valid agent config for every owner role.
func Check(workspace string, agents []AgentConfig, stages []Stage) Readiness {
	var missing []string

	if strings.TrimSpace(workspace) == "" {
		missing = append(missing, "workspace directory is not set")
	}

	byRole := map[Role]AgentConfig{}
	for _, config := range agents {
		byRole[config.Role] = config
	}

	for _, role := range Roles(stages) {
		config, ok := byRole[role]
		if !ok {
			missing = append(missing, fmt.Sprintf("agent config for role %q is missing", role))
			continue
		}
		if err := config.Validate(); err != nil {
			missing = append(missing, fmt.Sprintf("agent config for role %q is invalid: %v", role, err))
		}
	}

	return Readiness{Ready: len(missing) == 0, Missing: missing}
}
