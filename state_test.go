package main

import "testing"

func newTestState(data string) *State {
	return NewState(NewLines([]byte(data)))
}

func press(s *State, k KeyKind) Action {
	return s.HandleKey(Key{Kind: k})
}

func runeKey(s *State, r rune) Action {
	return s.HandleKey(Key{Kind: KeyRune, Rune: r})
}

func TestNewState(t *testing.T) {
	s := newTestState("a b c\nd e f\n")
	if s.Mode != ModeColumn || s.ColHigh != 0 || s.RowHigh != 0 {
		t.Fatalf("mode=%v col=%d row=%d", s.Mode, s.ColHigh, s.RowHigh)
	}
	if len(s.RowSel) != 0 || len(s.ColSel) != 0 {
		t.Fatalf("selections not empty")
	}
	if len(s.Cols) != 3 {
		t.Fatalf("cols=%v", s.Cols)
	}
}

func TestTabTogglesMode(t *testing.T) {
	s := newTestState("a b\nc d\n")
	press(s, KeyTab)
	if s.Mode != ModeRow {
		t.Fatalf("want row")
	}
	press(s, KeyTab)
	if s.Mode != ModeColumn {
		t.Fatalf("want column")
	}
}

func TestMovementAndClamping(t *testing.T) {
	s := newTestState("a b c\nd e f\n")
	runeKey(s, 'l')
	if s.highlight() != 1 {
		t.Fatalf("highlight=%d", s.highlight())
	}
	runeKey(s, 'l')
	runeKey(s, 'l')
	if s.highlight() != 2 {
		t.Fatalf("clamp: highlight=%d", s.highlight())
	}
	press(s, KeyRight)
	if s.highlight() != 2 {
		t.Fatalf("key right clamp: %d", s.highlight())
	}
	runeKey(s, 'h')
	if s.highlight() != 1 {
		t.Fatalf("highlight=%d", s.highlight())
	}

	runeKey(s, 'j')
	if s.Mode != ModeRow || s.highlight() != 0 {
		t.Fatalf("switch to row: mode=%v highlight=%d", s.Mode, s.highlight())
	}
	runeKey(s, 'j')
	if s.highlight() != 1 {
		t.Fatalf("highlight=%d", s.highlight())
	}
	runeKey(s, 'j')
	if s.highlight() != 1 {
		t.Fatalf("row clamp: %d", s.highlight())
	}
	runeKey(s, 'h')
	if s.Mode != ModeColumn || s.highlight() != 1 {
		t.Fatalf("row h should restore column highlight: mode=%v highlight=%d", s.Mode, s.highlight())
	}
}

func TestHighlightRememberedAcrossModes(t *testing.T) {
	s := newTestState("r0\nr1\nr2\nr3\nr4\nr5\n")
	press(s, KeyTab)
	for i := 0; i < 5; i++ {
		runeKey(s, 'j')
	}
	if s.RowHigh != 5 {
		t.Fatalf("row highlight=%d", s.RowHigh)
	}
	press(s, KeyTab)
	press(s, KeyTab)
	if s.Mode != ModeRow || s.highlight() != 5 {
		t.Fatalf("tab tab should restore row 5: mode=%v highlight=%d", s.Mode, s.highlight())
	}
}

func TestHighlightSwitchDoesNotMove(t *testing.T) {
	s := newTestState("r0\nr1\nr2\nr3\nr4\nr5\nr6\nr7\n")
	press(s, KeyTab)
	for i := 0; i < 5; i++ {
		runeKey(s, 'j')
	}
	press(s, KeyTab)
	runeKey(s, 'j')
	if s.Mode != ModeRow || s.highlight() != 5 {
		t.Fatalf("switch should keep row 5: mode=%v highlight=%d", s.Mode, s.highlight())
	}
	runeKey(s, 'j')
	if s.highlight() != 6 {
		t.Fatalf("second down should move to 6: %d", s.highlight())
	}
}

func TestSelectionColumnMode(t *testing.T) {
	s := newTestState("a b c\nd e f\n")
	runeKey(s, ' ')
	if !s.ColSel[0] {
		t.Fatalf("space did not toggle col 0")
	}
	runeKey(s, ' ')
	if s.ColSel[0] {
		t.Fatalf("space did not untoggle col 0")
	}
	runeKey(s, '2')
	if !s.ColSel[1] {
		t.Fatalf("digit did not select col 1")
	}
	runeKey(s, '9')
	if len(s.ColSel) != 1 {
		t.Fatalf("out of range digit changed selection")
	}
	runeKey(s, 'a')
	if len(s.ColSel) != 3 {
		t.Fatalf("select all cols: %v", s.ColSel)
	}
	runeKey(s, 'i')
	if len(s.ColSel) != 0 {
		t.Fatalf("invert to empty: %v", s.ColSel)
	}
	runeKey(s, 'i')
	if len(s.ColSel) != 3 {
		t.Fatalf("invert back: %v", s.ColSel)
	}
}

