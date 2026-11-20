package bench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAnAttemptThatCommitsIsGradedAgainstItsBase: an agent that commits its
// work moves HEAD. The oracle grades against the base the runner captured,
// so an unrelated edit committed beside the fix is still outside, and an
// exact task's reference is rebuilt from the base, not from the agent's
// commit (#706).
func TestAnAttemptThatCommitsIsGradedAgainstItsBase(t *testing.T) {
	t.Run("a committed stray edit is outside", func(t *testing.T) {
		// The fix (beta to gamma) and a stray edit (alpha to ALPHA).
		r := committedAttempt(t, false, "@@ -1,4 +1,4 @@\n-alpha\n-beta\n+ALPHA\n+gamma\n delta\n epsilon\n")
		if r.Oracle.Solved || len(r.Oracle.Outside) == 0 {
			t.Errorf("a committed stray edit was graded %+v, want it outside and the attempt not solved", r.Oracle)
		}
		if r.Diff.Files != 1 || r.Diff.Insertions != 2 || r.Diff.Deletions != 2 {
			t.Errorf("the diff stats of a committed change are %+v, want one file, two lines in and out", r.Diff)
		}
	})
	t.Run("a committed exact fix is the reference result", func(t *testing.T) {
		// A fix that adds a line: rebuilt from the agent's commit instead
		// of the base, the reference would add it a second time.
		r := committedAttempt(t, true, "@@ -1,4 +1,5 @@\n alpha\n beta\n+gamma\n delta\n epsilon\n")
		if !r.Oracle.Solved || len(r.Oracle.Missing) != 0 {
			t.Errorf("a committed reference fix of an exact task was graded %+v, want it solved", r.Oracle)
		}
	})
}

// committedAttempt runs one attempt of the fixture task whose stub agent
// applies hunk to notes.txt and commits the result, and returns its receipt.
// The stub leaves a marker only when the commit succeeded, so a case cannot
// pass on an uncommitted tree.
// An exact task takes hunk as its reference fix too, with gamma required.
func committedAttempt(t *testing.T, exact bool, hunk string) Receipt {
	t.Helper()
	opts, _ := runOptions(t)
	work := filepath.Dir(opts.Out)
	patch := filepath.Join(work, "agent.patch")
	if err := os.WriteFile(patch, []byte("diff --git a/notes.txt b/notes.txt\n--- a/notes.txt\n+++ b/notes.txt\n"+hunk), 0o644); err != nil {
		t.Fatal(err)
	}
	claude, _ := stubClaude(t, work, patch, "done")
	b, err := os.ReadFile(claude)
	if err != nil {
		t.Fatal(err)
	}
	apply := `git apply "` + patch + `" >/dev/null 2>&1`
	if !strings.Contains(string(b), apply) {
		t.Fatal("the stub no longer applies its patch the way this test expects")
	}
	script := strings.Replace(string(b), apply, apply+"\ngit -c user.name=agent -c user.email=agent@localhost -c commit.gpgsign=false commit -q --no-verify -am agent >/dev/null 2>&1 || exit 9\ntouch \""+filepath.Join(work, "committed")+"\"", 1)
	if err := os.WriteFile(claude, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	opts.Claude, opts.Configs = claude, []string{"A"}
	if exact {
		dir := opts.Tasks[0].Dir
		def := `{"id":"fix-one","shape":"fixture","repo":"fixture","exact":true,"instruction":"Add gamma after beta in notes.txt.","require":[{"file":"notes.txt","match":"(?m)^gamma$"}]}`
		if err := os.WriteFile(filepath.Join(dir, "task.json"), []byte(def), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(patch, filepath.Join(dir, "reference.patch")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(dir, "reference.patch"), patch); err != nil {
			t.Fatal(err)
		}
		task, err := LoadTask(dir)
		if err != nil {
			t.Fatal(err)
		}
		opts.Tasks[0] = task
	}
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(work, "committed")); err != nil {
		t.Fatalf("the stub agent did not commit: %v", err)
	}
	got := receipts(t, opts.Out)
	if len(got) != 1 {
		t.Fatalf("receipts: %d, want 1", len(got))
	}
	var r Receipt
	for name, b := range got {
		if err := json.Unmarshal(b, &r); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	return r
}
