package usecase

import (
	"context"
	"testing"
)

// The cancel book stops exactly the runs the operator named: one role, or the
// whole pipeline. A cancel of one pipeline never reaches another one.
func TestActivityCancelStopsTheNamedRuns(t *testing.T) {
	book := NewActivityService()
	architect, architectStop := context.WithCancel(context.Background())
	defer architectStop()
	analyst, analystStop := context.WithCancel(context.Background())
	defer analystStop()
	other, otherStop := context.WithCancel(context.Background())
	defer otherStop()

	book.RegisterRun("3", "architect", architectStop)
	book.RegisterRun("3", "analyst", analystStop)
	book.RegisterRun("4", "architect", otherStop)

	if count := book.Cancel("3", "architect"); count != 1 {
		t.Fatalf("cancel one role: want 1 stopped run, got %d", count)
	}
	if architect.Err() == nil {
		t.Fatal("the architect run keeps running after the stop")
	}
	if analyst.Err() != nil || other.Err() != nil {
		t.Fatal("the stop reached runs it was not aimed at")
	}

	// A second cancel of the same key is a no-op: the handle was dropped.
	if count := book.Cancel("3", "architect"); count != 0 {
		t.Fatalf("double cancel: want 0, got %d", count)
	}
	if count := book.Cancel("3", ""); count != 1 {
		t.Fatalf("cancel the whole pipeline: want 1 (the analyst), got %d", count)
	}
	if analyst.Err() == nil {
		t.Fatal("the whole-pipeline stop left the analyst running")
	}
}
