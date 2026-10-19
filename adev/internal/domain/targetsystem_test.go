package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTargetSystem_Validate tests the Validate method with various scenarios
func TestTargetSystem_Validate(t *testing.T) {
	tests := []struct {
		name         string
		targetSystem TargetSystem
		wantErr      bool
		errMsg       string
	}{
		{
			name: "valid entity",
			targetSystem: TargetSystem{
				Name:        "John Doe",
				RepoURL:     "valid value",
				Description: "A valid description",
			},
			wantErr: false,
		},
		{
			name: "invalid targetSystem - empty string",
			targetSystem: TargetSystem{
				Name:        "",
				RepoURL:     "valid value",
				Description: "A valid description",
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "invalid targetSystem - empty string",
			targetSystem: TargetSystem{
				Name:        "John Doe",
				RepoURL:     "",
				Description: "A valid description",
			},
			wantErr: true,
			errMsg:  "repourl",
		},
		{
			name: "invalid targetSystem - empty string",
			targetSystem: TargetSystem{
				Name:        "John Doe",
				RepoURL:     "valid value",
				Description: "",
			},
			wantErr: true,
			errMsg:  "description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.targetSystem.Validate()
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

// TestTargetSystem_Initialization tests entity field initialization
func TestTargetSystem_Initialization(t *testing.T) {
	targetSystem := &TargetSystem{
		Name:        "John Doe",
		RepoURL:     "valid value",
		Description: "A valid description",
	}

	assert.Equal(t, "John Doe", targetSystem.Name, "Name should be set correctly")
	assert.Equal(t, "valid value", targetSystem.RepoURL, "RepoURL should be set correctly")
	assert.Equal(t, "A valid description", targetSystem.Description, "Description should be set correctly")
}

// TestTargetSystem_Name_EdgeCases tests edge cases for Name field
func TestTargetSystem_Name_EdgeCases(t *testing.T) {
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
			targetSystem := &TargetSystem{
				Name:        tt.value,
				RepoURL:     "valid value",
				Description: "A valid description",
			}

			err := targetSystem.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTargetSystem_RepoURL_EdgeCases tests edge cases for RepoURL field
func TestTargetSystem_RepoURL_EdgeCases(t *testing.T) {
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
			targetSystem := &TargetSystem{
				Name:        "John Doe",
				RepoURL:     tt.value,
				Description: "A valid description",
			}

			err := targetSystem.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTargetSystem_Description_EdgeCases tests edge cases for Description field
func TestTargetSystem_Description_EdgeCases(t *testing.T) {
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
			targetSystem := &TargetSystem{
				Name:        "John Doe",
				RepoURL:     "valid value",
				Description: tt.value,
			}

			err := targetSystem.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
