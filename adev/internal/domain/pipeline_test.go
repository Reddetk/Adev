package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPipeline_Validate tests the Validate method with various scenarios
func TestPipeline_Validate(t *testing.T) {
	tests := []struct {
		name     string
		pipeline Pipeline
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid entity",
			pipeline: Pipeline{
				TargetSystemID: 1,
				Route:          "valid value",
				Status:         "active",
				CurrentStage:   "valid value",
			},
			wantErr: false,
		},
		{
			name: "invalid pipeline - negative number",
			pipeline: Pipeline{
				TargetSystemID: -1,
				Route:          "valid value",
				Status:         "active",
				CurrentStage:   "valid value",
			},
			wantErr: true,
			errMsg:  "targetsystemid",
		},
		{
			name: "invalid pipeline - empty string",
			pipeline: Pipeline{
				TargetSystemID: 1,
				Route:          "",
				Status:         "active",
				CurrentStage:   "valid value",
			},
			wantErr: true,
			errMsg:  "route",
		},
		{
			name: "invalid pipeline - empty string",
			pipeline: Pipeline{
				TargetSystemID: 1,
				Route:          "valid value",
				Status:         "",
				CurrentStage:   "valid value",
			},
			wantErr: true,
			errMsg:  "status",
		},
		{
			name: "invalid pipeline - empty string",
			pipeline: Pipeline{
				TargetSystemID: 1,
				Route:          "valid value",
				Status:         "active",
				CurrentStage:   "",
			},
			wantErr: true,
			errMsg:  "currentstage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pipeline.Validate()
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

// TestPipeline_Initialization tests entity field initialization
func TestPipeline_Initialization(t *testing.T) {
	pipeline := &Pipeline{
		TargetSystemID: 1,
		Route:          "valid value",
		Status:         "active",
		CurrentStage:   "valid value",
	}

	assert.Equal(t, 1, pipeline.TargetSystemID, "TargetSystemID should be set correctly")
	assert.Equal(t, "valid value", pipeline.Route, "Route should be set correctly")
	assert.Equal(t, "active", pipeline.Status, "Status should be set correctly")
	assert.Equal(t, "valid value", pipeline.CurrentStage, "CurrentStage should be set correctly")
}

// TestPipeline_TargetSystemID_NumericValidation tests numeric validation for TargetSystemID
func TestPipeline_TargetSystemID_NumericValidation(t *testing.T) {
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
			pipeline := &Pipeline{
				TargetSystemID: tt.value,
				Route:          "valid value",
				Status:         "active",
				CurrentStage:   "valid value",
			}

			err := pipeline.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPipeline_Route_EdgeCases tests edge cases for Route field
func TestPipeline_Route_EdgeCases(t *testing.T) {
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
			pipeline := &Pipeline{
				TargetSystemID: 1,
				Route:          tt.value,
				Status:         "active",
				CurrentStage:   "valid value",
			}

			err := pipeline.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPipeline_Status_EdgeCases tests edge cases for Status field
func TestPipeline_Status_EdgeCases(t *testing.T) {
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
			pipeline := &Pipeline{
				TargetSystemID: 1,
				Route:          "valid value",
				Status:         tt.value,
				CurrentStage:   "valid value",
			}

			err := pipeline.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPipeline_CurrentStage_EdgeCases tests edge cases for CurrentStage field
func TestPipeline_CurrentStage_EdgeCases(t *testing.T) {
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
			pipeline := &Pipeline{
				TargetSystemID: 1,
				Route:          "valid value",
				Status:         "active",
				CurrentStage:   tt.value,
			}

			err := pipeline.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
