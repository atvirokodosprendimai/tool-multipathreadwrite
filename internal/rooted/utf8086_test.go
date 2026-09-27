package rooted

import (
	"testing"
	"unicode/utf8"
)

// ADR-086 T3. Windows maps a byte that is not valid UTF-8 to U+FFFD when it
// converts a path to UTF-16, and reports no error, so a create of bad\xffname
// landed as bad<U+FFFD>name on Windows even after the plan header kept the byte.
// win32Alias names such a component with the name Windows would open; it is a
// pure function, so this runs on every platform.
func TestWin32AliasNamesAComponentThatIsNotValidUTF8(t *testing.T) {
	comp, reads := win32Alias("d/bad\xffname.txt")
	want := "bad" + string(utf8.RuneError) + "name.txt"
	if comp != "bad\xffname.txt" || reads != want {
		t.Errorf("win32Alias = (%q, %q), want (%q, %q)", comp, reads, "bad\xffname.txt", want)
	}
	if comp, _ := win32Alias("d/fé.txt"); comp != "" {
		t.Errorf("valid UTF-8 was named an alias: %q", comp)
	}
}
