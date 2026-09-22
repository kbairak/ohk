package main

import "testing"

func TestDecodeSingleBytes(t *testing.T) {
	cases := []struct {
		in   byte
		kind KeyKind
	}{
		{0x09, KeyTab},
		{0x0D, KeyEnter},
		{0x0A, KeyEnter},
		{0x7F, KeyBackspace},
		{0x08, KeyBackspace},
		{0x17, KeyCtrlW},
		{0x03, KeyCtrlC},
	}
	for _, c := range cases {
		k, n := DecodeKey([]byte{c.in})
		if k.Kind != c.kind || n != 1 {
			t.Errorf("byte %#x: got kind %v n=%d; want %v n=1", c.in, k.Kind, n, c.kind)
		}
	}
}

func TestDecodePrintableRune(t *testing.T) {
	k, n := DecodeKey([]byte{'A'})
	if k.Kind != KeyRune || k.Rune != 'A' || n != 1 {
		t.Fatalf("got %+v n=%d", k, n)
	}
}

func TestDecodeArrows(t *testing.T) {
	cases := []struct {
		in   string
		kind KeyKind
	}{
		{"\x1b[A", KeyUp},
		{"\x1b[B", KeyDown},
		{"\x1b[C", KeyRight},
		{"\x1b[D", KeyLeft},
	}
	for _, c := range cases {
		k, n := DecodeKey([]byte(c.in))
		if k.Kind != c.kind || n != 3 {
			t.Errorf("%q: got kind %v n=%d; want %v n=3", c.in, k.Kind, n, c.kind)
		}
	}
}

func TestDecodeEscapeAmbiguity(t *testing.T) {
	k, n := DecodeKey([]byte{0x1B})
	if k.Kind != KeyEsc || n != 1 {
		t.Fatalf("lone ESC: got %v n=%d", k.Kind, n)
	}
	// Unknown CSI falls back to Esc consuming 1 byte, so the caller can
	// re-decode the remainder.
	k, n = DecodeKey([]byte("\x1b[Z"))
	if k.Kind != KeyEsc || n != 1 {
		t.Fatalf("unknown CSI: got %v n=%d", k.Kind, n)
	}
}
