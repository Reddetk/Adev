package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/Reddetk/Adev/internal/domain"
)

// StageChain executes one stage run and follows the route to the next stage.
type StageChain interface {
	Run(ctx context.Context, request StageRequest) (StageOutcome, error)
}

// PipelineLookup reads the pipeline a run belongs to.
type PipelineLookup interface {
	FindByID(id int) (*domain.Pipeline, error)
}

// StageRunService starts stage runs in the background and keeps their state in the
// runs table. The HTTP request that starts a run answers at once, so a closed tab
// or a server restart no longer decides how long an agent works: the runs that
// outlive their process resume on the next boot.
type StageRunService struct {
	chain     StageChain
	runs      RunRepository
	pipelines PipelineLookup

	mu   sync.Mutex
	live map[uint]bool // run rows owned by this process: resume never doubles them
}

// NewStageRunService builds the service.
func NewStageRunService(chain StageChain, runs RunRepository, pipelines PipelineLookup) *StageRunService {
	return &StageRunService{chain: chain, runs: runs, pipelines: pipelines, live: map[uint]bool{}}
}

// Start records the run as running and executes the stage chain in the background;
// the caller gets the run row right away.
func (s *StageRunService) Start(ctx context.Context, request StageRequest) (*domain.Run, error) {
	pipelineID, err := strconv.Atoi(request.PipelineID)
	if err != nil || pipelineID < 0 {
		return nil, fmt.Errorf("%w: %q", domain.ErrInvalidRunPipelineID, request.PipelineID)
	}
	run := &domain.Run{
		PipelineID:   pipelineID,
		Stage:        request.StageID,
		Status:       "running",
		CheckpointID: string(request.Route),
	}
	if err := run.Validate(); err != nil {
		return nil, err
	}
	if err := s.runs.Save(run); err != nil {
		return nil, fmt.Errorf("stage runs: record run: %w", err)
	}
	s.claim(run.ID)
	s.execute(ctx, request, run)
	return run, nil
}

// Resume relaunches the runs that outlived their process: their rows still say
// running, but nothing executes them any more. The chain restarts at the stored
// stage, and the stages the cursor passed already confirm themselves as it goes.
func (s *StageRunService) Resume(ctx context.Context) int {
	all, err := s.runs.FindAll()
	if err != nil {
		log.Printf("stage runs: resume: %v", err)
		return 0
	}
	resumed := 0
	for i := range all {
		row := all[i]
		if row.Status != "running" {
			continue
		}
		if s.owns(row.ID) {
			continue // this process already runs it
		}
		pipeline, err := s.pipelines.FindByID(row.PipelineID)
		if err != nil {
			// The pipeline went away: the run can never find its inputs again.
			row.Status, row.LastError = "failed", "pipeline is gone"
			_ = s.runs.Update(&row)
			continue
		}
		request := StageRequest{
			PipelineID:     strconv.Itoa(row.PipelineID),
			TargetSystemID: int64(pipeline.TargetSystemID),
			Route:          domain.Route(pipeline.Route),
			StageID:        row.Stage,
		}
		s.claim(row.ID)
		s.execute(ctx, request, &row)
		resumed++
	}
	return resumed
}

// execute carries out the chain on a context no HTTP client can cancel: the
// operator may close the tab, the run keeps going. Stop stays possible through
// the activity book, which holds the cancel of the stage context.
func (s *StageRunService) execute(parent context.Context, request StageRequest, run *domain.Run) {
	go func() {
		// LIFO: the recover runs first, so a panic still closes the run row.
		defer func() {
			if err := s.runs.Update(run); err != nil {
				log.Printf("run %d stage %s: update status %s: %v", run.ID, run.Stage, run.Status, err)
			}
			s.release(run.ID)
		}()
		defer func() {
			if rec := recover(); rec != nil {
				run.Status, run.LastError = "failed", fmt.Sprintf("panic: %v", rec)
				log.Printf("run %d stage %s: panic: %v", run.ID, run.Stage, rec)
			}
		}()

		outcome, err := s.chain.Run(context.WithoutCancel(parent), request)
		switch {
		case errors.Is(err, ErrStageStopped):
			// The operator stopped the run: a normal terminal state.
			run.Status = "stopped"
		case err != nil:
			run.Status, run.LastError = "failed", err.Error()
		case outcome.Blocked:
			// Blocked is a normal pause: the questions wait in the dashboard.
			run.Status = "blocked"
		default:
			run.Status = "done"
		}
		run.Attempts = outcome.Attempts
	}()
}

func (s *StageRunService) claim(id uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.live[id] = true
}

func (s *StageRunService) release(id uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.live, id)
}

func (s *StageRunService) owns(id uint) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.live[id]
}
