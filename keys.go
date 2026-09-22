package main

type KeyKind int

const (
	KeyRune KeyKind = iota
	KeyEnter
	KeyEsc
	KeyTab
	KeyBackspace
	KeyCtrlW
	KeyCtrlC
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyOther
)

type Key struct {
	Kind KeyKind
	Rune rune
}

func DecodeKey(b []byte) (Key, int) {
	if len(b) == 0 {
		return Key{Kind: KeyOther}, 0
	}
	c := b[0]
	switch c {
	case 0x09:
		return Key{Kind: KeyTab}, 1
	case 0x0D, 0x0A:
		return Key{Kind: KeyEnter}, 1
	case 0x7F, 0x08:
		return Key{Kind: KeyBackspace}, 1
	case 0x17:
		return Key{Kind: KeyCtrlW}, 1
	case 0x03:
		return Key{Kind: KeyCtrlC}, 1
	case 0x1B:
		if len(b) >= 3 && b[1] == '[' {
			switch b[2] {
			case 'A':
				return Key{Kind: KeyUp}, 3
			case 'B':
				return Key{Kind: KeyDown}, 3
			case 'C':
				return Key{Kind: KeyRight}, 3
			case 'D':
				return Key{Kind: KeyLeft}, 3
			}
		}
		return Key{Kind: KeyEsc}, 1
	}
	if c >= 0x20 && c <= 0x7E {
		return Key{Kind: KeyRune, Rune: rune(c)}, 1
	}
	return Key{Kind: KeyOther}, 1
}
