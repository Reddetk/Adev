package domain

import (
	"testing"
	"time"
)

// The generated validation used to demand values that are legitimately empty or
// zero: an error text on a successful run, a parent on a root task, a latency on
// a cached call, a timestamp before the store sets it.

func TestRunAcceptsAnEmptyLastError(t *testing.T) {
	run := Run{PipelineID: 1, Stage: "00", Status: "running", CheckpointID: "cp-1"}
	if err := run.Validate(); err != nil {
		t.Fatalf("a successful run must pass without lasterror: %v", err)
	}
}

func TestRunRejectsAnEmptyStage(t *testing.T) {
	run := Run{PipelineID: 1, Status: "running", CheckpointID: "cp-1"}
	if err := run.Validate(); err == nil {
		t.Fatal("want an error for an empty stage")
	}
}

func TestTaskAcceptsARootWithoutParent(t *testing.T) {
	task := Task{PipelineID: 1, Owner: "dev", TaskType: "feature", Status: "todo", ContractRef: "CT-1"}
	if err := task.Validate(); err != nil {
		t.Fatalf("a root task must pass with parentId 0: %v", err)
	}
}

func TestModelCallAcceptsZeroUsage(t *testing.T) {
	call := ModelCall{RunID: 1, Profile: "worker", Model: "qwen3-coder"}
	if err := call.Validate(); err != nil {
		t.Fatalf("a cached call must pass with zero tokens and zero latency: %v", err)
	}
}

func TestAuditEventAcceptsAMissingTimestamp(t *testing.T) {
	event := AuditEvent{PipelineID: 1, RunID: 1, EventType: "stage_started"}
	if err := event.Validate(); err != nil {
		t.Fatalf("an event without a timestamp must pass, the store fills it: %v", err)
	}
	event.CreatedAt = time.Now().UTC()
	if err := event.Validate(); err != nil {
		t.Fatalf("an event with a timestamp must pass: %v", err)
	}
}
