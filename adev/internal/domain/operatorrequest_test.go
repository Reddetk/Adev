package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOperatorRequest_Validate tests the Validate method with various scenarios
func TestOperatorRequest_Validate(t *testing.T) {
	tests := []struct {
		name            string
		operatorRequest OperatorRequest
		wantErr         bool
		errMsg          string
	}{
		{
			name: "valid entity",
			operatorRequest: OperatorRequest{
				PipelineID: 1,
				Pointer:    "valid value",
				Question:   "valid value",
				Blocking:   true,
				Status:     "active",
			},
			wantErr: false,
		},
		{
			name: "invalid operatorRequest - negative number",
			operatorRequest: OperatorRequest{
				PipelineID: -1,
				Pointer:    "valid value",
				Question:   "valid value",
				Blocking:   true,
				Status:     "active",
			},
			wantErr: true,
			errMsg:  "pipelineid",
		},
		{
			name: "invalid operatorRequest - empty string",
			operatorRequest: OperatorRequest{
				PipelineID: 1,
				Pointer:    "",
				Question:   "valid value",
				Blocking:   true,
				Status:     "active",
			},
			wantErr: true,
			errMsg:  "pointer",
		},
		{
			name: "invalid operatorRequest - empty string",
			operatorRequest: OperatorRequest{
				PipelineID: 1,
				Pointer:    "valid value",
				Question:   "",
				Blocking:   true,
				Status:     "active",
			},
			wantErr: true,
			errMsg:  "question",
		},
		{
			name: "invalid operatorRequest - empty string",
			operatorRequest: OperatorRequest{
				PipelineID: 1,
				Pointer:    "valid value",
				Question:   "valid value",
				Blocking:   true,
				Status:     "",
			},
			wantErr: true,
			errMsg:  "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operatorRequest.Validate()
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

// TestOperatorRequest_Initialization tests entity field initialization
func TestOperatorRequest_Initialization(t *testing.T) {
	operatorRequest := &OperatorRequest{
		PipelineID: 1,
		Pointer:    "valid value",
		Question:   "valid value",
		Blocking:   true,
		Status:     "active",
	}

	assert.Equal(t, 1, operatorRequest.PipelineID, "PipelineID should be set correctly")
	assert.Equal(t, "valid value", operatorRequest.Pointer, "Pointer should be set correctly")
	assert.Equal(t, "valid value", operatorRequest.Question, "Question should be set correctly")
	assert.Equal(t, true, operatorRequest.Blocking, "Blocking should be set correctly")
	assert.Equal(t, "active", operatorRequest.Status, "Status should be set correctly")
}

// TestOperatorRequest_PipelineID_NumericValidation tests numeric validation for PipelineID
func TestOperatorRequest_PipelineID_NumericValidation(t *testing.T) {
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
			operatorRequest := &OperatorRequest{
				PipelineID: tt.value,
				Pointer:    "valid value",
				Question:   "valid value",
				Blocking:   true,
				Status:     "active",
			}

			err := operatorRequest.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestOperatorRequest_Pointer_EdgeCases tests edge cases for Pointer field
func TestOperatorRequest_Pointer_EdgeCases(t *testing.T) {
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
			operatorRequest := &OperatorRequest{
				PipelineID: 1,
				Pointer:    tt.value,
				Question:   "valid value",
				Blocking:   true,
				Status:     "active",
			}

			err := operatorRequest.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestOperatorRequest_Question_EdgeCases tests edge cases for Question field
func TestOperatorRequest_Question_EdgeCases(t *testing.T) {
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
			operatorRequest := &OperatorRequest{
				PipelineID: 1,
				Pointer:    "valid value",
				Question:   tt.value,
				Blocking:   true,
				Status:     "active",
			}

			err := operatorRequest.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestOperatorRequest_Status_EdgeCases tests edge cases for Status field
func TestOperatorRequest_Status_EdgeCases(t *testing.T) {
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
			operatorRequest := &OperatorRequest{
				PipelineID: 1,
				Pointer:    "valid value",
				Question:   "valid value",
				Blocking:   true,
				Status:     tt.value,
			}

			err := operatorRequest.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
