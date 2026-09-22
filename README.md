# ohk — interactive grep + awk

`ohk` sits in the middle of a pipe and lets you interactively **filter rows**,
**pick columns**, and **project** the result — then hands the final selection to
the next command in the pipe.

```sh
some-command | ohk | some-other-command
```

You know the drill:

1. `some-command` — output is too large
2. `some-command | grep pattenr` — mistyped the pattern
3. `some-command | grep pattern` — fine, but too many columns
4. `some-command | grep pattern | awk '{print $3, $4}'` — counted columns wrong
5. `some-command | grep pattern | awk '{print $4, $5}'` — finally

Each retry means re-running the whole upstream command, which is painful if it
is slow. `ohk` collapses steps 2-5 into one interactive pass:

```sh
some-command | ohk
```

Today's answer for "what is that column again?" is often `fzf`, but `fzf` knows
nothing about columns. `ohk` does: it splits aligned input into a column grid,
so you can filter _and_ cut in one go.

## Install

Requires Go and a Unix-like OS (uses `/dev/tty`).

```sh
go install github.com/kbairak/ohk@latest
```

This builds the binary into `$GOBIN`, or `$GOPATH/bin` (`$HOME/go/bin` by
default) if `GOBIN` is unset. Make sure that directory is on your `PATH`:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

Or build from a local checkout:

```sh
git clone https://github.com/kbairak/ohk
cd ohk
go build -o ohk .
install -m 0755 ohk /usr/local/bin/ohk
```

Or from the repo directory, `make build` then put `./ohk` on your `PATH`.

## What it looks like

Column mode (the default) — the header shows one numbered checkbox per column:

```
    1.[ ]    2.[ ]    3.[ ]    4.[ ]
[ ] row1col1 row1col2 row1col3 row1col4
[ ] row2col1 row2col2 row2col3 row2col4
[ ] row3col1 row3col2 row3col3 row3col4
```

Press `/row3` to filter rows down to just the match:

```
    1.[ ]    2.[ ]    3.[ ]    4.[ ]
[ ] row3col1 row3col2 row3col3 row3col4
```

Press `ENTER` and only the matching row is printed to stdout.

Or press `3` to select column 3 (no filter):

```
    1.[ ]    2.[ ]    3.[X]    4.[ ]
[ ] row1col1 row1col2 row1col3 row1col4
[ ] row2col1 row2col2 row2col3 row2col4
[ ] row3col1 row3col2 row3col3 row3col4
```

Press `ENTER` and only column 3 is printed.

Press `TAB` to switch to **row mode** — the checkboxes move to the row
prefixes, which is also how you can tell which mode you are in:

```
      [ ]      [ ]      [ ]      [ ]
1.[ ] row1col1 row1col2 row1col3 row1col4
2.[ ] row2col1 row2col2 row2col3 row2col4
3.[ ] row3col1 row3col2 row3col3 row3col4
```

## Keyboard shortcuts

| Key                    | Action                                                             |
| ---------------------- | ------------------------------------------------------------------ |
| `TAB`                  | toggle row / column mode                                           |
| `1`–`9`                | select/deselect the numbered row/column                            |
| `h` / `l`, `←` / `→`   | in row mode: switch to column mode; in column mode: move highlight |
| `j` / `k`, `↓` / `↑`   | in column mode: switch to row mode; in row mode: move highlight    |
| `SPACE`                | select/deselect the highlighted row/column                         |
| `a`                    | select all (respects the current mode)                             |
| `i`                    | invert selection (respects the current mode)                       |
| `/`                    | enter filter mode                                                  |
| `>` or `.`             | commit output as new input, push a snapshot                        |
| `<` or `,`             | pop the latest snapshot                                            |
| `ENTER`                | finalize and print the result to stdout                            |
| `q` / `ESC` / `CTRL-C` | quit without printing anything (exit 1)                            |

In filter mode:

| Key                  | Action                                               |
| -------------------- | ---------------------------------------------------- |
| printable characters | append to the query                                  |
| `BACKSPACE`          | delete the last character                            |
| `CTRL-W`             | delete the last word                                 |
| `ENTER`              | apply the filter (no-op if it would hide everything) |
| `ESC`                | clear the filter and leave filter mode               |

