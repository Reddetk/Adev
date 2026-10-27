package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	apphttp "github.com/Reddetk/Adev/internal/handler/http"
)

type fakeLogSource struct{ lines []string }

func (f *fakeLogSource) Lines() []string { return f.lines }

func newLogAPI(lines []string) *mux.Router {
	router := mux.NewRouter()
	version := router.PathPrefix("/api/v1").Subrouter()
	apphttp.SetupLogRoutes(version, apphttp.NewLogHandler(&fakeLogSource{lines: lines}))
	return router
}

func TestLogListReturnsTheNewestLines(t *testing.T) {
	router := newLogAPI([]string{"one", "two", "three"})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/logs?limit=2", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"total":3`, "the total counts every kept line")
	assert.Contains(t, recorder.Body.String(), `"lines":["two","three"]`)
	assert.NotContains(t, recorder.Body.String(), `"one"`)
}

func TestLogListDefaultsToOnePage(t *testing.T) {
	lines := make([]string, 300)
	for index := range lines {
		lines[index] = "line"
	}
	router := newLogAPI(lines)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "null")
	assert.Equal(t, 200, countOccurrences(recorder.Body.String(), `"line"`))
}

func TestLogListAnswersWithAnEmptyListBeforeAnyLine(t *testing.T) {
	router := newLogAPI(nil)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"lines":[]`, "the console must not parse null")
}

func countOccurrences(haystack, needle string) int {
	count := 0
	for index := 0; index+len(needle) <= len(haystack); index++ {
		if haystack[index:index+len(needle)] == needle {
			count++
		}
	}
	return count
}
