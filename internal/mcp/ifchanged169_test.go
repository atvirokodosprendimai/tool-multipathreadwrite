package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-169 Decision 6: the handshake an MCP host reads at the start of a session
// names the guard among its recipes, within the 4096-byte bound.
func TestTheHandshakeTeachesIfChanged(t *testing.T) {
	got := instructionsText()
	if !strings.Contains(got, "mrw_read with if_changed") {
		t.Errorf("the handshake's recipes do not name if_changed:\n%s", got)
	}
	if len(got) > maxInstructionsChars {
		t.Errorf("instructions are %d bytes, over %d", len(got), maxInstructionsChars)
	}
}

// ADR-169 T2. mrw_read takes if_changed, an object of path to the sha prefix
// the caller holds, and answers a spec of an unchanged file with one line and no
// checkpoint. That answer delivered what was asked for, so it is not an error
// (ADR-025 amended); a read that served nothing and answered nothing unchanged
// still is.
func TestIfChangedAnswersUnchangedWithoutAnError(t *testing.T) {
	root, _ := checkout(t, "a.txt", "alpha\nbravo\ncharlie\n")
	put := func(name, body string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(body))
		return hex.EncodeToString(sum[:])[:8]
	}
	a := put("a.txt", "alpha\nbravo\ncharlie\n")
	b := put("b.txt", "kilo\nlima\n")
	unchanged := func(res map[string]any) []any {
		t.Helper()
		l, _ := receipt(t, res)["unchanged"].([]any)
		return l
	}

	// An unchanged file: one line, no checkpoint, no error, the receipt names it.
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}, "if_changed": map[string]any{"a.txt": a}})
	if want := "==> a.txt  unchanged since sha " + a + " — 3 lines, 20 bytes not served"; !strings.Contains(resultText(res), want) {
		t.Errorf("the answer lacks %q:\n%s", want, resultText(res))
	}
	if res["isError"] == true || strings.Contains(resultText(res), "-- ck ") || strings.Contains(resultText(res), "alpha") {
		t.Errorf("an unchanged answer is an error, carries a checkpoint or serves text: %v", res)
	}
	if u := unchanged(res); len(u) != 1 || fmt.Sprint(u[0]) != fmt.Sprint(map[string]any{"path": "a.txt", "sha": a, "lines": float64(3), "bytes": float64(20)}) {
		t.Errorf("the receipt's unchanged = %v", u)
	}
	if obs, _ := receipt(t, res)["observed"].(map[string]any); len(obs) != 0 {
		t.Errorf("an unchanged answer observed %v", obs)
	}

	// The pair: the same call without the guard serves the lines and its checkpoints.
	if plain := call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}}); !strings.Contains(resultText(plain), "-- ck ") || unchanged(plain) != nil {
		t.Errorf("the unguarded read did not serve checkpoints, or lists unchanged:\n%s", resultText(plain))
	}

	// A changed file is served with its checkpoints, as an unguarded read serves it.
	put("a.txt", "alpha\nbravo\ncharlie\ndelta\n")
	res = call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}, "if_changed": map[string]any{"a.txt": a}})
	if !strings.Contains(resultText(res), "-- ck ") || !strings.Contains(resultText(res), "delta") || unchanged(res) != nil || res["isError"] == true {
		t.Errorf("a changed file was not served whole:\n%v", res)
	}

	// A guard on a named path no spec serves is a tool error naming it; so is a
	// value that is not an object, and a sha that is not hex.
	for name, g := range map[string]any{"an unserved file": map[string]any{"b.txt": b}, "not an object": "a.txt=" + a, "not hex": map[string]any{"a.txt": "zzzzzzzz"}, "too short": map[string]any{"a.txt": "abc"}} {
		res = call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}, "if_changed": g})
		if res["isError"] != true || resultText(res) == "" || strings.Contains(resultText(res), "-- ck ") {
			t.Errorf("%s: want a tool error, got %v", name, res)
		}
	}
	res = call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}, "if_changed": map[string]any{"b.txt": b}})
	if !strings.Contains(resultText(res), "b.txt") {
		t.Errorf("the refusal does not name the guard's file:\n%s", resultText(res))
	}
	res = call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}, "stat": true, "if_changed": map[string]any{"a.txt": a}})
	if res["isError"] != true {
		t.Errorf("if_changed beside stat was not refused: %v", res)
	}

	// ADR-025 still holds: a read that served nothing and answered nothing
	// unchanged is an error.
	if res = call(t, root, "mrw_read", map[string]any{"specs": []any{"nope.txt"}, "if_changed": map[string]any{"nope.txt": a}}); res["isError"] != true {
		t.Errorf("a read of a missing file under a guard was not an error: %v", res)
	}

	// Under grep, a guarded file the walk did not reach is named and the call is not an error.
	a = put("a.txt", "alpha\nbravo\ncharlie\n")
	res = call(t, root, "mrw_read", map[string]any{"grep": "alpha", "if_changed": map[string]any{"b.txt": b, "a.txt": a}})
	if res["isError"] == true || !strings.Contains(resultText(res), "-- guard not used: b.txt (this read did not serve it)") || !strings.Contains(resultText(res), "unchanged since sha "+a) {
		t.Errorf("a walk: %v", res)
	}

	// A grep that matches nothing still refuses two guards on one file, as the
	// CLI does (the go-reviewer pass of ADR-169).
	res = call(t, root, "mrw_read", map[string]any{"grep": "NO_SUCH_TEXT_ANYWHERE", "if_changed": map[string]any{"a.txt": a, "./a.txt": a}})
	if res["isError"] != true || !strings.Contains(resultText(res), "two guards on one file") {
		t.Errorf("a grep that matched nothing accepted two guards on one file: %v", res)
	}
	// An index is not judged; its specs resent with if_changed are.
	idxRoot := grepTree(t, 60, 400)
	idx := call(t, idxRoot, "mrw_read", map[string]any{"grep": "NEEDLE"})
	entries, _ := receipt(t, idx)["index"].([]any)
	if len(entries) < 2 {
		t.Fatalf("the fixture did not produce an index: %v", idx)
	}
	guard, specs := map[string]any{}, []any{}
	for _, e := range entries[:2] {
		name := e.(string)
		body, err := os.ReadFile(filepath.Join(idxRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		guard[name], specs = hex.EncodeToString(sum[:])[:8], append(specs, name)
	}
	res = call(t, idxRoot, "mrw_read", map[string]any{"grep": "NEEDLE", "specs": specs, "if_changed": guard})
	if u := unchanged(res); len(u) != 2 || res["isError"] == true || strings.Contains(resultText(res), "the NEEDLE is here") {
		t.Errorf("the resent index specs were not answered unchanged: %v", res)
	}

	// An ack sent with a guarded call is recorded as without the guard.
	root2, _ := checkout(t, "a.txt", "alpha\nbravo\n")
	put2 := func(name, body string) string {
		if err := os.WriteFile(filepath.Join(root2, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(body))
		return hex.EncodeToString(sum[:])[:8]
	}
	a2 := put2("a.txt", "alpha\nbravo\n")
	put2("b.txt", "kilo\nlima\n")
	acks := checkpointsIn(served0(t, call(t, root2, "mrw_read", map[string]any{"specs": []any{"b.txt"}})))
	call(t, root2, "mrw_read", map[string]any{"specs": []any{"a.txt"}, "if_changed": map[string]any{"a.txt": a2}, "ack": acks})
	w := structured(t, call(t, root2, "mrw_write", map[string]any{"plan": "@@ b.txt 1 replace\nKILO\n", "dry_run": true}))
	if n, _ := w["failed"].(float64); n != 0 {
		t.Errorf("an ack sent beside a guard was not recorded: %v", w["hunks"])
	}
	w = structured(t, call(t, root2, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nALPHA\n", "dry_run": true}))
	if n, _ := w["failed"].(float64); n != 1 {
		t.Errorf("an unchanged answer licensed a write to a.txt: %v", w["hunks"])
	}
}