func TestSelectionRowMode(t *testing.T) {
	s := newTestState("a b\nc d\ne f\n")
	press(s, KeyTab)
	runeKey(s, '2')
	if !s.RowSel[1] || len(s.RowSel) != 1 {
		t.Fatalf("digit row: %v", s.RowSel)
	}
	runeKey(s, ' ')
	if !s.RowSel[1] {
		t.Fatalf("space untoggled via highlight?")
	}
	runeKey(s, 'i')
	if len(s.RowSel) != 1 || !s.RowSel[2] {
		t.Fatalf("invert rows: %v", s.RowSel)
	}
}

func TestFilterLifecycle(t *testing.T) {
	s := newTestState("foobar\nfoo\nbar\n")

	runeKey(s, '/')
	if !s.Filtering {
		t.Fatalf("not filtering")
	}
	for _, r := range "foo" {
		s.HandleFilterKey(Key{Kind: KeyRune, Rune: r})
	}
	s.HandleFilterKey(Key{Kind: KeyEnter})
	if s.Filtering || s.FilterQuery != "foo" {
		t.Fatalf("filter not applied: filtering=%v q=%q", s.Filtering, s.FilterQuery)
	}
	if got := len(s.VisibleRows()); got != 2 {
		t.Fatalf("visible=%d", got)
	}

	runeKey(s, '/')
	for _, r := range "bar" {
		s.HandleFilterKey(Key{Kind: KeyRune, Rune: r})
	}
	s.HandleFilterKey(Key{Kind: KeyEnter})
	if s.FilterQuery != "foobar" {
		t.Fatalf("concatenate: q=%q", s.FilterQuery)
	}

	// zero-match query stays in filter mode
	runeKey(s, '/')
	for _, r := range "zz" {
		s.HandleFilterKey(Key{Kind: KeyRune, Rune: r})
	}
	s.HandleFilterKey(Key{Kind: KeyEnter})
	if !s.Filtering {
		t.Fatalf("zero-match should stay filtering")
	}
}

func TestFilterEnterEmptyExits(t *testing.T) {
	s := newTestState("a\nb\n")
	runeKey(s, '/')
	s.HandleFilterKey(Key{Kind: KeyEnter})
	if s.Filtering || s.FilterQuery != "" {
		t.Fatalf("empty enter: filtering=%v q=%q", s.Filtering, s.FilterQuery)
	}
}

func TestFilterEscClears(t *testing.T) {
	s := newTestState("a\nb\n")
	runeKey(s, '/')
	s.HandleFilterKey(Key{Kind: KeyRune, Rune: 'a'})
	s.HandleFilterKey(Key{Kind: KeyEnter})
	if s.FilterQuery != "a" {
		t.Fatalf("setup: %q", s.FilterQuery)
	}
	runeKey(s, '/')
	s.HandleFilterKey(Key{Kind: KeyEsc})
	if s.Filtering || s.FilterQuery != "" || s.SessionQuery != "" {
		t.Fatalf("esc: filtering=%v q=%q session=%q", s.Filtering, s.FilterQuery, s.SessionQuery)
	}
}

func TestFilterCtrlW(t *testing.T) {
	s := newTestState("a\n")
	s.SessionQuery = "foo bar  "
	s.HandleFilterKey(Key{Kind: KeyCtrlW})
	if s.SessionQuery != "foo " {
		t.Fatalf("ctrl-w: %q", s.SessionQuery)
	}
}

func TestFuzzyMatch(t *testing.T) {
	cases := []struct {
		s, q string
		want bool
	}{
		{"foobar", "fb", true},
		{"foobar", "fz", false},
		{"foobar", "", true},
		{"foobar", "foobar", true},
		{"foobar", "foobarbaz", false},
		{"abc", "bc", true},
	}
	for _, c := range cases {
		if got := fuzzyMatch(c.s, c.q); got != c.want {
			t.Errorf("fuzzyMatch(%q, %q) = %v; want %v", c.s, c.q, got, c.want)
		}
	}
}

func TestFilterModeDefaultIsFuzzy(t *testing.T) {
	s := newTestState("Foo Bar\nfoo baz\nquux\n")
	if s.FilterMode != FilterFuzzy {
		t.Fatalf("default filter mode = %v; want fuzzy", s.FilterMode)
	}
	s.FilterQuery = "fb"
	got := s.VisibleRows()
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("fuzzy subsequence rows = %v; want [0 1]", got)
	}
}

