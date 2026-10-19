package mocks

import (
	"github.com/stretchr/testify/mock"
	"net/http"
)

// MockRunHandler is a mock implementation of HTTP handler
type MockRunHandler struct {
	mock.Mock
}

// CreateRun mocks the CreateRun HTTP handler method
func (m *MockRunHandler) CreateRun(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

// GetRun mocks the GetRun HTTP handler method
func (m *MockRunHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

// UpdateRun mocks the UpdateRun HTTP handler method
func (m *MockRunHandler) UpdateRun(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

// DeleteRun mocks the DeleteRun HTTP handler method
func (m *MockRunHandler) DeleteRun(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

// ListRuns mocks the ListRuns HTTP handler method
func (m *MockRunHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

// NewMockRunHandler creates a new mock handler
func NewMockRunHandler() *MockRunHandler {
	return &MockRunHandler{}
}
