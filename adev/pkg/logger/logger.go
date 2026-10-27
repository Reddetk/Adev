package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ringSize is how many recent lines the operator console can read back.
const ringSize = 500

// Ring keeps the last log lines in memory. A server started as a background process
// has no console to watch, so the console reads them over HTTP instead.
type Ring struct {
	mu    sync.Mutex
	lines []string
	next  int
	full  bool
}

// NewRing builds a ring of the given capacity.
func NewRing(size int) *Ring {
	if size < 1 {
		size = 1
	}
	return &Ring{lines: make([]string, size)}
}

// Write accepts a log write, so a Ring can sit in an io.MultiWriter.
func (r *Ring) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		r.lines[r.next] = line
		r.next = (r.next + 1) % len(r.lines)
		if r.next == 0 {
			r.full = true
		}
	}
	return len(p), nil
}

// Lines returns the kept lines, oldest first.
func (r *Ring) Lines() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.full {
		out := make([]string, r.next)
		copy(out, r.lines[:r.next])
		return out
	}
	out := make([]string, 0, len(r.lines))
	out = append(out, r.lines[r.next:]...)
	out = append(out, r.lines[:r.next]...)
	return out
}

var defaultRing *Ring

// InfoLogger and ErrorLogger keep the prefixes the rest of the program uses.
var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

// Info writes one informational line.
func Info(v ...interface{}) {
	if InfoLogger != nil {
		InfoLogger.Println(v...)
	}
}

// Error writes one error line.
func Error(v ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Println(v...)
	}
}

// Init sends every log line to the console, to the log file and to the ring the
// operator console reads. Everything in this program logs through the standard
// logger, so one output covers it all.
func Init() {
	defaultRing = NewRing(ringSize)
	var output io.Writer = io.MultiWriter(os.Stdout, defaultRing)

	// LOG_FILE is the record that survives a background process.
	path := getEnv("LOG_FILE", filepath.Join("logs", "adev.log"))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("Warning: cannot create the log directory for %s: %v", path, err)
	} else if file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err != nil {
		log.Printf("Warning: cannot write the log file %s: %v", path, err)
	} else {
		output = io.MultiWriter(os.Stdout, defaultRing, file)
		log.Printf("Writing the log to %s", path)
	}

	log.SetOutput(output)
	InfoLogger = log.New(output, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(output, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Default returns the ring the console reads, or nil before Init.
func Default() *Ring { return defaultRing }

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
