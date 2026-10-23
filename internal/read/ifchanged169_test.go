package read

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-169 T1. A caller who holds a file's sha asks, with the read, whether the
// file is still those bytes: if it is, the read answers one line per spec in
// place of the spec's text, says how much it did not serve, and records
// nothing. The answer is about text, never about permission.

// guarded runs specs over root with the guard and returns the answer, what the
// read observed, its problem count and the unchanged entries it reported.
func guarded(t *testing.T, root string, guard map[string]string, opt Options, specs ...string) (string, map[string]seen.Observation, int, []Unchanged) {
	t.Helper()
	var parsed []Spec
	for _, s := range specs {
		sp, err := ParseSpec(s)
		if err != nil {
			t.Fatal(err)
		}
		parsed = append(parsed, sp)
	}
	var got []Unchanged
	opt.Numbers = true
	opt.IfChanged = guard
	opt.Unchanged = &got
	var out bytes.Buffer
	obs, problems := Run(&out, root, parsed, opt)
	return out.String(), obs, problems, got
}

func shaOf(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func guardFile(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// numbered matches a served line, `NNN| text`.
var numbered = regexp.MustCompile(`(?m)^ *[0-9]+\| `)

func TestAGuardedReadOfAnUnchangedFileServesNothing(t *testing.T) {
	root := t.TempDir()
	body := "alpha\nbravo\ncharlie\n" // 3 lines, 20 bytes
	guardFile(t, root, "a.txt", body)
	full := shaOf([]byte(body))
	short := full[:8]

	// The 8 hex digits the header prints, the whole digest, and upper case: one
	// line, with the spec's own lines and bytes, and the sha as the caller wrote it.
	for _, given := range []string{short, full, strings.ToUpper(short)} {
		out, obs, problems, un := guarded(t, root, map[string]string{"a.txt": given}, Options{}, "a.txt")
		want := fmt.Sprintf("==> a.txt  unchanged since sha %s — 3 lines, 20 bytes not served\n", given)
		if out != want {
			t.Errorf("guard %q: got %q, want %q", given, out, want)
		}
		if numbered.MatchString(out) || problems != 0 || len(obs) != 0 {
			t.Errorf("guard %q: served %q, problems %d, observed %v; want nothing served, no problem, nothing observed", given, out, problems, obs)
		}
		if want := []Unchanged{{Path: "a.txt", SHA: given, Lines: 3, Bytes: 20}}; !reflect.DeepEqual(un, want) {
			t.Errorf("guard %q: reported %v, want %v", given, un, want)
		}
	}

	// Two specs of one file are two answers, each with what that spec would
	// have served: the lines 1-2 are 12 bytes, line 3 is 8.
	out, _, _, un := guarded(t, root, map[string]string{"a.txt": short}, Options{}, "a.txt:1-2", "a.txt:3")
	want := fmt.Sprintf("==> a.txt  unchanged since sha %[1]s — 2 lines, 12 bytes not served\n==> a.txt  unchanged since sha %[1]s — 1 lines, 8 bytes not served\n", short)
	if out != want {
		t.Errorf("two specs: got %q, want %q", out, want)
	}
	if len(un) != 2 {
		t.Errorf("two specs: reported %v, want two entries", un)
	}

	// The right 8 digits followed by wrong ones is not the file's sha: served.
	if out, _, _, un := guarded(t, root, map[string]string{"a.txt": short + "00"}, Options{}, "a.txt"); strings.Contains(out, "unchanged") || len(un) != 0 {
		t.Errorf("a guard of the right 8 digits and a wrong tail answered unchanged: %q", out)
	}

	// A range the read would not have served is reported as it is without the
	// guard, beside the unchanged line (Zy, 2026-10-23).
	out, _, problems, _ := guarded(t, root, map[string]string{"a.txt": short}, Options{}, "a.txt:/zzz/")
	if !strings.Contains(out, "unchanged since sha") || problems != 1 {
		t.Errorf("a missed range under a guard: %q, problems %d; want the unchanged line and one problem", out, problems)
	}

	// --max-lines caps what the spec would have served, so it caps N and B.
	one := 1
	if out, _, _, _ := guarded(t, root, map[string]string{"a.txt": short}, Options{MaxLines: &one}, "a.txt"); !strings.Contains(out, "— 1 lines, 6 bytes not served") {
		t.Errorf("--max-lines 1: %q, want 1 line and 6 bytes", out)
	}

	// Another name for the file — a hard link, which needs no privilege on any
	// platform — is the same file to a guard (ADR-007).
	if err := os.Link(filepath.Join(root, "a.txt"), filepath.Join(root, "hard.txt")); err != nil {
		t.Fatal(err)
	}
	sp, err := ParseSpec("hard.txt")
	if err != nil {
		t.Fatal(err)
	}
	keyed, _, err := ResolveGuards(root, []Guard{{Path: "a.txt", SHA: short}}, []Spec{sp}, false)
	if err != nil {
		t.Fatal(err)
	}
	if out, _, _, _ := guarded(t, root, keyed, Options{}, "hard.txt"); !strings.HasPrefix(out, "==> hard.txt  unchanged since sha") {
		t.Errorf("a guard on a.txt did not answer its hard link hard.txt: %q", out)
	}
}

func TestAGuardedReadOfAChangedFileServesIt(t *testing.T) {
	root := t.TempDir()
	guardFile(t, root, "a.txt", "alpha\nbravo\n")
	guardFile(t, root, "b.txt", "kilo\nlima\n")
	oldA := shaOf([]byte("alpha\nbravo\n"))[:8]
	bSha := shaOf([]byte("kilo\nlima\n"))[:8]
	guardFile(t, root, "a.txt", "alpha\nbravo\ncharlie\n")
	newA := shaOf([]byte("alpha\nbravo\ncharlie\n"))[:8]

	// A changed file is served exactly as an unguarded read serves it.
	plain, _, _, _ := guarded(t, root, nil, Options{}, "a.txt")
	out, obs, problems, un := guarded(t, root, map[string]string{"a.txt": oldA}, Options{}, "a.txt")
	if out != plain {
		t.Errorf("a changed file was not served as an unguarded one:\n got %q\nwant %q", out, plain)
	}
	if !strings.Contains(out, "sha "+newA) || strings.Contains(out, "unchanged") {
		t.Errorf("the header does not name the new sha %s, or says unchanged:\n%s", newA, out)
	}
	if problems != 0 || len(un) != 0 || len(obs) != 1 {
		t.Errorf("problems %d, unchanged %v, observed %v; want 0, none, one entry", problems, un, obs)
	}

	// One unchanged and one changed file in one read: only the changed one is served.
	out, obs, _, un = guarded(t, root, map[string]string{"a.txt": oldA, "b.txt": bSha}, Options{}, "a.txt", "b.txt")
	if !strings.Contains(out, "==> a.txt  3L") || !strings.Contains(out, "charlie") {
		t.Errorf("the changed file was not served:\n%s", out)
	}
	if !strings.Contains(out, "==> b.txt  unchanged since sha "+bSha) || strings.Contains(out, "lima") {
		t.Errorf("the unchanged file was not answered in one line:\n%s", out)
	}
	if _, ok := obs["b.txt"]; ok || len(un) != 1 || un[0].Path != "b.txt" {
		t.Errorf("observed %v, unchanged %v; want only a.txt observed and only b.txt unchanged", obs, un)
	}
}

func TestAnUnchangedAnswerLicensesNothing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	guardFile(t, root, "a.txt", "alpha\nbravo\ncharlie\ndelta\n")
	guardFile(t, root, "b.txt", "kilo\nlima\n")

	// A prior entry for a.txt: lines 1-2 (with the line after), and none of 4.
	_, obs, _, _ := guarded(t, root, nil, Options{NextLine: true}, "a.txt:1-2")
	if err := seen.Record(root, obs); err != nil {
		t.Fatal(err)
	}
	before, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 1 {
		t.Fatalf("the prior ledger holds %v, want one entry", before)
	}

	for name, spec := range map[string]string{"a.txt": "a.txt", "b.txt": "b.txt"} {
		sha := shaOf(mustRead(t, root, name))[:8]
		_, obs, _, un := guarded(t, root, map[string]string{name: sha}, Options{NextLine: true}, spec)
		if len(obs) != 0 || len(un) != 1 {
			t.Fatalf("%s: observed %v, unchanged %v; want nothing observed and one unchanged", name, obs, un)
		}
		if err := seen.Record(root, obs); err != nil {
			t.Fatal(err)
		}
	}
	after, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Errorf("an unchanged answer moved the ledger:\nbefore %v\n after %v", before, after)
	}
	if _, ok := after["b.txt"]; ok {
		t.Errorf("an unchanged answer gave b.txt, which had no entry, one: %v", after)
	}
	if got := after["a.txt"].Spans; !reflect.DeepEqual(got, [][2]int{{1, 3}}) {
		t.Errorf("a.txt licenses %v after the unchanged answer, want only the lines its first read served, [[1 3]]", got)
	}
}

func mustRead(t *testing.T, root, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
