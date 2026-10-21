package usecase

import (
	"sort"
	"strconv"
	"sync"
	"time"
)

// Activity is the live state of one stage-run fill: which agent works on which
// stage of which pipeline, how many model attempts it made, and where it ended.
type Activity struct {
	ID         string     `json:"id"`
	PipelineID string     `json:"pipelineId"`
	Stage      string     `json:"stage"`
	Role       string     `json:"role"`
	Output     string     `json:"output"`
	Attempts   int        `json:"attempts"`
	Status     string     `json:"status"` // running | done | blocked | failed
	StartedAt  time.Time  `json:"startedAt"`
	EndedAt    *time.Time `json:"endedAt,omitempty"`
	LastError  string     `json:"lastError,omitempty"`

	book *ActivityService
}

// ActivityService is the in-memory book of recent stage runs. The operator
// dashboard reads it to see what the agents work on right now. No storage: a
// restart clears the book, and the finished runs live in the run table.
type ActivityService struct {
	mu   sync.Mutex
	book []*Activity
}

// NewActivityService builds the activity book.
func NewActivityService() *ActivityService {
	return &ActivityService{}
}

// Start registers a running activity and returns its live handle.
func (s *ActivityService) Start(pipelineID, stage, role, output string) *Activity {
	if s == nil {
		return nil
	}
	activity := &Activity{
		ID:         "act-" + strconv.FormatInt(time.Now().UnixNano(), 36),
		PipelineID: pipelineID,
		Stage:      stage,
		Role:       role,
		Output:     output,
		Status:     "running",
		StartedAt:  time.Now(),
		book:       s,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.book = append(s.book, activity)
	// The book is a window on the recent past, not an archive.
	if len(s.book) > 60 {
		s.book = s.book[len(s.book)-60:]
	}
	return activity
}

// Attempt counts one model call the activity is waiting on.
func (a *Activity) Attempt() {
	if a == nil || a.book == nil {
		return
	}
	a.book.mu.Lock()
	defer a.book.mu.Unlock()
	a.Attempts++
}

// Finish marks the activity ended. A finished activity never changes status again.
func (a *Activity) Finish(status, lastError string) {
	if a == nil || a.book == nil {
		return
	}
	now := time.Now()
	a.book.mu.Lock()
	defer a.book.mu.Unlock()
	if a.Status != "running" {
		return
	}
	a.Status, a.LastError, a.EndedAt = status, lastError, &now
}

// Finished reports whether the activity already reached a final status.
func (a *Activity) Finished() bool {
	if a == nil {
		return true
	}
	return a.Status != "running"
}

// List copies the book: the running activities first, then the newest finished.
func (s *ActivityService) List() []Activity {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Activity, 0, len(s.book))
	for _, entry := range s.book {
		out = append(out, *entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		left, right := out[i].Status == "running", out[j].Status == "running"
		if left != right {
			return left
		}
		return out[i].StartedAt.After(out[j].StartedAt)
	})
	return out
}
