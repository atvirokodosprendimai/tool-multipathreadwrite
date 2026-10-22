package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// skillRun runs argv as main does: the --skill spelling first, then the parser.
func skillRun(t *testing.T, argv ...string) (string, error) {
	t.Helper()
	var w, e bytes.Buffer
	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		cmd.Writer, cmd.ErrWriter = &w, &e
		return cmd.Run(context.Background(), skillSpelling(cmd, append([]string{"mrw"}, argv...)))
	})
	return out + w.String(), err
}

// ADR-168 Decision 1. `mrw skill` and a bare `--skill` where the subcommand goes
// print the skill; a --skill that is a root flag's value is left as one; every
// other spelling keeps ADR-097's pointer; and the command reads and writes no
// state: no legacy migration and no ledger notice, while a read on the same
// fixture does both.
func TestSkillCommandPrintsTheSkill(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	want := guide.Skill(versionString())
	dir := t.TempDir()
	for _, argv := range [][]string{{"skill"}, {"--skill"}, {"-C", dir, "--skill"}} {
		if out, err := skillRun(t, argv...); err != nil || out != want {
			t.Errorf("%q: err %v, stdout is not guide.Skill(versionString()):\n%s", argv, err, out)
		}
	}
	if _, err := skillRun(t, "skill", "x"); err == nil || exitCode(err) != exitUsage {
		t.Errorf("skill x: %v, want exit %d", err, exitUsage)
	}
	if out, err := skillRun(t, "--root", "--skill", "instructions"); err != nil || out != guide.CLI() {
		t.Errorf("--root --skill instructions: the value was rewritten (err %v)", err)
	}
	for _, argv := range [][]string{{"-skill"}, {"--skill=x"}, {"--skill "}} {
		_, err := skillRun(t, argv...)
		if err == nil || exitCode(err) != exitUsage || !strings.Contains(err.Error(), "only a bare --skill prints the skill") || strings.Contains(err.Error(), "is not a flag") {
			t.Errorf("%q: %v, want the refusal that names the bare spelling and does not deny --skill", argv, err)
		}
	}
	if _, err := skillRun(t, "read", "--skill"); err == nil || exitCode(err) != exitUsage {
		t.Errorf("read --skill: %v, want exit %d", err, exitUsage)
	}

	for _, argv := range [][]string{{"skill"}, {"--skill"}} {
		t.Run("legacy "+argv[0], func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			if err := os.MkdirAll(filepath.Join(root, ".mrw"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".mrw", "seen"), []byte("#mrw-seen v4\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("one\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Chdir(root)
			sd, err := state.Dir(".")
			if err != nil {
				t.Fatal(err)
			}
			if got := stderrOf(t, func() { _, _ = skillRun(t, argv...) }); strings.Contains(got, "moved") {
				t.Errorf("%v migrated: %q", argv, got)
			}
			if _, err := os.Stat(filepath.Join(sd, "seen")); err == nil {
				t.Errorf("%v copied the legacy seen file into %s", argv, sd)
			}
			if pair := stderrOf(t, func() { runIn(t, root, "read", "f.txt") }); !strings.Contains(pair, "moved seen from ./.mrw/") {
				t.Errorf("the pair: a read after %v did not migrate: %q", argv, pair)
			}
		})
		t.Run("newer ledger "+argv[0], func(t *testing.T) {
			root := newerRoot(t)
			if got := stderrOf(t, func() { _, _ = skillRun(t, append([]string{"-C", root}, argv...)...) }); strings.Contains(got, newerKey) {
				t.Errorf("%v read the ledger: %q", argv, got)
			}
			if pair := stderrOf(t, func() { runIn(t, root, "read", "f.txt") }); !strings.Contains(pair, newerKey) {
				t.Errorf("the pair: a read did not report the newer ledger: %q", pair)
			}
		})
	}
}

// ADR-168 Decision 2. --dir DIR writes DIR/mrw/SKILL.md through the write
// engine: created, then unchanged with nothing written, then replaced when the
// stamp differs, its mode kept. It never replaces a SKILL.md a person wrote —
// the marker counts only in the frontmatter's metadata — nor writes through a
// link at mrw/ or SKILL.md, nor into .git, nor makes a DIR that is not there.
// A sibling file survives.
func TestSkillDirInstallsAndReplacesOnlyItsOwn(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	want := guide.Skill(versionString())
	d := t.TempDir()
	target := filepath.Join(d, "mrw", "SKILL.md")
	read := func(p string) string {
		t.Helper()
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	if out, err := skillRun(t, "skill", "--dir", d); err != nil || !strings.Contains(out, "created "+target) || read(target) != want {
		t.Fatalf("first --dir: err %v, out %q; the file is the skill: %v", err, out, read(target) == want)
	}
	if fi, _ := os.Stat(target); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o644 {
		t.Errorf("a created skill has mode %v, want 0644", fi.Mode().Perm())
	}
	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(target, old, old); err != nil {
		t.Fatal(err)
	}
	if out, err := skillRun(t, "skill", "--dir", d); err != nil || !strings.Contains(out, "unchanged "+target) {
		t.Errorf("second --dir: err %v, out %q, want unchanged", err, out)
	}
	if fi, _ := os.Stat(target); !fi.ModTime().Equal(old) {
		t.Errorf("an unchanged skill was rewritten: mtime %v, want %v", fi.ModTime(), old)
	}

	if err := os.WriteFile(filepath.Join(d, "mrw", "notes.txt"), []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(guide.Skill("v0.0.1 (1111111)")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := skillRun(t, "--skill", "--dir", d)
	if err != nil || !strings.Contains(out, "replaced "+target) || !strings.Contains(out, "v0.0.1 (1111111) → "+versionString()) || read(target) != want {
		t.Errorf("--skill --dir over an older stamp: err %v, out %q", err, out)
	}
	if fi, _ := os.Stat(target); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Errorf("a replaced skill's mode became %v, want the 0600 it had", fi.Mode().Perm())
	}
	if read(filepath.Join(d, "mrw", "notes.txt")) != "mine\n" {
		t.Error("a sibling file in DIR/mrw did not survive")
	}

	for name, body := range map[string]string{
		"hand-written":          "---\nname: mrw\ndescription: mine\n---\nmy own skill\n",
		"marker in the body":    "---\nname: mrw\ndescription: mine\n---\nmetadata:\n  generated-by: \"mrw skill\"\n",
		"marker not in meta":    "---\nname: mrw\n  generated-by: \"mrw skill\"\n---\nx\n",
		"malformed frontmatter": "generated-by: \"mrw skill\"\n",
	} {
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := skillRun(t, "skill", "--dir", d); err == nil || exitCode(err) != exitNotApplied {
			t.Errorf("%s: err %v, want exit %d", name, err, exitNotApplied)
		}
		if read(target) != body {
			t.Errorf("%s: the file was changed", name)
		}
	}

	if runtime.GOOS != "windows" {
		l := t.TempDir()
		if err := os.MkdirAll(filepath.Join(l, "mrw"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(l, "real.md"), []byte(guide.Skill("v0.0.1 (1111111)")), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(l, "real.md"), filepath.Join(l, "mrw", "SKILL.md")); err != nil {
			t.Fatal(err)
		}
		// The target is an older stamped skill, so only the link check refuses it.
		if _, err := skillRun(t, "skill", "--dir", l); err == nil || read(filepath.Join(l, "real.md")) != guide.Skill("v0.0.1 (1111111)") {
			t.Errorf("SKILL.md as a link to a stamped skill inside DIR: err %v, the target was written through", err)
		}
		outside, m := t.TempDir(), t.TempDir()
		if err := os.Symlink(outside, filepath.Join(m, "mrw")); err != nil {
			t.Fatal(err)
		}
		if _, err := skillRun(t, "skill", "--dir", m); err == nil {
			t.Error("DIR/mrw as a link to a directory elsewhere was written through")
		}
		if _, err := os.Stat(filepath.Join(outside, "SKILL.md")); err == nil {
			t.Error("the skill landed outside DIR, through the link")
		}
		// A link inside DIR stays inside the engine's root, so only --dir's own
		// check refuses it.
		in := t.TempDir()
		if err := os.Mkdir(filepath.Join(in, "real"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(in, "real"), filepath.Join(in, "mrw")); err != nil {
			t.Fatal(err)
		}
		if _, err := skillRun(t, "skill", "--dir", in); err == nil {
			t.Error("DIR/mrw as a link to a folder inside DIR was written through")
		}
		if _, err := os.Stat(filepath.Join(in, "real", "SKILL.md")); err == nil {
			t.Error("the skill landed in DIR/real, through the mrw link")
		}
	}

	// A --dir write is a write: a legacy ./.mrw/ in the working directory moves
	// first, and a DIR whose ledger a newer mrw wrote is named.
	legacy := t.TempDir()
	if err := os.MkdirAll(filepath.Join(legacy, ".mrw"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, ".mrw", "seen"), []byte("#mrw-seen v4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(legacy)
	if got := stderrOf(t, func() { _, _ = skillRun(t, "skill", "--dir", ".") }); !strings.Contains(got, "moved seen from ./.mrw/") {
		t.Errorf("--dir . did not migrate the legacy state first: %q", got)
	}
	newer := newerRoot(t)
	if got := stderrOf(t, func() { _, _ = skillRun(t, "skill", "--dir", newer) }); !strings.Contains(got, newerKey) {
		t.Errorf("--dir into a root whose ledger a newer mrw wrote said nothing: %q", got)
	}
	g := filepath.Join(t.TempDir(), ".git")
	if err := os.Mkdir(g, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := skillRun(t, "skill", "--dir", g); err == nil {
		t.Error("--dir inside .git was written")
	}
	if _, err := os.Stat(filepath.Join(g, "mrw", "SKILL.md")); err == nil {
		t.Error("a skill landed in .git")
	}

	missing := filepath.Join(t.TempDir(), "skils")
	if _, err := skillRun(t, "skill", "--dir", missing); err == nil || exitCode(err) != exitUsage {
		t.Errorf("a missing DIR: %v, want exit %d", err, exitUsage)
	}
	if _, err := os.Stat(missing); err == nil {
		t.Error("a missing DIR was created")
	}
	file := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := skillRun(t, "skill", "--dir", file); err == nil || exitCode(err) != exitUsage {
		t.Errorf("DIR that is a file: %v, want exit %d", err, exitUsage)
	}
}

// ADR-168 Decision 4. `mrw --help` and a bare `mrw` open with the line in
// capitals asked for on 2026-10-22, then how to install the skill and the rules
// alone; the help stays a screen (25 lines measured that day); the full contract
// ends naming `mrw skill` and never spells --skill, a rewrite rather than a
// declared flag; the MCP handshake is untouched; a subcommand's help is too.
func TestRootHelpPointsAgentsAtTheSkill(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for _, argv := range [][]string{{"--help"}, {}} {
		out, err := skillRun(t, argv...)
		if err != nil {
			t.Fatalf("mrw %v: %v", argv, err)
		}
		if first, _, _ := strings.Cut(out, "\n"); first != "AGENTS: READ THE SKILL FIRST: mrw --skill" {
			t.Errorf("mrw %v opens with %q", argv, first)
		}
		for _, want := range []string{"mrw --skill --dir", "mrw instructions --core"} {
			if !strings.Contains(out, want) {
				t.Errorf("mrw %v does not name %q", argv, want)
			}
		}
		if n := strings.Count(strings.TrimRight(out, "\n"), "\n") + 1; n > 29 {
			t.Errorf("mrw %v is %d lines, over 29", argv, n)
		}
	}
	full := strings.Split(strings.TrimRight(guide.CLI(), "\n"), "\n")
	if last := full[len(full)-1]; !strings.Contains(last, "mrw skill") {
		t.Errorf("mrw instructions does not end naming mrw skill: %q", last)
	}
	if strings.Contains(guide.CLI(), "--skill") {
		t.Error("mrw instructions spells --skill, which no command declares")
	}
	if strings.Contains(guide.Shared(), "skill") {
		t.Error("the MCP handshake names the skill")
	}
	if out, _ := skillRun(t, "read", "--help"); strings.HasPrefix(out, "AGENTS") {
		t.Error("a subcommand's help carries the root's agents line")
	}
}

// ADR-168 Decision 3. The skill Claude Code loads in this checkout is the
// binary's own: what `mrw skill` prints for the version stamped in the file,
// so it cannot drift from the binary as the hand mirror did.
func TestTheRepositorySkillIsTheBinarysSkill(t *testing.T) {
	b, err := os.ReadFile("../../.claude/skills/mrw/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	v, _, _, ours := skillStamp(string(b))
	if !ours {
		t.Fatal(".claude/skills/mrw/SKILL.md was not written by mrw skill: regenerate it with mrw skill --dir .claude/skills")
	}
	if string(b) != guide.Skill(v) {
		t.Errorf(".claude/skills/mrw/SKILL.md differs from mrw skill's output for its own stamp %q: regenerate it with mrw skill --dir .claude/skills", v)
	}
}

// ADR-168 Decision 2, the Codex review of 2026-10-22. --dir judges a SKILL.md by
// the exact shape mrw skill prints, so a generated-by line inside a quoted
// description is a person's file; a copy whose line endings a host's tools
// changed, or whose final newline is gone, is still mrw's, is replaced once and
// then unchanged; a FIFO swapped in is refused, not waited on; a file larger
// than any skill mrw writes is not judged mrw's.
func TestSkillDirReadsOnlyTheShapeItPrints(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	install := func(t *testing.T, content string) (string, string, error) {
		t.Helper()
		d := t.TempDir()
		if err := os.MkdirAll(filepath.Join(d, "mrw"), 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(d, "mrw", "SKILL.md")
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := skillRun(t, "skill", "--dir", d)
		return d, out, err
	}
	decoy := "---\nname: mrw\ndescription: 'mine,\n  metadata:\n  generated-by: \"mrw skill\"'\nmetadata:\n  mrw-version: \"v0.0.1\"\n  generated-by: \"mrw skill\"\n---\nmy own\n"
	if _, _, err := install(t, decoy); err == nil || exitCode(err) != exitNotApplied {
		t.Errorf("a quoted description holding the marker: %v, want exit %d", err, exitNotApplied)
	}
	big := guide.Skill("v0.0.1") + strings.Repeat("x", maxSkillBytes)
	if _, _, err := install(t, big); err == nil || exitCode(err) != exitNotApplied {
		t.Errorf("a stamped file over %d bytes: %v, want exit %d", maxSkillBytes, err, exitNotApplied)
	}
	crlf := func(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }
	if _, out, err := install(t, crlf(guide.Skill(versionString()))); err != nil || !strings.Contains(out, "unchanged ") {
		t.Errorf("a CRLF copy of the current skill: err %v, out %q, want unchanged", err, out)
	}
	for name, old := range map[string]string{
		"CRLF":             crlf(guide.Skill("v0.0.1")),
		"no final newline": strings.TrimSuffix(guide.Skill("v0.0.1"), "\n"),
	} {
		d, out, err := install(t, old)
		if err != nil || !strings.Contains(out, "replaced ") {
			t.Errorf("%s older skill: err %v, out %q, want replaced", name, err, out)
		}
		if again, err := skillRun(t, "skill", "--dir", d); err != nil || !strings.Contains(again, "unchanged ") {
			t.Errorf("%s older skill, second run: err %v, out %q, want unchanged", name, err, again)
		}
	}
}

// ADR-168 Decision 2, the chaos tests of 2026-10-22 (five Mac sessions, two
// Windows). A body edited under the stamped frontmatter is a person's text and
// is never replaced; a file over the bound says so, not "not mrw's"; a byte
// order mark is read as CRLF is, and "unchanged" says when the bytes differ; a
// skill a newer release stamped is not rolled back; --dir twice is refused; a
// control character in DIR is quoted.
func TestSkillDirKeepsWhatAPersonChanged(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	put := func(t *testing.T, content string) (string, string) {
		t.Helper()
		d := t.TempDir()
		if err := os.MkdirAll(filepath.Join(d, "mrw"), 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(d, "mrw", "SKILL.md")
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return d, p
	}
	cur := guide.Skill(versionString())
	for name, edited := range map[string]string{
		"a line appended":       cur + "- my local rule\n",
		"the body replaced":     cur[:strings.Index(cur, "\n---\n")+5] + "MY PRIVATE NOTES\n",
		"an older skill edited": guide.Skill("v0.0.1") + "- mine\n",
	} {
		d, p := put(t, edited)
		_, err := skillRun(t, "skill", "--dir", d)
		if err == nil || exitCode(err) != exitNotApplied || !strings.Contains(err.Error(), "was edited after `mrw skill` wrote it") {
			t.Errorf("%s: %v, want exit %d naming the edit", name, err, exitNotApplied)
		}
		if b, _ := os.ReadFile(p); string(b) != edited {
			t.Errorf("%s: the edited file was changed", name)
		}
	}
	bigDir, _ := put(t, cur+strings.Repeat("x", maxSkillBytes))
	if _, err := skillRun(t, "skill", "--dir", bigDir); err == nil || !strings.Contains(err.Error(), "larger than any skill") {
		t.Errorf("a file over the bound: %v, want the size named", err)
	}
	bom := "\ufeff"
	if d, _ := put(t, bom+cur); true {
		out, err := skillRun(t, "skill", "--dir", d)
		if err != nil || !strings.Contains(out, "unchanged ") || !strings.Contains(out, "it has a byte order mark, which mrw skill's output does not, and they are kept") {
			t.Errorf("a BOM copy of the current skill: err %v, out %q, want unchanged with the byte order mark named", err, out)
		}
	}
	if d, _ := put(t, bom+guide.Skill("v0.0.1")); true {
		if out, err := skillRun(t, "skill", "--dir", d); err != nil || !strings.Contains(out, "replaced ") {
			t.Errorf("a BOM older skill: err %v, out %q, want replaced", err, out)
		}
		if out, err := skillRun(t, "skill", "--dir", d); err != nil || !strings.Contains(out, "unchanged ") {
			t.Errorf("a BOM older skill, second run: err %v, out %q, want unchanged", err, out)
		}
	}
	saved := version
	version = "v1.73.0"
	t.Cleanup(func() { version = saved })
	newer := guide.Skill("v99.0.0 (abcdef0)")
	d, p := put(t, newer)
	if _, err := skillRun(t, "skill", "--dir", d); err == nil || exitCode(err) != exitNotApplied || !strings.Contains(err.Error(), "a newer release than this mrw") {
		t.Errorf("a skill a newer release stamped: %v, want exit %d", err, exitNotApplied)
	}
	if b, _ := os.ReadFile(p); string(b) != newer {
		t.Error("a newer skill was rolled back")
	}
	if !newerRelease("v1.74.0", "v1.73.9-dev+abc") || newerRelease("v1.73.0", "v1.73.0-dev") || newerRelease("dev", "v1.0.0") || newerRelease("v2.0.0", "dev") {
		t.Error("newerRelease compares the wrong versions")
	}
	version = saved
	if _, err := skillRun(t, "skill", "--dir", t.TempDir(), "--dir", t.TempDir()); err == nil || exitCode(err) != exitUsage {
		t.Errorf("--dir twice: %v, want exit %d", err, exitUsage)
	}
	if runtime.GOOS != "windows" {
		nl := filepath.Join(t.TempDir(), "a\nb")
		if err := os.Mkdir(nl, 0o755); err != nil {
			t.Fatal(err)
		}
		out, err := skillRun(t, "skill", "--dir", nl)
		if err != nil || strings.Count(strings.TrimRight(out, "\n"), "\n") != 0 || !strings.Contains(out, `a\nb`) {
			t.Errorf("a DIR holding a newline: err %v, out %q, want one line naming it quoted", err, out)
		}
	}
}

// ADR-168 Decision 2, the chaos re-runs of 2026-10-22. The content-sha256
// covers the description as well as the body, so an edit to the trigger is a
// person's too; a byte order mark stays through a replace, as CRLF does; an
// empty SKILL.md is the engine's probe of a racing run (ADR-105) until it stays
// empty; and an engine refusal names the folder as the caller did without
// touching an absolute path that holds mrw/, nor showing a temporary name.
func TestSkillDirKeepsTheTriggerAndTheBOM(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	put := func(t *testing.T, content string) (string, string) {
		t.Helper()
		d := t.TempDir()
		if err := os.MkdirAll(filepath.Join(d, "mrw"), 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(d, "mrw", "SKILL.md")
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return d, p
	}
	cur := guide.Skill(versionString())
	edited := strings.Replace(cur, "Plan one read", "PLAN ONE READ", 1)
	if edited == cur {
		t.Fatal("the fixture did not edit the description")
	}
	d, p := put(t, edited)
	if _, err := skillRun(t, "skill", "--dir", d); err == nil || exitCode(err) != exitNotApplied || !strings.Contains(err.Error(), "was edited after `mrw skill` wrote it") {
		t.Errorf("an edited description: %v, want exit %d naming the edit", err, exitNotApplied)
	}
	if b, _ := os.ReadFile(p); string(b) != edited {
		t.Error("the edited description was replaced")
	}
	const bom = "\xef\xbb\xbf"
	d, p = put(t, bom+strings.ReplaceAll(guide.Skill("v0.0.1"), "\n", "\r\n"))
	if out, err := skillRun(t, "skill", "--dir", d); err != nil || !strings.Contains(out, "replaced ") || !strings.Contains(out, "v0.0.1 → ") {
		t.Errorf("a BOM and CRLF older skill: err %v, out %q, want replaced naming the old version", err, out)
	}
	if out, err := skillRun(t, "skill", "--dir", d); err != nil || !strings.Contains(out, "unchanged ") || !strings.Contains(out, "it has a byte order mark and CRLF line endings") {
		t.Errorf("the replaced BOM and CRLF skill, run again: err %v, out %q, want unchanged naming both", err, out)
	}
	if b, _ := os.ReadFile(p); !strings.HasPrefix(string(b), bom+"---") || normalised(string(b)) != cur {
		t.Errorf("a replace dropped the byte order mark or wrote other text: %q", string(b[:min(len(b), 12)]))
	}
	d, p = put(t, "")
	if _, err := skillRun(t, "skill", "--dir", d); err == nil || exitCode(err) != exitNotApplied || !strings.Contains(err.Error(), "is empty") {
		t.Errorf("an empty SKILL.md that stays empty: %v, want exit %d", err, exitNotApplied)
	}
	if b, _ := os.ReadFile(p); len(b) != 0 {
		t.Error("an empty SKILL.md was written")
	}
	// Windows PowerShell 5.1's redirect of mrw --skill writes UTF-16 with a
	// byte order mark (the chaos test of 2026-10-22): the refusal says so.
	if _, err := func() (string, error) {
		d, _ := put(t, "\xff\xfe-\x00-\x00-\x00")
		return skillRun(t, "skill", "--dir", d)
	}(); err == nil || exitCode(err) != exitNotApplied || !strings.Contains(err.Error(), "is UTF-16") {
		t.Errorf("a UTF-16 SKILL.md: %v, want exit %d naming UTF-16", err, exitNotApplied)
	}
	for _, c := range []struct{ reason, want string }{
		{"mrw/SKILL.md is read-only; chmod u+w mrw/SKILL.md", "D/mrw/SKILL.md is read-only; chmod u+w D/mrw/SKILL.md"},
		{"held open (open /x/lk/mrw/SKILL.md: busy)", "held open (open /x/lk/mrw/SKILL.md: busy)"},
		{"openat mrw/.mrw-c65058739c09edc2: permission denied", "openat D/mrw/(a temporary file): permission denied"},
		{"mrw/SKILL.md is refused", "D/mrw/SKILL.md is refused"},
	} {
		if got := callersReason(c.reason, "D"); got != filepath.FromSlash(c.want) && got != c.want {
			t.Errorf("callersReason(%q) = %q, want %q", c.reason, got, c.want)
		}
	}
}

// The Windows chaos tests of 2026-10-22: Win32 reads "d1." and "d1 " as d1, so a
// DIR spelled with a trailing dot or space in any component names another
// folder than the one it reads as, and --dir refuses it on Windows (ADR-071's
// rule for a name, applied to the folder --dir is given).
func TestUnkeptComponentFindsATrailingDotOrSpace(t *testing.T) {
	for dir, want := range map[string]string{
		"d1.": "d1.", "d1 ": "d1 ", `a\b.\c`: "b.", "a/b /c": "b ",
		"d1": "", ".": "", "..": "", "./d1/": "", `..\d1`: "", `C:\x\y`: "", "": "",
	} {
		if got := unkeptComponent(dir); got != want {
			t.Errorf("unkeptComponent(%q) = %q, want %q", dir, got, want)
		}
	}
}
