//go:build !windows

package rooted

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func realRoot727(t *testing.T) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return real
}

func slurp727(t *testing.T, f *os.File) string {
	t.Helper()
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func notRoot727(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root reads a directory whatever its mode")
	}
}

// #727 review P2-1. os.Root opens each directory for reading, so a directory
// that may be searched but not read (chmod 111) refused a file that can be
// read. The open falls back to the name, with an identity check.
func TestAFileBelowASearchOnlyDirectoryIsStillRead(t *testing.T) {
	notRoot727(t)
	root := realRoot727(t)
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.txt"), []byte("A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sub, 0o111); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })
	f, _, err := OpenRead(root, filepath.Join(sub, "a.txt"))
	if err != nil {
		t.Fatalf("a readable file below a search-only directory: %v", err)
	}
	if got := slurp727(t, f); got != "A\n" {
		t.Errorf("read %q", got)
	}
}

// A root at mode 0311 cannot be opened as an os.Root at all.
func TestAFileBelowARootThatCannotBeReadIsStillRead(t *testing.T) {
	notRoot727(t)
	root := realRoot727(t)
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o311); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	f, _, err := OpenRead(root, filepath.Join(root, "a.txt"))
	if err != nil {
		t.Fatalf("a readable file in a root at 0311: %v", err)
	}
	if got := slurp727(t, f); got != "A\n" {
		t.Errorf("read %q", got)
	}
}

// A directory swapped for a link out of the root BEFORE the by-name open of the
// fallback is refused, because the opened descriptor says where it is now: the
// path of the file the descriptor names is outside the root. One swapped for a
// link to another directory of the root is read: what is opened is inside it.
func TestASwapBeforeTheByNameFallbackIsRefusedByWhereTheDescriptorIs(t *testing.T) {
	notRoot727(t)
	if _, ok := fdPath(os.Stdin); !ok {
		t.Skip("this platform cannot ask a descriptor for its path")
	}
	for name, c := range map[string]struct {
		target func(root, outside string) string
		refuse bool
	}{
		"outside":         {func(root, outside string) string { return outside }, true},
		"inside, another": {func(root, outside string) string { return filepath.Join(root, "other") }, false},
	} {
		root, outside := realRoot727(t), realRoot727(t)
		sub := filepath.Join(root, "sub")
		for _, d := range []string{sub, filepath.Join(root, "other"), outside} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(d, "a.txt"), []byte("X\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Chmod(sub, 0o111); err != nil {
			t.Fatal(err)
		}
		aim := c.target(root, outside)
		prev := byNameGapFn
		byNameGapFn = func() {
			byNameGapFn = prev
			_ = os.Chmod(sub, 0o755)
			if err := os.Rename(sub, sub+".old"); err != nil {
				t.Error(err)
				return
			}
			if err := os.Symlink(aim, sub); err != nil {
				t.Error(err)
			}
		}
		f, _, err := OpenRead(root, filepath.Join(sub, "a.txt"))
		byNameGapFn = prev
		_ = os.Chmod(sub+".old", 0o755)
		if f != nil {
			_ = f.Close()
		}
		if c.refuse && err == nil {
			t.Errorf("%s: a swap before the fallback was not refused", name)
		}
		if !c.refuse && err != nil {
			t.Errorf("%s: a file inside the root was refused: %v", name, err)
		}
	}
}

// #727 review P2-2. A link at the end of the path that resolves to nothing is a
// file that does not exist, not "path escapes from parent", whether it points
// inside the root by an absolute or a relative name, or outside it.
func TestADanglingLinkIsAMissingFileNotAnEscape(t *testing.T) {
	root, outside := realRoot727(t), realRoot727(t)
	for name, target := range map[string]string{
		"relative": "nowhere.txt",
		"abs-in":   filepath.Join(root, "nowhere.txt"),
		"abs-out":  filepath.Join(outside, "nowhere.txt"),
	} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
		_, _, err := OpenRead(root, filepath.Join(root, name))
		if !errors.Is(err, fs.ErrNotExist) || (err != nil && strings.Contains(err.Error(), "escapes")) {
			t.Errorf("%s: %v, want a file that does not exist", name, err)
		}
	}
}

