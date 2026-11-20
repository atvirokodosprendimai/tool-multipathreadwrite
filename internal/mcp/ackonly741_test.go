package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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

// fixture741 is a 500-line file with its three checkpoints held pending, and the
// ledger it starts from.
func fixture741(t *testing.T) (root string, spans map[string][2]int, sum string) {
	t.Helper()
	root = t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	var body string
	for i := 1; i <= 500; i++ {
		body += fmt.Sprintf("line %d\n", i)
	}
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s := sha256.Sum256([]byte(body))
	sum = hex.EncodeToString(s[:])
	_, spans = interleave(served(1, 500), nil)
	if err := hold(root, "f.txt", sum, spans); err != nil {
		t.Fatal(err)
	}
	return root, spans, sum
}

func recorded741(t *testing.T, root string) (seen.Observation, bool) {
	t.Helper()
	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	o, ok := l["f.txt"]
	return o, ok
}

// #741 review. A no-match answer that only becomes an error at the final size
// check (the ceiling) must not have promoted its acknowledgement: it is
// promoted after that decision. A 1000-Z grep matches nothing, and its answer
// does not fit 600 bytes.
func TestAnAnswerRefusedForItsSizeAfterTheReadPromotesNoAcknowledgement(t *testing.T) {
	root, spans, _ := fixture741(t)
	var ck string
	for c, sp := range spans {
		if sp[0] == 1 {
			ck = c
		}
	}
	withCeiling(t, 600)
	res := call(t, root, "mrw_read", map[string]any{"ack": []string{ck}, "grep": strings.Repeat("Z", 1000)})
	if res["isError"] != true {
		t.Fatalf("the answer was not refused for its size: %v", res)
	}
	if _, ok := recorded741(t, root); ok {
		t.Error("a call whose answer was refused for its size promoted its acknowledgement")
	}
}

// #741 review. A serve holds new checkpoints and a full store drops its oldest,
// which may be the one the call acknowledges: with 512 pending, a read that
// acknowledges the oldest still promotes it, because it was taken before the
// serve.
func TestAReadThatEvictsTheCheckpointItAcknowledgesStillPromotesIt(t *testing.T) {
	root, spans, sum := fixture741(t)
	for i := 0; i < maxPending/3+2; i++ {
		_, more := interleave(served(1, 500), nil)
		if err := hold(root, "f.txt", sum, more); err != nil {
			t.Fatal(err)
		}
	}
	store, err := loadPending(root)
	if err != nil || len(store) != maxPending {
		t.Fatalf("the store holds %d entries (%v), want %d", len(store), err, maxPending)
	}
	oldest := ""
	for ck, p := range store {
		if oldest == "" || p.Seq < store[oldest].Seq {
			oldest = ck
		}
	}
	span := [2]int{store[oldest].Start, store[oldest].End}
	_ = spans
	if res := call(t, root, "mrw_read", map[string]any{"specs": []string{"f.txt:1-3"}, "ack": []string{oldest}}); res["isError"] == true {
		t.Fatalf("the read failed: %v", res)
	}
	o, ok := recorded741(t, root)
	if !ok || !o.Covers(span[0], span[1]) {
		t.Errorf("the oldest checkpoint %v was evicted before it was promoted: %+v", span, o)
	}
}

// #741 review. A quiet read licenses lines in the ledger as it serves; if the
// acknowledgements cannot be read it must not have recorded anything: the
// pending store is read before the call serves.
func TestAQuietReadWhoseAcknowledgementsCannotBeReadRecordsNothing(t *testing.T) {
	root, _, _ := fixture741(t)
	prev := readPendingFile
	readPendingFile = func(string) ([]byte, error) { return nil, errors.New("input/output error") }
	t.Cleanup(func() { readPendingFile = prev })
	res, rpcErr := readTool(root, json.RawMessage(`{"specs":["f.txt"],"quiet":true,"ack":["x"]}`))
	readPendingFile = prev
	if rpcErr == nil && !res.IsError {
		t.Fatalf("the failure was not reported: %+v", res)
	}
	if _, ok := recorded741(t, root); ok {
		t.Error("a quiet read licensed lines although its acknowledgements could not be read")
	}
}

// #741 beside #733. One call carries two acknowledgements, one for a readable file
// and one for a file that cannot be read just now. Answered with an error, the
// call changes nothing: both stay pending and the ledger is empty. Answered
// without one, the readable ack is consumed and recorded, the unreadable one
// stays pending and the answer names it (every kept id, as the note says).
func TestAReadThatKeepsAnAckPromotesTheOthersOnlyWhenAnswered(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("a mode-000 file is still readable here")
	}
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	for _, n := range []string{"ok.txt", "locked.txt"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte("a\nb\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		sha, err := currentSHA(root, n)
		if err != nil {
			t.Fatal(err)
		}
		if err := hold(root, n, sha, map[string][2]int{"ck-" + n: {1, 2}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(filepath.Join(root, "locked.txt"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "locked.txt"), 0o644) })
	acks := []string{"ck-ok.txt", "ck-locked.txt"}
	pendingIDs := func() string {
		store, err := loadPending(root)
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for id := range store {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		return strings.Join(ids, ",")
	}

	res := call(t, root, "mrw_read", map[string]any{"ack": acks})
	if res["isError"] != true {
		t.Fatalf("an ack with nothing to read was not an error: %v", res)
	}
	if got := pendingIDs(); got != "ck-locked.txt,ck-ok.txt" {
		t.Errorf("an error answer changed the pending store: %q", got)
	}
	if l, _ := seen.Load(root); len(l) != 0 {
		t.Errorf("an error answer recorded %v", l)
	}

	res = call(t, root, "mrw_read", map[string]any{"specs": []string{"ok.txt:1"}, "ack": acks})
	text := firstText(t, res)
	if res["isError"] == true || !strings.Contains(text, "ck-locked.txt") || !strings.Contains(text, "kept") {
		t.Errorf("the answer does not name the kept ack (isError %v):\n%.400s", res["isError"], text)
	}
	if got := pendingIDs(); !strings.Contains(got, "ck-locked.txt") || strings.Contains(got, "ck-ok.txt") {
		t.Errorf("after the answered call pending is %q, want the kept one and not the consumed one", got)
	}
	if l, _ := seen.Load(root); l["ok.txt"].SHA == "" {
		t.Errorf("the readable ack was not recorded: %v", l)
	}
}
