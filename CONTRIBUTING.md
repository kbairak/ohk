# Contributing

## Layout

Flat `package main` at the repo root — no compiler-enforced package boundaries:

| File | Responsibility |
| --- | --- |
| `lines.go` | input splitting, tab expansion, column detection (pure) |
| `keys.go` | byte → key decoding (pure) |
| `state.go` | modes, selections, filter, snapshots (pure) |
| `output.go` | the single output projection function (pure) |
| `render.go` | ANSI frame string building (pure string in, string out) |
| `term.go` | `/dev/tty` open, raw mode, alt screen (syscalls) |
| `save.go` | session result file for `ohke`, with permission checks |
| `main.go` | wiring: stdin, tty, loop |

## Invariants

These are easy to break and hard to notice. Keep them.

- **stdout carries only the final result.** It may be piped onward. All UI drawing
  and key reading go through `/dev/tty`. Never print UI bytes to stdout.
- **Purity by convention.** `lines.go`, `keys.go`, `state.go`, `output.go`,
  `render.go` must not import `golang.org/x/term` and must not touch `/dev/tty`.
  That is what makes them unit-testable without a terminal.
- **Restore the terminal before writing the result.** Leave alt-screen + show the
  cursor + `term.Restore()` must happen *before* the result reaches stdout, or it
  is written into the alt-screen buffer and vanishes.
- **`Raw` vs `Expanded`.** Column offsets, cells, and row rendering use `Expanded`.
  Filter matching and raw output use `Raw` (as-read, tabs intact). Mixing them up
  breaks both filters and output bytes.
- **Highlight and digit keys address *visible* rows.** `RowSel` is keyed by
  original row index; `Highlight`, `1`–`9`, and `SPACE` in row mode address the
  Nth *visible* row and map back through `VisibleRows()`. Forgetting the mapping
  is the most likely bug.
- **Selection maps delete on deselect.** A false-valued key would make
  `len(map) == 0` checks (and iteration) treat it as selected.
- **`Column.End` is `math.MaxInt` for the last column.** Always clamp with
  `min(End, len(Expanded))` and `max(Start, ...)` before slicing.
- **Escape ambiguity is accepted.** A lone `ESC` is treated as quit; decode only
  within one read buffer, never block waiting for sequence continuation bytes.
- **Raw mode has no signals.** `CTRL-C` arrives as byte `0x03` and is handled as
  quit explicitly.

## Testing

```sh
make lint    # gofmt check + go vet
go test .    # pure unit tests, no terminal or I/O
make build   # go build -o ohk .
```

All automated tests are pure. The TUI and pipeline behavior cannot be automated
without a pty harness, so verify changes manually.

## Manual smoke test

Run in a real terminal:

- [ ] `./ohk` with a terminal stdin prints an error to stderr and exits `1`.
- [ ] `printf 'a b c\nd e f\n' | ./ohk` renders; stdin is fully consumed first.
- [ ] Press `2` → header shows `2.[X]`; `ENTER` prints `b\ne`.
- [ ] Press `/`, type `d`, `ENTER` → only row 2 visible; `ENTER` prints `d e f`.
- [ ] `2` then `/d` `ENTER` then `ENTER` prints `e`.
- [ ] `2` `ENTER`, then `>` (input becomes `b\ne`), then `1` `ENTER` prints `b`.
- [ ] After `>`, `<` restores the previous input *and* selection state.
- [ ] `TAB` switches modes without the content shifting; highlight is remembered.
- [ ] `q` exits with no stdout and exit code `1`; the terminal is fully restored.
- [ ] `printf 'a b\n' | ./ohk | cat` — UI on the tty, only `a b` reaches `cat`.
- [ ] Input taller/wider than the terminal is cropped without crashing.
- [ ] With `OHK_SESSION` set, an accepted result lands in
  `${XDG_STATE_HOME:-$HOME/.local/state}/ohk/<session>.stdout` (`0600`), and an
  abort removes it. `ohke` exports and removes it.