> The highlight and mode persist across `TAB` switches, so you can bounce back
> and forth without losing your place. Columns beyond the first nine are reached
> with the arrow keys.

## Use cases

**Stop containers without pasting IDs**

```sh
docker ps | ohk | xargs docker stop
```

**Find and kill a process**

```sh
ps aux | ohk | xargs kill
```

**Pick fields out of a formatted listing**

```sh
mount | ohk                      # keep only the mountpoint column
ls -l | ohk                      # pick owner, size, name from ls output
lsof -i | ohk                    # find the pid holding a port
kubectl get pods -A | ohk        # keep namespace + name + status columns
```

**Trim noisy logs**

```sh
journalctl -u my-service | ohk   # filter to the relevant lines, keep only the message
```

**Pick a commit / branch / tag**

```sh
git log --oneline --all | ohk
```

**Inspect command output you rarely look at**

```sh
env | ohk
pip list | ohk
brew outdated | ohk
```

The pattern is always the same: a slow or unwieldy command on the left, a
column-aware interactive selection in the middle, a small pipeline on the right.

## How it works

- `ohk` reads **all of stdin to EOF**, then opens the TUI on `/dev/tty`
  (`PoC`: no streaming; see limitations).
- The **UI and keyboard** come from `/dev/tty`. **stdout is reserved for the
  final result**, so `ohk` can sit in the middle of a pipeline. Nothing the UI
  draws ever reaches stdout.
- If stdin is itself a terminal (nothing piped), `ohk` prints an error and exits.

### Column detection

Columns are the runs of characters between whitespace positions that **every
non-blank row shares**:

1. Expand tabs to 8-column tab stops.
2. For each row, collect the byte positions of spaces.
3. Intersect those sets across all non-blank rows.
4. Adjacent positions collapse into one gap; the cells between gaps are the columns.

Blank rows are skipped during detection but still displayed and selectable.
Leading/trailing spaces are trimmed per cell. Input is assumed to be **aligned**;
ragged input degenerates to fewer columns (or one), which just makes `ohk` a less
capable `fzf`.

### Output rules

Row and column selections are independent and combine into a cartesian
projection:

- **Row selection only** — the raw bytes of the selected rows, in input order.
- **Column selection only** — for each row, the selected cells trimmed, padded to
  the widest cell across the _output_ rows, and joined with a single space
  (`column -t` style).
- **Neither selected** — all rows, raw bytes.
- **Filter active, nothing selected** — the matching rows, raw bytes (a live grep).
- Output order is always input order. Empty selection means "all".

### Filter mode

- `a-z 0-9` and other printable symbols fill the query.
- `ENTER` applies the filter — but only if it would leave at least one row.
- The query is **remembered between sessions**: `/foo<ENTER>` then `/bar<ENTER>`
  filters by `foobar`.
- `ESC` clears the filter entirely.
- Rows hidden by a filter lose their selection.
- Exact (case-sensitive substring) matching in the PoC; other modes are deferred.

### Snapshots

`>` commits the current output as the new input and pushes a snapshot; `<` pops
back to the previous input and selection state.

Narrow repeatedly:

```
kubernetes pods ──> (>) pick bad namespace ──> (>) pick the line
```

Regret a step? Press `<`. The stack is unlimited, and a snapshot restores the
filter, selections, highlight, and mode too.

## Wiring `ohke` (export a result into a shell variable)

A child process can never set a variable in your shell, so `ohk` cannot export
anything itself. Instead, `ohk` saves its result to a per-shell file, and a tiny
shell function reads that file back.

- `ohk` only writes when the environment variable `OHK_SESSION` is set. A thin
  `ohk` wrapper sets it to the shell's PID (`$$`) for that invocation; direct
  `command ohk` runs save nothing.
- The result is written to
  `${XDG_STATE_HOME:-$HOME/.local/state}/ohk/<shell-pid>.stdout`, created `0600`
  inside a `0700` directory. `ohk` refuses to write if the directory is not owned
  by you or is group/world-accessible, and writes via an exclusive temp file that
  is renamed into place.
