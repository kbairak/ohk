package main

import "strings"

type Mode int

const (
	ModeColumn Mode = iota
	ModeRow
)

type FilterMode int

const (
	FilterFuzzy FilterMode = iota
	FilterInsensitive
	FilterSensitive
	filterModeCount
)

func (m FilterMode) String() string {
	switch m {
	case FilterInsensitive:
		return "case-insensitive"
	case FilterSensitive:
		return "case-sensitive"
	default:
		return "fuzzy"
	}
}

func matchLine(l Line, q string, mode FilterMode) bool {
	switch mode {
	case FilterSensitive:
		return strings.Contains(l.Raw, q)
	case FilterInsensitive:
		return strings.Contains(strings.ToLower(l.Raw), strings.ToLower(q))
	default:
		return fuzzyMatch(strings.ToLower(l.Raw), strings.ToLower(q))
	}
}

func fuzzyMatch(s, q string) bool {
	if q == "" {
		return true
	}
	qi := 0
	for i := 0; i < len(s) && qi < len(q); i++ {
		if s[i] == q[qi] {
			qi++
		}
	}
	return qi == len(q)
}

type Action int

const (
	ActNone Action = iota
	ActQuit
	ActAccept
)

type StateSnapshot struct {
	Lines       []Line
	Cols        []Column
	RowSel      map[int]bool
	ColSel      map[int]bool
	Mode        Mode
	ColHigh     int
	RowHigh     int
	FilterQuery string
	FilterMode  FilterMode
}

type State struct {
	Lines        []Line
	Cols         []Column
	RowSel       map[int]bool
	ColSel       map[int]bool
	Mode         Mode
	ColHigh      int
	RowHigh      int
	Filtering    bool
	SessionQuery string
	FilterQuery  string
	FilterMode   FilterMode
	Snapshots    []StateSnapshot
}

func NewState(lines []Line) *State {
	return &State{
		Lines:  lines,
		Cols:   ComputeColumns(lines),
		RowSel: map[int]bool{},
		ColSel: map[int]bool{},
		Mode:   ModeColumn,
	}
}

func (s *State) VisibleRows() []int {
	out := make([]int, 0, len(s.Lines))
	for i, l := range s.Lines {
		if s.FilterQuery == "" || matchLine(l, s.FilterQuery, s.FilterMode) {
			out = append(out, i)
		}
	}
	return out
}

func (s *State) HandleKey(k Key) Action {
	switch k.Kind {
	case KeyTab:
		if s.Mode == ModeColumn {
			s.Mode = ModeRow
		} else {
			s.Mode = ModeColumn
		}
		return ActNone
	case KeyLeft:
		s.moveLeft()
		return ActNone
	case KeyRight:
		s.moveRight()
		return ActNone
	case KeyUp:
		s.moveUp()
		return ActNone
	case KeyDown:
		s.moveDown()
		return ActNone
	case KeyEnter:
		return ActAccept
	case KeyEsc, KeyCtrlC:
		return ActQuit
	case KeyRune:
		if k.Rune == 'q' {
			return ActQuit
		}
		switch k.Rune {
		case 'h':
			s.moveLeft()
		case 'l':
			s.moveRight()
		case 'k':
			s.moveUp()
		case 'j':
			s.moveDown()
		case ' ':
			s.toggleHighlight()
		case 'a':
			s.selectAll()
		case 'i':
			s.invert()
		case '/':
			s.Filtering = true
			s.SessionQuery = ""
		case '>', '.':
			s.commit()
		case '<', ',':
			s.pop()
		default:
			if k.Rune >= '1' && k.Rune <= '9' {
				s.toggleDigit(int(k.Rune - '1'))
			}
		}
	}
	return ActNone
}

func (s *State) HandleFilterKey(k Key) {
	switch k.Kind {
	case KeyRune:
		s.SessionQuery += string(k.Rune)
	case KeyBackspace:
		if len(s.SessionQuery) > 0 {
			s.SessionQuery = s.SessionQuery[:len(s.SessionQuery)-1]
		}
	case KeyCtrlW:
		s.SessionQuery = dropLastWord(s.SessionQuery)
	case KeyEnter:
		candidate := s.FilterQuery + s.SessionQuery
		if candidate == "" {
			s.Filtering = false
			return
		}
		if s.wouldLeaveVisible(candidate) {
			s.FilterQuery = candidate
			s.SessionQuery = ""
			s.Filtering = false
			s.pruneRowSel()
		}
	case KeyEsc:
		s.Filtering = false
		s.SessionQuery = ""
		s.FilterQuery = ""
	case KeyTab:
		s.FilterMode = (s.FilterMode + 1) % filterModeCount
	default:
	}
}

func dropLastWord(s string) string {
	i := len(s)
	for i > 0 && s[i-1] == ' ' {
		i--
	}
	for i > 0 && s[i-1] != ' ' {
		i--
	}
	return s[:i]
}