func TestFilterModeMatching(t *testing.T) {
	s := newTestState("Foo Bar\nfoo baz\nquux\n")

	s.FilterMode = FilterInsensitive
	s.FilterQuery = "foo"
	if got := s.VisibleRows(); len(got) != 2 {
		t.Fatalf("case-insensitive rows = %v; want 2", got)
	}

	s.FilterMode = FilterSensitive
	s.FilterQuery = "Foo"
	if got := s.VisibleRows(); len(got) != 1 || got[0] != 0 {
		t.Fatalf("case-sensitive rows = %v; want [0]", got)
	}
	s.FilterQuery = "FOO"
	if got := s.VisibleRows(); len(got) != 0 {
		t.Fatalf("case-sensitive FOO rows = %v; want none", got)
	}
}

func TestFilterTabCyclesMode(t *testing.T) {
	s := newTestState("a\n")
	runeKey(s, '/')
	if s.FilterMode != FilterFuzzy {
		t.Fatalf("start = %v", s.FilterMode)
	}
	s.HandleFilterKey(Key{Kind: KeyTab})
	if s.FilterMode != FilterInsensitive {
		t.Fatalf("after 1 tab = %v", s.FilterMode)
	}
	s.HandleFilterKey(Key{Kind: KeyTab})
	if s.FilterMode != FilterSensitive {
		t.Fatalf("after 2 tabs = %v", s.FilterMode)
	}
	s.HandleFilterKey(Key{Kind: KeyTab})
	if s.FilterMode != FilterFuzzy {
		t.Fatalf("after 3 tabs = %v", s.FilterMode)
	}
}

func TestFilterPrunesHiddenRowSel(t *testing.T) {
	s := newTestState("foobar\nfoo\nbar\n")
	press(s, KeyTab)
	runeKey(s, '3')
	if !s.RowSel[2] {
		t.Fatalf("setup row sel: %v", s.RowSel)
	}
	press(s, KeyTab)
	runeKey(s, '/')
	for _, r := range "foo" {
		s.HandleFilterKey(Key{Kind: KeyRune, Rune: r})
	}
	s.HandleFilterKey(Key{Kind: KeyEnter})
	if s.RowSel[2] {
		t.Fatalf("hidden row still selected: %v", s.RowSel)
	}
}

func TestHighlightVisibleRows(t *testing.T) {
	s := newTestState("a b\nc d\ne f\n")
	s.FilterQuery = "c"
	if got := s.VisibleRows(); len(got) != 1 || got[0] != 1 {
		t.Fatalf("visible=%v", got)
	}
	press(s, KeyTab)
	runeKey(s, ' ')
	if !s.RowSel[1] || s.RowSel[0] {
		t.Fatalf("highlight maps to visible position: %v", s.RowSel)
	}
}

func TestSnapshotCommitAndPop(t *testing.T) {
	s := newTestState("a b c\nd e f\n")
	runeKey(s, '2')
	runeKey(s, '>')
	if len(s.Snapshots) != 1 {
		t.Fatalf("snapshot not pushed")
	}
	if len(s.Lines) != 2 || s.Lines[0].Raw != "b" || s.Lines[1].Raw != "e" {
		t.Fatalf("committed lines: %v", s.Lines)
	}
	if len(s.RowSel) != 0 || len(s.ColSel) != 0 || s.Mode != ModeColumn || s.ColHigh != 0 || s.RowHigh != 0 || s.FilterQuery != "" {
		t.Fatalf("commit did not reset: %+v", s)
	}

	runeKey(s, '<')
	if len(s.Snapshots) != 0 {
		t.Fatalf("snapshot not popped")
	}
	if len(s.Lines) != 2 || s.Lines[0].Raw != "a b c" || s.Lines[1].Raw != "d e f" {
		t.Fatalf("restored lines: %v", s.Lines)
	}
	if !s.ColSel[1] {
		t.Fatalf("restored col sel: %v", s.ColSel)
	}
}

func TestSnapshotRestoresFilterMode(t *testing.T) {
	s := newTestState("a b c\nd e f\n")
	s.FilterMode = FilterSensitive
	runeKey(s, '>')
	s.FilterMode = FilterFuzzy
	runeKey(s, '<')
	if s.FilterMode != FilterSensitive {
		t.Fatalf("filter mode not restored: %v", s.FilterMode)
	}
}

func TestPopEmptyStack(t *testing.T) {
	s := newTestState("a b\n")
	runeKey(s, '<')
	if len(s.Snapshots) != 0 {
		t.Fatalf("no-op failed")
	}
}

