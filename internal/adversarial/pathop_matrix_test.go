package adversarial

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestRandomisedPathOpWriteMatrix is ADR-057 arm 3: the built binary, random
// flags, .txt trees (prose, so the default check does not fire). Oracle is
// ADR-057 + ADR-001: unlink/rename after a whole-file read apply (exit 0,
// path gone / dest landed); dry-run writes nothing; an unread sibling is
// exit 1 and restores; dest-exists is exit 1 and restores. Pool (grep
// `case Op` in internal/plan/plan.go): unlink, rename, plus the two teaching
// misses from the 2026-09-15 Zeus field report — unread unlink (no ledger)
// and `to=` as an unknown option (exit 2).
func TestRandomisedPathOpWriteMatrix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh -c checks are not this arm; path ops are prose here anyway")
	}
	seed := int64(54)
	if s := os.Getenv("MRW_SEED"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		seed = v
	}
	r := rand.New(rand.NewSource(seed))
	const iterations = 80
	for i := 0; i < iterations; i++ {
		state := t.TempDir()
		root := tree(t, map[string]string{"a.txt": "aaa\n", "b.txt": "bbb\n"})
		flags := [][]string{nil, {"--dry-run"}, {"--no-check"}, {"--quiet"}}[r.Intn(4)]
		kind := r.Intn(7)
		var plan string
		switch kind {
		case 0:
			plan = "@@ a.txt - unlink\n"
		case 1:
			plan = "@@ a.txt - rename\nc.txt\n"
		case 2:
			plan = "@@ a.txt - unlink\n@@ b.txt 1 replace anchor=\"bbb\"\nnope\n"
		case 3:
			plan = "@@ a.txt - rename\nb.txt\n"
		case 4:
			plan = "@@ a.txt - unlink\n@@ a.txt 1 replace\nx\n"
		case 5:
			// Zeus 2026-09-15: unread unlink used to talk about a line address.
			plan = "@@ a.txt - unlink\n"
		default:
			plan = "@@ a.txt - rename to=c.txt\n"
		}
		planFile := filepath.Join(t.TempDir(), "p.mrw")
		if err := os.WriteFile(planFile, []byte(plan), 0o644); err != nil {
			t.Fatal(err)
		}
		if kind != 5 {
			if _, code := run(t, state, root, "read", "a.txt"); code != 0 {
				t.Fatalf("seed=%d iter=%d: read a.txt failed", seed, i)
			}
		}
		// kind 2 leaves b.txt unread on purpose. kind 5 reads nothing.
		if kind != 2 && kind != 5 {
			if _, code := run(t, state, root, "read", "b.txt"); code != 0 {
				t.Fatalf("seed=%d iter=%d: read b.txt failed", seed, i)
			}
		}
		out, code := run(t, state, root, append(append([]string{"write"}, flags...), planFile)...)
		label := fmt.Sprintf("seed=%d iter=%d kind=%d flags=%v\n%s", seed, i, kind, flags, out)
		dry := has(flags, "--dry-run")
		aGone := !fileExists(root, "a.txt")
		if kind == 6 {
			if code != 2 {
				t.Fatalf("%s\nrename to= exit %d want 2", label, code)
			}
			if aGone || readFile(t, root, "a.txt") != "aaa\n" {
				t.Fatalf("%s\nto= wrote the tree", label)
			}
			continue
		}
		wouldApply := kind == 0 || kind == 1
		switch {
		case dry:
			if wouldApply && code != 0 {
				t.Fatalf("%s\ndry-run of an applying plan exit %d want 0", label, code)
			}
			if !wouldApply && code != 1 {
				t.Fatalf("%s\ndry-run of a refusing plan exit %d want 1", label, code)
			}
			if aGone {
				t.Fatalf("%s\ndry-run removed a.txt", label)
			}
			if kind == 5 {
				assertUnreadPathOpWording(t, label, out)
			}
		case kind == 0:
			if code != 0 || !aGone {
				t.Fatalf("%s\nunlink exit %d gone=%v", label, code, aGone)
			}
			if readFile(t, root, "b.txt") != "bbb\n" {
				t.Fatalf("%s\nsibling mutated", label)
			}
		case kind == 1:
			if code != 0 || aGone == false || !fileExists(root, "c.txt") {
				t.Fatalf("%s\nrename exit %d aGone=%v c=%v", label, code, aGone, fileExists(root, "c.txt"))
			}
			if readFile(t, root, "c.txt") != "aaa\n" {
				t.Fatalf("%s\ndest content", label)
			}
		case kind == 2, kind == 3, kind == 4, kind == 5:
			if code != 1 {
				t.Fatalf("%s\nrefuse exit %d want 1", label, code)
			}
			if aGone || readFile(t, root, "a.txt") != "aaa\n" || readFile(t, root, "b.txt") != "bbb\n" {
				t.Fatalf("%s\nrefusal wrote", label)
			}
			if kind == 5 {
				assertUnreadPathOpWording(t, label, out)
			}
		}
	}
}

func fileExists(root, name string) bool {
	_, err := os.Stat(filepath.Join(root, name))
	return err == nil
}

func assertUnreadPathOpWording(t *testing.T, label, out string) {
	t.Helper()
	if !strings.Contains(out, "takes no line address") {
		t.Fatalf("%s\nunread path-op missing 'takes no line address'", label)
	}
	if strings.Contains(strings.ToLower(out), "line address means nothing") {
		t.Fatalf("%s\nunread path-op still talks about a line address", label)
	}
}
