package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

func TestCopiesAndEditsUseTheCurrentLinesAcrossAliases(t *testing.T) {
	const body = "one\ntwo\nthree\nfour\n"
	sha := seen.SHA([]byte(body))
	for _, op := range []string{"copy", "delete"} {
		for _, test := range []struct {
			name, spelling string
			second         Seen
			pass           bool
		}{
			{"third spelling", "third", Seen{SHA: sha, Spans: [][2]int{{3, 3}}}, true},
			{"partial exact spelling", "first", Seen{SHA: sha, Spans: [][2]int{{3, 3}}}, true},
			{"stale exact spelling", "old", Seen{SHA: sha, Spans: [][2]int{{3, 3}}}, true},
			{"stale alias", "third", Seen{SHA: "older", Spans: [][2]int{{3, 3}}}, false},
			{"only licensed", "third", Seen{SHA: sha, Written: true, Spans: [][2]int{{3, 3}}}, op == "delete"},
		} {
			t.Run(op+"/"+test.name, func(t *testing.T) {
				root := t.TempDir()
				write(t, root, "first", body)
				for _, alias := range []string{"second", "third", "old"} {
					if err := os.Link(filepath.Join(root, "first"), filepath.Join(root, alias)); err != nil {
						t.Skipf("cannot make hard link: %v", err)
					}
				}
				write(t, root, "other", body)
				ledger := map[string]Seen{
					"first": {SHA: sha, Spans: [][2]int{{2, 2}}}, "second": test.second,
					"old": {SHA: "older"}, "other": {SHA: sha},
				}
				input := Input{Path: test.spelling, Start: 2, End: 3, Op: "delete", Lines: -1}
				target, want := test.spelling, "one\nfour\n"
				if op == "copy" {
					input = copyInput167("out", "create", 0, test.spelling, 2, 3)
					target, want = "out", "two\nthree\n"
				}
				res, err := Apply(root, []Input{input}, Options{Seen: ledger})
				if err != nil || res.Applied != test.pass {
					t.Fatalf("applied %t, want %t: %+v, %v", res.Applied, test.pass, res.Hunks, err)
				}
				if test.pass && read(t, root, target) != want {
					t.Fatalf("%s did not contain %q", target, want)
				}
				if !test.pass && (res.Failed != 1 || read(t, root, "first") != body) {
					t.Fatalf("refused request changed the file: %+v", res)
				}
			})
		}
	}
}

func TestAWrittenAliasLicensesButDoesNotShowAnUnreadNumber(t *testing.T) {
	root := t.TempDir()
	write(t, root, "first", "one\ntwo\n")
	if err := os.Link(filepath.Join(root, "first"), filepath.Join(root, "second")); err != nil {
		t.Skipf("cannot make hard link: %v", err)
	}
	sha := shaOfFile(t, root, "first")
	ledger := map[string]Seen{"first": {SHA: sha, Spans: [][2]int{{1, 1}}}, "second": {SHA: sha, Written: true}}
	res, err := Apply(root, []Input{{Path: "first", Start: 2, End: 2, Op: "delete", Lines: -1}}, Options{Seen: ledger})
	if err != nil || res.Applied || res.Failed != 1 || res.Hunks[0].Kind != "stale-number" || read(t, root, "first") != "one\ntwo\n" {
		t.Fatalf("unshown line licensed only through a whole written alias: %+v, %v", res, err)
	}
}

func TestDuplicateTargetIndexKeepsTheFirstSpelling(t *testing.T) {
	root := t.TempDir()
	var group groupedFiles
	for i := 1; i <= 20; i++ {
		path := fmt.Sprint(i)
		write(t, root, path, "same contents")
		info, err := os.Stat(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		if _, dup := group.add(groupedFile{path: path, info: info, line: i}); dup {
			t.Fatalf("distinct file %s was counted twice", path)
		}
	}
	for _, path := range []string{"alias", "another"} {
		if err := os.Link(filepath.Join(root, "12"), filepath.Join(root, path)); err != nil {
			t.Skipf("cannot make hard link: %v", err)
		}
		info, err := os.Stat(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		prior, dup := group.add(groupedFile{path: path, info: info, line: 21})
		if !dup || prior.path != "12" || prior.line != 12 {
			t.Fatalf("%s duplicated %+v, found %t", path, prior, dup)
		}
	}
}

// Only duplicate detection is timed. File creation and identity capture happen once.
func BenchmarkDuplicateTargets(b *testing.B) {
	for _, count := range []int{5, 500} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			root := b.TempDir()
			files := make([]groupedFile, count)
			for i := range files {
				path := filepath.Join(root, fmt.Sprint(i))
				if err := os.WriteFile(path, nil, 0o600); err != nil {
					b.Fatal(err)
				}
				info, err := os.Stat(path)
				if err != nil || !os.SameFile(info, info) {
					b.Fatalf("stat %s: %v", path, err)
				}
				files[i] = groupedFile{path: path, info: info, line: i + 1}
			}
			b.ReportAllocs()
			for b.Loop() {
				grouped := groupedFiles{planned: len(files)}
				for _, f := range files {
					if _, dup := grouped.add(f); dup {
						b.Fatal("unexpected duplicate")
					}
				}
			}
		})
	}
}

func TestAnOccurrenceRefusalDescribesTheShownLinesAcrossSpellings(t *testing.T) {
	root := t.TempDir()
	write(t, root, "x", "marker\nmarker\nmarker\nmarker\n")
	if err := os.Link(filepath.Join(root, "x"), filepath.Join(root, "alias")); err != nil {
		t.Skipf("cannot make hard link: %v", err)
	}
	sha := shaOfFile(t, root, "x")
	ledger := map[string]Seen{"x": {SHA: sha, Spans: [][2]int{{1, 1}}}, "alias": {SHA: sha, Spans: [][2]int{{2, 2}}}}
	h := Input{Path: "x", Op: "delete", StartPat: regexp.MustCompile("^marker$"), Occurrence: 4, Lines: -1}
	res, err := Apply(root, []Input{h}, Options{Seen: ledger})
	if err != nil || res.Applied || res.Failed != 1 || !strings.Contains(res.Hunks[0].Reason, "mrw served lines 1-2") {
		t.Fatalf("occurrence refusal omitted a showing: %+v, %v", res.Hunks, err)
	}
}

// A read ledger can contain many single lines before a large served range.
// The selected partial licence should be checked once for a numbered edit.
func BenchmarkPartialLicenceRange(b *testing.B) {
	root := b.TempDir()
	const start, end = 1025, 5120
	body := strings.Repeat("line\n", end)
	if err := os.WriteFile(filepath.Join(root, "large"), []byte(body), 0o600); err != nil {
		b.Fatal(err)
	}
	var spans [][2]int
	for i := 1; i < start; i += 2 {
		spans = append(spans, [2]int{i, i})
	}
	spans = append(spans, [2]int{start, end})
	opt := Options{DryRun: true, Seen: map[string]Seen{"large": {SHA: seen.SHA([]byte(body)), Spans: spans}}}
	h := []Input{{Path: "large", Op: "delete", Start: start, End: end, Lines: -1}}
	for b.Loop() {
		r, err := Apply(root, h, opt)
		if err != nil || r.Failed != 0 {
			b.Fatalf("%+v, %v", r.Hunks, err)
		}
	}
}
