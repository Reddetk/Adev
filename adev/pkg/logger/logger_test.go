package logger

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRingKeepsTheNewestLinesInOrder(t *testing.T) {
	ring := NewRing(3)

	for _, line := range []string{"one", "two", "three", "four"} {
		if _, err := ring.Write([]byte(line + "\n")); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	assert.Equal(t, []string{"two", "three", "four"}, ring.Lines(), "the oldest line leaves first")
}

func TestRingSplitsOneWriteIntoLines(t *testing.T) {
	ring := NewRing(5)

	if _, err := ring.Write([]byte("first\nsecond\nthird\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	assert.Equal(t, []string{"first", "second", "third"}, ring.Lines())
}

// The standard logger writes every line with its own call, but a wrapped writer can
// hand over several at once and may leave a trailing newline.
func TestRingIgnoresEmptyLinesAndTrailingNewline(t *testing.T) {
	ring := NewRing(5)

	if _, err := ring.Write([]byte("only\n\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	assert.Equal(t, []string{"only"}, ring.Lines())
}

func TestRingReportsNothingBeforeTheFirstLine(t *testing.T) {
	assert.Empty(t, NewRing(4).Lines())
}

func TestRingKeepsTheLineLimit(t *testing.T) {
	ring := NewRing(50)

	if _, err := ring.Write([]byte(strings.Repeat("x", 120) + "\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	lines := ring.Lines()
	if len(lines) != 1 {
		t.Fatalf("want one line, got %d", len(lines))
	}
	assert.Equal(t, 120, len(lines[0]), "a long line is kept whole")
}
