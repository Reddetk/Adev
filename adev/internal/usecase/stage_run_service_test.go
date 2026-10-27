package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
)

// TestStageRunService: the run row opens running, closes with the terminal state
// of the chain, and the runs of a dead process resume on the next boot.
func TestStageRunService(t *testing.T) {
	pipelines := map[int]*domain.Pipeline{
		1: {ID: 1, TargetSystemID: 3, Route: "greenfield", Status: "ready", CurrentStage: "00"},
	}
	request := StageRequest{PipelineID: "1", TargetSystemID: 3, Route: domain.RouteGreenfield, StageID: "00"}

	fixture := func(outcome StageOutcome, chainErr error) (*StageRunService, *fakeChain, *fakeRunStore) {
		chain := &fakeChain{outcome: outcome, err: chainErr}
		runs := newFakeRunStore()
		return NewStageRunService(chain, runs, &fakePipelineStore{pipelines: pipelines}), chain, runs
	}
	// waitFor: the row closes when the background chain finishes with its state.
	waitFor := func(runs *fakeRunStore, id uint) *domain.Run {
		require.Eventually(t, func() bool { return runs.byID(id) != nil && runs.byID(id).Status != "running" }, 2*time.Second, 10*time.Millisecond)
		return runs.byID(id)
	}

	t.Run("a finished chain closes the run done", func(t *testing.T) {
		service, _, runs := fixture(StageOutcome{Stage: "00", Attempts: 2}, nil)
		run, err := service.Start(t.Context(), request)
		require.NoError(t, err)

		row := waitFor(runs, run.ID)
		assert.Equal(t, "done", row.Status)
		assert.Equal(t, 2, row.Attempts)
	})

	t.Run("a blocked chain closes the run blocked", func(t *testing.T) {
		service, _, runs := fixture(StageOutcome{Stage: "00", Blocked: true, Gaps: []string{"goals: is empty"}}, nil)
		run, err := service.Start(t.Context(), request)
		require.NoError(t, err)
		assert.Equal(t, "blocked", waitFor(runs, run.ID).Status)
	})

	t.Run("an operator stop closes the run stopped", func(t *testing.T) {
		service, _, runs := fixture(StageOutcome{}, ErrStageStopped)
		run, err := service.Start(t.Context(), request)
		require.NoError(t, err)
		assert.Equal(t, "stopped", waitFor(runs, run.ID).Status)
	})

	t.Run("a model failure closes the run failed with the reason", func(t *testing.T) {
		service, _, runs := fixture(StageOutcome{}, assert.AnError)
		run, err := service.Start(t.Context(), request)
		require.NoError(t, err)
		row := waitFor(runs, run.ID)
		assert.Equal(t, "failed", row.Status)
		assert.NotEmpty(t, row.LastError)
	})

	t.Run("a chain panic still closes the run failed", func(t *testing.T) {
		chain := &fakeChain{panicWith: "gone"}
		runs := newFakeRunStore()
		service := NewStageRunService(chain, runs, &fakePipelineStore{pipelines: pipelines})
		run, err := service.Start(t.Context(), request)
		require.NoError(t, err)
		row := waitFor(runs, run.ID)
		assert.Equal(t, "failed", row.Status)
		assert.Contains(t, row.LastError, "panic: gone")
	})

	t.Run("a non numeric pipeline id does not even start", func(t *testing.T) {
		service, chain, runs := fixture(StageOutcome{}, nil)
		_, err := service.Start(t.Context(), StageRequest{PipelineID: "PIPE-9", TargetSystemID: 3, Route: domain.RouteGreenfield, StageID: "00"})
		assert.ErrorIs(t, err, domain.ErrInvalidRunPipelineID)
		assert.False(t, chain.isCalled(), "the chain never starts for a pipeline the run table cannot keep")
		assert.Empty(t, runs.rows, "no orphan row either")
	})

	t.Run("resume relaunches the rows of a dead process", func(t *testing.T) {
		runs := newFakeRunStore()
		chain := &fakeChain{hold: make(chan struct{})} // the chain stays in flight
		service := NewStageRunService(chain, runs, &fakePipelineStore{pipelines: pipelines})

		// Simulate a dead process: a running row the current process did not start.
		orphan := &domain.Run{PipelineID: 1, Stage: "00", Status: "running", CheckpointID: "greenfield"}
		require.NoError(t, runs.Save(orphan))

		assert.Equal(t, 1, service.Resume(t.Context()), "one orphan row relaunches")
		require.Eventually(t, func() bool { return chain.isCalled() }, 2*time.Second, 10*time.Millisecond)

		close(chain.hold) // the chain finishes on its own
		row := waitFor(runs, orphan.ID)
		assert.Equal(t, "done", row.Status)
		assert.Equal(t, 0, service.Resume(t.Context()), "the closed row never resumes again")
	})

	t.Run("resume does not double the runs of the live process", func(t *testing.T) {
		chain := &fakeChain{hold: make(chan struct{})}
		runs := newFakeRunStore()
		service := NewStageRunService(chain, runs, &fakePipelineStore{pipelines: pipelines})
		run, err := service.Start(t.Context(), request)
		require.NoError(t, err)
		assert.Equal(t, 0, service.Resume(t.Context()), "the live process owns its row and passes it by")
		close(chain.hold)
		require.Eventually(t, func() bool { return runs.byID(run.ID).Status == "done" }, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("resume marks the run of a lost pipeline failed", func(t *testing.T) {
		service, _, runs := fixture(StageOutcome{}, nil)
		orphan := &domain.Run{PipelineID: 42, Stage: "00", Status: "running", CheckpointID: "greenfield"}
		require.NoError(t, runs.Save(orphan))
		assert.Equal(t, 0, service.Resume(t.Context()))
		assert.Equal(t, "failed", runs.byID(orphan.ID).Status)
	})
}

// --- fakes -------------------------------------------------------------------

type fakeChain struct {
	mu        sync.Mutex
	outcome   StageOutcome
	err       error
	panicWith string
	hold      chan struct{} // when set, the chain stays in flight until the test closes it
	request   StageRequest
	called    bool
}

func (f *fakeChain) Run(ctx context.Context, request StageRequest) (StageOutcome, error) {
	f.mu.Lock()
	f.called, f.request = true, request
	f.mu.Unlock()
	if f.hold != nil {
		<-f.hold
	}
	if f.panicWith != "" {
		panic(f.panicWith)
	}
	return f.outcome, f.err
}

func (f *fakeChain) isCalled() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.called
}

type fakeRunStore struct {
	mu   sync.Mutex
	rows map[uint]*domain.Run
	next uint
}

func newFakeRunStore() *fakeRunStore { return &fakeRunStore{rows: map[uint]*domain.Run{}, next: 1} }

func (f *fakeRunStore) Save(run *domain.Run) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	run.ID = f.next
	f.next++
	f.rows[run.ID] = run
	return nil
}

func (f *fakeRunStore) Update(run *domain.Run) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[run.ID] = run
	return nil
}

