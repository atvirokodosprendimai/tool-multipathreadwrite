package apply

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// ADR-114 chaos. Random multi-file plans mixing edits, edit+rename, plain
// renames and unlinks, with a failure injected at a random commit rename
// (content or path op), checked against invariants that must hold whatever
// happened: a clean apply leaves exactly the model; any outcome leaves every
// file in a state the plan could explain; the receipt matches the disk; an ok
// verdict describes something on disk; no .mrw-* file is left unnamed.
// MRW_CHAOS_N raises the iteration count.
func TestEditRenameChaos(t *testing.T) {
	n := 300
	if v, err := strconv.Atoi(os.Getenv("MRW_CHAOS_N")); err == nil && v > 0 {
		n = v
	}
	for seed := int64(1); seed <= int64(n); seed++ {
		chaosOnce(t, seed)
		if t.Failed() {
			t.Fatalf("seed %d broke an invariant (MRW_CHAOS_N=%d reruns it)", seed, seed)
		}
	}
}

type chaosFile struct {
	name     string
	orig     []string
	final    []string // nil when unlinked
	dest     string   // "" when it stays
	edited   bool
	unlinked bool
}

func chaosOnce(t *testing.T, seed int64) {
	r := rand.New(rand.NewSource(seed))
	root := t.TempDir()
	nf := 2 + r.Intn(4)
	files := make([]*chaosFile, nf)
	seen := map[string]Seen{}
	for i := range files {
		f := &chaosFile{name: fmt.Sprintf("f%d.go", i)}
		for k := 1; k <= 6; k++ {
			f.orig = append(f.orig, fmt.Sprintf("f%d-l%d", i, k))
		}
		write(t, root, f.name, strings.Join(f.orig, "\n")+"\n")
		seen[f.name] = Seen{SHA: shaOfFile(t, root, f.name)}
		files[i] = f
	}
	var in []Input
	add := func(h Input) {
		h.Index = len(in)
		h.SrcLine = len(in) + 1
		in = append(in, h)
	}
	for i, f := range files {
		f.final = append([]string(nil), f.orig...)
		switch r.Intn(5) {
		case 0: // untouched
		case 1, 2: // edit, maybe rename
			lines := r.Perm(6)[:1+r.Intn(2)]
			for _, l := range lines {
				body := fmt.Sprintf("%s-x%d", f.orig[l], seed)
				f.final[l] = body
				add(Input{Path: f.name, Start: l + 1, End: l + 1, Op: "replace", Body: []string{body}, Lines: -1})
			}
			f.edited = true
			if r.Intn(2) == 0 {
				f.dest = fmt.Sprintf("d%d/m%d.go", r.Intn(2), i)
				add(Input{Path: f.name, Op: "rename", Body: []string{f.dest}, Lines: -1})
			}
		case 3:
			f.dest = fmt.Sprintf("g%d.go", i)
			add(Input{Path: f.name, Op: "rename", Body: []string{f.dest}, Lines: -1})
		case 4:
			f.unlinked = true
			f.final = nil
			add(Input{Path: f.name, Op: "unlink", Lines: -1})
		}
	}
	if len(in) == 0 {
		return
	}
	r.Shuffle(len(in), func(i, j int) { in[i], in[j] = in[j], in[i] })
	for i := range in {
		in[i].Index = i
	}

	failAt := 0
	if r.Intn(3) == 0 {
		failAt = 1 + r.Intn(2*len(files))
	}
	calls := 0
	real := commitRenameFn
	defer func() { commitRenameFn = real }()
	commitRenameFn = func(tr *tree, from, to string) error {
		calls++
		if calls == failAt {
			return errors.New("chaos: rename refused")
		}
		return real(tr, from, to)
	}

	res, err := Apply(root, in, Options{Seen: seen})
	commitRenameFn = real

	disk := chaosDisk(t, root)
	if err == nil && res.Applied {
		want := map[string]string{}
		for _, f := range files {
			if f.unlinked {
				continue
			}
			p := f.name
			if f.dest != "" {
				p = f.dest
			}
			want[p] = strings.Join(f.final, "\n") + "\n"
		}
		if fmt.Sprint(sortedKeys(want)) != fmt.Sprint(sortedKeys(disk)) {
			t.Errorf("seed %d: applied, files %v, want %v", seed, sortedKeys(disk), sortedKeys(want))
		}
		for p, w := range want {
			if disk[p] != w {
				t.Errorf("seed %d: applied, %s = %q, want %q", seed, p, disk[p], w)
			}
		}
	}
	// Every file is in a state the plan explains: untouched, fully done, or —
	// for an edit+rename whose rename failed or was undone — edited in place.
	for _, f := range files {
		origS := strings.Join(f.orig, "\n") + "\n"
		finalS := strings.Join(f.final, "\n") + "\n"
		at, atOK := disk[f.name]
		switch {
		case atOK && at == origS:
		case atOK && f.edited && at == finalS:
		case !atOK && f.unlinked:
		case !atOK && f.dest != "" && disk[f.dest] == finalS:
		default:
			t.Errorf("seed %d: %s is in no state the plan explains (at source %v %q, at %q: %q; err %v)", seed, f.name, atOK, at, f.dest, disk[f.dest], err)
		}
	}
	// The receipt matches the disk, one record per path.
	count := map[string]int{}
	for _, fr := range res.Files {
		// Records spell paths with the OS separator; the disk map uses /.
		p := filepath.ToSlash(fr.Path)
		count[p]++
		if count[p] > 1 {
			t.Errorf("seed %d: %s has %d records", seed, p, count[p])
		}
		if !fr.Written {
			continue
		}
		body, ok := disk[p]
		switch {
		case fr.Removed && ok:
			t.Errorf("seed %d: %s recorded removed, still on disk", seed, fr.Path)
		case !fr.Removed && !ok:
			t.Errorf("seed %d: %s recorded written, not on disk (err %v)", seed, fr.Path, err)
		case !fr.Removed && fr.SHAAfter != "" && fr.SHAAfter != shaOfFile(t, root, fr.Path):
			t.Errorf("seed %d: %s recorded sha %s, disk %q", seed, fr.Path, fr.SHAAfter, body)
		}
	}
	// An ok verdict describes something on disk; failed counts agree.
	failed := 0
	for _, h := range res.Hunks {
		if h.Status == StatusFailed {
			failed++
		}
		if res.DryRun {
			continue
		}
		var f *chaosFile
		for _, c := range files {
			if c.name == h.Path {
				f = c
			}
		}
		if f == nil {
			continue
		}
		finalS := strings.Join(f.final, "\n") + "\n"
		// A verdict that is not ok says its change is not on disk: an edit
		// skipped or failed has not landed, and neither has such a rename.
		if h.Status != StatusOK {
			switch h.Op {
			case "rename":
				if _, ok := disk[f.dest]; ok {
					t.Errorf("seed %d: rename %s %s, but %s is on disk", seed, f.name, h.Status, f.dest)
				}
			case "unlink":
				if _, ok := disk[f.name]; !ok {
					t.Errorf("seed %d: unlink %s %s, but the file is gone", seed, f.name, h.Status)
				}
			default:
				if disk[f.name] == finalS || disk[f.dest] == finalS {
					t.Errorf("seed %d: edit on %s %s, but the edit is on disk (err %v)", seed, f.name, h.Status, err)
				}
			}
			continue
		}
		switch h.Op {
		case "rename":
			if _, ok := disk[f.name]; ok || disk[f.dest] != finalS {
				t.Errorf("seed %d: rename %s ok, but source there %v / dest %q", seed, f.name, ok, disk[f.dest])
			}
		case "unlink":
			if _, ok := disk[f.name]; ok {
				t.Errorf("seed %d: unlink %s ok, file still there", seed, f.name)
			}
		default:
			if disk[f.name] != finalS && disk[f.dest] != finalS {
				t.Errorf("seed %d: edit on %s ok, edit not on disk (err %v)", seed, f.name, err)
			}
		}
	}
	if failed != res.Failed {
		t.Errorf("seed %d: %d failed hunks, Failed=%d", seed, failed, res.Failed)
	}
	left := map[string]bool{}
	for _, p := range res.LeftBehind {
		left[filepath.ToSlash(p)] = true
	}
	for p := range disk {
		if strings.Contains(filepath.Base(p), ".mrw-") && !left[p] {
			t.Errorf("seed %d: %s left in the tree, not named", seed, p)
		}
	}
}

func chaosDisk(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatalf("walking the chaos tree: %v", err)
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	k := make([]string, 0, len(m))
	for p := range m {
		k = append(k, p)
	}
	sort.Strings(k)
	return k
}
