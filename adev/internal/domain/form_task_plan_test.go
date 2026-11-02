package domain

import "testing"

// TestRelationsForTasksKeepsOnlyTheRelatedEdges: the task plan gets the HLD edges
// whose endpoint a task names, and nothing else. A test for an integration names
// both containers; a task that names no container gets no relation.
func TestRelationsForTasksKeepsOnlyTheRelatedEdges(t *testing.T) {
	hld := HLD{ContextRelations: []ContextRelation{
		{From: "c-1", To: "c-2", Description: Text{Value: "checks the session"}, CauseRef: "fr-2"},
		{From: "c-2", To: "c-3", Description: Text{Value: "shows the progress"}, CauseRef: "fr-3"},
		{From: "c-4", To: "c-5", Description: Text{Value: "passes the grade"}, CauseRef: "fr-5"},
	}}

	tasks := []PlanTask{
		{ID: "t-1", ContainerRefs: []string{"c-1"}},
		{ID: "t-2", ContainerRefs: []string{"c-1", "c-2"}},
	}
	relations := RelationsForTasks(hld, tasks)
	if len(relations) != 2 {
		t.Fatalf("want the two edges touching c-1 or c-2, got %+v", relations)
	}
	if relations[0].CauseRef != "fr-2" || relations[1].CauseRef != "fr-3" {
		t.Fatalf("unexpected relation subset: %+v", relations)
	}

	empty := RelationsForTasks(hld, []PlanTask{{ID: "t-3"}})
	if len(empty) != 0 {
		t.Fatalf("a task with no container reference gets no relation, got %+v", empty)
	}
}