// #727 review P3-5. mrw's own state is refused after resolution too (ADR-077):
// the name was judged once and may have changed.
func TestOpenReadRefusesMrwsOwnState(t *testing.T) {
	root := realRoot727(t)
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".st"))
	if err := os.MkdirAll(filepath.Join(root, ".st", "mrw", "k"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{".st/mrw/k/seen": "ledger\n", ".st/notes.txt": "notes\n"} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if f, _, err := OpenRead(root, filepath.Join(root, ".st", "mrw", "k", "seen")); !errors.Is(err, ErrOwnState) {
		if f != nil {
			_ = f.Close()
		}
		t.Errorf("mrw's ledger: %v, want ErrOwnState", err)
	}
	f, _, err := OpenRead(root, filepath.Join(root, ".st", "notes.txt"))
	if err != nil {
		t.Fatalf("a sibling of mrw's state: %v", err)
	}
	_ = f.Close()
}

// fdCount727 counts the descriptors this process holds, through /dev/fd.
func fdCount727(t *testing.T) int {
	t.Helper()
	ents, err := os.ReadDir("/dev/fd")
	if err != nil {
		t.Skipf("no /dev/fd here: %v", err)
	}
	return len(ents)
}

// manyDirs727 makes n directories d<i>/sub each holding f.txt.
func manyDirs727(t *testing.T, root string, n int) []string {
	t.Helper()
	var files []string
	for i := 0; i < n; i++ {
		d := filepath.Join(root, "d"+strconv.Itoa(i), "sub")
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		f := filepath.Join(d, "f.txt")
		if err := os.WriteFile(f, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	return files
}

// #727 review (descriptors). A walk reader holds the directories it reads from,
// up to a bound, evicting BEFORE it opens the next: the descriptors held never
// pass the bound plus the reads in flight, and Close gives every one back. The
// count is taken during the reads, by a sampler, and after Close.
func TestAWalkReaderHoldsNoMoreDescriptorsThanItsBoundAndReleasesThemAll(t *testing.T) {
	root := realRoot727(t)
	files := manyDirs727(t, root, 80)
	rd, err := NewWalkReader(root)
	if err != nil {
		t.Fatal(err)
	}
	rd.maxDirs = 8
	const inFlight = 4
	base := fdCount727(t) // the held root is counted
	var peak atomic.Int64
	stop := make(chan struct{})
	sampled := make(chan struct{})
	go func() {
		defer close(sampled)
		for {
			select {
			case <-stop:
				return
			default:
			}
			if n := int64(fdCount727(t)); n > peak.Load() {
				peak.Store(n)
			}
		}
	}()
	sem := make(chan struct{}, inFlight)
	var wg sync.WaitGroup
	for round := 0; round < 3; round++ {
		for _, p := range files {
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer func() { <-sem; wg.Done() }()
				f, _, err := rd.Open(p)
				if err != nil {
					t.Errorf("%s: %v", p, err)
					return
				}
				_ = f.Close()
			}()
		}
	}
	wg.Wait()
	close(stop)
	<-sampled
	// Bound: what was held at the start, the handles kept, the reads in flight,
	// and the sampler's own directory listing.
	if limit := int64(base + rd.maxDirs + inFlight + 2); peak.Load() > limit {
		t.Errorf("the walk held %d descriptors at its peak, want at most %d (%d at the start, %d kept, %d in flight)", peak.Load(), limit, base, rd.maxDirs, inFlight)
	}
	rd.Close()
	if after := fdCount727(t); after > base-1 {
		t.Errorf("%d descriptors held after Close, want at most %d: handles leaked", after, base-1)
	}
}

// #727 review (descriptors). Under a descriptor limit the reads still succeed: a
// walk under `ulimit -n 24` read nothing, having opened a 33rd handle before it
// closed one. The reader sizes its cache from the limit, and gives the cache
// up when the process runs out.
func TestAWalkReaderReadsEverythingUnderALowDescriptorLimit(t *testing.T) {
	root := realRoot727(t)
	files := manyDirs727(t, root, 60)
	var old syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &old); err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { _ = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &old) })
	for _, soft := range []uint64{64, 24} {
		low := syscall.Rlimit{Cur: soft, Max: old.Max}
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &low); err != nil {
			t.Skip(err)
		}
		rd, err := NewWalkReader(root)
		if err != nil {
			t.Fatal(err)
		}
		sem := make(chan struct{}, 8)
		var wg sync.WaitGroup
		for _, p := range files {
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer func() { <-sem; wg.Done() }()
				f, _, err := rd.Open(p)
				if err != nil {
					t.Errorf("limit %d: %s: %v", soft, p, err)
					return
				}
				_ = f.Close()
			}()
		}
		wg.Wait()
		rd.Close()
	}
}

// #727 review (FIFO). A directory replaced by a FIFO made OpenRoot wait for a
// writer, and the walk with it. It is asked what it is first, and the read of a
// file below it is refused at once.
func TestADirectoryReplacedByAFIFODoesNotBlockTheWalk(t *testing.T) {
	root := realRoot727(t)
	if err := syscall.Mkfifo(filepath.Join(root, "d"), 0o644); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	rd, err := NewWalkReader(root)
	if err != nil {
		t.Fatal(err)
	}
	defer rd.Close()
	done := make(chan error, 1)
	go func() {
		f, _, err := rd.Open(filepath.Join(root, "d", "f.txt"))
		if f != nil {
			_ = f.Close()
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("a file below a FIFO was opened")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the walk blocked on a directory replaced by a FIFO")
	}
}

// #727 review (state alias). On a filesystem that folds case, --root /work/Repo
// with the state home at /work/repo/.st has the state inside the root, and a
// walk must see it so; compared by string it did not.
func TestAWalkSeesMrwsStateUnderADifferentSpellingOfTheRoot(t *testing.T) {
	tmp := realRoot727(t)
	if err := os.Mkdir(filepath.Join(tmp, "Repo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "repo")); err != nil {
		t.Skip("this filesystem keeps case, so Repo and repo are two directories")
	}
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "repo", ".st"))
	if !stateTouches(filepath.Join(tmp, "Repo")) {
		t.Error("the state directory under a case-spelled alias of the root was not seen")
	}
	if stateTouches(realRoot727(t)) {
		t.Error("the pair: an unrelated root was taken for one holding the state")
	}
}
