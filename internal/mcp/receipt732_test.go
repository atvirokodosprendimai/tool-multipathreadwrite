package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer"
)

const (
	commitErr  = "could not restore c.txt from dir/.mrw-aside-123456: rename refused; b.go: rename failed (a.go already written)"
	asidePath  = "dir/.mrw-aside-123456"
	receiptLed = "The full receipt is at "
)

// partialResult is a commit that failed after a.go landed, with okHunks verdicts
// padded to make the whole receipt large.
func partialResult(okHunks int) apply.Result {
	res := apply.Result{
		Root:       "/tmp/whatever",
		Files:      []apply.FileResult{{Path: "a.go", Written: true}, {Path: "b.go"}},
		LeftBehind: []string{asidePath},
		Failed:     1,
	}
	for i := 0; i < okHunks; i++ {
		res.Hunks = append(res.Hunks, apply.HunkResult{Path: "a.go", Addr: fmt.Sprint(i), Op: "replace", Status: apply.StatusOK, Reason: strings.Repeat("pad ", 40)})
	}
	res.Hunks = append(res.Hunks, apply.HunkResult{Path: "b.go", Addr: "1", Op: "replace", Status: apply.StatusFailed, Reason: "commit failed"})
	return res
}

func withMax(t *testing.T, n int) {
	t.Helper()
	restore := MaxResultChars
	t.Cleanup(func() { MaxResultChars = restore })
	MaxResultChars = n
}

func structuredOf(t *testing.T, out callToolResult) (sc struct {
	Elided      string `json:"elided"`
	FullReceipt string `json:"full_receipt"`
	Mutation    string `json:"mutation"`
}, ok bool) {
	t.Helper()
	raw, isRaw := out.StructuredContent.(json.RawMessage)
	if !isRaw {
		return sc, false
	}
	if err := json.Unmarshal(raw, &sc); err != nil {
		t.Fatal(err)
	}
	return sc, true
}

func keptDir(t *testing.T) string {
	t.Helper()
	base, err := state.Base()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(filepath.Dir(base), "mrw-receipts")
}

func keptCount(t *testing.T) int {
	t.Helper()
	ents, _ := os.ReadDir(keptDir(t))
	n := 0
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".receipt") {
			n++
		}
	}
	return n
}

