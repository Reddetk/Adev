package mocks

import (
	"github.com/stretchr/testify/mock"
	"net/http"
)

// MockPipelineHandler is a mock implementation of HTTP handler
type MockPipelineHandler struct {
	mock.Mock
}

// CreatePipeline mocks the CreatePipeline HTTP handler method
func (m *MockPipelineHandler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

// GetPipeline mocks the GetPipeline HTTP handler method
func (m *MockPipelineHandler) GetPipeline(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

// UpdatePipeline mocks the UpdatePipeline HTTP handler method
func (m *MockPipelineHandler) UpdatePipeline(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

// DeletePipeline mocks the DeletePipeline HTTP handler method
func (m *MockPipelineHandler) DeletePipeline(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

// ListPipelines mocks the ListPipelines HTTP handler method
func (m *MockPipelineHandler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

// NewMockPipelineHandler creates a new mock handler
func NewMockPipelineHandler() *MockPipelineHandler {
	return &MockPipelineHandler{}
}
