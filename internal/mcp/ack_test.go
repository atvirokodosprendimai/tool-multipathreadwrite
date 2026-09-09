package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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

func TestACheckpointCoversTheSpanItBrackets(t *testing.T) {
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
	if err := hold(root, "f.txt", "deadbeef", spans); err != nil {
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
// page survives that cut, and licenses 2,553 lines nobody saw.
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

	// The real digest: promotion records only the version ON DISK, because a
	// span acknowledged against a version that is gone cannot license anything
	// and the ledger's SHA check would refuse it anyway.
	sum0 := sha256.Sum256([]byte(body))
	_, spans := interleave(served(1, 500))
	if err := hold(root, "f.txt", hex.EncodeToString(sum0[:]), spans); err != nil {
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

	// ⚠ THE BOUNDARIES, EXACTLY. Probing the middle of an unacknowledged span
	// leaves an off-by-one invisible: [Start, End+1] licenses line 201 while
	// 1-200 was acknowledged, and 250 is refused either way. The fifth review of
	// PR #132 found that. So the adjacent lines on both sides are asserted.
	if o.Covers(201, 201) {
		t.Error("line 201 is licensed although only 1-200 was acknowledged — promotion is over-licensing by one at the span's end")
	}
	if o.Covers(400, 400) {
		t.Error("line 400 is licensed although the acknowledged span starts at 401 — promotion is over-licensing by one at the span's start")
	}
	if !o.Covers(200, 200) || !o.Covers(401, 401) {
		t.Errorf("an acknowledged span does not reach its own edges: %s", o.Served())
	}
	// ⚠ THE EXACT SPAN SET. Probing 201, 250 and 400 leaves a rogue interior
	// span anywhere else undetected, while T2 claims the ledger holds EXACTLY
	// the two acknowledged spans (sixth review of PR #132).
	want := [][2]int{{1, 200}, {401, 500}}
	if len(o.Spans) != len(want) {
		t.Fatalf("the ledger holds %v, want exactly %v", o.Spans, want)
	}
	for i := range want {
		if o.Spans[i] != want[i] {
			t.Errorf("span %d is %v, want %v", i, o.Spans[i], want[i])
		}
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
	if err := hold(root2, "f.txt", sha, spans2); err != nil {
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
	if !strings.Contains(res.Hunks[0].Reason, AckRule) {
		t.Errorf("the refusal does not carry the canonical rule, so it teaches an abbreviated one: %s", res.Hunks[0].Reason)
	}

	// Through an ALIAS, because apply treats a symlink as the same file and the
	// remedy used to go missing for exactly that caller (ADR-029).
	if err := os.Symlink("f.txt", filepath.Join(root2, "alias.txt")); err == nil {
		aliased := apply.Input{Path: "alias.txt", Start: 250, End: 250, Op: "replace", Body: []string{"X"}, Lines: -1}
		ares, err := apply.Apply(root2, []apply.Input{aliased}, apply.Options{Seen: ledger, DryRun: true})
		if err != nil {
			t.Fatal(err)
		}
		if ares.Failed != 1 {
			t.Fatalf("an alias-spelled write into the unacknowledged middle was allowed: %d", ares.Failed)
		}
		nameTheAck(root2, &ares)
		if !strings.Contains(ares.Hunks[0].Reason, AckRule) {
			t.Errorf("the remedy is missing for an ALIAS spelling: %s", ares.Hunks[0].Reason)
		}
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
	if err := hold(root, "f.txt", "0000aaaa", oldSpans); err != nil {
		t.Fatal(err)
	}
	if err := hold(root, "f.txt", "1111bbbb", newSpans); err != nil {
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

// TestAStaleAcknowledgementDoesNotRevokeTheCurrentOne is the other half, and the
// review of PR #132 found it missing. Recording one observation per version by
// ranging over a map let map ORDER decide the outcome, and seen.merge replaces
// on a SHA change — so a stale acknowledgement could overwrite a valid current
// licence and consume both ids doing it.
func TestAStaleAcknowledgementDoesNotRevokeTheCurrentOne(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	body := strings.Repeat("y\n", 500)
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(body))
	live := hex.EncodeToString(sum[:])

	_, stale := interleave(served(1, 200))
	_, current := interleave(served(201, 400))
	var staleCk, curCk string
	for ck := range stale {
		staleCk = ck
	}
	for ck := range current {
		curCk = ck
	}
	if err := hold(root, "f.txt", "00000000deadbeef", stale); err != nil {
		t.Fatal(err)
	}
	if err := hold(root, "f.txt", live, current); err != nil {
		t.Fatal(err)
	}
	// ⚠ The pending records are RECREATED each round. An earlier version looped
	// twenty times over the same two ids, and promotion CONSUMES a pending entry
	// — so rounds 2-20 were no-ops and the map-order implementation could still
	// pass by luck. Found by the review of PR #132.
	for i := 0; i < 20; i++ {
		if err := hold(root, "f.txt", "00000000deadbeef", map[string][2]int{staleCk: stale[staleCk]}); err != nil {
			t.Fatal(err)
		}
		if err := hold(root, "f.txt", live, map[string][2]int{curCk: current[curCk]}); err != nil {
			t.Fatal(err)
		}
		if err := promote(root, []string{staleCk, curCk}); err != nil {
			t.Fatal(err)
		}
	}
	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	o := l["f.txt"]
	if o.SHA != live || !o.Covers(201, 400) {
		t.Errorf("the CURRENT licence did not survive a stale acknowledgement in the same call: sha=%s served=%s", o.SHA, o.Served())
	}
}

// TestEverySurfaceCarriesTheOneRule is the structural answer to a gate that kept
// passing on presence. Three paraphrases drifted from the mechanism and two
// mutants survived by leaving a token or a heading in place, so the rule is now
// ONE constant and every surface must carry it BYTE FOR BYTE.
func TestEverySurfaceCarriesTheOneRule(t *testing.T) {
	// ⚠ AN INDEPENDENT ORACLE FIRST. Every assertion below is
	// strings.Contains(surface, AckRule), which is satisfied by ANY surface when
	// AckRule is empty — so setting the constant to "" strips the requirement
	// from the instructions, the footer, the refusal and both schemas while
	// every check passes. The fifth review of PR #132 demonstrated exactly that.
	// A rule compared only against itself is not pinned, so the sentence is
	// written out HERE, literally, and the constant is checked against it.
	const want = "Send an id in ack only if you hold BOTH its open and close markers AND counted the N numbered lines the open marker says follow: one marker is not enough, because a cut starting inside a span leaves the other end."
	if AckRule != want {
		t.Fatalf("AckRule no longer states the rule this record decided:\n got: %q\nwant: %q", AckRule, want)
	}

	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(filepath.Dir(root))
	for _, f := range []string{"README.md", "AGENTS.md"} {
		b, err := os.ReadFile(filepath.Join(repo, f))
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if !strings.Contains(string(b), AckRule) {
			t.Errorf("%s does not carry the acknowledgement rule verbatim, so it can drift from what the server enforces", f)
		}
		if strings.Contains(string(b), "A page licenses nothing until you acknowledge it") {
			t.Errorf("%s still teaches acknowledgement as pages-only", f)
		}
	}
	if !strings.Contains(instructionsText(), AckRule) {
		t.Error("the MCP instructions do not carry the acknowledgement rule verbatim")
	}
	if strings.Contains(instructionsText(), "A PAGE LICENSES NOTHING") {
		t.Error("the MCP instructions still teach acknowledgement as pages-only")
	}

	// ⚠ AND THE SURFACES A CALLER ACTUALLY MEETS. Checking the two documents and
	// the instructions left the page FOOTER and both schema descriptions as
	// independent paraphrases — a schema could teach the old unsafe rule with
	// every check green, which the fourth review of PR #132 pointed out. Both
	// ack descriptions are built from the constant, and this asserts that.
	for _, tl := range tools() {
		schema, _ := tl.InputSchema.(map[string]any)
		props, _ := schema["properties"].(map[string]any)
		ack, ok := props["ack"].(map[string]any)
		if !ok {
			continue
		}
		if d, _ := ack["description"].(string); !strings.Contains(d, AckRule) {
			t.Errorf("%s's ack description paraphrases the rule instead of carrying it: %s", tl.Name, d)
		} else if strings.Contains(d, "paged read") {
			t.Errorf("%s's ack description still frames the requirement as pages-only: %s", tl.Name, d)
		}
	}
}

// TestCheckpointsAreNotADenseSequence catches the cheapest wrong implementation
// and CLAIMS NO MORE THAN THAT.
//
// ⚠ A statistical test on output cannot establish unpredictability. The first
// version of this test was called "…AreNotASequence" and the fourth review of
// PR #132 showed it green against a counter emitting i<<48 — adjacent gaps of
// 2^48, a total spread far past the threshold. What actually buys the property
// is the crypto/rand dependency in checkpoint(); this test rules out a dense
// counter and the non-collision case, and the name says so.
func TestCheckpointsAreNotADenseSequence(t *testing.T) {
	_, spans := interleave(served(1, 4000))
	if len(spans) < 8 {
		t.Fatalf("need several checkpoints to judge a sequence, got %d", len(spans))
	}
	vals := make([]uint64, 0, len(spans))
	for ck := range spans {
		n, err := strconv.ParseUint(ck, 16, 64)
		if err != nil {
			t.Fatalf("checkpoint %q is not 16 hex digits: %v", ck, err)
		}
		vals = append(vals, n)
	}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })
	// A DENSE counter's sorted values are consecutive or nearly so. This says
	// nothing about a sparse one — see the doc comment.
	gap := vals[len(vals)-1] - vals[0]
	if gap < uint64(len(vals))*1<<40 {
		t.Errorf("checkpoints span only %d across %d values — that is a dense counter, not 64 bits of draw", gap, len(vals))
	}
	// And no two are adjacent, which a dense counter's neighbours would be.
	for i := 1; i < len(vals); i++ {
		if vals[i]-vals[i-1] < 1<<32 {
			t.Errorf("two checkpoints are within 2^32 of each other, which 64-bit draws do not do at this sample size")
		}
	}
}

// TestThePendingStoreIsBounded gives the cap an oracle. T1 marked the bound
// [proof: mutation] with no mutant against it, and disabling eviction was
// observed by nothing (sixth review of PR #132).
func TestThePendingStoreIsBounded(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxPending+64; i++ {
		spans := map[string][2]int{checkpoint(): {i + 1, i + 1}}
		if err := hold(root, "f.txt", "deadbeef", spans); err != nil {
			t.Fatal(err)
		}
	}
	store, err := loadPending(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(store) > maxPending {
		t.Errorf("the pending store holds %d entries, over the %d bound — a caller that never acknowledges grows it without limit", len(store), maxPending)
	}
}

// TestTheRemedyMatchesTheRefusedAddress pins the P2 the seventh review of
// PR #132 found: nameTheAck appended the acknowledgement remedy whenever the
// FILE had anything pending, so a caller refused at line 2000 with lines 1-1000
// pending was told to acknowledge a page that cannot license line 2000. A
// refusal naming a fix that cannot work is ADR-015's failure wearing the shape
// of help.
func TestTheRemedyMatchesTheRefusedAddress(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	body := strings.Repeat("z\n", 3000)
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(body))
	if err := hold(root, "f.txt", hex.EncodeToString(sum[:]),
		map[string][2]int{checkpoint(): {1, 1000}}); err != nil {
		t.Fatal(err)
	}
	ledger, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		name   string
		line   int
		remedy bool
	}{
		{"inside the pending span", 500, true},
		{"outside every pending span", 2000, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			res, err := apply.Apply(root, []apply.Input{{
				Path: "f.txt", Start: c.line, End: c.line, Op: "replace",
				Body: []string{"X"}, Lines: -1,
			}}, apply.Options{Seen: ledger, DryRun: true})
			if err != nil {
				t.Fatal(err)
			}
			if res.Failed != 1 {
				t.Fatalf("failed=%d, want 1", res.Failed)
			}
			nameTheAck(root, &res)
			got := strings.Contains(res.Hunks[0].Reason, AckRule)
			if got != c.remedy {
				t.Errorf("remedy present = %v, want %v — a page covering 1-1000 cannot license line %d: %s",
					got, c.remedy, c.line, res.Hunks[0].Reason)
			}
		})
	}
}

// TestAFittingReadLicensesOnlyWhatCameBack is ADR-039's Enforced-by. A fitting
// MCP serve used to call seen.Record before returning, so a host that cut it
// licensed lines nobody received — the ADR-031 hole, one size class down.
func TestAFittingReadLicensesOnlyWhatCameBack(t *testing.T) {
	root, path := checkout(t, "a.txt", "one\ntwo\nthree\n")
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}})
	text := served0(t, res)
	acks := checkpointsIn(text)
	if len(acks) == 0 {
		t.Fatal("a fitting MCP read carries no checkpoints, so nothing can be acknowledged")
	}
	if !strings.Contains(text, "-- ck ") || !strings.Contains(text, " open lines ") || !strings.Contains(text, " close") {
		t.Fatalf("a fitting serve is not bracketed the way a page is:\n%s", text)
	}

	ledger, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ledger[path]; ok {
		t.Fatal("a fitting MCP read licensed the file on serve — that is the hole ADR-039 closes")
	}

	unacked := structured(t, call(t, root, "mrw_write", map[string]any{
		"plan": "@@ a.txt 2 replace\nTWO\n"}))
	if n, _ := unacked["failed"].(float64); n == 0 {
		t.Fatalf("a write without ack after a fitting serve applied: %v", unacked)
	}
	hunks, _ := unacked["hunks"].([]any)
	if len(hunks) == 0 {
		t.Fatal("the refused write named no hunk")
	}
	h0, _ := hunks[0].(map[string]any)
	reason, _ := h0["reason"].(string)
	if !strings.Contains(reason, "ack") && !strings.Contains(reason, AckRule) {
		t.Errorf("the refusal does not name ack / AckRule: %s", reason)
	}
	body, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "one\ntwo\nthree\n" {
		t.Errorf("an unacked write changed the tree: %q", body)
	}

	ok := structured(t, call(t, root, "mrw_write", map[string]any{
		"plan": "@@ a.txt 2 replace\nTWO\n", "ack": acks}))
	if n, _ := ok["failed"].(float64); n != 0 {
		t.Errorf("the same write with ack was refused: %v", ok["hunks"])
	}
	body, err = os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "one\nTWO\nthree\n" {
		t.Errorf("acked write did not apply: %q", body)
	}
}

