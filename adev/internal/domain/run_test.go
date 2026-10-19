package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRun_Validate tests the Validate method with various scenarios
func TestRun_Validate(t *testing.T) {
	tests := []struct {
		name    string
		run     Run
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid entity",
			run: Run{
				PipelineID:   1,
				Stage:        "valid value",
				Status:       "active",
				CheckpointID: "valid value",
				Attempts:     1,
				LastError:    "valid value",
			},
			wantErr: false,
		},
		{
			name: "invalid run - negative number",
			run: Run{
				PipelineID:   -1,
				Stage:        "valid value",
				Status:       "active",
				CheckpointID: "valid value",
				Attempts:     1,
				LastError:    "valid value",
			},
			wantErr: true,
			errMsg:  "pipelineid",
		},
		{
			name: "invalid run - empty string",
			run: Run{
				PipelineID:   1,
				Stage:        "",
				Status:       "active",
				CheckpointID: "valid value",
				Attempts:     1,
				LastError:    "valid value",
			},
			wantErr: true,
			errMsg:  "stage",
		},
		{
			name: "invalid run - empty string",
			run: Run{
				PipelineID:   1,
				Stage:        "valid value",
				Status:       "",
				CheckpointID: "valid value",
				Attempts:     1,
				LastError:    "valid value",
			},
			wantErr: true,
			errMsg:  "status",
		},
		{
			name: "invalid run - empty string",
			run: Run{
				PipelineID:   1,
				Stage:        "valid value",
				Status:       "active",
				CheckpointID: "",
				Attempts:     1,
				LastError:    "valid value",
			},
			wantErr: true,
			errMsg:  "checkpointid",
		},
		{
			name: "invalid run - negative number",
			run: Run{
				PipelineID:   1,
				Stage:        "valid value",
				Status:       "active",
				CheckpointID: "valid value",
				Attempts:     -1,
				LastError:    "valid value",
			},
			wantErr: true,
			errMsg:  "attempts",
		},
		{
			name: "invalid run - empty string",
			run: Run{
				PipelineID:   1,
				Stage:        "valid value",
				Status:       "active",
				CheckpointID: "valid value",
				Attempts:     1,
				LastError:    "",
			},
			wantErr: true,
			errMsg:  "lasterror",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run.Validate()
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

// TestRun_Initialization tests entity field initialization
func TestRun_Initialization(t *testing.T) {
	run := &Run{
		PipelineID:   1,
		Stage:        "valid value",
		Status:       "active",
		CheckpointID: "valid value",
		Attempts:     1,
		LastError:    "valid value",
	}

	assert.Equal(t, 1, run.PipelineID, "PipelineID should be set correctly")
	assert.Equal(t, "valid value", run.Stage, "Stage should be set correctly")
	assert.Equal(t, "active", run.Status, "Status should be set correctly")
	assert.Equal(t, "valid value", run.CheckpointID, "CheckpointID should be set correctly")
	assert.Equal(t, 1, run.Attempts, "Attempts should be set correctly")
	assert.Equal(t, "valid value", run.LastError, "LastError should be set correctly")
}

// TestRun_PipelineID_NumericValidation tests numeric validation for PipelineID
func TestRun_PipelineID_NumericValidation(t *testing.T) {
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
			run := &Run{
				PipelineID:   tt.value,
				Stage:        "valid value",
				Status:       "active",
				CheckpointID: "valid value",
				Attempts:     1,
				LastError:    "valid value",
			}

			err := run.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRun_Stage_EdgeCases tests edge cases for Stage field
func TestRun_Stage_EdgeCases(t *testing.T) {
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
			run := &Run{
				PipelineID:   1,
				Stage:        tt.value,
				Status:       "active",
				CheckpointID: "valid value",
				Attempts:     1,
				LastError:    "valid value",
			}

			err := run.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRun_Status_EdgeCases tests edge cases for Status field
func TestRun_Status_EdgeCases(t *testing.T) {
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
			run := &Run{
				PipelineID:   1,
				Stage:        "valid value",
				Status:       tt.value,
				CheckpointID: "valid value",
				Attempts:     1,
				LastError:    "valid value",
			}

			err := run.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRun_CheckpointID_EdgeCases tests edge cases for CheckpointID field
func TestRun_CheckpointID_EdgeCases(t *testing.T) {
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
			run := &Run{
				PipelineID:   1,
				Stage:        "valid value",
				Status:       "active",
				CheckpointID: tt.value,
				Attempts:     1,
				LastError:    "valid value",
			}

			err := run.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRun_Attempts_NumericValidation tests numeric validation for Attempts
func TestRun_Attempts_NumericValidation(t *testing.T) {
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
			run := &Run{
				PipelineID:   1,
				Stage:        "valid value",
				Status:       "active",
				CheckpointID: "valid value",
				Attempts:     tt.value,
				LastError:    "valid value",
			}

			err := run.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRun_LastError_EdgeCases tests edge cases for LastError field
func TestRun_LastError_EdgeCases(t *testing.T) {
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
			run := &Run{
				PipelineID:   1,
				Stage:        "valid value",
				Status:       "active",
				CheckpointID: "valid value",
				Attempts:     1,
				LastError:    tt.value,
			}

			err := run.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
