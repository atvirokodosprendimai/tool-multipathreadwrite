package mcp

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// keptFile is the path the terminal sentence names, read from its text.
func keptFile(t *testing.T, text string) string {
	t.Helper()
	i := strings.Index(text, receiptLed)
	if i < 0 {
		t.Fatalf("no kept receipt named:\n%s", text)
	}
	return strings.TrimSuffix(strings.TrimSpace(text[i+len(receiptLed):]), ".")
}

// Review of #739, 1. A state home whose path alone is longer than the ceiling
// leaves no room to name the kept receipt: the answer is the terminal refusal that
// names the counts, and the receipt that nobody can be told of is not left behind.
func TestAKeptReceiptThatCannotBeNamedWithinTheCeilingIsRemoved(t *testing.T) {
	home := t.TempDir()
	for range 10 {
		home = filepath.Join(home, strings.Repeat("d", 80))
	}
	t.Setenv("XDG_STATE_HOME", home)
	withMax(t, 1100)
	out, rpcErr := boundedReceipt(t.TempDir(), partialResult(60), errors.New(commitErr))
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	text := out.Content[0].Text
	if strings.Contains(text, receiptLed) || !strings.Contains(text, "PARTIALLY APPLIED") || encodedSize(out) > MaxResultChars {
		t.Errorf("the refusal names a path that does not fit, or lost its counts (%d bytes):\n%s", encodedSize(out), text)
	}
	if n := keptCount(t); n != 0 {
		t.Errorf("%d kept receipt(s) remain that no answer names", n)
	}
	// The pair: with a short state home the same plan names its kept receipt.
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	out, _ = boundedReceipt(t.TempDir(), partialResult(60), errors.New(commitErr))
	if p := keptFile(t, out.Content[0].Text); !strings.HasSuffix(p, ".receipt") {
		t.Errorf("path %q", p)
	}
}

// Review of #739, 2. Other writers publishing and trimming while one is between
// writing its receipt and naming it must not take it: the path an answer names
// exists when the answer leaves.
func TestAKeptReceiptStillExistsAfterSixteenOthersPublishMeanwhile(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	restoreGrace, restoreHook := trimGrace, beforePublish
	t.Cleanup(func() { trimGrace, beforePublish = restoreGrace, restoreHook })
	trimGrace = 0 // the bare count: only the name and the order protect it
	beforePublish = func() {
		beforePublish = func() {}
		for range keptReceiptsMax + 1 {
			if _, err := keepFullReceipt("another writer's", nil); err != nil {
				t.Error(err)
			}
		}
	}
	path, err := keepFullReceipt("mine", nil)
	if err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(path); err != nil || !strings.Contains(string(b), "mine") {
		t.Fatalf("the reported receipt is gone or not mine (%v)", err)
	}
	// The next writer to publish trims: what this one named must survive it, since
	// it was stamped after the receipts that were published meanwhile.
	if _, err := keepFullReceipt("the next writer", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the reported receipt is gone once the next writer trimmed: %v", err)
	}
	if n := keptCount(t); n > keptReceiptsMax {
		t.Errorf("%d receipts stay, want at most %d", n, keptReceiptsMax)
	}
}

// Review of #739, 5. A receipt over the cap loses its ok and skipped rows first:
// a failed hunk at the end of more than a mebibyte of rows survives, and the file
// stays within the cap with its note.
func TestAKeptReceiptOverTheCapKeepsEveryFailedHunk(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	withMax(t, 1100)
	res := partialResult(9000)
	res.Hunks[len(res.Hunks)-1].Reason = "LATE-FAILURE-REASON"
	out, rpcErr := boundedReceipt(t.TempDir(), res, errors.New(commitErr))
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	path := keptFile(t, out.Content[0].Text)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) > keptReceiptMaxBytes {
		t.Errorf("the kept receipt is %d bytes, over the %d cap", len(b), keptReceiptMaxBytes)
	}
	s := string(b)
	if !strings.Contains(s, "LATE-FAILURE-REASON") || !strings.Contains(s, "the kept receipt is over") || strings.Contains(s, "[truncated:") {
		t.Errorf("the failed hunk did not survive the cap, or the cut was not by rows:\n%.400s", s)
	}
}

// Review of #739, 6. No field that carries text of a file is kept.
func TestAKeptReceiptHoldsNoFileContent(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	withMax(t, 1100)
	res := partialResult(60)
	res.Hunks = append(res.Hunks, apply.HunkResult{
		Path: "c.go", Addr: "1", Op: "delete", Status: apply.StatusOK,
		Before: []string{"SECRET-BEFORE"}, After: []string{"SECRET-AFTER"},
		RemovedFirst: "SECRET-FIRST", RemovedLast: "SECRET-LAST",
		Echo: []string{"SECRET-ECHO"}, Closer: "SECRET-CLOSER",
		CopiedFirst: "SECRET-COPIED", CopiedLast: "SECRET-COPIED-LAST",
	})
	out, _ := boundedReceipt(t.TempDir(), res, errors.New(commitErr))
	b, err := os.ReadFile(keptFile(t, out.Content[0].Text))
	if err != nil || strings.Contains(string(b), "SECRET") {
		t.Errorf("the kept receipt holds file content (%v):\n%s", err, b)
	}
}
