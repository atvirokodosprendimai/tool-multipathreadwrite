package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
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
		// ⚠ PLACEMENT, not presence. Asserting only that a marker appears
		// somewhere passes when every marker is moved to the end of the page —
		// which is the single-token design this record rejects, and which the
		// first version of this test could not tell apart. The open marker must
		// come BEFORE the first line of its span and the close AFTER the last.
		open := strings.Index(text, "-- ck "+ck+" open")
		closed := strings.Index(text, "-- ck "+ck+" close")
		firstLine := strings.Index(text, fmt.Sprintf("%5d| line %d\n", sp[0], sp[0]))
		lastLine := strings.Index(text, fmt.Sprintf("%5d| line %d\n", sp[1], sp[1]))
		if open < 0 || closed < 0 {
			t.Errorf("checkpoint %s does not bracket its span: open=%d close=%d", ck, open, closed)
			continue
		}
		if !(open < firstLine && firstLine <= lastLine && lastLine < closed) {
			t.Errorf("checkpoint %s does not bracket lines %d-%d: open=%d first=%d last=%d close=%d",
				ck, sp[0], sp[1], open, firstLine, lastLine, closed)
		}
		// The open marker states the count, which is what lets an honest caller
		// notice it received fewer lines than the span claims.
		if !strings.Contains(text, fmt.Sprintf("-- ck %s open lines %d-%d (%d lines follow)", ck, sp[0], sp[1], sp[1]-sp[0]+1)) {
			t.Errorf("checkpoint %s does not state its range and count, so a cut caller cannot tell it was cut", ck)
		}
		if len(ck) != 16 {
			t.Errorf("checkpoint %s is %d hex digits; 8 was 32 bits of entropy against a 512-entry store", ck, len(ck))
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

	// ⚠ The WRITES, not only the ledger's opinion of them. Inspecting Covers
	// asserts what the ledger holds; it does not assert that a write is refused
	// or that the refusal carries the remedy, which is what a caller meets. The
	// review of PR #132 found this half missing.
	root2 := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root2, "state"))
	if err := os.WriteFile(filepath.Join(root2, "f.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(body))
	sha := hex.EncodeToString(sum[:])
	_, spans2 := interleave(served(1, 500))
	var f2, l2 string
	for ck, sp := range spans2 {
		if sp[0] == 1 {
			f2 = ck
		}
		if sp[0] == 401 {
			l2 = ck
		}
	}
	if err := hold(root2, map[string]seen.Observation{"f.txt": {SHA: sha, Spans: [][2]int{{1, 500}}}}, spans2); err != nil {
		t.Fatal(err)
	}
	if err := promote(root2, []string{f2, l2}); err != nil {
		t.Fatal(err)
	}
	ledger, err := seen.Load(root2)
	if err != nil {
		t.Fatal(err)
	}
	midHunk := apply.Input{Path: "f.txt", Start: 250, End: 250, Op: "replace", Body: []string{"X"}, Lines: -1}
	res, err := apply.Apply(root2, []apply.Input{midHunk}, apply.Options{Seen: ledger, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("a write into the UNACKNOWLEDGED middle was allowed: failed=%d", res.Failed)
	}
	nameTheAck(root2, &res)
	if !strings.Contains(res.Hunks[0].Reason, "ack") {
		t.Errorf("the refusal does not name the remedy: %s", res.Hunks[0].Reason)
	}
	endHunk := apply.Input{Path: "f.txt", Start: 450, End: 450, Op: "replace", Body: []string{"X"}, Lines: -1}
	res2, err := apply.Apply(root2, []apply.Input{endHunk}, apply.Options{Seen: ledger, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Failed != 0 {
		t.Errorf("a write into an ACKNOWLEDGED span was refused: %s", res2.Hunks[0].Reason)
	}

	// And an ack nobody issued licenses nothing, rather than failing the call.
	if err := promote(root, []string{"00000000"}); err != nil {
		t.Errorf("a stale ack should be ignored, not an error: %v", err)
	}
}

// TestAStaleAcknowledgementDoesNotLicenseTheCurrentFile pins the second P0 the
// review of PR #132 found: promotion used to key observations by PATH alone and
// let the last checkpoint's SHA win, so acknowledging a stale page and a current
// one together recorded the OLD file's spans under the NEW file's SHA.
func TestAStaleAcknowledgementDoesNotLicenseTheCurrentFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(strings.Repeat("x\n", 500)), 0o644); err != nil {
		t.Fatal(err)
	}
	_, oldSpans := interleave(served(1, 200))
	_, newSpans := interleave(served(301, 400))
	var oldCk, newCk string
	for ck := range oldSpans {
		oldCk = ck
	}
	for ck := range newSpans {
		newCk = ck
	}
	if err := hold(root, map[string]seen.Observation{"f.txt": {SHA: "0000aaaa"}}, oldSpans); err != nil {
		t.Fatal(err)
	}
	if err := hold(root, map[string]seen.Observation{"f.txt": {SHA: "1111bbbb"}}, newSpans); err != nil {
		t.Fatal(err)
	}
	if err := promote(root, []string{oldCk, newCk}); err != nil {
		t.Fatal(err)
	}

	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	o := l["f.txt"]
	if o.SHA == "1111bbbb" && o.Covers(1, 200) {
		t.Errorf("a span acknowledged against the OLD file is licensed under the CURRENT sha: %s %s", o.SHA, o.Served())
	}
}
