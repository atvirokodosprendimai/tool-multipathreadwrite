package read

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ADR-104 T3. Every served or searched file was read whole with no size check.
// One over the cap is refused before it is read, by name, with its size and the
// limit; a smaller one is served as before.
func TestAFileOverTheReadCapIsRefusedByName(t *testing.T) {
	old := maxFileBytes
	maxFileBytes = 16
	t.Cleanup(func() { maxFileBytes = old })
	root := t.TempDir()
	for name, body := range map[string]string{"big.txt": strings.Repeat("0123456789", 4), "small.txt": "tiny\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, problems := run(t, root, Options{Numbers: true}, "big.txt", "small.txt")
	if problems != 1 || !strings.Contains(out, "big.txt  UNREADABLE") || !strings.Contains(out, "40 bytes") || !strings.Contains(out, "16") || !strings.Contains(out, "tiny") {
		t.Errorf("problems %d:\n%s", problems, out)
	}
	specs, probs, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile(".")})
	if err != nil {
		t.Fatal(err)
	}
	var named bool
	for _, p := range probs {
		named = named || (p.Path == "big.txt" && strings.Contains(p.Reason, "16"))
	}
	if !named || len(specs) != 1 || specs[0].Path != "small.txt" {
		t.Errorf("the walk: specs %v, problems %v; want small.txt matched and big.txt reported with the limit", specs, probs)
	}
}

// ADR-104 T4. ast-grep's whole answer was read with io.ReadAll. One over the cap
// is refused, naming its size and the limit.
func TestAnAstGrepAnswerOverTheCapIsRefused(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	answer := `[{"file":"a.go","range":{"start":{"line":0},"end":{"line":0}}}]`
	answer += strings.Repeat(" ", 200-len(answer))
	installFakeAstGrepJSON(t, answer, 0)
	old := maxAstGrepBytes
	t.Cleanup(func() { maxAstGrepBytes = old })
	maxAstGrepBytes = 64
	if _, _, err := AstGrep(root, nil, "package $A", nil); err == nil || !strings.Contains(err.Error(), "200 bytes") || !strings.Contains(err.Error(), "64") {
		t.Errorf("an answer over the cap: err %v, want one naming 200 bytes and the limit", err)
	}
	maxAstGrepBytes = 1024
	if specs, _, err := AstGrep(root, nil, "package $A", nil); err != nil || len(specs) != 1 {
		t.Errorf("the same answer under the cap: specs %v, err %v", specs, err)
	}
}
