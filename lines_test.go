package main

import (
	"math"
	"testing"
)

func TestExpandTabs(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"a\tb", "a       b"},
		{"1234567\tX", "1234567 X"},
		{"12345678\tX", "12345678        X"},
		{"\t\tX", "                X"},
		{"no tabs", "no tabs"},
		{"", ""},
	}
	for _, c := range cases {
		if got := expandTabs(c.in); got != c.want {
			t.Errorf("expandTabs(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestNewLines(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"trailing blank", "a\nb\n\n", []string{"a", "b", ""}},
		{"crlf", "a\r\n", []string{"a"}},
		{"empty", "", nil},
		{"no trailing newline", "abc", []string{"abc"}},
		{"simple", "a\nb\n", []string{"a", "b"}},
	}
	for _, c := range cases {
		got := NewLines([]byte(c.in))
		if len(got) != len(c.want) {
			t.Fatalf("%s: got %d lines, want %d", c.name, len(got), len(c.want))
		}
		for i := range c.want {
			if got[i].Raw != c.want[i] {
				t.Errorf("%s: line %d = %q; want %q", c.name, i, got[i].Raw, c.want[i])
			}
		}
	}
}

func spaceLine(positions []int, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'x'
	}
	for _, p := range positions {
		b[p] = ' '
	}
	return string(b)
}

func TestComputeColumnsDesignExample(t *testing.T) {
	pos := []int{5, 8, 9, 10, 15, 16, 20}
	l := spaceLine(pos, 24)
	got := ComputeColumns([]Line{{Raw: l, Expanded: l}, {Raw: l, Expanded: l}})
	want := []Column{{0, 5}, {6, 8}, {11, 15}, {17, 20}, {21, math.MaxInt}}
	assertCols(t, got, want)
}

func TestComputeColumnsAdjacentMerge(t *testing.T) {
	l := spaceLine([]int{5, 6}, 10)
	got := ComputeColumns([]Line{{Raw: l, Expanded: l}})
	want := []Column{{0, 5}, {7, math.MaxInt}}
	assertCols(t, got, want)
}

func TestComputeColumnsNoCommonSpaces(t *testing.T) {
	a := spaceLine([]int{1}, 6)
	b := spaceLine([]int{2}, 6)
	got := ComputeColumns([]Line{{Raw: a, Expanded: a}, {Raw: b, Expanded: b}})
	assertCols(t, got, []Column{{0, math.MaxInt}})
}

func TestComputeColumnsBlankRowsSkipped(t *testing.T) {
	l := spaceLine([]int{3}, 8)
	got := ComputeColumns([]Line{
		{Raw: l, Expanded: l},
		{Raw: "", Expanded: ""},
		{Raw: "    ", Expanded: "    "},
		{Raw: l, Expanded: l},
	})
	want := []Column{{0, 3}, {4, math.MaxInt}}
	assertCols(t, got, want)
}

func TestComputeColumnsLeadingSpace(t *testing.T) {
	l := spaceLine([]int{0, 2}, 6)
	got := ComputeColumns([]Line{{Raw: l, Expanded: l}, {Raw: l, Expanded: l}})
	want := []Column{{1, 2}, {3, math.MaxInt}}
	assertCols(t, got, want)
}

func TestComputeColumnsZeroNonBlank(t *testing.T) {
	got := ComputeColumns([]Line{{Raw: "", Expanded: ""}})
	assertCols(t, got, []Column{{0, math.MaxInt}})
}

func assertCols(t *testing.T, got, want []Column) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("col %d: got %v; want %v", i, got[i], want[i])
		}
	}
}
