package read

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
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
	if _, _, err := AstGrep(root, nil, "package $A", nil, AstGrepOptions{}); err == nil || !strings.Contains(err.Error(), "200 bytes") || !strings.Contains(err.Error(), "64") {
		t.Errorf("an answer over the cap: err %v, want one naming 200 bytes and the limit", err)
	}
	maxAstGrepBytes = 1024
	if specs, _, err := AstGrep(root, nil, "package $A", nil, AstGrepOptions{}); err != nil || len(specs) != 1 {
		t.Errorf("the same answer under the cap: specs %v, err %v", specs, err)
	}
}

// ADR-104 T3, the review of #295. A size taken before the read is outrun by a
// file that grows, so readCapped also reads through a bound. ADR-109 refuses an
// endless device at the open, before any read, so /dev/zero is now that
// refusal; the bound is driven on Linux by a /proc file, which is a regular file
// whose stat says 0 bytes and whose read returns more.
func TestReadCappedRefusesAStreamOverTheCap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/dev/zero is a unix device")
	}
	old := maxFileBytes
	maxFileBytes = 16
	t.Cleanup(func() { maxFileBytes = old })
	if _, err := readCapped("/dev/zero"); !errors.Is(err, regular.ErrNotRegular) {
		t.Errorf("an endless stream: err %v; want it refused as not a regular file, unread", err)
	}
	if runtime.GOOS == "linux" {
		var over errOverFileCap
		if _, err := readCapped("/proc/self/maps"); !errors.As(err, &over) || over.exact {
			t.Errorf("a file whose stat is outrun by its read: err %v; want the bounded read's refusal", err)
		}
	}
}

// ADR-104 T3, the review of #295. ast-grep's CR-only probe read each hit file
// whole. A hit on a file over the cap is reported once, naming the limit, and not
// served.
func TestAnAstGrepHitOnAFileOverTheCapIsReportedOnce(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "big.go"), []byte("package big // "+strings.Repeat("x", 40)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hit := `{"file":"big.go","range":{"start":{"line":0},"end":{"line":0}}}`
	installFakeAstGrepJSON(t, "["+hit+","+hit+"]", 0)
	old := maxFileBytes
	maxFileBytes = 16
	t.Cleanup(func() { maxFileBytes = old })
	specs, probs, err := AstGrep(root, nil, "package $A", nil, AstGrepOptions{})
	if err != nil || len(specs) != 0 || len(probs) != 1 || probs[0].Path != "big.go" || !strings.Contains(probs[0].Reason, "16-byte limit") {
		t.Errorf("specs %v, problems %v, err %v; want no spec and one problem naming the limit", specs, probs, err)
	}
}
