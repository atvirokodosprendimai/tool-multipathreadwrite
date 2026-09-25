package ingest

import (
	"bytes"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

// ADR-070 T2. The compilers left a body uncounted unless it held an @@ line,
// so a patch whose content starts with body=3 would compile to a plan the new
// guard refuses, naming a header the caller never wrote. emit counts it.
func TestACompiledBodyStartingWithBodyIsCounted(t *testing.T) {
	root := writeTree(t, map[string]string{"a.go": "a\n"})
	for name, compile := range map[string]func() ([]byte, error){
		"apply_patch add": func() ([]byte, error) {
			return CompileApplyPatch(root, []byte("*** Begin Patch\n*** Add File: n.txt\n+body=3\n+x\n*** End Patch\n"))
		},
		"search_replace": func() ([]byte, error) {
			return CompileSearchReplace(root, []byte("a.go\n<<<<<<< SEARCH\na\n=======\nbody=3\n>>>>>>> REPLACE\n"))
		},
	} {
		text, err := compile()
		if err != nil {
			t.Fatalf("%s: compile: %v", name, err)
		}
		hs, err := plan.Parse(bytes.NewReader(text))
		if err != nil {
			t.Errorf("%s: the compiled plan is refused: %v\n%s", name, err, text)
			continue
		}
		if len(hs) != 1 || len(hs[0].Body) == 0 || hs[0].Body[0] != "body=3" {
			t.Errorf("%s: hunks = %+v, want a first body line \"body=3\"", name, hs)
		}
	}
}
