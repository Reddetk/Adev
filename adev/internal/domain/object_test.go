package domain

import "testing"

func TestObjectKindAndStatusAreValidated(t *testing.T) {
	if !KindModule.Valid() {
		t.Fatal("module must be a known kind")
	}
	if Kind("wormhole").Valid() {
		t.Fatal("an unknown kind must not be valid")
	}
	if !ObjectPlanned.Valid() || ObjectStatus("draft").Valid() {
		t.Fatal("status validation is wrong")
	}
}

func TestObjectLifecycleTransitions(t *testing.T) {
	cases := []struct {
		from, to ObjectStatus
		want     bool
	}{
		{ObjectPlanned, ObjectActive, true},
		{ObjectPlanned, ObjectRemoved, true},
		{ObjectActive, ObjectRemoved, true},
		{ObjectActive, ObjectActive, false},
		{ObjectRemoved, ObjectActive, false},
		{ObjectRemoved, ObjectRemoved, false},
	}
	for _, item := range cases {
		if got := CanChange(item.from, item.to); got != item.want {
			t.Errorf("CanChange(%s, %s) = %v, want %v", item.from, item.to, got, item.want)
		}
	}
}