func (s *State) wouldLeaveVisible(q string) bool {
	for _, l := range s.Lines {
		if matchLine(l, q, s.FilterMode) {
			return true
		}
	}
	return false
}

func (s *State) pruneRowSel() {
	visible := map[int]bool{}
	for _, i := range s.VisibleRows() {
		visible[i] = true
	}
	for i := range s.RowSel {
		if !visible[i] {
			delete(s.RowSel, i)
		}
	}
}

func (s *State) highlight() int {
	if s.Mode == ModeRow {
		n := len(s.VisibleRows())
		if n == 0 {
			return 0
		}
		if s.RowHigh >= n {
			return n - 1
		}
		if s.RowHigh < 0 {
			return 0
		}
		return s.RowHigh
	}
	n := len(s.Cols)
	if n == 0 {
		return 0
	}
	if s.ColHigh >= n {
		return n - 1
	}
	if s.ColHigh < 0 {
		return 0
	}
	return s.ColHigh
}

func (s *State) moveLeft() {
	if s.Mode == ModeRow {
		s.Mode = ModeColumn
		return
	}
	if h := s.highlight(); h > 0 {
		s.ColHigh = h - 1
	}
}

func (s *State) moveRight() {
	if s.Mode == ModeRow {
		s.Mode = ModeColumn
		return
	}
	if h := s.highlight(); h < len(s.Cols)-1 {
		s.ColHigh = h + 1
	}
}

func (s *State) moveUp() {
	if s.Mode == ModeColumn {
		s.Mode = ModeRow
		return
	}
	if h := s.highlight(); h > 0 {
		s.RowHigh = h - 1
	}
}

func (s *State) moveDown() {
	if s.Mode == ModeColumn {
		s.Mode = ModeRow
		return
	}
	if h := s.highlight(); h < len(s.VisibleRows())-1 {
		s.RowHigh = h + 1
	}
}

func (s *State) toggleHighlight() {
	if s.Mode == ModeColumn {
		toggleSel(s.ColSel, s.highlight())
		return
	}
	vr := s.VisibleRows()
	if h := s.highlight(); h >= 0 && h < len(vr) {
		toggleSel(s.RowSel, vr[h])
	}
}

func toggleSel(m map[int]bool, i int) {
	if m[i] {
		delete(m, i)
	} else {
		m[i] = true
	}
}

func (s *State) selectAll() {
	if s.Mode == ModeColumn {
		for i := range s.Cols {
			s.ColSel[i] = true
		}
		return
	}
	for _, i := range s.VisibleRows() {
		s.RowSel[i] = true
	}
}

func (s *State) invert() {
	if s.Mode == ModeColumn {
		for i := range s.Cols {
			toggleSel(s.ColSel, i)
		}
		return
	}
	for _, i := range s.VisibleRows() {
		toggleSel(s.RowSel, i)
	}
}

func (s *State) toggleDigit(d int) {
	if s.Mode == ModeColumn {
		if d < 0 || d >= len(s.Cols) {
			return
		}
		toggleSel(s.ColSel, d)
		return
	}
	vr := s.VisibleRows()
	if d < 0 || d >= len(vr) {
		return
	}
	toggleSel(s.RowSel, vr[d])
}

func (s *State) commit() {
	out := Output(s)
	snap := StateSnapshot{
		Lines:       append([]Line(nil), s.Lines...),
		Cols:        append([]Column(nil), s.Cols...),
		RowSel:      copyMap(s.RowSel),
		ColSel:      copyMap(s.ColSel),
		Mode:        s.Mode,
		ColHigh:     s.ColHigh,
		RowHigh:     s.RowHigh,
		FilterQuery: s.FilterQuery,
		FilterMode:  s.FilterMode,
	}
	s.Snapshots = append(s.Snapshots, snap)

	data := strings.Join(out, "\n")
	if len(out) > 0 {
		data += "\n"
	}
	s.Lines = NewLines([]byte(data))
	s.Cols = ComputeColumns(s.Lines)
	s.RowSel = map[int]bool{}
	s.ColSel = map[int]bool{}
	s.Mode = ModeColumn
	s.ColHigh = 0
	s.RowHigh = 0
	s.Filtering = false
	s.SessionQuery = ""
	s.FilterQuery = ""
}

func (s *State) pop() {
	if len(s.Snapshots) == 0 {
		return
	}
	snap := s.Snapshots[len(s.Snapshots)-1]
	s.Snapshots = s.Snapshots[:len(s.Snapshots)-1]
	s.Lines = snap.Lines
	s.Cols = snap.Cols
	s.RowSel = snap.RowSel
	s.ColSel = snap.ColSel
	s.Mode = snap.Mode
	s.ColHigh = snap.ColHigh
	s.RowHigh = snap.RowHigh
	s.FilterQuery = snap.FilterQuery
	s.FilterMode = snap.FilterMode
	s.Filtering = false
	s.SessionQuery = ""
}

func copyMap(m map[int]bool) map[int]bool {
	out := make(map[int]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
