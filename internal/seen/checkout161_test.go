package seen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-161 Decision 14. The ledger belongs to the checkout that holds the root:
// a record from a subdirectory is stored under the root's path below the top
// and loads back without it, so a read in one directory licenses a write from
// another.
func TestALedgerBelongsToTheCheckoutNotTheDirectory(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	sha := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	mk := func(dirs ...string) string {
		t.Helper()
		top := t.TempDir()
		for _, d := range dirs {
			if err := os.MkdirAll(filepath.Join(top, d), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		return top
	}
	has := func(l Ledger, k string) bool { _, ok := l[k]; return ok }
	rec := func(root string, ks ...string) {
		t.Helper()
		obs := map[string]Observation{}
		for _, k := range ks {
			obs[k] = Observation{SHA: sha}
		}
		if err := Record(root, obs); err != nil {
			t.Fatal(err)
		}
	}
	load := func(root string) Ledger {
		t.Helper()
		l, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		return l
	}

	t.Run("a record from the subdirectory loads from the top and back", func(t *testing.T) {
		top := mk(".git", "sub")
		sub := filepath.Join(top, "sub")
		rec(sub, "a.go")
		if !has(load(top), filepath.Join("sub", "a.go")) {
			t.Errorf("a record from sub is not sub/a.go at the top; the top holds %q", keys(load(top)))
		}
		if !has(load(sub), "a.go") {
			t.Errorf("a record from sub does not load in sub as a.go; sub holds %q", keys(load(sub)))
		}
	})

	t.Run("a record from the top loads in the subdirectory, and the rest of the top does not", func(t *testing.T) {
		top := mk(".git", "sub", "other")
		rec(top, filepath.Join("sub", "b.go"), "c.go", filepath.Join("other", "d.go"), filepath.Join("subway", "x.go"), "sub.go")
		l := load(filepath.Join(top, "sub"))
		if !has(l, "b.go") {
			t.Errorf("sub/b.go recorded at the top did not load in sub as b.go; sub holds %q", keys(l))
		}
		if len(l) != 1 {
			t.Errorf("sub loaded entries outside its own path: %q", keys(l))
		}
	})

	t.Run("a record from the subdirectory keeps what the top already held", func(t *testing.T) {
		top := mk(".git", "sub")
		rec(top, "c.go")
		rec(filepath.Join(top, "sub"), "a.go")
		if l := load(top); !has(l, "c.go") || !has(l, filepath.Join("sub", "a.go")) {
			t.Errorf("the checkout's ledger lost an entry; it holds %q", keys(l))
		}
	})

	t.Run("a drop from the subdirectory removes only its own entry", func(t *testing.T) {
		top := mk(".git", "sub")
		rec(top, "c.go", filepath.Join("sub", "a.go"))
		if err := Drop(filepath.Join(top, "sub"), []string{"a.go"}); err != nil {
			t.Fatal(err)
		}
		if l := load(top); !has(l, "c.go") || has(l, filepath.Join("sub", "a.go")) {
			t.Errorf("after the drop the checkout's ledger holds %q", keys(l))
		}
	})

	t.Run("a record from the subdirectory takes the checkout's seen.lock, not its own", func(t *testing.T) {
		top := mk(".git", "sub")
		rec(filepath.Join(top, "sub"), "a.go")
		topLock, err := state.Path(top, Name+".lock")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(topLock); err != nil {
			t.Errorf("the checkout's seen.lock was not taken: %v", err)
		}
		subLock, err := state.Path(filepath.Join(top, "sub"), Name+".lock")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(subLock); err == nil {
			t.Error("the subdirectory's own seen.lock was taken")
		}
	})

	t.Run("a ledger from a newer mrw at the top is left byte for byte", func(t *testing.T) {
		top := mk(".git", "sub")
		p, err := ReadPath(top)
		if err != nil {
			t.Fatal(err)
		}
		newer := []byte("#mrw-seen v7 v9.9.9\n" + sha + "  -  c.go\n")
		if err := os.WriteFile(p, newer, 0o600); err != nil {
			t.Fatal(err)
		}
		rec(filepath.Join(top, "sub"), "a.go")
		if got, err := os.ReadFile(p); err != nil || string(got) != string(newer) {
			t.Errorf("a record from sub wrote over a newer mrw's ledger: %q (%v)", got, err)
		}
		if _, isNewer, err := Newer(filepath.Join(top, "sub")); err != nil || !isNewer {
			t.Errorf("sub did not see the top's newer ledger: newer %v, err %v", isNewer, err)
		}
	})

	t.Run("a licence does not widen or change on the way across", func(t *testing.T) {
		top := mk(".git", "sub", "other")
		sub, other := filepath.Join(top, "sub"), filepath.Join(top, "other")
		if err := Record(sub, map[string]Observation{
			"a.go": {SHA: sha, Spans: [][2]int{{2, 3}}},
			"w.go": {SHA: sha, Spans: [][2]int{{4, 4}}, Written: true, Shown: [][2]int{}},
		}); err != nil {
			t.Fatal(err)
		}
		a := load(top)[filepath.Join("sub", "a.go")]
		if !a.ServedLine(2) || !a.ServedLine(3) || a.ServedLine(1) || a.ServedLine(5) {
			t.Errorf("lines 2-3 read in sub are served as %q at the top", a.Served())
		}
		w := load(sub)["w.go"]
		if !w.Written || !w.Covers(4, 4) || w.Covers(5, 5) || w.ServedLine(4) || len(w.Shown) != 0 {
			t.Errorf("a written, unshown observation came back as %+v", w)
		}
		// The same bytes at the same name in another directory are another key.
		if err := Record(other, map[string]Observation{"a.go": {SHA: sha, Spans: [][2]int{{9, 9}}}}); err != nil {
			t.Fatal(err)
		}
		if a := load(sub)["a.go"]; !a.ServedLine(2) || a.ServedLine(9) {
			t.Errorf("a read in other changed what sub holds for a.go: %q", a.Served())
		}
		if _, ok := load(other)["w.go"]; ok {
			t.Error("a file read in sub is licensed in other because the contents match")
		}
	})

	t.Run("a .git file counts as a checkout", func(t *testing.T) {
		top := mk("sub")
		if err := os.WriteFile(filepath.Join(top, ".git"), []byte("gitdir: /nowhere\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		rec(filepath.Join(top, "sub"), "a.go")
		if !has(load(top), filepath.Join("sub", "a.go")) {
			t.Errorf("a worktree's .git file did not make the top the checkout; the top holds %q", keys(load(top)))
		}
	})

	t.Run("a root with no .git above keeps its own ledger", func(t *testing.T) {
		top := mk("sub")
		rec(filepath.Join(top, "sub"), "a.go")
		if has(load(top), filepath.Join("sub", "a.go")) || has(load(top), "a.go") {
			t.Errorf("a directory with no .git shared its ledger with its parent: %q", keys(load(top)))
		}
		if !has(load(filepath.Join(top, "sub")), "a.go") {
			t.Errorf("a root with no .git lost its own entry")
		}
	})

	t.Run("a held write lock in the subdirectory holds the top", func(t *testing.T) {
		t.Setenv("MRW_WRITE_LOCK_TIMEOUT", "0")
		top := mk(".git", "sub")
		release, err := LockWrites(filepath.Join(top, "sub"))
		if err != nil {
			t.Fatal(err)
		}
		defer release()
		if r, err := LockWrites(top); err == nil {
			r()
			t.Error("a second writer at the top took the write lock while one held it from sub")
		}
	})
}

// The Codex review of ADR-161 T11: a key holding a line feed — a file name, or
// a directory name joined to it as the prefix — is never written as a second
// record that licenses a file nobody read.
func TestALineFeedInAKeyWritesNoSecondRecord(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	sha := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	top := t.TempDir()
	if err := os.MkdirAll(filepath.Join(top, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	forged := "x\n" + sha + "  -  victim.txt"
	if err := Record(top, map[string]Observation{forged: {SHA: sha, Spans: [][2]int{{2, 2}}}, "a.txt": {SHA: sha}}); err != nil {
		t.Fatal(err)
	}
	l, err := Load(top)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := l["victim.txt"]; ok {
		t.Fatalf("a line feed in a key wrote a record licensing victim.txt: %v", l)
	}
	if _, ok := l["a.txt"]; !ok {
		t.Errorf("the ordinary key beside it was lost: %v", l)
	}
}
