package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestSetupSnippet(t *testing.T) {
	for _, sh := range []string{"zsh", "bash", "sh"} {
		s, ok := setupSnippet(sh)
		if !ok || s == "" {
			t.Fatalf("%s: ok=%v len=%d", sh, ok, len(s))
		}
		if !strings.Contains(s, "ohk()") || !strings.Contains(s, "ohke()") {
			t.Fatalf("%s: snippet missing functions:\n%s", sh, s)
		}
	}
	if _, ok := setupSnippet("fish"); ok {
		t.Fatalf("unknown shell should not resolve")
	}
}

func TestSetupSnippetSyntax(t *testing.T) {
	for _, sh := range []string{"zsh", "bash", "sh"} {
		path, err := exec.LookPath(sh)
		if err != nil {
			t.Logf("skipping %s: not on PATH", sh)
			continue
		}
		s, _ := setupSnippet(sh)
		cmd := exec.Command(path, "-n")
		cmd.Stdin = strings.NewReader(s)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s -n rejected the snippet: %v\n%s", sh, err, out)
		}
	}
}

func TestHandleArgsNoArgs(t *testing.T) {
	if handled, _ := handleArgs(nil, &bytes.Buffer{}, &bytes.Buffer{}); handled {
		t.Fatalf("no args should not be handled")
	}
}

func TestHandleArgsSetup(t *testing.T) {
	var out bytes.Buffer
	handled, code := handleArgs([]string{"--setup-zsh"}, &out, &bytes.Buffer{})
	if !handled || code != 0 {
		t.Fatalf("handled=%v code=%d", handled, code)
	}
	if !strings.Contains(out.String(), "ohke()") {
		t.Fatalf("stdout missing snippet: %q", out.String())
	}
}

func TestHandleArgsHelp(t *testing.T) {
	var out bytes.Buffer
	handled, code := handleArgs([]string{"--help"}, &out, &bytes.Buffer{})
	if !handled || code != 0 {
		t.Fatalf("handled=%v code=%d", handled, code)
	}
	if !strings.Contains(out.String(), "usage:") {
		t.Fatalf("stdout missing usage: %q", out.String())
	}
}

func TestHandleArgsUnknown(t *testing.T) {
	for _, args := range [][]string{{"--nope"}, {"--setup-fish"}, {"positional"}} {
		var out, errBuf bytes.Buffer
		handled, code := handleArgs(args, &out, &errBuf)
		if !handled || code != 2 {
			t.Fatalf("%v: handled=%v code=%d", args, handled, code)
		}
		if out.Len() != 0 {
			t.Fatalf("%v: unexpected stdout %q", args, out.String())
		}
		if errBuf.Len() == 0 {
			t.Fatalf("%v: expected stderr message", args)
		}
	}
}
