//go:build tdd

package acceptance

// pr9_details tests the level-by-level filling of the LLD: one model call per
// document nesting level. The skeleton names the containers; the later calls
// detail one entry of one level each — containers, layers, packages, modules,
// and the contracts of the module.

import (
	"encoding/json"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLDIsFilledOneLevelPerCall(t *testing.T) {
	fixture := newFixture(t, respondWith(t, map[domain.FormType]json.RawMessage{
		domain.TypeLLD: validLLD(t),
	}))

	fixture.submit(t, "FRM-1-03", domain.TypeHLD, "03", validHLD(t))
	fixture.submit(t, "FRM-1-04-ADR", domain.TypeADR, "04", validADR(t))
	fixture.run(t, "05")

	// skeleton + 1 container + 1 layer + 1 package + 1 module + 1 contract.
	require.Len(t, fixture.llm.requests, 6)
	assert.Contains(t, fixture.llm.requests[0].User, "List every entry")
	assert.Contains(t, fixture.llm.requests[1].User, `the containers entry with containerId "obj-container-1"`)
	assert.Contains(t, fixture.llm.requests[2].User, `the layers entry with name "app"`)
	assert.Contains(t, fixture.llm.requests[3].User, `the packages entry with path "internal/app"`)
	assert.Contains(t, fixture.llm.requests[4].User, `the modules entry with objectId "obj-module-1"`)
	assert.Contains(t, fixture.llm.requests[5].User, `the contracts entry with id "contract-1"`)
	assert.Contains(t, fixture.llm.requests[5].User, `whose contractRef is "contract-1"`)
}
