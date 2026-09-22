package main

import (
	"math"
	"testing"
)

func colState() *State {
	lines := []Line{
		{Raw: "aa  bb c", Expanded: "aa  bb c"},
		{Raw: "d   ee f", Expanded: "d   ee f"},
	}
	return &State{
		Lines:  lines,
		Cols:   []Column{{0, 2}, {4, 6}, {7, math.MaxInt}},
		RowSel: map[int]bool{},
		ColSel: map[int]bool{},
		Mode:   ModeColumn,
	}
}

func assertOut(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %q; want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d: got %q; want %q", i, got[i], want[i])
		}
	}
}

func TestOutputIdentity(t *testing.T) {
	s := colState()
	assertOut(t, Output(s), []string{"aa  bb c", "d   ee f"})
}

func TestOutputRowSelectionRaw(t *testing.T) {
	s := colState()
	s.RowSel[0] = true
	assertOut(t, Output(s), []string{"aa  bb c"})
}

func TestOutputColumnSelection(t *testing.T) {
	s := colState()
	s.ColSel[0] = true
	s.ColSel[1] = true
	assertOut(t, Output(s), []string{"aa bb", "d  ee"})
}

func TestOutputColumnSelectionNonAdjacent(t *testing.T) {
	s := colState()
	s.ColSel[0] = true
	s.ColSel[2] = true
	assertOut(t, Output(s), []string{"aa c", "d  f"})
}

func TestOutputColumnOrderAscending(t *testing.T) {
	s := colState()
	s.ColSel[2] = true
	s.ColSel[0] = true
	assertOut(t, Output(s), []string{"aa c", "d  f"})
}

func TestOutputRowsAndColumns(t *testing.T) {
	s := colState()
	s.RowSel[0] = true
	s.ColSel[0] = true
	s.ColSel[1] = true
	s.ColSel[2] = true
	assertOut(t, Output(s), []string{"aa bb c"})
}

func TestOutputFilterNoSelection(t *testing.T) {
	s := NewState(NewLines([]byte("foo x\nbar y\n")))
	s.FilterQuery = "foo"
	assertOut(t, Output(s), []string{"foo x"})
}

func TestOutputFilterAndRows(t *testing.T) {
	s := NewState(NewLines([]byte("foo x\nfoo y\n")))
	s.FilterQuery = "foo"
	s.RowSel[1] = true
	assertOut(t, Output(s), []string{"foo y"})
}

func TestOutputEmptyChosenRows(t *testing.T) {
	s := colState()
	s.RowSel[99] = true
	s.ColSel[0] = true
	if got := Output(s); len(got) != 0 {
		t.Fatalf("got %q; want empty", got)
	}
}
