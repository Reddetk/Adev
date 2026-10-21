//go:build tdd

package acceptance

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
)

// PR7 phase A (2.8): a contract hangs off a module, so container -> layer ->
// module -> contract is one chain. The key names are the contract of the phase.
func TestPR7PhaseAContractsBelongToAModule(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"containers": []any{map[string]any{
			"containerId": "obj-container-1",
			"layers": []any{map[string]any{
				"name": "app",
				"packages": []any{map[string]any{
					"path":    "internal/app",
					"modules": []any{map[string]any{"objectId": "obj-module-1", "path": "internal/app/book"}},
				}},
			}},
		}},
		"contracts": []any{map[string]any{"id": "contract-1", "kind": "http"}},
		"paths": []any{
			map[string]any{"id": "p1", "contractRef": "contract-1", "kind": "happy"},
			map[string]any{"id": "p2", "contractRef": "contract-1", "kind": "negative", "expectedErrorCode": "E-1"},
		},
	})
	require.NoError(t, err)

	assert.NotEmpty(t, gateFailures(t, domain.TypeLLD, payload),
		"a contract without a module has no place in the container structure")
}

// PR7 phase B (2.5): the DDD cycle starts with the nouns of the domain, so a
// bounded context without a ubiquitous definition is not analyzed.
func TestPR7PhaseBTheDDDCycleStartsWithTheNouns(t *testing.T) {
	hld := domain.HLD{
		BoundedContexts: []domain.BoundedContext{{ID: "bc-1", Name: "Lending", Purpose: ste("The desk lends books.")}},
		Containers: []domain.Container{{
			ObjectID: "obj-container-1", BoundedContextID: "bc-1",
			Purpose: ste("Serve the lending desk."), Tech: "go",
		}},
	}

	err := domain.ValidateHLD(hld)

	require.Error(t, err, "a bounded context without ubiquitous language skips the DDD cycle")
}

// PR7 phase C (2.2): ATAM records the quality scenarios and their influence.
func TestPR7PhaseCTheHLDCarriesTheATAMScenarios(t *testing.T) {
	shape, err := domain.PayloadShape(domain.TypeHLD)
	require.NoError(t, err)

	assert.Contains(t, shape, "qualityScenarios",
		"2.2: the HLD stores the quality scenarios with their influence on the project")
}

// PR7 phase D (2.7): the tool matrix of the NFR and FR pairs is a document field,
// not a paragraph inside a design.
func TestPR7PhaseDTheLLDCarriesTheToolMatrix(t *testing.T) {
	shape, err := domain.PayloadShape(domain.TypeLLD)
	require.NoError(t, err)

	assert.Contains(t, shape, "toolMatrix",
		"2.7: the LLD stores which tool serves which NFR/FR pair")
}