func TestCommitPopAlternateKeys(t *testing.T) {
	s := newTestState("a b c\nd e f\n")
	runeKey(s, '2')
	runeKey(s, '.')
	if len(s.Snapshots) != 1 || s.Lines[0].Raw != "b" {
		t.Fatalf("'.' should commit: %v", s.Lines)
	}
	runeKey(s, ',')
	if len(s.Snapshots) != 0 || s.Lines[0].Raw != "a b c" {
		t.Fatalf("',' should pop: %v", s.Lines)
	}
}

func assertInts(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %v; want %v", i, got, want)
		}
	}
}

func TestSortCycles(t *testing.T) {
	s := newTestState("b\na\nc\n")
	runeKey(s, 's')
	if s.SortDir != SortAsc || s.SortCol != 0 {
		t.Fatalf("first s: dir=%v col=%d", s.SortDir, s.SortCol)
	}
	assertInts(t, s.VisibleRows(), []int{1, 0, 2})
	runeKey(s, 's')
	if s.SortDir != SortDesc {
		t.Fatalf("second s: dir=%v", s.SortDir)
	}
	assertInts(t, s.VisibleRows(), []int{2, 0, 1})
	runeKey(s, 's')
	if s.SortDir != SortOff || s.SortCol != -1 {
		t.Fatalf("third s: dir=%v col=%d", s.SortDir, s.SortCol)
	}
	assertInts(t, s.VisibleRows(), []int{0, 1, 2})
}

func TestSortOnlyInColumnMode(t *testing.T) {
	s := newTestState("b\na\n")
	press(s, KeyTab)
	runeKey(s, 's')
	if s.SortDir != SortOff || s.SortCol != -1 {
		t.Fatalf("row mode sort should be no-op: dir=%v col=%d", s.SortDir, s.SortCol)
	}
	if s.Mode != ModeRow {
		t.Fatalf("row mode changed: %v", s.Mode)
	}
}

func TestSortNumeric(t *testing.T) {
	s := newTestState("10\n9\n2\n")
	runeKey(s, 's')
	assertInts(t, s.VisibleRows(), []int{2, 1, 0})
}

func TestSortCaseInsensitiveString(t *testing.T) {
	s := newTestState("banana\nApple\ncherry\n")
	runeKey(s, 's')
	assertInts(t, s.VisibleRows(), []int{1, 0, 2})
}

func TestSortStableTies(t *testing.T) {
	s := newTestState("b 1\na 1\nc 1\n")
	runeKey(s, 'l')
	runeKey(s, 's')
	assertInts(t, s.VisibleRows(), []int{0, 1, 2})
	runeKey(s, 's')
	if s.SortDir != SortDesc {
		t.Fatalf("dir=%v", s.SortDir)
	}
	assertInts(t, s.VisibleRows(), []int{0, 1, 2})
}

func TestSortSwitchColumnResetsAsc(t *testing.T) {
	s := newTestState("b 2\na 1\nc 3\n")
	runeKey(s, 's')
	if s.SortCol != 0 || s.SortDir != SortAsc {
		t.Fatalf("setup: col=%d dir=%v", s.SortCol, s.SortDir)
	}
	runeKey(s, 'l')
	runeKey(s, 's')
	if s.SortCol != 1 || s.SortDir != SortAsc {
		t.Fatalf("switch: col=%d dir=%v", s.SortCol, s.SortDir)
	}
	assertInts(t, s.VisibleRows(), []int{1, 0, 2})
}

func TestSortWithFilter(t *testing.T) {
	s := newTestState("foo 3\nbar 1\nfoo 1\nfoo 2\n")
	s.FilterQuery = "foo"
	runeKey(s, 'l')
	runeKey(s, 's')
	assertInts(t, s.VisibleRows(), []int{2, 3, 0})
}

func TestSortResetsOnCommitRestoresOnPop(t *testing.T) {
	s := newTestState("b\na\n")
	runeKey(s, 's')
	runeKey(s, '>')
	if s.SortDir != SortOff || s.SortCol != -1 {
		t.Fatalf("commit did not reset sort: dir=%v col=%d", s.SortDir, s.SortCol)
	}
	runeKey(s, '<')
	if s.SortDir != SortAsc || s.SortCol != 0 {
		t.Fatalf("pop did not restore sort: dir=%v col=%d", s.SortDir, s.SortCol)
	}
}

func TestQuitAndAccept(t *testing.T) {
	s := newTestState("a\n")
	if press(s, KeyEsc) != ActQuit {
		t.Fatalf("esc should quit")
	}
	if press(s, KeyCtrlC) != ActQuit {
		t.Fatalf("ctrl-c should quit")
	}
	if runeKey(s, 'q') != ActQuit {
		t.Fatalf("q should quit")
	}
	if press(s, KeyEnter) != ActAccept {
		t.Fatalf("enter should accept")
	}
}
