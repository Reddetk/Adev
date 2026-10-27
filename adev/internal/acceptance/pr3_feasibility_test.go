//go:build tdd

package acceptance

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
)

// PR3 acceptance: the shape carries the report type. The greenfield route was
// changed to TEJ-only (2026-10 decision), so the RAR choice left the shape.
func TestPR3FeasibilityCarriesTheReportType(t *testing.T) {
	shape, err := domain.PayloadShape(domain.TypeFeasibility)
	require.NoError(t, err)

	assert.Contains(t, shape, "reportType",
		"the model must see the report type in the payload shape")
	assert.Contains(t, shape, `"tej`)
	assert.NotContains(t, shape, `"rar`,
		"the greenfield route writes a TEJ, the RAR branch waits for the change route")
}

// PR3 acceptance: a TEJ report without a cost estimate does not pass the gate.
func TestPR3ATEJReportNeedsCostScheduleAndConfidence(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"reportType": "tej",
		"options": []any{map[string]any{
			"id": "OPT-1", "summary": map[string]any{"text": "Build it.", "lang": "STE100"},
			"scheduleDays": 30, "confidence": "high",
		}},
		"recommendedOption": "OPT-1",
		"verdict":           "go",
		"reason":            map[string]any{"text": "The team can build it.", "lang": "STE100"},
	})
	require.NoError(t, err)

	assert.NotEmpty(t, gateFailures(t, domain.TypeFeasibility, payload),
		"a technical and economic justification without the money is not a TEJ")
}

// PR3 acceptance: a greenfield stage writes a TEJ: the system that does not
// exist yet has no risk register to assess.
func TestPR3AReportThatIsNotATechnoEconomicJustificationFailsTheGates(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"reportType": "rar",
		"options": []any{map[string]any{
			"id": "OPT-1", "summary": map[string]any{"text": "Build it.", "lang": "STE100"},
			"cost":         map[string]any{"currency": "EUR", "amount": 1000},
			"scheduleDays": 30, "confidence": "high",
		}},
		"recommendedOption": "OPT-1",
		"verdict":           "go",
		"reason":            map[string]any{"text": "The team can build it.", "lang": "STE100"},
	})
	require.NoError(t, err)

	assert.Contains(t, gateFailures(t, domain.TypeFeasibility, payload), "writes a TEJ",
		"a greenfield run prices the options, it does not assess risks of a system that does not exist")
}

// gateFailures runs the payload gates of one form type on a raw payload.
func gateFailures(t *testing.T, formType domain.FormType, payload json.RawMessage) string {
	t.Helper()
	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-1-01", FormType: formType, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusValid, Route: domain.RouteGreenfield, Stage: "01", Mode: domain.ModeFull,
		PipelineID: testPipeline, TargetSystemID: "1", ProducedBy: domain.RoleAnalyst,
	}}
	require.NoError(t, form.SetRawPayload(payload))
	err := form.ValidatePayload()
	if err == nil {
		return ""
	}
	return err.Error()
}
