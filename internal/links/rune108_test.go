package links

import (
	"runtime"
	"slices"
	"testing"
)

// ADR-108 T2. components narrowed each rune to a byte before asking whether it
// was a path separator, so on Windows U+042F (Я, low byte 0x2F) and U+015C (Ŝ,
// low byte 0x5C) split a valid name into directories. Only an ASCII separator
// splits a path.
func TestAUnicodeRuneNeverSplitsAPath(t *testing.T) {
	for _, name := range []string{"aЯb.txt", "aŜb.txt"} {
		if got := components(name); !slices.Equal(got, []string{name}) {
			t.Errorf("components(%q) = %q, want it whole", name, got)
		}
	}
	for r := rune(0x80); r <= 0x2FFFF; r++ {
		if low := r & 0xFF; low != '/' && low != '\\' {
			continue
		}
		name := "x" + string(r) + "y"
		if got := components(name); len(got) != 1 {
			t.Fatalf("U+%04X split %q into %q", r, name, got)
		}
	}
	if got := components("a/b"); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("components(a/b) = %q", got)
	}
	if runtime.GOOS == "windows" {
		if got := components(`a\b`); !slices.Equal(got, []string{"a", "b"}) {
			t.Errorf(`components(a\b) = %q`, got)
		}
	}
}
