package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const sessionEnv = "OHK_SESSION"

func stateDir() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "ohk"), nil
}

func validSession(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

func resultPath() (string, bool, error) {
	sess := os.Getenv(sessionEnv)
	if sess == "" {
		return "", false, nil
	}
	if !validSession(sess) {
		return "", false, fmt.Errorf("invalid %s %q", sessionEnv, sess)
	}
	dir, err := stateDir()
	if err != nil {
		return "", false, err
	}
	return filepath.Join(dir, sess+".stdout"), true, nil
}

func ensureStateDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	fi, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("state path %s is not a directory", dir)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("state dir %s has insecure permissions %04o", dir, fi.Mode().Perm())
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		if int(st.Uid) != os.Geteuid() {
			return fmt.Errorf("state dir %s is not owned by uid %d", dir, os.Geteuid())
		}
	}
	return nil
}

func writeFileSecure(dir, path string, data []byte) error {
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	fail := func(e error) error {
		tmp.Close()
		os.Remove(tmpName)
		return e
	}
	if err := tmp.Chmod(0o600); err != nil {
		return fail(err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fail(err)
	}
	fi, err := tmp.Stat()
	if err != nil {
		return fail(err)
	}
	if !fi.Mode().IsRegular() || fi.Mode().Perm()&0o077 != 0 {
		return fail(fmt.Errorf("refusing to write insecure temp file %s", tmpName))
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

func saveResult(out []string) error {
	path, ok, err := resultPath()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	dir := filepath.Dir(path)
	if err := ensureStateDir(dir); err != nil {
		return err
	}
	var data []byte
	if len(out) > 0 {
		data = []byte(joinWithTrailingNewline(out))
	}
	return writeFileSecure(dir, path, data)
}

func clearResult() {
	path, ok, err := resultPath()
	if err != nil || !ok {
		return
	}
	os.Remove(path)
}

func joinWithTrailingNewline(out []string) string {
	s := ""
	for i, l := range out {
		if i > 0 {
			s += "\n"
		}
		s += l
	}
	return s + "\n"
}
