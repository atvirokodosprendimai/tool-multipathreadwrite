package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer"
)

// landCompiled lands compiled plan text the way the write command lands an
// apply_patch: Computed, so the stale-number refusal does not judge it.
func landCompiled(t *testing.T, root string, compiled []byte) *writer.Landed {
	t.Helper()
	hunks, err := plan.Parse(bytes.NewReader(compiled))
	if err != nil {
		t.Fatal(err)
	}
	var in []apply.Input
	for _, h := range hunks {
		in = append(in, writer.InputFromHunk(h, h.Path))
	}
	prep, err := writer.Prepare(writer.Request{Root: root, In: in, Check: writer.CheckOff, Opts: apply.Options{Computed: true}})
	if err != nil {
		t.Fatal(err)
	}
	land, err := prep.Land()
	if err != nil {
		t.Fatal(err)
	}
	return land
}

// Issue #729. apply_patch compiles before the write lock and its numbers are the
// file as it was then. When another mrw writer lands a line above the target in
// between, the ledger holds the new file as wholly known, and a one-line match
// carried no sha= and no anchor, so the compiled hunk replaced the wrong line
// with status ok. It now carries the sha of the file it was compiled from.
func TestAnApplyPatchCompiledBeforeAnotherWriterLandsIsRefused(t *testing.T) {
	const patch = "*** Begin Patch\n*** Update File: f.txt\n@@\n-l3\n+L3\n*** End Patch\n"
	root := t.TempDir()
	f := filepath.Join(root, "f.txt")
	if err := os.WriteFile(f, []byte("l1\nl2\nl3\nl4\nl5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "read", "f.txt"); code != 0 {
		t.Fatalf("read: %d %s", code, out)
	}
	compiled, err := ingest.CompileApplyPatch(root, []byte(patch))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(compiled), "sha=") {
		t.Errorf("the compiled one-line replace carries no sha=:\n%s", compiled)
	}
	// another mrw writer inserts a line at the top, between compile and land
	pf := filepath.Join(t.TempDir(), "other")
	if err := os.WriteFile(pf, []byte("@@ f.txt 0 insert-after\nNEW\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "write", "--no-check", pf); code != 0 {
		t.Fatalf("other writer: %d %s", code, out)
	}
	land := landCompiled(t, root, compiled)
	got, _ := os.ReadFile(f)
	if land.Res.Applied || string(got) != "NEW\nl1\nl2\nl3\nl4\nl5\n" {
		t.Errorf("the stale compiled plan applied (applied=%v), file now %q", land.Res.Applied, got)
	}
	if len(land.Res.Hunks) != 1 || !strings.Contains(land.Res.Hunks[0].Reason, "file changed") {
		t.Errorf("the refusal does not say the file changed: %+v", land.Res.Hunks)
	}
}

// The pair, over every file shape: the compiled sha= is the engine's own hash of
// the file, which is the sha256 of its bytes, so a plan nothing moved is never
// refused.
func TestAnApplyPatchWithNothingMovedStillApplies(t *testing.T) {
	for name, content := range map[string]string{
		"lf":             "l1\nl2\nl3\nl4\n",
		"crlf":           "l1\r\nl2\r\nl3\r\nl4\r\n",
		"no final eol":   "l1\nl2\nl3\nl4",
		"leading blanks": "\n\nl1\nl2\nl3\nl4\n",
		"cr only":        "l1\rl2\rl3\rl4\r",
		"blank last":     "l1\nl2\nl3\nl4\n\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			f := filepath.Join(root, "f.txt")
			if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			if out, code := runIn(t, root, "read", "f.txt"); code != 0 {
				t.Fatalf("read: %d %s", code, out)
			}
			compiled, err := ingest.CompileApplyPatch(root, []byte("*** Begin Patch\n*** Update File: f.txt\n@@\n-l3\n+L3\n*** End Patch\n"))
			if err != nil {
				t.Fatal(err)
			}
			land := landCompiled(t, root, compiled)
			if !land.Res.Applied {
				t.Errorf("an untouched file's plan was refused: %+v", land.Res.Hunks)
			}
		})
	}
}

// Review of #736. The test above lands the compiled plan through a helper that
// supplies the sha itself, so it cannot see the public command drop it. This one
// goes through `mrw write --format apply_patch`: another writer's change lands
// between the compile and the lock (ingest.AfterCompile), and the command must
// refuse with "file changed" and leave the file as that writer made it.
func TestTheWriteCommandForwardsTheCompiledShaToTheLock(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	f := filepath.Join(root, "f.txt")
	if err := os.WriteFile(f, []byte("l1\nl2\nl3\nl4\nl5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "read", "f.txt"); code != 0 {
		t.Fatalf("read: %d %s", code, out)
	}
	const other = "NEW\nl1\nl2\nl3\nl4\nl5\n"
	ingest.AfterCompile = func() {
		// What another mrw write leaves: the new content, wholly known to the ledger.
		if err := os.WriteFile(f, []byte(other), 0o644); err != nil {
			t.Error(err)
		}
		if err := seen.Record(root, map[string]seen.Observation{"f.txt": {SHA: seen.SHA([]byte(other)), Written: true}}); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { ingest.AfterCompile = nil })
	patch := filepath.Join(t.TempDir(), "patch")
	if err := os.WriteFile(patch, []byte("*** Begin Patch\n*** Update File: f.txt\n@@\n-l3\n+L3\n*** End Patch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runIn(t, root, "write", "--format", "apply_patch", patch)
	got, _ := os.ReadFile(f)
	if code != exitNotApplied || string(got) != other || !strings.Contains(out, "file changed") {
		t.Errorf("exit %d, want %d and the other writer's file untouched; file now %q:\n%s", code, exitNotApplied, got, out)
	}
}