- On abort (`q`/`ESC`) `ohk` removes the session file, so `ohke` can never export
  a stale result — it errors instead.

Then:

```sh
ll | ohk          # interact, press ENTER
ohke name         # exports `name` from the last result, then deletes the file
echo "$name"
```

`ohke` must run in the **same shell instance** that ran `ohk` (same `$$`); it
won't work from a child script or subshell.

Add the snippet for your shell to your rc file.

### zsh (`~/.zshrc`)

```sh
ohk() { OHK_SESSION=$$ command ohk "$@"; }

ohke() {
  local _ohk_name=${1:?usage: ohke VARNAME}
  local _ohk_dir=${XDG_STATE_HOME:-$HOME/.local/state}/ohk
  local _ohk_file=$_ohk_dir/$$.stdout
  if [ ! -f "$_ohk_file" ]; then
    printf '%s\n' "ohke: no ohk result for this shell (run ohk first)" >&2
    return 1
  fi
  local _ohk_val
  _ohk_val=$(command cat -- "$_ohk_file")
  export "$_ohk_name=$_ohk_val"
  command rm -f -- "$_ohk_file"
}
```

### bash (`~/.bashrc`)

```bash
ohk() { OHK_SESSION=$$ command ohk "$@"; }

ohke() {
  local _ohk_name=${1:?usage: ohke VARNAME}
  local _ohk_dir=${XDG_STATE_HOME:-$HOME/.local/state}/ohk
  local _ohk_file=$_ohk_dir/$$.stdout
  if [ ! -f "$_ohk_file" ]; then
    printf '%s\n' "ohke: no ohk result for this shell (run ohk first)" >&2
    return 1
  fi
  local _ohk_val
  _ohk_val=$(command cat -- "$_ohk_file")
  export "$_ohk_name=$_ohk_val"
  command rm -f -- "$_ohk_file"
}
```

### POSIX sh (`~/.profile`)

`local` is not portable, so this variant uses prefixed names instead.

```sh
ohk() { OHK_SESSION=$$ command ohk "$@"; }

ohke() {
  _ohk_name=${1:?usage: ohke VARNAME}
  _ohk_dir=${XDG_STATE_HOME:-$HOME/.local/state}/ohk
  _ohk_file=$_ohk_dir/$$.stdout
  if [ ! -f "$_ohk_file" ]; then
    printf '%s\n' "ohke: no ohk result for this shell (run ohk first)" >&2
    return 1
  fi
  _ohk_val=$(command cat -- "$_ohk_file")
  export "$_ohk_name=$_ohk_val"
  command rm -f -- "$_ohk_file"
}
```

> The repo also ships `ohk.sh` with the same functions if you prefer
> `source ohk.sh`; the snippets above are equivalent and let you keep your rc
> self-contained.

## Limitations (PoC)

- Reads all of stdin **before** showing the UI — not for infinite streams.
- Filtering is exact-match only (case-insensitive / fuzzy / regex are deferred).
- ASCII byte offsets only; no rune/wide-character awareness.
- No SIGWINCH handling (the size is re-read every frame, so resize mostly works).
- Lines wider than the terminal are cropped.

## Roadmap

- **Streaming input.** Render rows as they arrive instead of waiting for EOF, and
  let `ENTER` stop the upstream command. This is the `docker ps | ohk | docker stop`
  flow taken to its logical end.
- **Filter modes.** Cycle `TAB` through exact → case-insensitive → fuzzy → regex:
  - _case-insensitive_: substring match ignoring case.
  - _fuzzy_: query matches as a subsequence (characters in order, gaps allowed).
  - _regex_: RE2 syntax (Go's regexp).
- **Resize handling** via `SIGWINCH` instead of polling the size each frame.
- **Rune- and wide-character aware columns** (currently byte offsets, ASCII only).

## Development

```sh
make fmt     # go fmt ./...
make lint    # gofmt check + go vet
make build   # go build -o ohk .
make clean   # rm -f ohk
go test .    # unit tests for the pure logic
```

