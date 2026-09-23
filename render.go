package main

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	invOn    = "\x1b[7m"
	invOff   = "\x1b[27m"
	statusBG = "\x1b[48;5;238m"
	statusFG = "\x1b[38;5;252m"
	sgrReset = "\x1b[0m"
)

func numWidth(s *State) int {
	maxN := len(s.Cols)
	if v := len(s.VisibleRows()); v > maxN {
		maxN = v
	}
	return len(strconv.Itoa(maxN)) + 1
}

func labelWidth(i int) int {
	return len(strconv.Itoa(i+1)) + 4
}

func displayWidths(s *State, visible []int) []int {
	ws := make([]int, len(s.Cols))
	for i, col := range s.Cols {
		ws[i] = labelWidth(i)
		for _, ri := range visible {
			c := trimCell(s.Lines[ri].Expanded, col.Start, col.End)
			if len(c) > ws[i] {
				ws[i] = len(c)
			}
		}
	}
	return ws
}

func Render(s *State, w, h int) string {
	visible := s.VisibleRows()
	widths := displayWidths(s, visible)
	numW := numWidth(s)

	lines := make([]string, 0, h)
	lines = append(lines, renderStatus(s, w))
	lines = append(lines, renderHeader(s, widths, numW))

	maxRows := max(h-2, 0)
	shown := visible
	if len(shown) > maxRows {
		shown = shown[:maxRows]
	}
	for dn, orig := range shown {
		lines = append(lines, renderRow(s, dn+1, orig, w, widths, numW))
	}

	var frame strings.Builder
	frame.WriteString("\x1b[H\x1b[2J")
	for i, l := range lines {
		if i > 0 {
			frame.WriteString("\r\n")
		}
		if i == 0 {
			frame.WriteString(l)
		} else {
			frame.WriteString(cropBytes(l, w))
		}
		frame.WriteString("\x1b[K")
	}
	return frame.String()
}

func renderHeader(s *State, widths []int, numW int) string {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", numW+4))
	hl := s.highlight()
	for i := range s.Cols {
		var label string
		box := byte(' ')
		if s.ColSel[i] {
			box = 'X'
		}
		if s.Mode == ModeColumn {
			label = fmt.Sprintf("%d.[%c]", i+1, box)
		} else {
			label = fmt.Sprintf("[%c]", box)
		}
		shown := label
		if s.Mode == ModeColumn && hl == i {
			shown = invOn + label + invOff
		}
		b.WriteString(shown)
		if i < len(s.Cols)-1 {
			if pad := widths[i] - len(label); pad > 0 {
				b.WriteString(strings.Repeat(" ", pad))
			}
			b.WriteString(" ")
		}
	}
	return b.String()
}

func renderRow(s *State, displayNum, orig, w int, widths []int, numW int) string {
	var numField string
	if s.Mode == ModeColumn {
		numField = strings.Repeat(" ", numW)
	} else {
		numField = strconv.Itoa(displayNum) + "."
		if pad := numW - len(numField); pad > 0 {
			numField += strings.Repeat(" ", pad)
		}
	}
	box := ' '
	if s.RowSel[orig] {
		box = 'X'
	}
	prefix := numField + fmt.Sprintf("[%c] ", box)
	if s.Mode == ModeRow && s.highlight() == displayNum-1 {
		prefix = invOn + prefix + invOff
	}

	var b strings.Builder
	b.WriteString(prefix)
	for i, col := range s.Cols {
		cell := trimCell(s.Lines[orig].Expanded, col.Start, col.End)
		b.WriteString(cell)
		if i < len(s.Cols)-1 {
			if pad := widths[i] - len(cell); pad > 0 {
				b.WriteString(strings.Repeat(" ", pad))
			}
			b.WriteString(" ")
		}
	}
	return cropBytes(b.String(), w)
}

func renderStatus(s *State, w int) string {
	var text string
	if s.Filtering {
		text = fmt.Sprintf("[%s] /%s%s█", s.FilterMode, s.FilterQuery, s.SessionQuery)
	} else {
		modeName := "column"
		if s.Mode == ModeRow {
			modeName = "row"
		}
		sortInfo := ""
		if s.SortDir != SortOff {
			arrow := "^"
			if s.SortDir == SortDesc {
				arrow = "v"
			}
			sortInfo = fmt.Sprintf("sort:%s col %d | ", arrow, s.SortCol+1)
		}
		text = fmt.Sprintf("[%s] %sfilter:%q (%s) | TAB mode  / filter (TAB cycles match mode)  s sort  ENTER output  a all  i invert  >/. commit  </, pop  q quit", modeName, sortInfo, s.FilterQuery, s.FilterMode)
	}
	runes := []rune(text)
	if w < 0 {
		w = 0
	}
	if len(runes) > w {
		runes = runes[:w]
	}
	line := statusBG + statusFG + string(runes)
	if pad := w - len(runes); pad > 0 {
		line += strings.Repeat(" ", pad)
	}
	return line + sgrReset
}

func cropBytes(s string, n int) string {
	if n < 0 {
		n = 0
	}
	if len(s) <= n {
		return s
	}
	return s[:n]
}
