package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// A stage run waits on the model for minutes, so its handler extends the write
// deadline. The logging wrapper must not block that extension: without Unwrap the
// response dies on the server WriteTimeout and the client sees a socket hang up.
func TestLoggingLetsTheHandlerExtendTheWriteDeadline(t *testing.T) {
	handler := Logging()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Errorf("set write deadline: %v", err)
		}
		time.Sleep(600 * time.Millisecond)
		_, _ = io.WriteString(w, "done")
	}))

	server := httptest.NewUnstartedServer(handler)
	server.Config.WriteTimeout = 200 * time.Millisecond
	server.Start()
	defer server.Close()

	response, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("the server dropped the connection: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if response.StatusCode != http.StatusOK || string(body) != "done" {
		t.Fatalf("want 200 %q, got %d %q", "done", response.StatusCode, body)
	}
}
