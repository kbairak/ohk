package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

func main() {
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

	if err := tty.MakeRaw(); err != nil {
		fmt.Fprintln(os.Stderr, "ohk:", err)
		os.Exit(1)
	}
	if err := tty.EnterAltScreen(); err != nil {
		tty.Restore()
		fmt.Fprintln(os.Stderr, "ohk:", err)
		os.Exit(1)
	}

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
