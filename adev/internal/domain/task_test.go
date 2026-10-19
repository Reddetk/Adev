package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTask_Validate tests the Validate method with various scenarios
func TestTask_Validate(t *testing.T) {
	tests := []struct {
		name    string
		task    Task
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid entity",
			task: Task{
				PipelineID:  1,
				ParentID:    1,
				Owner:       "valid value",
				TaskType:    "valid value",
				Status:      "active",
				ContractRef: "valid value",
			},
			wantErr: false,
		},
		{
			name: "invalid task - negative number",
			task: Task{
				PipelineID:  -1,
				ParentID:    1,
				Owner:       "valid value",
				TaskType:    "valid value",
				Status:      "active",
				ContractRef: "valid value",
			},
			wantErr: true,
			errMsg:  "pipelineid",
		},
		{
			name: "invalid task - negative number",
			task: Task{
				PipelineID:  1,
				ParentID:    -1,
				Owner:       "valid value",
				TaskType:    "valid value",
				Status:      "active",
				ContractRef: "valid value",
			},
			wantErr: true,
			errMsg:  "parentid",
		},
		{
			name: "invalid task - empty string",
			task: Task{
				PipelineID:  1,
				ParentID:    1,
				Owner:       "",
				TaskType:    "valid value",
				Status:      "active",
				ContractRef: "valid value",
			},
			wantErr: true,
			errMsg:  "owner",
		},
		{
			name: "invalid task - empty string",
			task: Task{
				PipelineID:  1,
				ParentID:    1,
				Owner:       "valid value",
				TaskType:    "",
				Status:      "active",
				ContractRef: "valid value",
			},
			wantErr: true,
			errMsg:  "tasktype",
		},
		{
			name: "invalid task - empty string",
			task: Task{
				PipelineID:  1,
				ParentID:    1,
				Owner:       "valid value",
				TaskType:    "valid value",
				Status:      "",
				ContractRef: "valid value",
			},
			wantErr: true,
			errMsg:  "status",
		},
		{
			name: "invalid task - empty string",
			task: Task{
				PipelineID:  1,
				ParentID:    1,
				Owner:       "valid value",
				TaskType:    "valid value",
				Status:      "active",
				ContractRef: "",
			},
			wantErr: true,
			errMsg:  "contractref",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
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

// TestTask_Initialization tests entity field initialization
func TestTask_Initialization(t *testing.T) {
	task := &Task{
		PipelineID:  1,
		ParentID:    1,
		Owner:       "valid value",
		TaskType:    "valid value",
		Status:      "active",
		ContractRef: "valid value",
	}

	assert.Equal(t, 1, task.PipelineID, "PipelineID should be set correctly")
	assert.Equal(t, 1, task.ParentID, "ParentID should be set correctly")
	assert.Equal(t, "valid value", task.Owner, "Owner should be set correctly")
	assert.Equal(t, "valid value", task.TaskType, "TaskType should be set correctly")
	assert.Equal(t, "active", task.Status, "Status should be set correctly")
	assert.Equal(t, "valid value", task.ContractRef, "ContractRef should be set correctly")
}

// TestTask_PipelineID_NumericValidation tests numeric validation for PipelineID
func TestTask_PipelineID_NumericValidation(t *testing.T) {
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
			task := &Task{
				PipelineID:  tt.value,
				ParentID:    1,
				Owner:       "valid value",
				TaskType:    "valid value",
				Status:      "active",
				ContractRef: "valid value",
			}

			err := task.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTask_ParentID_NumericValidation tests numeric validation for ParentID
func TestTask_ParentID_NumericValidation(t *testing.T) {
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
			task := &Task{
				PipelineID:  1,
				ParentID:    tt.value,
				Owner:       "valid value",
				TaskType:    "valid value",
				Status:      "active",
				ContractRef: "valid value",
			}

			err := task.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTask_Owner_EdgeCases tests edge cases for Owner field
func TestTask_Owner_EdgeCases(t *testing.T) {
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
			task := &Task{
				PipelineID:  1,
				ParentID:    1,
				Owner:       tt.value,
				TaskType:    "valid value",
				Status:      "active",
				ContractRef: "valid value",
			}

			err := task.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTask_TaskType_EdgeCases tests edge cases for TaskType field
func TestTask_TaskType_EdgeCases(t *testing.T) {
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
			task := &Task{
				PipelineID:  1,
				ParentID:    1,
				Owner:       "valid value",
				TaskType:    tt.value,
				Status:      "active",
				ContractRef: "valid value",
			}

			err := task.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTask_Status_EdgeCases tests edge cases for Status field
func TestTask_Status_EdgeCases(t *testing.T) {
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
			task := &Task{
				PipelineID:  1,
				ParentID:    1,
				Owner:       "valid value",
				TaskType:    "valid value",
				Status:      tt.value,
				ContractRef: "valid value",
			}

			err := task.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTask_ContractRef_EdgeCases tests edge cases for ContractRef field
func TestTask_ContractRef_EdgeCases(t *testing.T) {
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
			task := &Task{
				PipelineID:  1,
				ParentID:    1,
				Owner:       "valid value",
				TaskType:    "valid value",
				Status:      "active",
				ContractRef: tt.value,
			}

			err := task.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
