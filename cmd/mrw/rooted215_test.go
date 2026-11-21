package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-215 Decisions 5 and 6, driven through the built binary. mrw iter add
// keeps a rooted spelling, because a read takes a real path; a plan that
// expands a pointer to it is refused, and the refusal says which entry holds it.
func TestAnExpandedWorkingSetPointerThatIsRootedIsRefused(t *testing.T) {
	const entry = "the working-set entry @1 holds it: mrw iter lists it; add it again without the leading separator"
	const sentence = "is absolute, and every path in a plan is relative to the root"

	fresh := func(t *testing.T) *keptRig {
		t.Helper()
		k := newKeptRig(t)
		if err := os.MkdirAll(filepath.Join(k.root, "dd"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(k.root, "dd", "zz.txt"), []byte("a\nb\nc\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, e, code := k.run("", nil, "iter", "add", "/dd/zz.txt:1"); code != 0 {
			t.Fatalf("iter add keeps a rooted spelling: exit %d: %s", code, e)
		}
		if _, e, code := k.run("", nil, "read", "dd/zz.txt"); code != 0 {
			t.Fatalf("read: exit %d: %s", code, e)
		}
		return k
	}
	unchanged := func(t *testing.T, k *keptRig) {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(k.root, "dd", "zz.txt"))
		if err != nil || string(b) != "a\nb\nc\n" {
			t.Errorf("the file changed: %q %v", b, err)
		}
	}

	for _, c := range []struct {
		name  string
		stdin string
		args  []string
	}{
		{"a hunk path", "@@ @1 1 replace\nX\n", []string{"write", "--no-check", "-"}},
		{"--create", "X\n", []string{"write", "--no-check", "--create", "@1"}},
		{"an apply_patch Add File", "*** Begin Patch\n*** Add File: @1\n+X\n*** End Patch\n", []string{"write", "--no-check", "--format=apply_patch", "-"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			k := fresh(t)
			o, e, code := k.run(c.stdin, nil, c.args...)
			if code != 1 || !strings.Contains(o+e, sentence) || !strings.Contains(o+e, entry) {
				t.Errorf("exit %d, want 1 with %q and %q:\n%s%s", code, sentence, entry, o, e)
			}
			unchanged(t, k)
		})
	}

	// A piped plan with a rooted hunk that ends with nothing in the tree is kept
	// and named, now at exit 1 (Decision 6).
	t.Run("a piped rooted plan is kept and named", func(t *testing.T) {
		k := newKeptRig(t)
		if err := os.MkdirAll(filepath.Join(k.root, "dd"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(k.root, "dd", "zz.txt"), []byte("a\nb\nc\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, e, code := k.run("", nil, "read", "dd/zz.txt"); code != 0 {
			t.Fatalf("read: exit %d: %s", code, e)
		}
		o, e, code := k.run("@@ /dd/zz.txt 1 replace\nX\n", nil, "write", "--json", "--no-check", "-")
		if code != 1 || !strings.Contains(o, sentence) || !strings.Contains(o, `"refused_plan"`) {
			t.Errorf("exit %d, want 1 with the sentence and refused_plan:\n%s%s", code, o, e)
		}
		k.onlyKept("the refused rooted plan")
	})

	// What is not rooted keeps working: a plan maker that is given an absolute
	// path inside the root emits the relative one.
	t.Run("--sub over an absolute argument emits a relative hunk", func(t *testing.T) {
		k := newKeptRig(t)
		abs := filepath.Join(k.root, "dd", "zz.txt")
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("a\nb\nc\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, e, code := k.run("", nil, "read", "dd/zz.txt"); code != 0 {
			t.Fatalf("read: exit %d: %s", code, e)
		}
		o, e, code := k.run("", nil, "write", "--no-check", "--grep", "a", "--sub", "Z", abs)
		b, _ := os.ReadFile(abs)
		if code != 0 || string(b) != "Z\nb\nc\n" {
			t.Errorf("exit %d, file %q:\n%s%s", code, b, o, e)
		}
	})
	t.Run("--rewrite over an absolute argument emits a relative hunk", func(t *testing.T) {
		if _, err := exec.LookPath("ast-grep"); err != nil {
			return // no ast-grep here: the --sub case above covers the plan makers' shared path
		}
		k := newKeptRig(t)
		abs := filepath.Join(k.root, "dd", "f.go")
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("package p\n\nfunc f() { foo(1) }\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		o, e, code := k.run("", nil, "write", "--no-check", "--ast-grep", "foo($A)", "--rewrite", "bar($A)", abs)
		b, _ := os.ReadFile(abs)
		if code != 0 || !strings.Contains(string(b), "bar(1)") {
			t.Errorf("exit %d, file %q:\n%s%s", code, b, o, e)
		}
	})
}
