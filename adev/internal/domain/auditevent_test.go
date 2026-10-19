package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestAuditEvent_Validate tests the Validate method with various scenarios
func TestAuditEvent_Validate(t *testing.T) {
	tests := []struct {
		name       string
		auditEvent AuditEvent
		wantErr    bool
		errMsg     string
	}{
		{
			name: "valid entity",
			auditEvent: AuditEvent{
				PipelineID: 1,
				RunID:      1,
				EventType:  "valid value",
				CreatedAt:  time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid auditEvent - negative number",
			auditEvent: AuditEvent{
				PipelineID: -1,
				RunID:      1,
				EventType:  "valid value",
				CreatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "pipelineid",
		},
		{
			name: "invalid auditEvent - negative number",
			auditEvent: AuditEvent{
				PipelineID: 1,
				RunID:      -1,
				EventType:  "valid value",
				CreatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "runid",
		},
		{
			name: "invalid auditEvent - empty string",
			auditEvent: AuditEvent{
				PipelineID: 1,
				RunID:      1,
				EventType:  "",
				CreatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "eventtype",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.auditEvent.Validate()
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

// TestAuditEvent_Initialization tests entity field initialization
func TestAuditEvent_Initialization(t *testing.T) {
	auditEvent := &AuditEvent{
		PipelineID: 1,
		RunID:      1,
		EventType:  "valid value",
		CreatedAt:  time.Now(),
	}

	assert.Equal(t, 1, auditEvent.PipelineID, "PipelineID should be set correctly")
	assert.Equal(t, 1, auditEvent.RunID, "RunID should be set correctly")
	assert.Equal(t, "valid value", auditEvent.EventType, "EventType should be set correctly")
	assert.False(t, auditEvent.CreatedAt.IsZero(), "CreatedAt should be set correctly")
}

// TestAuditEvent_PipelineID_NumericValidation tests numeric validation for PipelineID
func TestAuditEvent_PipelineID_NumericValidation(t *testing.T) {
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
			auditEvent := &AuditEvent{
				PipelineID: tt.value,
				RunID:      1,
				EventType:  "valid value",
				CreatedAt:  time.Now(),
			}

			err := auditEvent.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAuditEvent_RunID_NumericValidation tests numeric validation for RunID
func TestAuditEvent_RunID_NumericValidation(t *testing.T) {
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
			auditEvent := &AuditEvent{
				PipelineID: 1,
				RunID:      tt.value,
				EventType:  "valid value",
				CreatedAt:  time.Now(),
			}

			err := auditEvent.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAuditEvent_EventType_EdgeCases tests edge cases for EventType field
func TestAuditEvent_EventType_EdgeCases(t *testing.T) {
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
			auditEvent := &AuditEvent{
				PipelineID: 1,
				RunID:      1,
				EventType:  tt.value,
				CreatedAt:  time.Now(),
			}

			err := auditEvent.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