func (f *fakeRunStore) byID(id uint) *domain.Run {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rows[id]
}

func (f *fakeRunStore) Delete(int) error { return nil }

func (f *fakeRunStore) FindByID(int) (*domain.Run, error)         { return nil, errNotFound }
func (f *fakeRunStore) FindByPipelineID(int) (*domain.Run, error) { return nil, errNotFound }
func (f *fakeRunStore) FindByStage(string) (*domain.Run, error)   { return nil, errNotFound }
func (f *fakeRunStore) FindByStatus(status string) (*domain.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, row := range f.rows {
		if row.Status == status {
			return row, nil
		}
	}
	return nil, errNotFound
}
func (f *fakeRunStore) FindByCheckpointID(string) (*domain.Run, error) { return nil, errNotFound }
func (f *fakeRunStore) FindByAttempts(int) (*domain.Run, error)        { return nil, errNotFound }
func (f *fakeRunStore) FindByLastError(string) (*domain.Run, error)    { return nil, errNotFound }
func (f *fakeRunStore) FindAll() ([]domain.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.Run, 0, len(f.rows))
	for _, row := range f.rows {
		out = append(out, *row)
	}
	return out, nil
}

// errNotFound stands for the sentinel the real repositories answer with; the use
// case layer may not import the repository package, so the fake keeps its own.
var errNotFound = errors.New("record not found")

type fakePipelineStore struct {
	pipelines map[int]*domain.Pipeline
}

func (f *fakePipelineStore) FindByID(id int) (*domain.Pipeline, error) {
	if pipeline, ok := f.pipelines[id]; ok {
		return pipeline, nil
	}
	return nil, errNotFound
}
