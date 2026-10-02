package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-116. A grep too large to serve answers with an INDEX, and an index says
// what the walk skipped as a served answer does: the `skipped` key and the
// note naming no_ignore. Found by the review of #315: only the served and
// no-match answers carried it, so an index skipped in silence.
func TestAnIndexSaysWhatTheWalkSkipped(t *testing.T) {
	root := grepTree(t, 60, 400)
	if err := os.WriteFile(filepath.Join(root, "bin.dat"), []byte("NEEDLE\x00\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := call(t, root, "mrw_read", map[string]any{"grep": "NEEDLE"})
	rc := receipt(t, res)
	if _, ok := rc["index"]; !ok {
		t.Fatalf("the fixture did not produce an INDEX: %v", rc)
	}
	sk, _ := rc["skipped"].(map[string]any)
	if sk["binary"] != float64(1) {
		t.Errorf("the index's skipped is %v, want binary 1", rc["skipped"])
	}
	if txt := firstText(t, res); !strings.Contains(txt, "-- skipped:") || !strings.Contains(txt, "no_ignore") {
		t.Errorf("the index text does not say what was skipped: %q", txt)
	}
}
