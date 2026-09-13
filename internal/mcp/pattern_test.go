package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

// ADR-056 T1: the structured receipt carries `pattern` on every write —
// present with fires:false on the first advisory write, fires:true with the
// counts on the third, and present on a refused plan too. ADR-055 put the
// line on the CLI receipt because the balance ROW was not read; a caller of
// this transport reads a JSON object, and a fact absent from it is a fact
// the caller does not have.
func TestTheMCPReceiptCarriesThePattern(t *testing.T) {
	root, path := checkout(t, "f.go", "func A() {\n\treturn\n}\n")
	const plan = "@@ f.go 1 replace anchor=\"func A\"\nfunc A() { return }\n"

	var receipts []map[string]any
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(filepath.Join(root, "f.go"), []byte("func A() {\n\treturn\n}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{path + ":1"}})))
		receipts = append(receipts, structured(t, call(t, root, "mrw_write", map[string]any{"plan": plan, "ack": acks})))
	}

	first := patternOf(t, receipts[0])
	if first["fires"] != false || first["advisory_writes"] != float64(1) || first["window"] != float64(1) {
		t.Errorf("first receipt pattern = %v, want {1 1 false}", first)
	}
	third := patternOf(t, receipts[2])
	if third["fires"] != true || third["advisory_writes"] != float64(3) || third["window"] != float64(3) {
		t.Errorf("third receipt pattern = %v, want {3 3 true}", third)
	}

	// A refused plan (a wrong anchor) still carries the object: it describes
	// the ring, not this write, and the ring is unchanged by a plan that
	// wrote nothing.
	refused := structured(t, call(t, root, "mrw_write", map[string]any{
		"plan": "@@ f.go 3 replace anchor=\"NOT HERE\"\n}\n",
	}))
	if applied, _ := refused["applied"].(bool); applied {
		t.Fatalf("the unread-line plan applied: %v", refused)
	}
	got := patternOf(t, refused)
	if got["window"] != float64(3) || got["fires"] != true {
		t.Errorf("a refused plan's pattern = %v, want the unchanged ring {3 3 true}", got)
	}
}

func patternOf(t *testing.T, receipt map[string]any) map[string]any {
	t.Helper()
	p, ok := receipt["pattern"].(map[string]any)
	if !ok {
		t.Fatalf("receipt has no pattern object: %v", receipt)
	}
	for _, k := range []string{"advisory_writes", "window", "fires"} {
		if _, ok := p[k]; !ok {
			t.Errorf("pattern lacks %q: %v", k, p)
		}
	}
	return p
}
