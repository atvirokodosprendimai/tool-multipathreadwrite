package read

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestAParallelWalkAnswersAsASerialOneWould: the walk reads files concurrently
// but must answer as a serial walk does. The tree is built so every answer is
// known without running one: 240 matching files across eight directories (so
// reads overlap), one path named and walked (served once), unreadable files in
// the middle and at the end of the walk (their Problems keep the walk's order,
// between the two named paths that fail), a binary and a gitignored file.
func TestAParallelWalkAnswersAsASerialOneWould(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a file chmod 000 refuses to open")
	}
	root := t.TempDir()
	put := func(name, body string, mode os.FileMode) {
		t.Helper()
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	put(".git/HEAD", "ref: refs/heads/main\n", 0o644)
	put(".gitignore", "*.log\n", 0o644)
	var want []string
	for d := 0; d < 8; d++ {
		for f := 0; f < 30; f++ {
			n := fmt.Sprintf("d%d/f%03d.txt", d, f)
			put(n, "one\nneedle\n", 0o644)
			want = append(want, n)
		}
	}
	put("x.log", "needle\n", 0o644)
	put("bin.dat", "nee\x00dle needle\n", 0o644)
	// Many, so that with eight reads at once some finish out of queue order.
	var unreadable []string
	for d := 0; d < 8; d++ {
		for u := 0; u < 12; u++ {
			unreadable = append(unreadable, fmt.Sprintf("d%d/u%02d.txt", d, u))
		}
	}
	unreadable = append(unreadable, "zz.txt")
	for _, n := range unreadable {
		put(n, "needle\n", 0o000)
	}

	var sk WalkSkipped
	specs, problems, err := Walk(root, []string{"missing1", "d0/f000.txt", ".", "missing2"},
		WalkOptions{Pattern: regexp.MustCompile("needle"), Skipped: &sk})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, s := range specs {
		got = append(got, s.Path)
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("specs: got %d paths, want %d, in path order:\n%v", len(got), len(want), got)
	}
	wantProblems := append(append([]string{"missing1"}, unreadable...), "missing2")
	if len(problems) != len(wantProblems) {
		t.Fatalf("problems: got %v, want paths %v", problems, wantProblems)
	}
	for i, p := range problems {
		if p.Path != wantProblems[i] {
			t.Fatalf("problem %d is %q, want %q (order of the walk): %v", i, p.Path, wantProblems[i], problems)
		}
		if !strings.HasPrefix(p.Path, "missing") {
			if !strings.Contains(p.Reason, "permission denied") {
				t.Fatalf("problem %q reason %q, want permission denied", p.Path, p.Reason)
			}
		}
	}
	if sk.Binary != 1 || sk.Ignored != 1 {
		t.Fatalf("skipped: got binary %d ignored %d, want 1 and 1", sk.Binary, sk.Ignored)
	}
}

// TestAFileFromTheBigSizeIsReadWithNothingBesideIt: eight reads at once each
// hold their file whole, which broke ADR-104's memory bound for large files, so
// a file of bigFile bytes or more takes every scan slot and a smaller one a
// single slot. The walk of three such files and a small one must still finish
// and serve all four.
func TestAFileFromTheBigSizeIsReadWithNothingBesideIt(t *testing.T) {
	if got := scanSlots(bigFile-1, maxScanners); got != 1 {
		t.Fatalf("a file one byte under bigFile takes %d slots, want 1", got)
	}
	if got := scanSlots(bigFile, maxScanners); got != maxScanners {
		t.Fatalf("a file of bigFile bytes takes %d slots, want all %d", got, maxScanners)
	}
	root := t.TempDir()
	big := strings.Repeat("padding line\n", bigFile/13+1) + "needle\n"
	for _, n := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte(big), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "d.txt"), []byte("needle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	specs, problems, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle")})
	if err != nil || len(problems) != 0 || len(specs) != 4 {
		t.Fatalf("walk: %d specs, problems %v, err %v; want 4 specs", len(specs), problems, err)
	}
}

// TestBigFilesAreReadAloneAndSmallOnesTogether counts the reads in flight, which
// the slot arithmetic alone cannot show: a slot count computed and then not
// used would keep every test of scanSlots green. While a file of bigFile bytes
// is read nothing else is, and many small files are read together.
func TestBigFilesAreReadAloneAndSmallOnesTogether(t *testing.T) {
	root := t.TempDir()
	write := func(name string, size int64) {
		t.Helper()
		p := filepath.Join(root, name)
		if err := os.WriteFile(p, []byte("needle\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if size > 0 {
			if err := os.Truncate(p, size); err != nil {
				t.Fatal(err)
			}
		}
	}
	for i := 0; i < 40; i++ {
		write(fmt.Sprintf("s%02d.txt", i), 0)
		if i%10 == 5 {
			write(fmt.Sprintf("s%02d_big.txt", i), bigFile)
		}
	}
	var (
		mu                       sync.Mutex
		cur, bigCur, peak, bigAt int
	)
	prev := scanRead
	scanRead = func(path string) ([]byte, error) {
		fi, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		big := fi.Size() >= bigFile
		mu.Lock()
		if cur > 0 && big || bigCur > 0 {
			bigAt++
		}
		cur++
		if big {
			bigCur++
		}
		peak = max(peak, cur)
		mu.Unlock()
		time.Sleep(15 * time.Millisecond) // long enough for the others to start if they may
		mu.Lock()
		cur--
		if big {
			bigCur--
		}
		mu.Unlock()
		return prev(path)
	}
	t.Cleanup(func() { scanRead = prev })

	specs, problems, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle")})
	// The four big files are NUL padded: read, matched, and skipped as binaries.
	if err != nil || len(problems) != 0 || len(specs) != 40 {
		t.Fatalf("walk: %d specs, problems %v, err %v; want 40 specs", len(specs), problems, err)
	}
	if bigAt != 0 {
		t.Fatalf("%d reads began beside a file of bigFile bytes, want none", bigAt)
	}
	if peak < 2 {
		t.Fatalf("at most %d read at a time, want small files read together", peak)
	}
}
