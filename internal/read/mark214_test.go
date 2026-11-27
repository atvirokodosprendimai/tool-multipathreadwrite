package read

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

// ADR-214 Decision 8. A read that prints line 1 of a file that begins with the
// UTF-8 byte order mark says so once, after the header, in report text that is
// not a line of the file. M is the mark, written as bytes.
func TestAReadOfAMarkedFileSaysSoOnce(t *testing.T) {
	const (
		m    = "\xEF\xBB\xBF"
		note = "-- note: a UTF-8 byte order mark begins this file and is part of line 1: ^one does not match it (^.one and ^\\x{FEFF}one do), and every edit keeps it at byte zero\n"
	)
	root := t.TempDir()
	put := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	put("f.txt", m+"one\ntwo\nthree\n")
	put("g.txt", "one\ntwo\n")
	put("only.txt", m)
	put("fold.txt", m+"x\nx\nx\nx\n")
	put("c.go", m+"// a\n// b\npackage x\n")
	specs := func(t *testing.T, raw ...string) []Spec {
		t.Helper()
		var out []Spec
		for _, r := range raw {
			s, err := ParseSpec(r)
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, s)
		}
		return out
	}
	run := func(t *testing.T, opt Options, raw ...string) (string, map[string][][2]int, int) {
		t.Helper()
		var out bytes.Buffer
		obs, problems := Run(&out, root, specs(t, raw...), opt)
		sp := map[string][][2]int{}
		for k, o := range obs {
			sp[k] = o.Spans
		}
		return out.String(), sp, problems
	}
	count := func(s string) int { return strings.Count(s, "-- note: a UTF-8 byte order mark") }

	t.Run("after_the_header_before_the_first_range", func(t *testing.T) {
		for _, numbers := range []bool{true, false} {
			out, _, _ := run(t, Options{Numbers: numbers}, "f.txt")
			lines := strings.SplitAfter(out, "\n")
			if len(lines) < 3 || !strings.HasPrefix(lines[0], "==> f.txt  3L  17B  sha ") || lines[1] != note || !strings.HasPrefix(lines[2], "@@ 1-3") {
				t.Errorf("numbers=%v: header, note, range expected:\n%s", numbers, out)
			}
		}
	})
	t.Run("once_for_two_ranges_of_one_file", func(t *testing.T) {
		out, _, _ := run(t, Options{Numbers: true}, "f.txt:1", "f.txt:1")
		if count(out) != 1 {
			t.Errorf("the note was printed %d times:\n%s", count(out), out)
		}
	})
	t.Run("the_note_earns_no_showing_and_no_line_count", func(t *testing.T) {
		out, spans, _ := run(t, Options{Numbers: true}, "f.txt")
		if spans["f.txt"] != nil {
			t.Errorf("a whole read records %v", spans["f.txt"])
		}
		if !strings.Contains(out, "3L  17B") {
			t.Errorf("the header is not the raw count:\n%s", out)
		}
		_, spans, _ = run(t, Options{Numbers: true}, "f.txt:1")
		if got := spans["f.txt"]; len(got) != 1 || got[0] != [2]int{1, 1} {
			t.Errorf("line 1 and its line after are licensed, and nothing else: %v", got)
		}
	})
	t.Run("a_cut_line_1_counts_as_printed_and_stays_unlicensed", func(t *testing.T) {
		out, spans, _ := run(t, Options{Numbers: true, MaxCols: 2}, "f.txt:1")
		if count(out) != 1 {
			t.Errorf("a line 1 cut by max-cols: the note was printed %d times:\n%s", count(out), out)
		}
		for _, s := range spans["f.txt"] {
			if s[0] <= 1 && s[1] >= 1 {
				t.Errorf("a cut line 1 is licensed: %v", spans["f.txt"])
			}
		}
	})
	t.Run("a_folded_display_counts_as_printed", func(t *testing.T) {
		out, _, _ := run(t, Options{Numbers: true, Fold: true}, "fold.txt")
		if count(out) != 1 {
			t.Errorf("a folded display: the note was printed %d times:\n%s", count(out), out)
		}
	})
	t.Run("not_for_a_read_that_does_not_print_line_1", func(t *testing.T) {
		zero := 0
		sub := &Sub{Re: regexp.MustCompile(`one`), Repl: "ONE"}
		cases := []struct {
			name string
			opt  Options
			spec string
		}{
			{"line_2", Options{Numbers: true}, "f.txt:2"},
			{"quiet", Options{Numbers: true, Quiet: true}, "f.txt"},
			{"stat", Options{Numbers: true, Stat: true}, "f.txt"},
			{"count", Options{Numbers: true, Count: true}, "f.txt"},
			{"max_lines_0", Options{Numbers: true, MaxLines: &zero}, "f.txt"},
			{"a_plan_maker", Options{Sub: sub}, "f.txt:/one/"},
			{"an_unmarked_file", Options{Numbers: true}, "g.txt"},
		}
		for _, c := range cases {
			out, _, _ := run(t, c.opt, c.spec)
			if count(out) != 0 {
				t.Errorf("%s printed the note:\n%s", c.name, out)
			}
		}
		// A reference to a file that has not changed prints no line of it either.
		stat, _, _ := run(t, Options{Stat: true}, "f.txt")
		sha := ""
		if mt := regexp.MustCompile(`sha ([0-9a-f]{8})`).FindStringSubmatch(stat); mt != nil {
			sha = mt[1]
		}
		if sha == "" {
			t.Fatal("no sha in the stat header")
		}
		out, _, _ := run(t, Options{Numbers: true, IfChanged: map[string]string{"f.txt": sha[:8]}}, "f.txt")
		if count(out) != 0 || !strings.Contains(out, "unchanged since") {
			t.Errorf("an if-changed hit:\n%s", out)
		}
		out, _, _ = run(t, Options{Numbers: true, IfChanged: map[string]string{"f.txt": "00000000"}}, "f.txt")
		if count(out) != 1 {
			t.Errorf("an if-changed miss prints the file, note included, once:\n%s", out)
		}
	})
	t.Run("the_file_that_is_only_the_mark", func(t *testing.T) {
		out, _, _ := run(t, Options{Numbers: true}, "only.txt:1")
		if count(out) != 1 {
			t.Errorf("the file M, line 1: the note was printed %d times:\n%s", count(out), out)
		}
		out, _, _ = run(t, Options{Numbers: true}, "only.txt:2")
		if count(out) != 0 {
			t.Errorf("the file M, line 2 (not served): the note was printed:\n%s", out)
		}
	})
	t.Run("not_when_line_1_opens_an_elided_comment_run", func(t *testing.T) {
		// --no-comments prints line 1 as part of a "-- … N comment line(s)" line,
		// not as line 1, so the note is not printed (the go-reviewer on T1).
		out, _, _ := run(t, Options{Numbers: true, NoComments: true}, "c.go")
		if !strings.Contains(out, "comment line(s), 1-2") || count(out) != 0 {
			t.Errorf("an elided line 1 printed the note:\n%s", out)
		}
	})
	t.Run("the_note_and_line_1_are_delivered_together_or_not_at_all", func(t *testing.T) {
		var whole bytes.Buffer
		Run(&whole, root, specs(t, "f.txt:1"), Options{Numbers: true})
		all := strings.SplitAfter(whole.String(), "\n") // header, note, range, line 1, ...
		runes := func(ls ...string) int { return utf8.RuneCountInString(strings.Join(ls, "")) }
		bud := func(n int) (string, string, map[string][][2]int, int) {
			var out, withheld bytes.Buffer
			obs, problems := Run(NewBudget(&out, n), root, specs(t, "f.txt:1"), Options{Numbers: true, Withheld: &withheld})
			sp := map[string][][2]int{}
			for k, o := range obs {
				sp[k] = o.Spans
			}
			return out.String(), withheld.String(), sp, problems
		}
		// The header fits and nothing else does.
		out, w, sp, problems := bud(runes(all[0]))
		if out != all[0] || problems == 0 || !strings.Contains(w, "f.txt: lines 1-") || len(sp["f.txt"]) != 0 {
			t.Errorf("header only: out %q withheld %q problems %d spans %v", out, w, problems, sp)
		}
		// The header, the note and the range line fit, line 1 does not: never the note alone.
		out, w, sp, problems = bud(runes(all[0], all[1], all[2]))
		if out != all[0] || count(out) != 0 || problems == 0 || !strings.Contains(w, "f.txt: lines 1-") || len(sp["f.txt"]) != 0 {
			t.Errorf("note and range fit, line 1 does not: out %q withheld %q problems %d spans %v", out, w, problems, sp)
		}
		// Everything fits.
		out, w, _, problems = bud(runes(all[:4]...) + 20)
		if count(out) != 1 || w != "" || problems != 0 {
			t.Errorf("a budget that holds the note and line 1: out %q withheld %q problems %d", out, w, problems)
		}
	})
}
