package main

import "testing"

func TestWidthsModeIndependent(t *testing.T) {
	s := NewState(NewLines([]byte("aa  bb c\nd   ee f\n")))
	visible := s.VisibleRows()

	s.Mode = ModeColumn
	colWidths := displayWidths(s, visible)
	colNum := numWidth(s)

	s.Mode = ModeRow
	rowWidths := displayWidths(s, visible)
	rowNum := numWidth(s)

	if colNum != rowNum {
		t.Fatalf("numWidth differs: column=%d row=%d", colNum, rowNum)
	}
	if len(colWidths) != len(rowWidths) {
		t.Fatalf("width count differs")
	}
	for i := range colWidths {
		if colWidths[i] != rowWidths[i] {
			t.Fatalf("col %d width differs: column=%d row=%d", i, colWidths[i], rowWidths[i])
		}
	}
}
