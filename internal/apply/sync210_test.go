package apply

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-210 T1. Every staged temp is synced before the first rename and every
// directory the write changed is synced once after the last change, with one
// drain per device in each phase on macOS and no directory sync on Windows. The
// seams record each call in order; a failure before the first rename writes
// nothing, one after it is a landed write with a SyncError.
func TestStagedFilesAndTheirDirectoriesAreSyncedBeforeTheWriteReturns(t *testing.T) {
	var events []string
	var sizes []int64 // each temp's size at the moment it was synced
	var tempCalls, drainCalls, dirCalls, devCalls atomic.Int32
	relOf := func(root, p string) string {
		for _, base := range []string{root, evalOr(root)} {
			if r, err := filepath.Rel(base, p); err == nil && !strings.HasPrefix(r, "..") {
				return filepath.ToSlash(r)
			}
		}
		return filepath.ToSlash(p)
	}
	type fault struct {
		temp, drain int32  // fail this call number (1-based), 0 never
		dir         string // fail the sync of this directory
		// noBarrierDrain and noBarrierDir make that drain, or that directory's
		// flush, answer ErrNoBarrier: the volume gives no barrier.
		noBarrierDrain int32
		noBarrierDir   string
	}
	// install puts the recording seams in place for one subtest.
	install := func(t *testing.T, root string, f fault, devices bool) {
		events, sizes = nil, nil
		tempCalls.Store(0)
		drainCalls.Store(0)
		dirCalls.Store(0)
		devCalls.Store(0)
		rt, rd, rs, rv, rr := SyncTempFn, drainFn, SyncDirFn, deviceOfFn, commitRenameFn
		t.Cleanup(func() { SyncTempFn, drainFn, SyncDirFn, deviceOfFn, commitRenameFn = rt, rd, rs, rv, rr })
		SyncTempFn = func(fh *os.File) error {
			events = append(events, "temp")
			// The bytes are in the temp before it is synced: a sync of an empty
			// file makes nothing durable.
			if fi, err := fh.Stat(); err == nil {
				sizes = append(sizes, fi.Size())
			}
			if f.temp != 0 && tempCalls.Add(1) == f.temp {
				return errors.New("injected temp sync failure")
			}
			return rt(fh)
		}
		drainFn = func(fh *os.File) error {
			events = append(events, "drain")
			n := drainCalls.Add(1)
			if f.drain != 0 && n == f.drain {
				return errors.New("injected drain failure")
			}
			if f.noBarrierDrain != 0 && n == f.noBarrierDrain {
				return ErrNoBarrier
			}
			return rd(fh)
		}
		SyncDirFn = func(fh *os.File) error {
			d := relOf(root, filepath.Join(root, fh.Name()))
			if filepath.IsAbs(fh.Name()) {
				d = relOf(root, fh.Name())
			}
			events = append(events, "dir:"+d)
			if f.noBarrierDir != "" && d == f.noBarrierDir {
				return ErrNoBarrier
			}
			if f.dir != "" && d == f.dir {
				return errors.New("injected directory sync failure")
			}
			return rs(fh)
		}
		if devices {
			deviceOfFn = func(os.FileInfo) uint64 { return uint64(devCalls.Add(1)%2) + 1 }
		}
		commitRenameFn = func(tr *tree, from, to string) error {
			events = append(events, "rename")
			return rr(tr, from, to)
		}
	}
	fixture := func(t *testing.T) (string, map[string]Seen) {
		root := t.TempDir()
		files := map[string]string{"a.txt": "one\ntwo\n", "s/b.txt": "bee\n", "t/keep.txt": "k\n", "other/c.txt": "see\n", "other/e.txt": "eee\n"}
		ledger := map[string]Seen{}
		for name, body := range files {
			p := filepath.Join(root, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			ledger[name] = Seen{SHA: seen.SHA([]byte(body))}
		}
		ledger["OTHER/e.txt"] = ledger["other/e.txt"] // the same file under another spelling
		return root, ledger
	}
	edit := Input{Path: "a.txt", Op: "replace", Start: 1, End: 1, Body: []string{"uno"}, Lines: -1}
	editOther := Input{Path: "other/c.txt", Op: "replace", Start: 1, End: 1, Body: []string{"cee"}, Lines: -1, Index: 1}
	editAlias := Input{Path: "OTHER/e.txt", Op: "replace", Start: 1, End: 1, Body: []string{"EEE"}, Lines: -1, Index: 2}
	create := Input{Path: "n/m/k.txt", Op: "create", Body: []string{"made"}, Lines: -1, Index: 1}
	move := Input{Path: "s/b.txt", Op: "rename", Body: []string{"t/b.txt"}, Lines: -1, Index: 2}
	createS := Input{Path: "s/new/k.txt", Op: "create", Body: []string{"made"}, Lines: -1}
	unlink := Input{Path: "s/b.txt", Op: "unlink", Lines: -1}

	darwin := runtime.GOOS == "darwin"
	dirsAfter := func(drain bool, dirs ...string) []string {
		var out []string
		for _, d := range dirs {
			out = append(out, "dir:"+d)
		}
		if drain && darwin {
			out = append(out, "drain")
		}
		return out
	}
	join := func(parts ...[]string) []string {
		var out []string
		for _, p := range parts {
			out = append(out, p...)
		}
		return out
	}
	drain := func() []string {
		if darwin {
			return []string{"drain"}
		}
		return nil
	}

	type row struct {
		name   string
		in     []Input
		opt    Options
		fault  fault
		device bool
		want   []string // the recorded calls, in order
		// foldCase compares the calls ignoring case: which of two spellings of one
		// directory is recorded first is the platform's, and the row asserts that the
		// directory is synced once, not which spelling names it (Windows CI on #750).
		foldCase bool
		// wantErr is "" for a landing, "sync" for a SyncError after it, "staging" for
		// a failure before the first rename.
		wantErr string
		wantDir string // the path a SyncError names
		// wantSizes is the size of each temp when it was synced, ascending.
		wantSizes []int64
		// wantNote is whether the receipt carries the no-barrier note.
		wantNote bool
		// wantFailed, when set, is the one file whose hunk is failed by a drain
		// failure: the file whose temp is on the device that failed.
		wantFailed string
	}
	rows := []row{
		{name: "edit_create_rename", in: []Input{edit, create, move}, wantSizes: []int64{5, 8},
			want: join([]string{"temp", "temp"}, drain(), []string{"rename", "rename", "rename"}, dirsAfter(true, ".", "n", "n/m", "s", "t"))},
		// A directory made under an existing one: both the new directory and its
		// parent hold a changed entry, and the root holds none.
		{name: "create_under_existing", in: []Input{createS}, wantSizes: []int64{5},
			want: join([]string{"temp"}, drain(), []string{"rename"}, dirsAfter(true, "s", "s/new"))},
		{name: "rename_only", in: []Input{move}, want: join([]string{"rename"}, dirsAfter(true, "s", "t"))},
		{name: "unlink_only", in: []Input{unlink}, want: join([]string{"rename"}, dirsAfter(true, "s"))},
		{name: "temp_sync_fails", in: []Input{edit, create, move}, fault: fault{temp: 2}, wantErr: "staging", wantSizes: []int64{5, 8},
			want: []string{"temp", "temp"}},
		{name: "dry_run", in: []Input{edit, create, move}, opt: Options{DryRun: true}, want: nil},
	}
	// One directory under two spellings is synced once, where the filesystem folds case.
	if foldsCase(t) {
		rows = append(rows, row{name: "one_directory_two_spellings", in: []Input{editOther, editAlias}, wantSizes: []int64{4, 4}, foldCase: true,
			want: join([]string{"temp", "temp"}, drain(), []string{"rename", "rename"}, dirsAfter(true, "OTHER"))})
	}
	if darwin {
		rows = append(rows,
			// Every device is drained though the first one's drain fails.
			// A volume whose F_FULLFSYNC is not supported answers ErrNoBarrier from the
			// drain: the write proceeds, both phases, with the note.
			// The first device falls back (ErrNoBarrier), the second fails (EIO): the
			// write stops with nothing written and carries no "files written" note.
			row{name: "no_barrier_then_eio", in: []Input{edit, editOther}, device: true, fault: fault{noBarrierDrain: 1, drain: 2}, wantErr: "staging", wantSizes: []int64{4, 8},
				want: []string{"temp", "temp", "drain", "drain"}},
			row{name: "drain_no_barrier", in: []Input{edit, create, move}, fault: fault{noBarrierDrain: 1}, wantNote: true, wantSizes: []int64{5, 8},
				want: join([]string{"temp", "temp", "drain", "rename", "rename", "rename"}, dirsAfter(true, ".", "n", "n/m", "s", "t"))},
			row{name: "second_device_drain_fails", in: []Input{edit, editOther}, device: true, fault: fault{drain: 2}, wantErr: "staging", wantSizes: []int64{4, 8}, wantFailed: "other/c.txt",
				want: []string{"temp", "temp", "drain", "drain"}},
			row{name: "two_devices_first_drain_fails", in: []Input{edit, editOther}, device: true, fault: fault{drain: 1}, wantErr: "staging", wantSizes: []int64{4, 8},
				want: []string{"temp", "temp", "drain", "drain"}},
			row{name: "two_devices", in: []Input{edit, editOther}, device: true, wantSizes: []int64{4, 8},
				want: join([]string{"temp", "temp", "drain", "drain", "rename", "rename"}, []string{"dir:.", "dir:other", "drain", "drain"})},
			row{name: "drain_before_fails", in: []Input{edit, create, move}, fault: fault{drain: 1}, wantErr: "staging", wantSizes: []int64{5, 8},
				want: []string{"temp", "temp", "drain"}},
			row{name: "drain_after_fails", in: []Input{edit, create, move}, fault: fault{drain: 2}, wantErr: "sync", wantSizes: []int64{5, 8},
				want: join([]string{"temp", "temp", "drain", "rename", "rename", "rename"}, dirsAfter(true, ".", "n", "n/m", "s", "t"))},
		)
	}
	// One failing directory does not stop the others: every directory is tried
	// and the drain still runs; the first failure is the error.
	rows = append(rows, row{name: "dir_sync_fails", in: []Input{edit, create, move}, fault: fault{dir: "n"}, wantErr: "sync", wantDir: "n", wantSizes: []int64{5, 8},
		want: join([]string{"temp", "temp"}, drain(), []string{"rename", "rename", "rename"}, dirsAfter(true, ".", "n", "n/m", "s", "t"))})
	// A directory the volume cannot flush (macOS has none here; Windows over SMB
	// answers ERROR_INVALID_FUNCTION) is skipped with a note, not a failure, and
	// the other directories are still synced.
	rows = append(rows, row{name: "dir_no_barrier", in: []Input{edit, create, move}, fault: fault{noBarrierDir: "n"}, wantNote: true, wantSizes: []int64{5, 8},
		want: join([]string{"temp", "temp"}, drain(), []string{"rename", "rename", "rename"}, dirsAfter(true, ".", "n", "n/m", "s", "t"))})

	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			root, ledger := fixture(t)
			install(t, root, r.fault, r.device)
			opt := r.opt
			opt.Seen = ledger
			res, err := Apply(root, r.in, opt)
			if got, want := strings.Join(events, " "), strings.Join(r.want, " "); got != want && !(r.foldCase && strings.EqualFold(got, want)) {
				t.Errorf("calls:\n got %s\nwant %s", got, want)
			}
			slices.Sort(sizes)
			if !slices.Equal(sizes, r.wantSizes) && (len(sizes) > 0 || len(r.wantSizes) > 0) {
				t.Errorf("temps were synced at sizes %v, want %v: a temp is synced after its bytes are written", sizes, r.wantSizes)
			}
			var se *SyncError
			switch r.wantErr {
			case "":
				if err != nil || (!r.opt.DryRun && !res.Applied) {
					t.Fatalf("applied=%v err=%v; want a clean landing", res.Applied, err)
				}
				if (res.SyncNote != "") != r.wantNote {
					t.Errorf("SyncNote %q, want a note: %v", res.SyncNote, r.wantNote)
				}
			case "staging":
				if err == nil || res.Applied || errors.As(err, &se) {
					t.Fatalf("applied=%v err=%v; want a staging failure, nothing written", res.Applied, err)
				}
				// Every failure before the first rename, the drain's included, is the
				// environment's (ErrStagingSync) and says nothing about files written.
				if !errors.Is(err, ErrStagingSync) {
					t.Errorf("a failure before the first rename is %v; want it to be ErrStagingSync", err)
				}
				if res.SyncNote != "" {
					t.Errorf("a write that failed carries the note %q", res.SyncNote)
				}
				if r.wantFailed != "" {
					failed := ""
					for _, h := range res.Hunks {
						if h.Status == StatusFailed {
							failed = h.Path
						}
					}
					if failed != r.wantFailed {
						t.Errorf("the failed hunk is %q, want %q (the file on the device that failed)", failed, r.wantFailed)
					}
				}
				if b, _ := os.ReadFile(filepath.Join(root, "a.txt")); string(b) != "one\ntwo\n" {
					t.Errorf("a.txt changed: %q", b)
				}
				if _, err := os.Stat(filepath.Join(root, "n")); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("n was left behind (%v)", err)
				}
				var stray []string
				_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, _ error) error {
					if d != nil && strings.HasPrefix(d.Name(), ".mrw-") {
						stray = append(stray, p)
					}
					return nil
				})
				if len(stray) != 0 {
					t.Errorf("temps left behind: %v", stray)
				}
			case "sync":
				if !res.Applied || !errors.As(err, &se) {
					t.Fatalf("applied=%v err=%v; want Applied with a SyncError", res.Applied, err)
				}
				if r.wantDir != "" && relOf(root, filepath.Join(root, se.Path)) != r.wantDir && se.Path != r.wantDir {
					t.Errorf("the error names %q, want %q", se.Path, r.wantDir)
				}
				for name, want := range map[string]string{"a.txt": "uno\ntwo\n", "n/m/k.txt": "made\n", "t/b.txt": "bee\n"} {
					if b, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(name))); string(b) != want {
						t.Errorf("%s = %q, want %q", name, b, want)
					}
				}
			}
		})
	}
	_ = fmt.Sprint
}

