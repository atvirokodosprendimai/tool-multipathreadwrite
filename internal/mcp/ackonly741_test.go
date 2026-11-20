package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// #741. An mrw_read that sent only `ack` answered isError "needs at least one
// spec" and had already promoted the acknowledgement into the ledger: a call
// reported as an error changed the ledger. ADR-031 Decision 3 has `ack` ride
// with a read or a write, and the tool's description has it name the ids of a
// previous read; an ack with nothing to read is the caller's mistake, so it
// stays an error and records nothing. The pair: the same ack with a spec is
// promoted.
func TestAnErrorFromReadPromotesNoAcknowledgement(t *testing.T) {
	setup := func(t *testing.T) (root, first string) {
		root = t.TempDir()
		t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
		var body string
		for i := 1; i <= 500; i++ {
			body += fmt.Sprintf("line %d\n", i)
		}
		if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(body))
		_, spans := interleave(served(1, 500), nil)
		if err := hold(root, "f.txt", hex.EncodeToString(sum[:]), spans); err != nil {
			t.Fatal(err)
		}
		for ck, sp := range spans {
			if sp[0] == 1 {
				first = ck
			}
		}
		if first == "" {
			t.Fatal("no checkpoint for the first span")
		}
		return root, first
	}
	recorded := func(t *testing.T, root string) bool {
		t.Helper()
		l, err := seen.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		_, ok := l["f.txt"]
		return ok
	}

	for name, args := range map[string]func(ck string) map[string]any{
		"ack alone": func(ck string) map[string]any { return map[string]any{"ack": []string{ck}} },
		"count without grep": func(ck string) map[string]any {
			return map[string]any{"ack": []string{ck}, "specs": []string{"f.txt"}, "count": true}
		},
	} {
		root, ck := setup(t)
		res := call(t, root, "mrw_read", args(ck))
		if res["isError"] != true {
			t.Fatalf("%s: not an error: %v", name, res)
		}
		if recorded(t, root) {
			t.Errorf("%s: a call that answered isError promoted its acknowledgement", name)
		}
	}

	root, ck := setup(t)
	if res := call(t, root, "mrw_read", map[string]any{"ack": []string{ck}, "specs": []string{"f.txt:1-3"}}); res["isError"] == true {
		t.Fatalf("the pair: an ack beside a spec was refused: %v", res)
	}
	if !recorded(t, root) {
		t.Error("the pair: an ack beside a spec was not promoted")
	}
}
