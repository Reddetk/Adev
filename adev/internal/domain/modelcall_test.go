package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestModelCall_Validate tests the Validate method with various scenarios
func TestModelCall_Validate(t *testing.T) {
	tests := []struct {
		name      string
		modelCall ModelCall
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid entity",
			modelCall: ModelCall{
				RunID:     1,
				Profile:   "valid value",
				Model:     "valid value",
				TokensIn:  1,
				TokensOut: 1,
				Cost:      10.5,
				LatencyMs: 1,
			},
			wantErr: false,
		},
		{
			name: "invalid modelCall - negative number",
			modelCall: ModelCall{
				RunID:     -1,
				Profile:   "valid value",
				Model:     "valid value",
				TokensIn:  1,
				TokensOut: 1,
				Cost:      10.5,
				LatencyMs: 1,
			},
			wantErr: true,
			errMsg:  "runid",
		},
		{
			name: "invalid modelCall - empty string",
			modelCall: ModelCall{
				RunID:     1,
				Profile:   "",
				Model:     "valid value",
				TokensIn:  1,
				TokensOut: 1,
				Cost:      10.5,
				LatencyMs: 1,
			},
			wantErr: true,
			errMsg:  "profile",
		},
		{
			name: "invalid modelCall - empty string",
			modelCall: ModelCall{
				RunID:     1,
				Profile:   "valid value",
				Model:     "",
				TokensIn:  1,
				TokensOut: 1,
				Cost:      10.5,
				LatencyMs: 1,
			},
			wantErr: true,
			errMsg:  "model",
		},
		{
			name: "invalid modelCall - negative number",
			modelCall: ModelCall{
				RunID:     1,
				Profile:   "valid value",
				Model:     "valid value",
				TokensIn:  -1,
				TokensOut: 1,
				Cost:      10.5,
				LatencyMs: 1,
			},
			wantErr: true,
			errMsg:  "tokensin",
		},
		{
			name: "invalid modelCall - negative number",
			modelCall: ModelCall{
				RunID:     1,
				Profile:   "valid value",
				Model:     "valid value",
				TokensIn:  1,
				TokensOut: -1,
				Cost:      10.5,
				LatencyMs: 1,
			},
			wantErr: true,
			errMsg:  "tokensout",
		},
		{
			name: "invalid modelCall - negative number",
			modelCall: ModelCall{
				RunID:     1,
				Profile:   "valid value",
				Model:     "valid value",
				TokensIn:  1,
				TokensOut: 1,
				Cost:      -1.0,
				LatencyMs: 1,
			},
			wantErr: true,
			errMsg:  "cost",
		},
		{
			name: "invalid modelCall - negative number",
			modelCall: ModelCall{
				RunID:     1,
				Profile:   "valid value",
				Model:     "valid value",
				TokensIn:  1,
				TokensOut: 1,
				Cost:      10.5,
				LatencyMs: -1,
			},
			wantErr: true,
			errMsg:  "latencyms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.modelCall.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestModelCall_Initialization tests entity field initialization
func TestModelCall_Initialization(t *testing.T) {
	modelCall := &ModelCall{
		RunID:     1,
		Profile:   "valid value",
		Model:     "valid value",
		TokensIn:  1,
		TokensOut: 1,
		Cost:      10.5,
		LatencyMs: 1,
	}

	assert.Equal(t, 1, modelCall.RunID, "RunID should be set correctly")
	assert.Equal(t, "valid value", modelCall.Profile, "Profile should be set correctly")
	assert.Equal(t, "valid value", modelCall.Model, "Model should be set correctly")
	assert.Equal(t, 1, modelCall.TokensIn, "TokensIn should be set correctly")
	assert.Equal(t, 1, modelCall.TokensOut, "TokensOut should be set correctly")
	assert.Equal(t, 10.5, modelCall.Cost, "Cost should be set correctly")
	assert.Equal(t, 1, modelCall.LatencyMs, "LatencyMs should be set correctly")
}

// TestModelCall_RunID_NumericValidation tests numeric validation for RunID
func TestModelCall_RunID_NumericValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{name: "positive value", value: 10, wantErr: false},
		{name: "zero value", value: 0, wantErr: false},
		{name: "negative value", value: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelCall := &ModelCall{
				RunID:     tt.value,
				Profile:   "valid value",
				Model:     "valid value",
				TokensIn:  1,
				TokensOut: 1,
				Cost:      10.5,
				LatencyMs: 1,
			}

			err := modelCall.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestModelCall_Profile_EdgeCases tests edge cases for Profile field
func TestModelCall_Profile_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty string", value: "", wantErr: true},
		{name: "valid value", value: "Valid Name", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelCall := &ModelCall{
				RunID:     1,
				Profile:   tt.value,
				Model:     "valid value",
				TokensIn:  1,
				TokensOut: 1,
				Cost:      10.5,
				LatencyMs: 1,
			}

			err := modelCall.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestModelCall_Model_EdgeCases tests edge cases for Model field
func TestModelCall_Model_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty string", value: "", wantErr: true},
		{name: "valid value", value: "Valid Name", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelCall := &ModelCall{
				RunID:     1,
				Profile:   "valid value",
				Model:     tt.value,
				TokensIn:  1,
				TokensOut: 1,
				Cost:      10.5,
				LatencyMs: 1,
			}

			err := modelCall.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestModelCall_TokensIn_NumericValidation tests numeric validation for TokensIn
func TestModelCall_TokensIn_NumericValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{name: "positive value", value: 10, wantErr: false},
		{name: "zero value", value: 0, wantErr: false},
		{name: "negative value", value: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelCall := &ModelCall{
				RunID:     1,
				Profile:   "valid value",
				Model:     "valid value",
				TokensIn:  tt.value,
				TokensOut: 1,
				Cost:      10.5,
				LatencyMs: 1,
			}

			err := modelCall.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestModelCall_TokensOut_NumericValidation tests numeric validation for TokensOut
func TestModelCall_TokensOut_NumericValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{name: "positive value", value: 10, wantErr: false},
		{name: "zero value", value: 0, wantErr: false},
		{name: "negative value", value: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelCall := &ModelCall{
				RunID:     1,
				Profile:   "valid value",
				Model:     "valid value",
				TokensIn:  1,
				TokensOut: tt.value,
				Cost:      10.5,
				LatencyMs: 1,
			}

			err := modelCall.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestModelCall_Cost_NumericValidation tests numeric validation for Cost
func TestModelCall_Cost_NumericValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		wantErr bool
	}{
		{name: "positive value", value: 10.5, wantErr: false},
		{name: "zero value", value: 0.0, wantErr: false},
		{name: "negative value", value: -1.5, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelCall := &ModelCall{
				RunID:     1,
				Profile:   "valid value",
				Model:     "valid value",
				TokensIn:  1,
				TokensOut: 1,
				Cost:      tt.value,
				LatencyMs: 1,
			}

			err := modelCall.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestModelCall_LatencyMs_NumericValidation tests numeric validation for LatencyMs
func TestModelCall_LatencyMs_NumericValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{name: "positive value", value: 10, wantErr: false},
		{name: "zero value", value: 0, wantErr: false},
		{name: "negative value", value: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelCall := &ModelCall{
				RunID:     1,
				Profile:   "valid value",
				Model:     "valid value",
				TokensIn:  1,
				TokensOut: 1,
				Cost:      10.5,
				LatencyMs: tt.value,
			}

			err := modelCall.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
