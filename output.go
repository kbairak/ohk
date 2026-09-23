package main

import "strings"

func Output(s *State) []string {
	visible := s.VisibleRows()

	var chosenRows []int
	if len(s.RowSel) == 0 {
		chosenRows = append(chosenRows, visible...)
	} else {
		for _, i := range visible {
			if s.RowSel[i] {
				chosenRows = append(chosenRows, i)
			}
		}
	}

	chosenCols := make([]int, 0, len(s.ColSel))
	for i := range s.ColSel {
		chosenCols = append(chosenCols, i)
	}
	sortInts(chosenCols)

	if len(s.ColSel) == 0 {
		if len(s.RowSel) == 0 && s.FilterQuery == "" && s.SortDir == SortOff {
			out := make([]string, 0, len(s.Lines))
			for _, l := range s.Lines {
				out = append(out, l.Raw)
			}
			return out
		}
		out := make([]string, 0, len(chosenRows))
		for _, i := range chosenRows {
			out = append(out, s.Lines[i].Raw)
		}
		return out
	}

	if len(chosenRows) == 0 {
		return nil
	}

	cells := make([][]string, len(chosenRows))
	widths := make([]int, len(chosenCols))
	for r, ri := range chosenRows {
		row := make([]string, len(chosenCols))
		for c, ci := range chosenCols {
			col := s.Cols[ci]
			cell := trimCell(s.Lines[ri].Expanded, col.Start, col.End)
			row[c] = cell
			if len(cell) > widths[c] {
				widths[c] = len(cell)
			}
		}
		cells[r] = row
	}

	out := make([]string, len(chosenRows))
	for r := range cells {
		var b strings.Builder
		for c := range chosenCols {
			if c == len(chosenCols)-1 {
				b.WriteString(cells[r][c])
			} else {
				b.WriteString(cells[r][c])
				b.WriteString(strings.Repeat(" ", widths[c]-len(cells[r][c])))
				b.WriteString(" ")
			}
		}
		out[r] = b.String()
	}
	return out
}
