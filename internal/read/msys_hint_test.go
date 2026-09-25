package read

import (
	"strings"
	"testing"
)

// ADR-074 T4. MSYS_NO_PATHCONV=1 stops the rewriting as well as
// MSYS2_ARG_CONV_EXCL='*' does, and the hint named only the second.
func TestTheMSYSHintNamesBothVariables(t *testing.T) {
	mangled := `cmd\mrw\main.go;C:\Users\me\AppData\Local\Programs\Git\^func main\`
	_, err := ParseSpec(mangled)
	if err == nil {
		t.Fatal("a mangled spec parsed cleanly")
	}
	for _, want := range []string{"MSYS_NO_PATHCONV=1", "MSYS2_ARG_CONV_EXCL='*'"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the hint does not name %s:\n%v", want, err)
		}
	}
}
