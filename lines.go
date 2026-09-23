package main

import "math"

type Line struct {
	Raw      string
	Expanded string
}

type Column struct {
	Start, End int
}

func NewLines(data []byte) []Line {
	s := string(data)
	if s == "" {
		return nil
	}
	parts := splitLines(s)
	lines := make([]Line, 0, len(parts))
	for _, p := range parts {
		raw := p
		if len(raw) > 0 && raw[len(raw)-1] == '\r' {
			raw = raw[:len(raw)-1]
		}
		lines = append(lines, Line{Raw: raw, Expanded: expandTabs(raw)})
	}
	return lines
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func expandTabs(s string) string {
	if !hasTab(s) {
		return s
	}
	var b []byte
	col := 0
	for i := range len(s) {
		c := s[i]
		if c == '\t' {
			n := 8 - (col % 8)
			for range n {
				b = append(b, ' ')
			}
			col += n
			continue
		}
		b = append(b, c)
		col++
	}
	return string(b)
}

func hasTab(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\t' {
			return true
		}
	}
	return false
}

func isBlank(l Line) bool {
	for i := 0; i < len(l.Expanded); i++ {
		if l.Expanded[i] != ' ' {
			return false
		}
	}
	return true
}

func ComputeColumns(lines []Line) []Column {
	var spacePositions []map[int]bool
	for _, l := range lines {
		if isBlank(l) {
			continue
		}
		pos := make(map[int]bool)
		for i := 0; i < len(l.Expanded); i++ {
			if l.Expanded[i] == ' ' {
				pos[i] = true
			}
		}
		spacePositions = append(spacePositions, pos)
	}
	if len(spacePositions) == 0 {
		return []Column{{0, math.MaxInt}}
	}

	var common []int
	first := true
	for _, pos := range spacePositions {
		if first {
			for p := range pos {
				common = append(common, p)
			}
			first = false
			continue
		}
		var kept []int
		for _, p := range common {
			if pos[p] {
				kept = append(kept, p)
			}
		}
		common = kept
	}

	if len(common) == 0 {
		return []Column{{0, math.MaxInt}}
	}

	sortInts(common)

	gaps := mergeRuns(common)

	cols := []Column{}
	prev := 0
	for _, g := range gaps {
		if g[0] > prev {
			cols = append(cols, Column{prev, g[0]})
		}
		prev = g[1] + 1
	}
	cols = append(cols, Column{prev, math.MaxInt})

	return dropEmptyColumns(lines, cols)
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		v := a[i]
		j := i - 1
		for j >= 0 && a[j] > v {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = v
	}
}

func mergeRuns(sorted []int) [][2]int {
	var runs [][2]int
	start := sorted[0]
	end := sorted[0]
	for _, p := range sorted[1:] {
		if p == end+1 {
			end = p
			continue
		}
		runs = append(runs, [2]int{start, end})
		start = p
		end = p
	}
	runs = append(runs, [2]int{start, end})
	return runs
}

func dropEmptyColumns(lines []Line, cols []Column) []Column {
	out := make([]Column, 0, len(cols))
	for _, c := range cols {
		nonEmpty := false
		for _, l := range lines {
			if trimCell(l.Expanded, c.Start, c.End) != "" {
				nonEmpty = true
				break
			}
		}
		if nonEmpty {
			out = append(out, c)
		}
	}
	return out
}

func trimCell(expanded string, start, end int) string {
	if start > len(expanded) {
		start = len(expanded)
	}
	if end > len(expanded) {
		end = len(expanded)
	}
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	cell := expanded[start:end]
	return trimSpaces(cell)
}

func trimSpaces(s string) string {
	start := 0
	for start < len(s) && s[start] == ' ' {
		start++
	}
	end := len(s)
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}