func evalOr(p string) string {
	if e, err := filepath.EvalSymlinks(p); err == nil {
		return e
	}
	return p
}

// foldsCase says whether the filesystem under the test's temp directory treats
// d and D as one directory (APFS by default, NTFS).
func foldsCase(t *testing.T) bool {
	t.Helper()
	d := t.TempDir()
	if err := os.Mkdir(filepath.Join(d, "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := os.Stat(filepath.Join(d, "X"))
	return err == nil
}

// ADR-210 review. The drain asks through each temp's DIRECTORY, so a staged temp
// whose inherited mode the owner cannot read (0244) is no obstacle to it: it is
// never reopened for reading.
func TestTheDrainDoesNotReopenAStagedTempForReading(t *testing.T) {
	root := evalOr(t.TempDir())
	tr, err := openTree(root, root)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.close()
	tmp := filepath.Join(root, ".mrw-0244")
	if err := os.WriteFile(tmp, []byte("x"), 0o244); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(tmp, 0o244); err != nil {
		t.Fatal(err)
	}
	if p, _, err := syncStaged(tr, []staged{{tmp: tmp}}); err != nil {
		t.Errorf("the drain failed at %s: %v", p, err)
	}
}

// ADR-210 final targeted pass. A directory whose identity was not recorded (the
// stat at staging failed) cannot be told from another object, such as a junction
// retargeted outside the checkout, so it is not opened and not flushed: a SyncError.
func TestADirectoryWithoutARecordedIdentityIsNotFlushed(t *testing.T) {
	root := evalOr(t.TempDir())
	tr, err := openTree(root, root)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.close()
	rs, rd := SyncsDirs, SyncDirFn
	t.Cleanup(func() { SyncsDirs, SyncDirFn = rs, rd })
	SyncsDirs = true
	flushed := 0
	SyncDirFn = func(f *os.File) error { flushed++; return rd(f) }
	_, err = syncChanged(tr, []dirRef{{path: root}})
	var se *SyncError
	if !errors.As(err, &se) || flushed != 0 {
		t.Errorf("err=%v, %d flush(es); want a SyncError and no flush for a directory with no recorded identity", err, flushed)
	}
}