// TestAFittingReadHoldsPendingPerFile pins the hold reshape. Today's hold
// picks one path from the observation map, so a two-file fixture is the only
// shape that can fail if that loop survives.
func TestAFittingReadHoldsPendingPerFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for name, body := range map[string]string{"a.txt": "A1\nA2\nA3\n", "b.txt": "B1\nB2\nB3\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt", "b.txt"}})
	text := served0(t, res)
	aAcks := checkpointsBetween(text, "a.txt", "b.txt")
	if len(aAcks) == 0 {
		t.Fatal("file A carried no checkpoints")
	}

	noB := structured(t, call(t, root, "mrw_write", map[string]any{
		"plan": "@@ b.txt 2 replace\nBB\n", "ack": aAcks}))
	if n, _ := noB["failed"].(float64); n == 0 {
		t.Fatalf("acking only A's ids licensed a write to B: %v", noB)
	}
	okA := structured(t, call(t, root, "mrw_write", map[string]any{
		"plan": "@@ a.txt 2 replace\nAA\n", "ack": aAcks}))
	if n, _ := okA["failed"].(float64); n != 0 {
		t.Errorf("acking A's ids did not license A: %v", okA["hunks"])
	}
}

// checkpointsBetween returns open-marker ids that sit after ==> path and
// before the next ==> header (or EOF). That is how a caller tells which file
// a checkpoint brackets on a multi-file serve.
func checkpointsBetween(text, path, next string) []any {
	start := strings.Index(text, "==> "+path)
	if start < 0 {
		return nil
	}
	rest := text[start:]
	if next != "" {
		if i := strings.Index(rest[1:], "\n==> "); i >= 0 {
			rest = rest[:i+1]
		}
	}
	return checkpointsIn(rest)
}

// TestACLIReadStillLicensesWithoutAck is T2: the CLI is the member this
// record leaves out. Drive the built binary the way internal/adversarial does.
func TestACLIReadStillLicensesWithoutAck(t *testing.T) {
	root, path := checkout(t, "a.txt", "one\ntwo\nthree\n")
	bin := buildCLI(t)
	read := exec.Command(bin, "--root", root, "read", path)
	if out, err := read.CombinedOutput(); err != nil {
		t.Fatalf("mrw read: %v\n%s", err, out)
	}
	write := exec.Command(bin, "--root", root, "write", "-")
	write.Stdin = strings.NewReader("@@ a.txt 2 replace\nTWO\n")
	out, err := write.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI write after CLI read (no ack) was refused: %v\n%s", err, out)
	}
	body, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "one\nTWO\nthree\n" {
		t.Errorf("CLI write did not apply: %q", body)
	}
}
