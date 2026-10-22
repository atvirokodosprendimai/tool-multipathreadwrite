package mcp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-167 T2 over mrw_write: the copy's source is licensed by an acked
// mrw_read (ADR-108) or a CLI read, in the checkout's one ledger.
const src167 = "secret-1\nsecret-2\nsecret-3\nsecret-4\nsecret-5\nsecret-6\n"

func tree167(t *testing.T) string {
	t.Helper()
	root, _ := checkout(t, "a.txt", src167)
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("x\ny\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func fileText167(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestAFromCopyThroughMrwWriteNeedsAnAckedSource(t *testing.T) {
	t.Run("an acked read of the source licenses a copy, a create, $ and A,+N", func(t *testing.T) {
		for _, c := range []struct{ plan, file, want string }{
			{"@@ b.txt 1 insert-after from=a.txt:3-5\n", "b.txt", "x\nsecret-3\nsecret-4\nsecret-5\ny\n"},
			{"@@ c.txt - create from=a.txt:3-5\n", "c.txt", "secret-3\nsecret-4\nsecret-5\n"},
			{"@@ c.txt - create from=a.txt:$\n", "c.txt", "secret-6\n"},
			{"@@ c.txt - create from=a.txt:2,+2\n", "c.txt", "secret-2\nsecret-3\nsecret-4\n"},
		} {
			root := tree167(t)
			acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt", "b.txt"}})))
			res := call(t, root, "mrw_write", map[string]any{"plan": c.plan, "ack": acks, "check": false})
			if res["isError"] == true {
				t.Errorf("%q: refused: %s", c.plan, firstText(t, res))
				continue
			}
			if got := fileText167(t, root, c.file); got != c.want {
				t.Errorf("%q: %s = %q, want %q", c.plan, c.file, got, c.want)
			}
			h, _ := structured(t, res)["hunks"].([]any)
			if len(h) != 1 || h[0].(map[string]any)["copied_first"] == nil || h[0].(map[string]any)["body_from"] == nil {
				t.Errorf("%q: the receipt hunk lacks the copy keys: %v", c.plan, h)
			}
		}
	})

	t.Run("an unacked read of the source licenses nothing", func(t *testing.T) {
		root := tree167(t)
		acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"b.txt"}})))
		call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}}) // served, never acknowledged
		res := call(t, root, "mrw_write", map[string]any{"plan": "@@ b.txt 1 insert-after from=a.txt:3-5\n", "ack": acks, "check": false})
		if res["isError"] != true && structured(t, res)["failed"] == float64(0) {
			t.Errorf("an unacked source was copied: %v", res)
		}
		if got := fileText167(t, root, "b.txt"); got != "x\ny\n" {
			t.Errorf("b.txt = %q after a refused copy", got)
		}
	})

	t.Run("a CLI read of the source licenses it too", func(t *testing.T) {
		root := tree167(t)
		bin := buildCLI(t)
		if out, err := exec.Command(bin, "--root", root, "read", "a.txt", "b.txt").CombinedOutput(); err != nil {
			t.Fatalf("mrw read: %v\n%s", err, out)
		}
		res := call(t, root, "mrw_write", map[string]any{"plan": "@@ b.txt 1 insert-after from=a.txt:3-5\n", "check": false})
		if res["isError"] == true || fileText167(t, root, "b.txt") != "x\nsecret-3\nsecret-4\nsecret-5\ny\n" {
			t.Errorf("a copy after a CLI read: %v / %q", res, fileText167(t, root, "b.txt"))
		}
	})
}

// A refusal for an unserved SOURCE is named by the source: the ack remedy is
// for the page the source was served on, and no target line is named.
func TestAnUnackedSourceIsNamedInTheRefusal(t *testing.T) {
	const plan = "@@ b.txt 1 insert-after from=a.txt:3-5\n"
	root := tree167(t)
	acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"b.txt"}})))
	call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}}) // the source page, never acknowledged
	res := call(t, root, "mrw_write", map[string]any{"plan": plan, "ack": acks, "check": false})
	text := firstText(t, res)
	if !strings.Contains(text, "a.txt") || !strings.Contains(text, "never acknowledged") {
		t.Errorf("the refusal does not offer the ack for the source page: %q", text)
	}
	// The reason, not the hunk line that heads the report: it speaks of the
	// source's lines and of no line of the target.
	hunks, _ := structured(t, res)["hunks"].([]any)
	reason, _ := hunks[0].(map[string]any)["reason"].(string)
	if strings.Contains(reason, "b.txt") || strings.Contains(reason, "secret-") {
		t.Errorf("the reason names the target or discloses a copied line: %q", reason)
	}

	t.Run("the pair: nothing pending, no remedy", func(t *testing.T) {
		root := tree167(t)
		acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"b.txt"}})))
		text := firstText(t, call(t, root, "mrw_write", map[string]any{"plan": plan, "ack": acks, "check": false}))
		if strings.Contains(text, "never acknowledged") {
			t.Errorf("a source that was never served is told to acknowledge: %q", text)
		}
	})

	t.Run("under a newer ledger the cause leads and names the source", func(t *testing.T) {
		root := tree167(t)
		plantLedger157(t, root, "#mrw-seen v7 v9.9.9")
		call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt", "b.txt"}})
		text := firstText(t, call(t, root, "mrw_write", map[string]any{"plan": plan, "check": false}))
		if !strings.Contains(text, "a.txt cannot be copied by this mrw: the read ledger was written by a newer mrw (v9.9.9)") {
			t.Errorf("the newer-ledger refusal does not lead with the source: %q", text)
		}
		if strings.Contains(text, "never acknowledged") {
			t.Errorf("an ack is advised under a newer ledger: %q", text)
		}
	})
}
