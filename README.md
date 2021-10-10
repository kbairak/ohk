# ohk, an interactive replacement to tools like grep and awk

[![asciicast](https://asciinema.org/a/Y0pXKP6UCryFR0YynA0EAT2bn.svg)](https://asciinema.org/a/Y0pXKP6UCryFR0YynA0EAT2bn)

## Features:

- Normal / fuzzy / regex search for lines
- Case sensitive / insensitive search
- Keyboard shortcuts

## Installation

```sh
→ pip install ohk
```

or

```sh
→ git clone https://github.com/kbairak/ohk
→ cd ohk
→ pip install -e .
```

_Add `sudo` or `--user` to your `pip install` command depending on your setup_

## Usage

```sh
→ [previous command | ] ohk [-f / -r / -i] [ | next command]
```

The `previous command`'s output will be piped into ohk which will start an
interactive session to help you filter its lines and columns. Once you press
enter inside the interactive session, the filtered output will piped into the
`next command`.

If the `previous command` is missing (ie if `ohk`'s standard input is the
keyboard), you will be asked to provide a command before the interactive
session starts:

```sh
➜  ohk | xargs docker stop
Enter command:
```

If the `next command` is missing (ie if `ohk`'s standard output is the terminal
screen), you will be asked to provide one after you make your selection.

You can even use pipes in the output command and even re-invoke `ohk`:

```sh
➜  docker ps -a | ohk
Tips:
  - use 'cat' (or leave empty) to print to terminal
  - use 'xargs' to pass output as argument
  - use '{}' placeholder to pass output as argument
  - use 'ohk' to re-run ohk on the results
  - append ' | ohk' to run ohk on the command's output
Pipe output to: sort | uniq | ohk
```

_Note: You should not use `ohk` twice in the chain of processes because they
will launch in parallel and both try to take over the screen and keyboard_

### Keyboard shortcuts

|                               |                                          |
|-------------------------------|------------------------------------------|
| Esc                           | Quit ohk                                 |
| Enter                         | Finalize selection                       |
| Alt-E                         | Change search mode                       |
| Alt-I                         | Change case sensitivity                  |
| ←↓↑→ / Alt-hjkl / (shift) tab | Focus rows/columns                       |
| Space                         | Select row/column                        |
| Alt-123456789                 | Select numbered column                   |
| Left click                    | Select column                            |
| Alt-A/C                       | Select all/none rows/columns             |
| Alt-R                         | Rerun ohk with currently selected output |

### Command line options

```
usage: ohk [-h] [-f] [-r] [-i]

optional arguments:
  -h, --help            show this help message and exit
  -f, --fuzzy
  -r, --regex
  -i, --case-insensitive
```

## TODOs

- [ ] Tests
- [ ] Use Ctrl- shortcuts
- [ ] Set output as environment variable on the outer shell (will probably need
  the user to set an alias)
- [ ] Scroll
- [ ] Search for column title
- [ ] Decorate
- [ ] Organize code better
