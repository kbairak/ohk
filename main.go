package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func main() {
	if handled, code := handleArgs(os.Args[1:], os.Stdout, os.Stderr); handled {
		os.Exit(code)
	}

	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintln(os.Stderr, "ohk: stdin is a terminal, pipe something in")
		os.Exit(1)
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ohk:", err)
		os.Exit(1)
	}

	tty, err := OpenTTY()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ohk: no controlling terminal:", err)
		os.Exit(1)
	}
	defer tty.Close()

	cleanup := func() {
		tty.ExitAltScreen()
		tty.Restore()
	}

	st := NewState(NewLines(data))
	buf := make([]byte, 16)

	quit := false
	accepted := false
	var out []string
	for {
		w, h, _ := tty.Size()
		if err := tty.WriteString(Render(st, w, h)); err != nil {
			quit = true
			break
		}
		n, err := tty.Read(buf)
		if err != nil || n == 0 {
			quit = true
			break
		}
		key, _ := DecodeKey(buf[:n])
		if st.Filtering {
			st.HandleFilterKey(key)
			continue
		}
		switch st.HandleKey(key) {
		case ActQuit:
			quit = true
		case ActAccept:
			out = Output(st)
			accepted = true
		}
		if quit || accepted {
			break
		}
	}

	cleanup()

	if quit {
		clearResult()
		os.Exit(1)
	}
	if err := saveResult(out); err != nil {
		fmt.Fprintln(os.Stderr, "ohk: could not save session result:", err)
	}
	if len(out) > 0 {
		os.Stdout.WriteString(joinWithTrailingNewline(out))
	}
}

func handleArgs(args []string, stdout, stderr io.Writer) (handled bool, code int) {
	if len(args) == 0 {
		return false, 0
	}
	if arg := args[0]; arg == "-h" || arg == "--help" {
		printUsage(stdout)
		return true, 0
	} else if shell, ok := strings.CutPrefix(arg, "--setup-"); ok {
		snippet, ok := setupSnippet(shell)
		if !ok {
			fmt.Fprintf(stderr, "ohk: unknown shell in %q (want zsh, bash or sh)\n", arg)
			printUsage(stderr)
			return true, 2
		}
		fmt.Fprint(stdout, snippet)
		return true, 0
	} else if strings.HasPrefix(arg, "-") {
		fmt.Fprintf(stderr, "ohk: unknown option %q\n", arg)
		printUsage(stderr)
		return true, 2
	}
	fmt.Fprintf(stderr, "ohk: unexpected argument %q\n", args[0])
	printUsage(stderr)
	return true, 2
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `usage:
  some-command | ohk                  interact, then print the result to stdout
  ohk --setup-zsh                     print the shell setup snippet (also: bash, sh)
  ohk --help                          show this help

See the README for the keyboard shortcuts and the ohke shell integration.
`)
}
