package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// served renders n lines the way read.Run does, so the interleave is exercised
// against the shape it actually meets rather than a convenient one.
func served(from, to int) string {
	var b strings.Builder
	for i := from; i <= to; i++ {
		fmt.Fprintf(&b, "%5d| line %d\n", i, i)
	}
	return b.String()
}

func TestACheckpointCoversTheSpanItFollows(t *testing.T) {
	text, spans := interleave(served(1, 500))
	if len(spans) != 3 {
		t.Fatalf("got %d checkpoints for 500 lines at ckEvery=%d, want 3", len(spans), ckEvery)
	}
	want := map[[2]int]bool{{1, 200}: true, {201, 400}: true, {401, 500}: true}
	for ck, sp := range spans {
		if !want[sp] {
			t.Errorf("checkpoint %s covers %v, which is not one of the three expected spans", ck, sp)
		}
		delete(want, sp)
		if !strings.Contains(text, "-- ck "+ck) {
			t.Errorf("checkpoint %s is not in the served text, so nobody can echo it", ck)
		}
	}
	if len(want) != 0 {
		t.Errorf("spans never produced: %v", want)
	}

	// The marker must be UNGUESSABLE. A content hash or a counter would be
	// derivable by a caller that received nothing, which is the whole point.
	_, again := interleave(served(1, 500))
	for ck := range spans {
		if _, collides := again[ck]; collides {
			t.Fatalf("checkpoint %s repeated across two reads — it is derived, not random", ck)
		}
	}

	// A page that served only part of a file yields checkpoints inside what it
	// served and none beyond: the span is what was SERVED, not what was asked.
	_, part := interleave(served(1, 90))
	for ck, sp := range part {
		if sp[0] < 1 || sp[1] > 90 {
			t.Errorf("checkpoint %s covers %v, outside the served range 1-90", ck, sp)
		}
	}
}

func TestAPendingRecordReachesNoLedger(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(strings.Repeat("x\n", 500)), 0o644); err != nil {
		t.Fatal(err)
	}
	_, spans := interleave(served(1, 500))
	obs := map[string]seen.Observation{"f.txt": {SHA: "deadbeef", Spans: [][2]int{{1, 500}}}}
	if err := hold(root, obs, spans); err != nil {
		t.Fatal(err)
	}

	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if o, ok := l["f.txt"]; ok {
		t.Fatalf("a served page reached the ledger before it was acknowledged: %v", o)
	}
}

// TestOnlyAckedSegmentsAreRecorded is ADR-031's Enforced-by.
//
// ⚠ THE FIXTURE CUTS THE MIDDLE, and that is the whole design of it. The
// truncation measured on 2026-09-05 kept lines 1-90 and 2644-2727 and discarded
// everything between, so a fixture that drops the TAIL is green against the
// single-token scheme ADR-031 explicitly rejects — one token at the end of the
// page survives that cut, and licenses 2,554 lines nobody saw.
func TestOnlyAckedSegmentsAreRecorded(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	body := ""
	for i := 1; i <= 500; i++ {
		body += fmt.Sprintf("line %d\n", i)
	}
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	_, spans := interleave(served(1, 500))
	obs := map[string]seen.Observation{"f.txt": {SHA: "deadbeef", Spans: [][2]int{{1, 500}}}}
	if err := hold(root, obs, spans); err != nil {
		t.Fatal(err)
	}

	// Ack the FIRST and LAST checkpoints, not the middle one — the shape the
	// host's cut actually produced.
	var first, mid, last string
	for ck, sp := range spans {
		switch sp[0] {
		case 1:
			first = ck
		case 201:
			mid = ck
		case 401:
			last = ck
		}
	}
	if first == "" || mid == "" || last == "" {
		t.Fatalf("fixture did not produce three checkpoints: %v", spans)
	}
	if err := promote(root, []string{first, last}); err != nil {
		t.Fatal(err)
	}

	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	o, ok := l["f.txt"]
	if !ok {
		t.Fatal("nothing was recorded for an acknowledged page")
	}
	if !o.Covers(1, 200) {
		t.Errorf("the first acknowledged segment is not licensed: %s", o.Served())
	}
	if !o.Covers(401, 500) {
		t.Errorf("the last acknowledged segment is not licensed: %s", o.Served())
	}
	// The middle is the whole point: it was sent, it was not received, and it
	// must not be writable.
	if o.Covers(250, 250) {
		t.Errorf("a line in the UNACKNOWLEDGED middle is licensed: %s — this is the defect ADR-031 exists to close", o.Served())
	}
	if o.Whole() {
		t.Error("the observation claims the whole file, so acking two of three segments licensed everything")
	}

	// And an ack nobody issued licenses nothing, rather than failing the call.
	if err := promote(root, []string{"00000000"}); err != nil {
		t.Errorf("a stale ack should be ignored, not an error: %v", err)
	}
}
