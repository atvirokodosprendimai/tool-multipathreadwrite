package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// dirReadCause is the error text a read of a directory ends with: Windows
// answers ReadFile on a directory "Incorrect function", unix "is a directory".
var dirReadCause = map[bool]string{true: "Incorrect function", false: "is a directory"}[runtime.GOOS == "windows"]

// A directory named in a plan is a load error that has a hunk (#703). Under a
// ceiling too small for the receipt, the bounded answer kept the error's own
// words only when the receipt had no hunks, so it dropped the cause and told
// the caller to send fewer hunks. The cause is kept whatever the hunk count.
func TestAnErrorWithHunksKeepsItsCauseUnderASmallCeiling(t *testing.T) {
	root, _ := checkout(t, "x.txt", "a\n")
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	withCeiling(t, 1100)
	res := call(t, root, "mrw_write", map[string]any{"plan": "@@ d 1 replace\nz\n"})
	text := firstText(t, res)
	if res["isError"] != true || !strings.Contains(text, dirReadCause) || strings.Contains(text, "Send fewer hunks") {
		t.Fatalf("the bounded answer lost the directory's own words (isError %v):\n%s", res["isError"], text)
	}
}

// The counts stay beside the cause: twenty directories under a small ceiling
// answer "20 of 20 hunk(s) failed" and the first error's words (AGENTS.md).
func TestTheBoundedAnswerKeepsTheFailureCountsAndTheFirstCause(t *testing.T) {
	root, _ := checkout(t, "x.txt", "a\n")
	var plan strings.Builder
	for i := 0; i < 20; i++ {
		d := fmt.Sprintf("dir%02d", i)
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&plan, "@@ %s 1 replace\nz\n", d)
	}
	withCeiling(t, 1100)
	text := firstText(t, call(t, root, "mrw_write", map[string]any{"plan": plan.String()}))
	if !strings.Contains(text, "20 of 20 hunk(s) failed") || !strings.Contains(text, dirReadCause) {
		t.Fatalf("the bounded answer lost the counts or the cause:\n%s", text)
	}
}

// A long directory name must not push the cause out: the path loses its
// middle, "is a directory" stays, and so do the counts.
func TestTheBoundedAnswerShortensALongPathNotTheCause(t *testing.T) {
	root, _ := checkout(t, "x.txt", "a\n")
	part := strings.Repeat("long-directory-name-", 10)
	name := part + "/" + part + "/" + part
	if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
		t.Fatal(err)
	}
	withCeiling(t, 1100)
	text := firstText(t, call(t, root, "mrw_write", map[string]any{"plan": "@@ " + name + " 1 replace\nz\n"}))
	if !strings.Contains(text, "1 of 1 hunk(s) failed and nothing was written: ") || !strings.Contains(text, dirReadCause) || !strings.Contains(text, " … ") {
		t.Fatalf("the bounded answer lost the counts or the cause, or did not shorten the path:\n%s", text)
	}
}
