package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResultPathUnset(t *testing.T) {
	t.Setenv(sessionEnv, "")
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if _, ok, err := resultPath(); err != nil || ok {
		t.Fatalf("want no path, got ok=%v err=%v", ok, err)
	}
}

func TestResultPathSet(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	t.Setenv(sessionEnv, "12345")
	path, ok, err := resultPath()
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	want := filepath.Join(base, "ohk", "12345.stdout")
	if path != want {
		t.Fatalf("path=%q want %q", path, want)
	}
}

func TestInvalidSession(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for _, s := range []string{"..", "a/b", "a b"} {
		t.Setenv(sessionEnv, s)
		if _, ok, err := resultPath(); ok || err == nil {
			t.Fatalf("session %q should be rejected (ok=%v err=%v)", s, ok, err)
		}
	}
	t.Setenv(sessionEnv, "")
	if _, ok, err := resultPath(); ok || err != nil {
		t.Fatalf("empty session should mean no write (ok=%v err=%v)", ok, err)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	t.Setenv(sessionEnv, "999")
	if err := saveResult([]string{"b", "e"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(base, "ohk", "999.stdout")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "b\ne\n" {
		t.Fatalf("content=%q", data)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%04o", fi.Mode().Perm())
	}
}

func TestSaveEmptyOutput(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	t.Setenv(sessionEnv, "1")
	if err := saveResult(nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(base, "ohk", "1.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("want empty file, got %q", data)
	}
}

func TestSaveNoSessionWritesNothing(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	t.Setenv(sessionEnv, "")
	if err := saveResult([]string{"x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, "ohk")); !os.IsNotExist(err) {
		t.Fatalf("state dir should not exist, err=%v", err)
	}
}

func TestSaveRejectsInsecureDir(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	t.Setenv(sessionEnv, "7")
	if err := os.MkdirAll(filepath.Join(base, "ohk"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := saveResult([]string{"x"}); err == nil {
		t.Fatalf("want error for insecure dir")
	}
}

func TestClearResult(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	t.Setenv(sessionEnv, "42")
	if err := saveResult([]string{"x"}); err != nil {
		t.Fatal(err)
	}
	clearResult()
	if _, err := os.Stat(filepath.Join(base, "ohk", "42.stdout")); !os.IsNotExist(err) {
		t.Fatalf("file should be removed, err=%v", err)
	}
}
