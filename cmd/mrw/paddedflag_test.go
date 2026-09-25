package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// paddedTree holds x and "x " with different bytes, and skips where the
// filesystem cannot keep the two names apart.
func paddedTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for n, b := range map[string]string{"x": "plain\n", "x ": "padded\n"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte(b), 0o644); err != nil {
			t.Skipf("this filesystem cannot hold %q: %v", n, err)
		}
	}
	if fi, err := os.ReadDir(root); err != nil || len(fi) != 2 {
		t.Skip("this filesystem folds names that differ by a trailing space")
	}
	return root
}

// plainTree holds x alone. The refusals these tests pin fire before any I/O
// and the reads they pair them with serve x, so they need no file named "x "
// — which Win32 cannot hold beside x, and whose absence skipped all thirteen
// padded-path tests on NTFS (ADR-071 T4). paddedTree stays for the five that
// serve a file whose name ends in whitespace — two of them create their own
// (" -1= "), which NTFS stores silently as " -1=" (found by the first Windows
// run of this change).
func plainTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "x"), []byte("plain\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// ADR-069 T5, from the Codex review of v1.25.0. The guard stopped at every
// "--", including one consumed as a flag's value, so `read --grep -- 'x '`
// still let the parser trim the path to x. A flag's own value is skipped, and
// only a "--" in argument position ends the guard. The same rule retires the
// false refusal of a padded flag value whose trim equals a positional.
func TestAFlagValueDoesNotEndThePaddedArgGuard(t *testing.T) {
	root := plainTree(t)
	out, code := runIn(t, root, "read", "--grep", "--", "x ")
	if code != exitUsage || !strings.Contains(out, "'x '") {
		t.Errorf("read --grep -- 'x ' exited %d, want %d naming 'x ':\n%s", code, exitUsage, out)
	}
	if out, code := runIn(t, root, "read", "--grep", "plain", "--exclude", " x", "x"); code != 0 {
		t.Errorf("a padded --exclude value beside the positional x was refused, exit %d:\n%s", code, out)
	}
	if out, code := runIn(t, root, "read", "--grep", "plain", "--", "x "); code != 1 || strings.Contains(out, "1| plain") {
		t.Errorf("read --grep plain -- 'x ' exited %d (want 1: x-space holds no match) or served x:\n%s", code, out)
	}
}

// ADR-069 T5. urfave trims a whole attached token, so --files-from='list '
// opened list. The value is refused, and the separate spelling, which the
// parser keeps as given, still works. Root flags never reach a subcommand's
// raw tail, so refusePaddedFlagValues checks the whole argv in main.
func TestAnAttachedFlagValueWithTrailingSpaceIsRefused(t *testing.T) {
	root := paddedTree(t)
	list := filepath.Join(t.TempDir(), "list")
	for n, b := range map[string]string{list: "x\n", list + " ": "x \n"} {
		if err := os.WriteFile(n, []byte(b), 0o644); err != nil {
			t.Skipf("cannot write %q: %v", n, err)
		}
	}
	out, code := runIn(t, root, "read", "--files-from="+list+" ")
	if code != exitUsage || !strings.Contains(out, "own argument") {
		t.Errorf("--files-from='list ' exited %d, want %d saying to pass the value as its own argument:\n%s", code, exitUsage, out)
	}
	if out, code := runIn(t, root, "read", "--files-from", list+" "); code != 0 || !strings.Contains(out, "padded") {
		t.Errorf("--files-from 'list ' exited %d or did not serve x-space:\n%s", code, out)
	}
	if err := refusePaddedFlagValues(rootCommand(), []string{"--root=/tmp/d ", "read", "x"}); err == nil {
		t.Error("a padded attached --root value was not refused")
	}
	if err := refusePaddedFlagValues(rootCommand(), []string{"read", "--", "--root=x "}); err != nil {
		t.Errorf("a path after -- was refused: %v", err)
	}
}

// ADR-069 T5. The iter refusal suggested `mrw iter -- 'x '`, which makes the
// path the verb; the suggestion keeps the verb.
func TestTheIterRefusalKeepsTheVerb(t *testing.T) {
	root := plainTree(t)
	out, code := runIn(t, root, "iter", "add", "x ")
	if code != exitUsage || !strings.Contains(out, "mrw iter add -- 'x '") {
		t.Errorf("iter add 'x ' exited %d, want %d suggesting mrw iter add -- 'x ':\n%s", code, exitUsage, out)
	}
}

// ADR-069 T6, from the Codex review of PR #222. The guard looked a flag name
// up as typed, so a padded boolean name (`'--no-numbers '`, which the parser
// trims to the flag) read as value-taking, and the padded path after it was
// skipped: `read '--no-numbers ' 'x '` served x. The name is classified the
// way the parser reads it, trimmed.
func TestAPaddedBooleanFlagNameDoesNotHideAPaddedPath(t *testing.T) {
	root := plainTree(t)
	out, code := runIn(t, root, "read", "--no-numbers ", "x ")
	if code != exitUsage || !strings.Contains(out, "'x '") {
		t.Errorf("read '--no-numbers ' 'x ' exited %d, want %d naming 'x ':\n%s", code, exitUsage, out)
	}
	if out, code := runIn(t, root, "read", "--no-numbers", "x"); code != 0 || !strings.Contains(out, "plain") {
		t.Errorf("read --no-numbers x exited %d or did not serve x:\n%s", code, out)
	}
	// A padded flag NAME before a padded value that equals a positional: the
	// trimmed lookup sees --exclude and consumes ' x'; looked up as typed it
	// is unknown, ' x' is judged as a positional, and x is falsely refused.
	// (--exclude needs --grep, or read refuses the pair as a usage error.)
	if out, code := runIn(t, root, "read", "--grep ", "plain", "--exclude ", " x", "x"); code != 0 || !strings.Contains(out, "plain") {
		t.Errorf("read '--grep ' plain '--exclude ' ' x' x exited %d or did not serve x:\n%s", code, out)
	}
}

// ADR-069 T6. The whole-argv check read every "-" token as a flag, so a
// separate value that looks like one (`--files-from '--list= '`) was refused
// though the parser keeps it; and it stopped at any "--", including one a
// root flag consumed, so `--root -- --root='dir '` reached the trimmed dir.
// It reads argv as the parser does: root flags and their values, the
// subcommand, its flags and their values; only a bare "--" ends it.
func TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator(t *testing.T) {
	root := rootCommand()
	if err := refusePaddedFlagValues(root, []string{"read", "--files-from", "--list= "}); err != nil {
		t.Errorf("a separate value that looks like an attached flag was refused: %v", err)
	}
	if err := refusePaddedFlagValues(root, []string{"--root", "--", "--root=dir ", "read", "x"}); err == nil {
		t.Error("a padded root value after a -- the first root flag consumed was not refused")
	}
	if err := refusePaddedFlagValues(root, []string{"read", "--grep", "--", "--exclude=x "}); err == nil {
		t.Error("a padded attached value after a -- that --grep consumed was not refused")
	}
	if err := refusePaddedFlagValues(root, []string{"read", "--", "--exclude=x "}); err != nil {
		t.Errorf("a path after a bare -- was refused: %v", err)
	}
	tree := paddedTree(t)
	if err := os.WriteFile(filepath.Join(tree, "--list= "), []byte("x \n"), 0o644); err != nil {
		t.Skipf("cannot write %q: %v", "--list= ", err)
	}
	t.Chdir(tree)
	if out, code := runIn(t, tree, "read", "--files-from", "--list= "); code != 0 || !strings.Contains(out, "padded") {
		t.Errorf("read --files-from '--list= ' exited %d or did not serve x-space:\n%s", code, out)
	}
}

// ADR-069 T6. padAttached checked for a trailing space or tab, while the
// parser trims every whitespace (strings.TrimSpace), so --files-from=$'list\n'
// opened list. Any trailing whitespace is refused; the separate spelling,
// which the parser keeps, serves the file.
func TestAnAttachedValueEndingInAnyWhitespaceIsRefused(t *testing.T) {
	root := paddedTree(t)
	list := filepath.Join(t.TempDir(), "list")
	for _, ws := range []string{"\n", "\u00a0"} {
		if err := os.WriteFile(list+ws, []byte("x \n"), 0o644); err != nil {
			t.Skipf("cannot write %q: %v", list+ws, err)
		}
		out, code := runIn(t, root, "read", "--files-from="+list+ws)
		if code != exitUsage || !strings.Contains(out, "own argument") {
			t.Errorf("--files-from=%q exited %d, want %d saying to pass the value as its own argument:\n%s", list+ws, code, exitUsage, out)
		}
		if out, code := runIn(t, root, "read", "--files-from", list+ws); code != 0 || !strings.Contains(out, "padded") {
			t.Errorf("--files-from %q exited %d or did not serve x-space:\n%s", list+ws, code, out)
		}
	}
}

// ADR-069 T7, from the second Codex review of PR #222. A parent's persistent
// flag is accepted by a subcommand that has no flag of the same name
// (command_parse.go:43-57): write and iter take --root after the verb; read,
// whose -C is context, does not. Neither guard knew it, so `write --root --
// --root='dir '` ended the walk at the -- the root flag consumed, and
// `iter --root ' x' add x` was falsely refused. Both guards read the flags
// the parser accepts for the command, ancestors included.
func TestAnInheritedRootFlagIsReadByBothGuards(t *testing.T) {
	root := rootCommand()
	if err := refusePaddedFlagValues(root, []string{"write", "--root", "--", "--root=dir ", "p.mrw"}); err == nil {
		t.Error("a padded attached root after a -- the inherited root flag consumed was not refused")
	}
	if err := refusePaddedFlagValues(root, []string{"iter", "--root", "--", "add", "x"}); err != nil {
		t.Errorf("a -- consumed by the inherited root flag was refused: %v", err)
	}
	tree := plainTree(t)
	if err := os.MkdirAll(filepath.Join(tree, " x"), 0o755); err != nil {
		t.Skipf("cannot make %q: %v", " x", err)
	}
	if err := os.WriteFile(filepath.Join(tree, " x", "x"), []byte("q\n"), 0o644); err != nil {
		t.Skipf("cannot write under %q: %v", " x", err)
	}
	t.Chdir(tree)
	if out, code := runIn(t, tree, "iter", "--root", " x", "add", "x"); code != 0 {
		t.Errorf("iter --root ' x' add x was refused, exit %d:\n%s", code, out)
	}
	if out, code := runIn(t, tree, "iter", "--root", " x", "add", "x "); code != exitUsage {
		t.Errorf("iter --root ' x' add 'x ' exited %d, want %d:\n%s", code, exitUsage, out)
	}
}

// ADR-069 T7. A single dash before a non-letter stops the parser, which keeps
// every remaining token as given (command_parse.go:134-138); the guard read
// ` -1= ` as an attached flag value and refused it. It stops where the parser
// stops, and a real attached padded value is still refused.
func TestASingleDashNonLetterTokenStopsTheParserAndTheGuard(t *testing.T) {
	root := paddedTree(t)
	if err := os.WriteFile(filepath.Join(root, " -1= "), []byte("dashfile\n"), 0o644); err != nil {
		t.Skipf("cannot write %q: %v", " -1= ", err)
	}
	t.Chdir(root)
	if out, code := runIn(t, root, "read", " -1= "); code != 0 || !strings.Contains(out, "dashfile") {
		t.Errorf("read ' -1= ' exited %d or did not serve the file:\n%s", code, out)
	}
	if out, code := runIn(t, root, "read", "--max-lines=1 ", "x"); code != exitUsage || !strings.Contains(out, "own argument") {
		t.Errorf("--max-lines='1 ' exited %d, want %d as an attached padded value:\n%s", code, exitUsage, out)
	}
}

// ADR-069 T8, from the third Codex review of PR #222. A "--" before the
// subcommand ends the ROOT's options only: the parser still dispatches the
// subcommand, which parses its own flags (command_run.go:282-315). The
// whole-argv guard ended its walk there, and the iter guard exempted a note
// entirely, so `-- iter note --root='dir ' revised` reached dir; stats has no
// guard of its own. The walk continues below the subcommand, and the note
// exemption covers the note's words, not the flags beside them.
func TestARootTerminatorDoesNotEndTheGuardBelowIt(t *testing.T) {
	root := rootCommand()
	if err := refusePaddedFlagValues(root, []string{"--", "iter", "note", "--root=dir ", "revised"}); err == nil {
		t.Error("a padded attached root below a root-level -- was not refused")
	}
	if err := refusePaddedFlagValues(root, []string{"--", "stats", "--root=dir "}); err == nil {
		t.Error("a padded attached root on stats below a root-level -- was not refused")
	}
	if err := refusePaddedFlagValues(root, []string{"--", "read", "--", "--root=x "}); err != nil {
		t.Errorf("a path after the subcommand's own -- was refused: %v", err)
	}
	tree := plainTree(t)
	if out, code := runIn(t, tree, "iter", "note", "--root=dir ", "revised"); code != exitUsage || !strings.Contains(out, "own argument") {
		t.Errorf("iter note --root='dir ' exited %d, want %d as an attached padded value:\n%s", code, exitUsage, out)
	}
	if out, code := runIn(t, tree, "iter", "note", "revised "); code != 0 {
		t.Errorf("a note's words are free: iter note 'revised ' exited %d:\n%s", code, out)
	}
}

// ADR-069 T8. A lone "-" ends the parse: the parser keeps it as a positional
// and drops every token after it (command_parse.go:123-125), so the guard,
// reading on, refused `write - '--format=plan '`, which v1.25.0 accepted.
// Both guards stop there; the same token before the "-" is still refused.
func TestALoneDashEndsTheParseAndTheGuard(t *testing.T) {
	root := rootCommand()
	if err := refusePaddedFlagValues(root, []string{"write", "-", "--format=plan "}); err != nil {
		t.Errorf("a token the parser drops after a lone - was refused: %v", err)
	}
	if err := refusePaddedFlagValues(root, []string{"write", "--format=plan ", "-"}); err == nil {
		t.Error("an attached padded value before a lone - was not refused")
	}
	tree := plainTree(t)
	if out, code := runIn(t, tree, "read", "-", "--max-lines=1 "); strings.Contains(out, "own argument") {
		t.Errorf("read - '--max-lines=1 ' was refused by the subcommand guard (exit %d):\n%s", code, out)
	}
}

// ADR-069 T9, from the fourth Codex review of PR #222. The parser trims a
// lone "-" and KEEPS it as the positional that ends its parse
// (command_parse.go:123-125), so `write ' - '` read stdin once T8 made the
// guard stop there before judging the token, which 97f8a02 had refused. The
// stop token is judged like any positional first; after a "--" the parser
// keeps ` - ` as given, and a bare "-" is stdin.
func TestAPaddedLoneDashIsRefusedBeforeTheGuardStops(t *testing.T) {
	root := plainTree(t)
	out, code := runIn(t, root, "write", "--dry-run", "--no-check", " - ")
	if code != exitUsage || !strings.Contains(out, "' - '") || !strings.Contains(out, "edge whitespace") {
		t.Errorf("write ' - ' exited %d, want %d refusing ' - ' as a padded positional:\n%s", code, exitUsage, out)
	}
	if out, _ := runIn(t, root, "write", "--dry-run", "--no-check", "--", " - "); strings.Contains(out, "edge whitespace") {
		t.Errorf("write -- ' - ' was refused as padded though the parser keeps it as given:\n%s", out)
	}
	if out, _ := runIn(t, root, "read", " - ", "x"); !strings.Contains(out, "edge whitespace") {
		t.Errorf("read ' - ' x was not refused as a padded positional:\n%s", out)
	}
}

// ADR-069 T10, from the fifth Codex review of PR #222. A single dash before a
// non-letter stops the parser, which keeps that token and everything after it
// as given (command_parse.go:134-138); T9 judged every stop token before
// ending the walk, so `read ' -1= ' '-1='`, both files present, was refused
// because the first's trimmed spelling is the second. Only the lone "-",
// which the parser trims and keeps, is judged before the stop.
func TestAPreservedStopTokenIsNotJudgedAgainstAPositional(t *testing.T) {
	root := paddedTree(t)
	for n, b := range map[string]string{" -1= ": "padded\n", "-1=": "plain\n"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte(b), 0o644); err != nil {
			t.Skipf("cannot write %q: %v", n, err)
		}
	}
	t.Chdir(root)
	if out, code := runIn(t, root, "read", " -1= ", "-1="); code != 0 || !strings.Contains(out, "padded") || !strings.Contains(out, "plain") {
		t.Errorf("read ' -1= ' '-1=' exited %d or did not serve both as given:\n%s", code, out)
	}
	if out, code := runIn(t, root, "read", " - ", "x"); code != exitUsage || !strings.Contains(out, "edge whitespace") {
		t.Errorf("read ' - ' x exited %d, want %d as a padded lone dash:\n%s", code, exitUsage, out)
	}
}

// ADR-069 T11, found by the random differential test. An iter VERB is not a
// path: the parser trims `'add '` to the verb, and nothing reaches a file by
// it, yet the guard judged it like a positional and refused `iter 'add ' x`.
// The verb is never judged; the paths after it still are.
func TestAnIterVerbIsNotJudgedAsAPath(t *testing.T) {
	root := plainTree(t)
	if out, code := runIn(t, root, "iter", "add ", "x"); code != 0 {
		t.Errorf("iter 'add ' x was refused, exit %d:\n%s", code, out)
	}
	if out, code := runIn(t, root, "iter", "add", "x "); code != exitUsage || !strings.Contains(out, "edge whitespace") {
		t.Errorf("iter add 'x ' exited %d, want %d as a padded path:\n%s", code, exitUsage, out)
	}
}