// Issue #732. A ceiling too small for the receipt of a partial commit left only
// the terminal sentence: no commit error, no written file names. The whole
// receipt is kept outside the tree and the sentence names where, and says the
// mutation in words (#735).
func TestAPartialCommitUnderASmallCeilingNamesTheKeptWholeReceipt(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	withMax(t, 1100) // just above the floor: the terminal sentence is all the answer has
	if !writeFloorFits() {
		t.Fatalf("the fixture ceiling %d is under the write floor %d", MaxResultChars, writeFloor())
	}
	out, rpcErr := boundedReceipt(t.TempDir(), partialResult(60), errors.New(commitErr))
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	if n := encodedSize(out); n > MaxResultChars {
		t.Fatalf("the answer is %d bytes against a %d ceiling", n, MaxResultChars)
	}
	text := out.Content[0].Text
	if !strings.Contains(text, "mutation: partial") {
		t.Errorf("the terminal sentence does not say the mutation in words:\n%s", text)
	}
	i := strings.Index(text, receiptLed)
	if i < 0 {
		t.Fatalf("the terminal sentence names no kept receipt:\n%s", text)
	}
	path := strings.TrimSuffix(strings.TrimSpace(text[i+len(receiptLed):]), ".")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the named file cannot be read: %v", err)
	}
	for _, want := range []string{"rename refused", "a.go", asidePath, `"mutation": "partial"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("the kept receipt lacks %q:\n%s", want, b)
		}
	}
	// Windows reports 0666 for a writable file and keeps the access to the
	// directory's inherited ACL, as for mrw's other state files.
	if fi, err := os.Stat(path); err != nil || (runtime.GOOS != "windows" && fi.Mode().Perm()&0o077 != 0) {
		t.Errorf("the kept receipt is %v (%v), want mode 0600", fi.Mode().Perm(), err)
	}
}

// The pair: a complete write whose cut drops only ok verdicts loses no recovery
// detail, so it keeps nothing and names nothing; the same goes for a partial
// commit whose cut still holds the failed hunk, the written file and the error.
func TestACutThatDropsOnlyOkVerdictsKeepsNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	withMax(t, 3000)
	complete := apply.Result{Root: "/tmp/whatever", Applied: true, Files: []apply.FileResult{{Path: "a.go", Written: true}}}
	for i := 0; i < 300; i++ {
		complete.Hunks = append(complete.Hunks, apply.HunkResult{Path: "a.go", Addr: fmt.Sprint(i), Op: "replace", Status: apply.StatusOK, Reason: strings.Repeat("pad ", 40)})
	}
	for name, tc := range map[string]struct {
		res apply.Result
		err error
	}{
		"a complete write": {complete, nil},
		"a partial commit whose failed hunk, written file and error all fit": {partialResult(200), errors.New(commitErr)},
	} {
		out, rpcErr := boundedReceipt(t.TempDir(), tc.res, tc.err)
		if rpcErr != nil {
			t.Fatal(rpcErr)
		}
		sc, ok := structuredOf(t, out)
		if !ok || sc.Elided == "" || sc.FullReceipt != "" || keptCount(t) != 0 {
			t.Errorf("%s: structured=%v elided=%q full_receipt=%q kept=%d; want an elided answer that keeps nothing", name, ok, sc.Elided, sc.FullReceipt, keptCount(t))
		}
	}
}

// lostTail is a receipt whose check tail must be cut and cannot be kept in a log
// of its own (TMPDIR names a directory that is not there): the one cut before the
// terminal sentence that loses detail no answer holds.
func lostTail(t *testing.T) (*writer.Verified, apply.Result) {
	t.Helper()
	gone := filepath.Join(t.TempDir(), "gone")
	t.Setenv("TMPDIR", gone)
	t.Setenv("TEMP", gone)
	t.Setenv("TMP", gone)
	var tail []string
	for i := 0; i < 40; i++ {
		tail = append(tail, fmt.Sprintf("check output line %02d %s", i, strings.Repeat("x", 60)))
	}
	chk := &check.Result{Ran: true, ExitCode: 1, Command: "false", Tail: tail}
	return &writer.Verified{Check: chk}, partialResult(10)
}

// A failure tail that no log could keep is detail the answer cannot hold: the
// whole receipt, tail included, is kept and named.
func TestALostCheckTailKeepsTheWholeReceipt(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	v, res := lostTail(t)
	withMax(t, 6000)
	out, rpcErr := checkedReceipt(t.TempDir(), res, v, errors.New(commitErr), true)
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	sc, ok := structuredOf(t, out)
	if !ok || !strings.Contains(sc.Elided, "lost: no log could be written") || sc.FullReceipt == "" {
		t.Fatalf("structured=%v elided=%q full_receipt=%q; want the lost tail named and a kept receipt", ok, sc.Elided, sc.FullReceipt)
	}
	b, err := os.ReadFile(sc.FullReceipt)
	if err != nil || !strings.Contains(string(b), "check output line 39") {
		t.Errorf("the kept receipt does not hold the lost tail (%v):\n%.300s", err, b)
	}
}

// Review of #739. The kept receipt's name outranks the ok verdicts a deeper cut
// drops: over a run of ceilings, an answer that is structured names the receipt
// unless no cut fits with the name, when it is the shallowest cut without it and
// the kept file is removed (final pass); once any answer is structured every
// larger ceiling's is too. Where the name does not fit a cut, the next, deeper
// cut is taken, not the same cut without it.
func TestACutThatCannotNameTheKeptReceiptGivesWayToADeeperOneThatCan(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	v, res := lostTail(t)
	for i := 0; i < 300; i++ {
		res.Hunks = append(res.Hunks, apply.HunkResult{Path: "a.go", Addr: fmt.Sprint(5000 + i), Op: "replace", Status: apply.StatusOK, Reason: "ok"})
	}
	restore := MaxResultChars
	t.Cleanup(func() { MaxResultChars = restore })
	structured, terminal := 0, 0
	for c := 1500; c <= 7000; c += 10 {
		MaxResultChars = c
		if !writeFloorFits() {
			continue
		}
		before := keptCount(t)
		out, rpcErr := checkedReceipt(t.TempDir(), res, v, errors.New(commitErr), true)
		if rpcErr != nil {
			t.Fatal(rpcErr)
		}
		sc, ok := structuredOf(t, out)
		switch {
		case ok && sc.FullReceipt == "" && keptCount(t) > before:
			t.Fatalf("at ceiling %d a structured answer names no kept receipt and leaves one on disk", c)
		case ok:
			structured++
		case structured > 0:
			t.Fatalf("at ceiling %d the answer fell back to the terminal sentence, though a smaller ceiling had a structured one", c)
		default:
			terminal++
		}
	}
	if structured == 0 || terminal == 0 {
		t.Errorf("the run saw %d structured and %d terminal answers: the ceilings do not cross the window", structured, terminal)
	}
}

// Review of #739. A kept receipt holds no file content: previews are taken off
// before it is written.
func TestAKeptReceiptHoldsNoPreview(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	withMax(t, 1100)
	res := partialResult(60)
	res.Hunks[0].Before, res.Hunks[0].After = []string{"SECRET-BEFORE"}, []string{"SECRET-AFTER"}
	out, _ := boundedReceipt(t.TempDir(), res, errors.New(commitErr))
	text := out.Content[0].Text
	i := strings.Index(text, receiptLed)
	if i < 0 {
		t.Fatalf("no kept receipt named:\n%s", text)
	}
	b, err := os.ReadFile(strings.TrimSuffix(strings.TrimSpace(text[i+len(receiptLed):]), "."))
	if err != nil || strings.Contains(string(b), "SECRET-") {
		t.Errorf("the kept receipt holds preview content (%v)", err)
	}
}

// Review of #739. One kept file is cut at 1 MiB, the note included, and says so.
func TestAKeptReceiptIsCapped(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	path, err := keepFullReceipt(strings.Repeat("x", 2*keptReceiptMaxBytes), map[string]string{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil || fi.Size() > keptReceiptMaxBytes {
		t.Fatalf("the kept receipt is %d bytes (%v), want at most %d", fi.Size(), err, keptReceiptMaxBytes)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "[truncated:") {
		t.Errorf("a cut receipt does not say so")
	}
}

// Review of #739. The lock guards only the trim, and a trim that cannot take it
// at once is skipped: a held lock delays nothing, and the file is kept all the same.
func TestAHeldTrimLockDoesNotDelayTheKeptReceipt(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := os.MkdirAll(keptDir(t), 0o700); err != nil {
		t.Fatal(err)
	}
	release, err := state.HoldFileWithin(filepath.Join(keptDir(t), ".lock"), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	withMax(t, 1100)
	start := time.Now()
	out, rpcErr := boundedReceipt(t.TempDir(), partialResult(60), errors.New(commitErr))
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("the answer took %v behind a held lock", d)
	}
	if !strings.Contains(out.Content[0].Text, receiptLed) || keptCount(t) != 1 {
		t.Errorf("the receipt was not kept behind a held lock (kept %d):\n%s", keptCount(t), out.Content[0].Text)
	}
}

// Review of #739. A keep that failed is not tried again on the way to the
// terminal sentence.
func TestAFailedKeepIsNotRetried(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	v, res := lostTail(t)
	calls := 0
	real := keepReceipt
	keepReceipt = func(string, any) (string, error) { calls++; return "", errors.New("no room") }
	t.Cleanup(func() { keepReceipt = real })
	withMax(t, 1100) // the tail is cut, then the terminal sentence is reached
	out, rpcErr := checkedReceipt(t.TempDir(), res, v, errors.New(commitErr), true)
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	if calls != 1 || strings.Contains(out.Content[0].Text, receiptLed) {
		t.Errorf("keep called %d times (want 1), answer:\n%s", calls, out.Content[0].Text)
	}
}

// Only the newest keptReceiptsMax stay, and they are the newest: the oldest go.
func TestOnlyTheNewestKeptReceiptsStay(t *testing.T) {
	restoreGrace := trimGrace
	t.Cleanup(func() { trimGrace = restoreGrace })
	trimGrace = 0
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var paths []string
	for i := 0; i < keptReceiptsMax+5; i++ {
		p, err := keepFullReceipt(fmt.Sprint("receipt ", i), nil)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	for i, p := range paths {
		_, err := os.Stat(p)
		wantKept := i >= len(paths)-keptReceiptsMax
		if (err == nil) != wantKept {
			t.Errorf("receipt %d: kept=%v, want %v (the newest %d stay)", i, err == nil, wantKept, keptReceiptsMax)
		}
	}
}

// failedPlan returns a partial commit whose failed hunks alone make the whole
// receipt many times the 1 MiB a kept receipt is capped at: n out-of-range
// replaces, plan lines 3, 5, 7, ..., each its own header and body, a.go written.
func failedPlan(n int) apply.Result {
	res := apply.Result{
		Root:   "/tmp/whatever",
		Files:  []apply.FileResult{{Path: "a.go", Written: true}, {Path: "b.go"}},
		Failed: n,
	}
	for i := 0; i < n; i++ {
		res.Hunks = append(res.Hunks, apply.HunkResult{Path: "b.go", Addr: fmt.Sprint(100 + i), Op: "replace", SrcLine: 3 + 2*i,
			Status: apply.StatusFailed, Reason: "replace of that line is out of range: the file has 4 lines, " + strings.Repeat("x", 30)})
	}
	return res
}

// #739 final pass. When the failed rows ALONE exceed the cap, the kept receipt is
// no longer cut at 1 MiB wherever the byte count falls: its summary (counts,
// mutation, written files, the commit error) comes first, then failed rows until
// the cap, and it ends with a note naming how many failed hunks it omitted and
// their first and last plan lines. Every byte of it is inside the cap.
func TestAKeptReceiptOfOnlyFailedHunksNamesWhatItOmits(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	withMax(t, 1100)
	res := failedPlan(40000)
	out, rpcErr := boundedReceipt(t.TempDir(), res, errors.New(commitErr))
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	text := out.Content[0].Text
	i := strings.Index(text, receiptLed)
	if i < 0 {
		t.Fatalf("no kept receipt named:\n%s", text)
	}
	path := strings.TrimSuffix(strings.TrimSpace(text[i+len(receiptLed):]), ".")
	fi, err := os.Stat(path)
	if err != nil || fi.Size() > keptReceiptMaxBytes {
		t.Fatalf("the kept receipt is %d bytes (%v), want at most %d", fi.Size(), err, keptReceiptMaxBytes)
	}
	b, _ := os.ReadFile(path)
	body := string(b)
	for _, want := range []string{"rename refused", "a.go", `"mutation": "partial"`, "40000 hunk(s), 2 file(s), 40000 failed"} {
		if !strings.Contains(body, want) {
			t.Errorf("the kept receipt lacks %q (its head: %.300s)", want, body)
		}
	}
	if strings.Contains(body, "[truncated:") {
		t.Errorf("the kept receipt was cut by the byte cap instead of by rows")
	}
	m := regexp.MustCompile(`(\d+) failed hunk\(s\) are not here: plan lines (\d+) to (\d+)`).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("the kept receipt names no omitted failed hunks")
	}
	omitted, _ := strconv.Atoi(m[1])
	first, _ := strconv.Atoi(m[2])
	last, _ := strconv.Atoi(m[3])
	if omitted <= 0 || omitted >= 40000 || last != 3+2*39999 || first != last-2*(omitted-1) {
		t.Errorf("the note says %d omitted, plan lines %d to %d; want the last %d of the 40000 (last line %d)", omitted, first, last, omitted, 3+2*39999)
	}
	// And the rows it does hold are the FIRST failed ones, in plan order.
	if !strings.Contains(body, "100 replace") {
		t.Errorf("the first failed hunk is not in the kept receipt")
	}
}

// #739 final pass, P2b. A long state home makes the kept receipt's name too big for
// any cut that keeps it; the answer that fits WITHOUT the name keeps the failed
// hunk, the written file and the commit error, and the kept file, which nobody
// can be told of, is removed. A normal state home names it in the same answer.
func TestAnOversizedKeptPathGivesWayToTheShallowestCutThatFitsWithoutIt(t *testing.T) {
	v, res := lostTail(t)
	for i := 0; i < 300; i++ {
		res.Hunks = append(res.Hunks, apply.HunkResult{Path: "a.go", Addr: fmt.Sprint(5000 + i), Op: "replace", Status: apply.StatusOK, Reason: "ok"})
	}
	answer := func() (callToolResult, bool) {
		out, rpcErr := checkedReceipt(t.TempDir(), res, v, errors.New(commitErr), true)
		if rpcErr != nil {
			t.Fatal(rpcErr)
		}
		_, ok := structuredOf(t, out)
		return out, ok
	}
	withMax(t, 1500)
	// s0: the smallest ceiling at which a structured answer exists when no kept
	// receipt can be made at all.
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	realKeep := keepReceipt
	keepReceipt = func(string, any) (string, error) { return "", errors.New("no room") }
	s0 := 0
	for c := 1500; c < 20000; c += 10 {
		MaxResultChars = c
		if !writeFloorFits() {
			continue
		}
		if _, ok := answer(); ok {
			s0 = c
			break
		}
	}
	keepReceipt = realKeep
	if s0 == 0 {
		t.Fatal("no ceiling gave a structured answer without a kept receipt")
	}

	// Test 1. At a normal state home and room for the name, the shallowest cut that
	// fits WITH the name is the answer, and it names the receipt.
	MaxResultChars = s0 + 1000
	if out, ok := answer(); !ok {
		t.Errorf("at ceiling %d the answer is not structured:\n%.200s", MaxResultChars, out.Content[0].Text)
	} else if sc, _ := structuredOf(t, out); sc.FullReceipt == "" {
		t.Errorf("at ceiling %d the answer does not name the kept receipt", MaxResultChars)
	}

	// Test 2. At a state home whose path is ~900 bytes, no cut fits with the name:
	// the shallowest cut that fits WITHOUT it is the answer, the receipt nobody can
	// be told of is removed, and the failed hunk, the written file and the commit
	// error are still in the answer.
	long := t.TempDir()
	for len(long) < 900 {
		long = filepath.Join(long, strings.Repeat("d", 100))
	}
	t.Setenv("XDG_STATE_HOME", long)
	MaxResultChars = s0
	out, ok := answer()
	if !ok {
		t.Fatalf("an oversized kept path made the answer the terminal sentence, losing the failed hunk and the commit error:\n%s", out.Content[0].Text)
	}
	if sc, _ := structuredOf(t, out); sc.FullReceipt != "" {
		t.Errorf("the answer names a kept receipt it cannot fit: %q", sc.FullReceipt)
	}
	if n := encodedSize(out); n > MaxResultChars {
		t.Errorf("the answer is %d bytes against %d", n, MaxResultChars)
	}
	all := out.Content[0].Text + fmt.Sprint(out.StructuredContent)
	for _, want := range []string{"commit failed", "a.go"} {
		if !strings.Contains(all, want) {
			t.Errorf("the answer lost %q", want)
		}
	}
	if !strings.Contains(string(out.StructuredContent.(json.RawMessage)), "rename refused") {
		t.Errorf("the answer lost the commit error")
	}
	if keptCount(t) != 0 {
		t.Errorf("the kept receipt nobody can be told of was left on disk")
	}
}

// #739 final pass, P3. The bound is the one the docs state: after a burst, every
// receipt younger than the grace period stays whatever the count, and the next
// save past the grace period brings the directory back to the newest 16.
func TestAfterABurstTheKeptReceiptsAreTheNewest16PlusThoseYoungerThanTheGrace(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for i := 0; i < keptReceiptsMax+4; i++ {
		if _, err := keepFullReceipt("r", nil); err != nil {
			t.Fatal(err)
		}
	}
	if n := keptCount(t); n != keptReceiptsMax+4 {
		t.Fatalf("after a burst %d receipts remain, want all %d (each younger than the grace)", n, keptReceiptsMax+4)
	}
	restore := trimGrace
	trimGrace = 0
	t.Cleanup(func() { trimGrace = restore })
	if _, err := keepFullReceipt("r", nil); err != nil {
		t.Fatal(err)
	}
	if n := keptCount(t); n != keptReceiptsMax {
		t.Errorf("after a save past the grace %d receipts remain, want %d", n, keptReceiptsMax)
	}
}
