package main

import (
	"os"

	"golang.org/x/term"
)

type TTY struct {
	f     *os.File
	saved *term.State
}

func OpenTTY() (*TTY, error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	return &TTY{f: f}, nil
}

func (t *TTY) MakeRaw() error {
	saved, err := term.MakeRaw(int(t.f.Fd()))
	if err != nil {
		return err
	}
	t.saved = saved
	return nil
}

func (t *TTY) Restore() error {
	if t.saved == nil {
		return nil
	}
	return term.Restore(int(t.f.Fd()), t.saved)
}

func (t *TTY) Size() (w, h int, err error) {
	return term.GetSize(int(t.f.Fd()))
}

func (t *TTY) Read(p []byte) (int, error) {
	return t.f.Read(p)
}

func (t *TTY) WriteString(s string) error {
	_, err := t.f.WriteString(s)
	return err
}

func (t *TTY) Close() error {
	return t.f.Close()
}

func (t *TTY) EnterAltScreen() error {
	return t.WriteString("\x1b[?1049h" + "\x1b[?25l")
}

func (t *TTY) ExitAltScreen() error {
	return t.WriteString("\x1b[?25h" + "\x1b[?1049l" + "\x1b[27m")
}
