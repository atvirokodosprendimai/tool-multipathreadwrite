package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// directCall742 is one tools/call straight to the handler, so a JSON-RPC error comes
// back as a value instead of failing the test.
func directCall742(t *testing.T, root, tool string, args map[string]any) (callToolResult, *rpcError) {
	t.Helper()
	params, err := json.Marshal(map[string]any{"name": tool, "arguments": args})
	if err != nil {
		t.Fatal(err)
	}
	return callToolCtx(context.Background(), root, params, false, nil)
}

// holdFile files one checkpoint for a file as it is now.
func holdFile(t *testing.T, root, name, ck string) {
	t.Helper()
	sha, err := currentSHA(root, name)
	if err != nil {
		t.Fatal(err)
	}
	if err := hold(root, name, sha, map[string][2]int{ck: {1, 2}}); err != nil {
		t.Fatal(err)
	}
}

func writeFile742(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func skipIfRoot(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("a mode bit does not stop this user")
	}
}

// #742 final pass, 1. A call whose acknowledgement commit fails grants nothing:
// the pending store is saved before any ledger record, and the quiet read's own
// record waits for the same step, so the error answer leaves the ledger as it was.
func TestAFailedAckCommitGrantsNoLicence(t *testing.T) {
	skipIfRoot(t)
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	writeFile742(t, root, "a.txt", "a\nb\n")
	writeFile742(t, root, "b.txt", "a\nb\n")
	holdFile(t, root, "a.txt", "ck-a")
	p, err := pendingPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(p, 0o644) })
	_, rpcErr := directCall742(t, root, "mrw_read", map[string]any{"specs": []string{"b.txt"}, "quiet": true, "ack": []string{"ck-a"}})
	if rpcErr == nil {
		t.Skip("the pending store could be saved here, so the commit did not fail")
	}
	if l, _ := seen.Load(root); len(l) != 0 {
		t.Errorf("a call that ended in an error granted licences: %v", l)
	}
}

// #742 final pass, 2. What the call says about an acknowledgement is what it does:
// commit acts on what prepare saw, whatever the file's mode does in between.
func TestACommitFollowsWhatPrepareSaw(t *testing.T) {
	skipIfRoot(t)
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	writeFile742(t, root, "a.txt", "a\nb\n")
	holdFile(t, root, "a.txt", "ck-a")
	if err := os.Chmod(filepath.Join(root, "a.txt"), 0o000); err != nil {
		t.Fatal(err)
	}
	b, err := prepareAcks(root, []string{"ck-a"})
	if err != nil || len(b.kept) != 1 {
		t.Fatalf("prepare: kept=%v err=%v; want ck-a kept", b.kept, err)
	}
	if err := os.Chmod(filepath.Join(root, "a.txt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := b.commitAll(root, nil); err != nil {
		t.Fatal(err)
	}
	store, _ := loadPending(root)
	if _, ok := store["ck-a"]; !ok {
		t.Errorf("an id the call reported kept was consumed")
	}
	if l, _ := seen.Load(root); len(l) != 0 {
		t.Errorf("a kept id licensed %v", l)
	}
}

// #742 final pass, 3. A full queue does not evict an id the same call keeps.
func TestAKeptAckSurvivesTheEvictionOfTheSameCall(t *testing.T) {
	skipIfRoot(t)
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	for _, n := range []string{"kept.txt", "ok.txt", "fill.txt", "fresh.txt"} {
		writeFile742(t, root, n, "a\nb\n")
	}
	holdFile(t, root, "kept.txt", "ck-kept")
	holdFile(t, root, "ok.txt", "ck-ok")
	fill := map[string][2]int{}
	for i := 0; i < maxPending-2; i++ {
		fill[fmt.Sprintf("ck-fill-%03d", i)] = [2]int{1, 2}
	}
	sha, _ := currentSHA(root, "fill.txt")
	if err := hold(root, "fill.txt", sha, fill); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(root, "kept.txt"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "kept.txt"), 0o644) })
	res, rpcErr := directCall742(t, root, "mrw_read", map[string]any{"specs": []string{"fresh.txt"}, "ack": []string{"ck-kept", "ck-ok"}})
	if rpcErr != nil || res.IsError {
		t.Fatalf("the read failed: %v %v", rpcErr, res)
	}
	if text := res.Content[0].Text; !strings.Contains(text, "ck-kept") {
		t.Errorf("the answer does not name the kept id:\n%.300s", text)
	}
	store, _ := loadPending(root)
	if _, ok := store["ck-kept"]; !ok {
		t.Errorf("the kept id was evicted by the same call's checkpoints")
	}
}

// #742 final pass, 4. A name that cleans to a recordable one gets checkpoints on
// every page: the observation key decides, not the spelling typed.
func TestAPagedReadOfANameThatCleansToARecordableOneIsMarked(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	var body strings.Builder
	for i := 1; i <= 400; i++ {
		fmt.Fprintf(&body, "line %05d of the fixture\n", i)
	}
	writeFile742(t, root, "f.txt", body.String())
	withCeiling(t, 4000)
	res, rpcErr := directCall742(t, root, "mrw_read", map[string]any{"specs": []string{"a\nb/../f.txt"}})
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	text := res.Content[0].Text
	if !strings.Contains(text, "-- PARTIAL:") {
		t.Skipf("the read was not paged:\n%.300s", text)
	}
	if !strings.Contains(text, "-- ck ") || strings.Contains(text, "name holds") {
		t.Errorf("a recordable file was served without checkpoints:\n%.500s", text)
	}
}

// #742 final pass, 5. A read with no acknowledgements takes no pending-store lock.
func TestAReadWithNoAckDoesNotWaitOnThePendingStoreLock(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	writeFile742(t, root, "f.txt", "a\nb\n")
	release, err := state.Hold(root, pendingLock)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = directCall742(t, root, "mrw_read", map[string]any{"grep": "NO_MATCH"})
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a read with no ack waited on the pending-store lock")
	}
}
