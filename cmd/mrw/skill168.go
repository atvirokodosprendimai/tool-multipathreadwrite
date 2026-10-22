package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer"
	"github.com/urfave/cli/v3"
)

// skillCmd prints mrw's Agent Skills SKILL.md (ADR-168): the frontmatter a
// skill host reads every session, then the core rules and the recipes, so a
// host that loads skills is handed the binary's own text rather than a copy.
func skillCmd() *cli.Command {
	return &cli.Command{
		Name:  "skill",
		Usage: "print mrw's Agent Skills SKILL.md: the core rules and the recipes, with the frontmatter a skill host reads; --dir DIR installs it as DIR/mrw/SKILL.md",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "dir", Usage: "write the skill to `DIR`/mrw/SKILL.md through the write engine, replacing only a SKILL.md mrw skill wrote (ADR-168)"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 0 {
				return cli.Exit("skill takes no arguments", exitUsage)
			}
			out := cmd.Root().Writer
			if out == nil {
				out = os.Stdout
			}
			if cmd.Count("dir") > 1 {
				return cli.Exit("--dir is given more than once; name one folder", exitUsage)
			}
			if cmd.IsSet("dir") {
				return installSkill(cmd.String("dir"), out)
			}
			_, err := fmt.Fprint(out, guide.Skill(versionString()))
			return err
		},
	}
}

// skillSpelling reads a bare --skill where the subcommand goes as the
// subcommand skill (ADR-168 Decision 1), because `mrw --help` tells agents to
// type it. Root options are walked as the parser reads them, so a --skill that
// is a root flag's value (mrw --root --skill …) is left as one. Only the exact
// token is taken: -skill, --skill=x and a padded "--skill " keep ADR-097's
// pointer at the subcommand, so no spelling list has to match the parser's.
// args[0] is the program.
func skillSpelling(root *cli.Command, args []string) []string {
	kinds := takesValue(root.Flags)
	for i := 1; i < len(args); i++ {
		if args[i] == "--skill" {
			out := append([]string{}, args...)
			out[i] = "skill"
			return out
		}
		role, next := flagRole(args[i], kinds)
		if role != roleFlag {
			return args
		}
		if next {
			i++
		}
	}
	return args
}

// installSkill writes the skill to DIR/mrw/SKILL.md through the write engine
// (ADR-168 Decision 2), as `mrw --root DIR write` would apply one hunk: a create
// when the file is absent, a whole-file replace when it is mrw's own and its
// content is the one mrw stamped. The engine keeps every guard a write has — a
// link that leaves DIR, a reserved name, .git, mrw's state, a held file — and
// --dir adds the ones a skill needs: DIR must already exist, a SKILL.md a person
// wrote or edited is never replaced, and a link inside DIR (which the engine
// would follow) is refused at mrw and at SKILL.md. A run that loses a race to
// another --dir judges again, at most five times, so installs that agree do not
// report a failure.
func installSkill(dir string, out io.Writer) error {
	if runtime.GOOS == "windows" {
		if c := unkeptComponent(dir); c != "" {
			return cli.Exit(fmt.Sprintf("--dir %s: Windows does not keep %s as written — it drops a trailing dot or space — so it would open another folder; name the folder as it is", quoted(dir), quoted(c)), exitUsage)
		}
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return cli.Exit(fmt.Sprintf("--dir %s: not an existing directory; mrw does not create the folder a host reads skills from", quoted(dir)), exitUsage)
	}
	// A --dir install is a write, so a legacy ./.mrw/ moves first, as it does
	// before any write (ADR-004); the Before hook skips it for plain skill.
	migrateLegacyState()
	for attempt := 1; ; attempt++ {
		raced, err := installOnce(dir, out)
		if !raced || attempt == 5 {
			return err
		}
		// The engine's probe of a new name is an empty file for a moment
		// (ADR-105); a pause lets a racing run finish before judging again.
		time.Sleep(time.Duration(attempt) * 20 * time.Millisecond)
	}
}

// bom is the UTF-8 byte order mark an editor may put before a SKILL.md.
const bom = "\xef\xbb\xbf"

// installOnce judges DIR/mrw/SKILL.md and writes the skill once. It reports
// raced when another --dir is mid-write — the file is the engine's empty probe,
// or the engine refused because the file changed between the judging and the
// write — so the caller judges again.
func installOnce(dir string, out io.Writer) (raced bool, err error) {
	text := guide.Skill(versionString())
	target := filepath.Join(dir, "mrw", "SKILL.md")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	in := apply.Input{Path: "mrw/SKILL.md", Op: "create", Body: lines, CountedBody: true, Lines: -1}
	verb, old, was := "created", "", ""
	if fi, err := os.Lstat(filepath.Join(dir, "mrw")); err == nil && (fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir()) {
		return false, cli.Exit(fmt.Sprintf("%s is not a directory (a file, a link or a junction); --dir writes only into a folder it can name", shownPath(filepath.Join(dir, "mrw"))), exitNotApplied)
	}
	var judged []byte
	switch fi, err := os.Lstat(target); {
	case err == nil && !fi.Mode().IsRegular():
		return false, cli.Exit(fmt.Sprintf("%s is not a regular file (a link, a junction, a FIFO, a device or a directory); --dir replaces only a SKILL.md it wrote", shownPath(target)), exitNotApplied)
	case err == nil:
		b, err := readSkillFile(target)
		if err != nil {
			return false, err
		}
		if len(b) == 0 {
			return true, cli.Exit(fmt.Sprintf("%s is empty, so it is left as it is; move it aside to install here", shownPath(target)), exitNotApplied)
		}
		if len(b) > maxSkillBytes {
			return false, cli.Exit(fmt.Sprintf("%s is larger than any skill `mrw skill` writes (over %d bytes), so it is left as it is; move it aside to install here", shownPath(target), maxSkillBytes), exitNotApplied)
		}
		version, sha, unsigned, ours := skillStamp(string(b))
		if !ours {
			// The two UTF-16 byte order marks, little- and big-endian, as bytes:
			// as strings both are invalid UTF-8, which a reader can take for one.
			if bytes.HasPrefix(b, []byte{0xff, 0xfe}) || bytes.HasPrefix(b, []byte{0xfe, 0xff}) {
				return false, cli.Exit(fmt.Sprintf("%s is UTF-16, which mrw skill never writes — a Windows PowerShell 5.1 redirect (mrw --skill > file) writes it so — so it is left as it is; move it aside, and install with mrw --skill --dir, which writes UTF-8", shownPath(target)), exitNotApplied)
			}
			return false, cli.Exit(fmt.Sprintf("%s was not written by `mrw skill` (its frontmatter is not the shape mrw skill prints), so it is left as it is; move it aside to install here", shownPath(target)), exitNotApplied)
		}
		if guide.ContentSHA(unsigned) != sha {
			return false, cli.Exit(fmt.Sprintf("%s was edited after `mrw skill` wrote it (it no longer matches the content-sha256 in its frontmatter), so it is left as it is; move it aside to install here", shownPath(target)), exitNotApplied)
		}
		if newerRelease(version, versionString()) {
			return false, cli.Exit(fmt.Sprintf("%s was written by mrw %s, a newer release than this mrw (%s), so it is left as it is; install with the newer mrw, or move it aside", shownPath(target), version, versionString()), exitNotApplied)
		}
		// Line endings, a final newline and a byte order mark are the file's
		// own — a replace keeps them — so text equal up to them is unchanged,
		// and the line says which of them differ from what mrw skill prints.
		if normalised(string(b)) == text {
			_, err := fmt.Fprintf(out, "unchanged %s (mrw-version %s%s)\n", shownPath(target), version, keptDifference(string(b), false))
			return false, err
		}
		// The bytes judged above are the bytes replaced: sha= refuses the write
		// if the file changed since, and Force stands for the read this was. A
		// byte order mark the file began with stays, as its line endings do.
		judged = b
		verb = "replaced"
		if version == versionString() {
			was = "; its content differed from what this mrw prints"
		} else {
			old = version + " → "
		}
		was += keptDifference(string(b), true)
		body := append([]string{}, lines...)
		if strings.HasPrefix(string(b), bom) {
			body[0] = bom + body[0]
		}
		in = apply.Input{Path: "mrw/SKILL.md", Start: 1, End: apply.EOF, Op: "replace", Body: body, CountedBody: true,
			SHA: seen.SHA(b), Anchor: "---", Lines: -1}
	case !errors.Is(err, fs.ErrNotExist):
		return false, targetErr(target, err)
	}
	prep, err := writer.Prepare(writer.Request{Root: dir, In: []apply.Input{in}, Check: writer.CheckOff, Opts: apply.Options{Force: in.Op == "replace"}})
	if err != nil {
		return false, exitWith(gateRefusal(err))
	}
	land, err := prep.Land()
	if err != nil {
		return false, exitWith(gateRefusal(err))
	}
	// A landing that left something behind is a failure to report, never a race
	// to retry: its leftovers and its own exit stand (the Codex review of 6860a48).
	if (land.Err != nil || land.Res.Failed > 0) && len(land.Res.LeftBehind) == 0 && changedSince(target, judged) {
		return true, cli.Exit(fmt.Sprintf("%s kept changing while --dir wrote it; nothing was written", shownPath(target)), exitNotApplied)
	}
	if land.Err != nil || land.Res.Failed > 0 {
		landed := ""
		if land.Err != nil {
			landed = fmt.Sprint(land.Err)
		}
		for _, h := range land.Res.Hunks {
			// The landing's own error is the exit's message: a reason that only
			// repeats it is not printed twice (the Windows chaos test of 2026-10-22).
			if h.Reason != "" && (landed == "" || !strings.Contains(h.Reason, landed)) {
				fmt.Fprintf(os.Stderr, "mrw: %s\n", callersReason(h.Reason, dir))
			}
		}
		// What the engine made and could not take away is named, as a write's
		// receipt names it (ADR-105): the refusal above does not carry it.
		for _, p := range land.Res.LeftBehind {
			fmt.Fprintf(os.Stderr, "mrw: left behind: %s\n", shownPath(filepath.Join(dir, p)))
		}
		if land.Err != nil {
			return false, exitWith(land.Err)
		}
		return false, cli.Exit(fmt.Sprintf("%s: the skill was refused — nothing was written", shownPath(target)), exitNotApplied)
	}
	if notice := writer.NewerUntold(dir, land.Res); notice != "" {
		fmt.Fprintln(os.Stderr, notice)
	}
	// The file changed whatever the ledger says, so the line that says so comes
	// first; a ledger that could not record it is then the exit's reason.
	_, err = fmt.Fprintf(out, "%s %s (mrw-version %s%s%s)\n", verb, shownPath(target), old, versionString(), was)
	if land.LedgerErr != nil {
		return false, exitWith(land.LedgerErr)
	}
	return false, err
}

// unkeptComponent is the first component of dir that Windows would not keep as
// written — one ending in a dot or a space, other than . and .. — or "". Win32
// opens such a name as the name without it (ADR-071), so a DIR spelled so
// names another folder than the one it reads as.
func unkeptComponent(dir string) string {
	for _, c := range strings.FieldsFunc(dir, func(r rune) bool { return r == '/' || r == '\\' }) {
		if c != "." && c != ".." && (strings.HasSuffix(c, ".") || strings.HasSuffix(c, " ")) {
			return c
		}
	}
	return ""
}

// keptDifference names how a file text-equal to the skill differs in its bytes,
// or is empty when it does not.
func keptDifference(file string, replaced bool) string {
	var kept []string
	if strings.HasPrefix(file, bom) {
		kept = append(kept, "a byte order mark")
	}
	// A file all in CRLF keeps it through a replace; a file of mixed endings
	// keeps them only while it is left alone, since a replace writes LF.
	switch crlf := strings.Count(file, "\r\n"); {
	case crlf > 0 && crlf == strings.Count(file, "\n"):
		kept = append(kept, "CRLF line endings")
	case crlf > 0 && !replaced:
		kept = append(kept, "mixed line endings")
	}
	if !strings.HasSuffix(file, "\n") {
		kept = append(kept, "no final newline")
	}
	if len(kept) == 0 {
		return ""
	}
	return "; it has " + strings.Join(kept, " and ") + ", which mrw skill's output does not, and they are kept"
}

// callerPath matches the engine's root-relative name of the skill or of a
// temporary file beside it where a path begins — at the start, or after a
// space, a quote or a bracket — and nothing else, so an absolute path that
// holds mrw/ (DIR "/tmp/host mrw" included) is left as it is.
var callerPath = regexp.MustCompile(`(^|[\s("'])mrw([/\\])(SKILL\.md|\(a temporary file\))`)

// callersReason is an engine refusal in the caller's spelling: DIR/mrw/… for
// the engine's mrw/…, with the separator the engine wrote there, and the name
// of a temporary file the engine could not make replaced by what it was. An
// absolute path the engine names under DIR is set aside first and put back
// last, so no spelling of DIR — "/tmp/host mrw/SKILL.md" included — can make a
// relative name appear inside it (the Codex review of 50c7c60).
func callersReason(reason, dir string) string {
	var spans []string
	if abs, err := filepath.Abs(dir); err == nil {
		spans = append(spans, abs)
		if real, err := filepath.EvalSymlinks(abs); err == nil && real != abs {
			spans = append(spans, real)
		}
	}
	// The longest first, so a resolved path that holds the other is taken whole.
	sort.Slice(spans, func(i, j int) bool { return len(spans[i]) > len(spans[j]) })
	for i, s := range spans {
		reason = strings.ReplaceAll(reason, s, fmt.Sprintf("\x00%d\x00", i))
	}
	reason = tempName.ReplaceAllString(reason, "${1}mrw${2}(a temporary file)")
	reason = callerPath.ReplaceAllStringFunc(reason, func(m string) string {
		sm := callerPath.FindStringSubmatch(m)
		folder := filepath.Join(dir, "mrw")
		if sm[2] == "/" {
			folder = filepath.ToSlash(folder)
		}
		return sm[1] + shownPath(folder) + sm[2] + sm[3]
	})
	for i, s := range spans {
		reason = strings.ReplaceAll(reason, fmt.Sprintf("\x00%d\x00", i), s)
	}
	return reason
}

// tempName is a temporary file the engine names in a failed open or create.
var tempName = regexp.MustCompile(`(^|[\s("'])mrw([/\\])\.mrw-[0-9a-f]+`)

// changedSince reports whether target no longer holds judged — a create that
// found a file, or a replace whose sha= no longer matched — which is a race
// with another --dir, not a refusal of the caller's.
func changedSince(target string, judged []byte) bool {
	// Read as readSkillFile reads, so a FIFO or a device swapped in after the
	// judging is no race — the engine's refusal of it stands — and a re-read
	// never waits on one or reads past the bound (the Codex review of 8c1f08b).
	f, _, err := regular.Open(target)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, maxSkillBytes+1))
	return err == nil && (judged == nil || string(b) != string(judged))
}

// quoted is s in double quotes, as a Windows or POSIX user reads a path, with
// Go's escaping only when s holds a control character: %q would double every
// backslash of a Windows path (the Windows chaos test of 2026-10-22).
func quoted(s string) string {
	if strings.IndexFunc(s, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return strconv.Quote(s)
	}
	return `"` + s + `"`
}

// shownPath is p as the caller can read it back: quoted when it holds a control
// character, so a newline in a name cannot make one line of output look like two.
func shownPath(p string) string {
	if strings.IndexFunc(p, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return strconv.Quote(p)
	}
	return p
}

// maxSkillBytes bounds what --dir reads before judging a SKILL.md: the one mrw
// prints is a few kilobytes, so a larger file is not one it wrote.
const maxSkillBytes = 1 << 20

// readSkillFile reads an existing SKILL.md through regular.Open (ADR-109), so a
// file swapped for a FIFO after the Lstat is refused rather than waited on, and
// reads no more than one byte past maxSkillBytes.
func readSkillFile(path string) ([]byte, error) {
	f, _, err := regular.Open(path)
	if err != nil {
		return nil, targetErr(path, err)
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, maxSkillBytes+1))
	if err != nil {
		return nil, targetErr(path, err)
	}
	return b, nil
}

// targetErr is a SKILL.md --dir could not read: a permission, a file that is
// not regular, or one another process holds is the target's cause, exit 1;
// anything else, an I/O error, is the environment's, exit 2 — the split a
// write's staging makes (ADR-132, ADR-125). A held file is worded as the
// engine words one, so the two locks a caller can meet read alike.
func targetErr(path string, err error) error {
	// Each keeps err to unwrap, as every command's failure does, so the state
	// directory's hint can still see one (TestNoCommandExitsWithAnErrorThroughCliExit).
	switch {
	case heldByAnother(err):
		return &exitError{err: err, msg: shownPath(path) + " cannot be opened: held open by another process; nothing was written", code: exitNotApplied}
	case errors.Is(err, fs.ErrPermission) || errors.Is(err, regular.ErrNotRegular):
		return &exitError{err: err, msg: fmt.Sprintf("%s: %v; nothing was written", shownPath(path), err), code: exitNotApplied}
	}
	return &exitError{err: err, msg: fmt.Sprintf("%s: %v", shownPath(path), err), code: exitUsage}
}

// skillStamp reports a SKILL.md's mrw-version, the content-sha256 it carries,
// the file without that line (what the digest covers), and whether `mrw skill`
// wrote it, by the exact frontmatter shape Skill prints: name: mrw, a folded
// description of indented lines, then metadata holding mrw-version,
// generated-by and content-sha256 and nothing else. Any other shape is a
// person's file: a generated-by line inside a quoted description, the body or a
// comment does not make it mrw's. CRLF, a missing final newline and a leading
// byte order mark are read as the engine keeps them.
func skillStamp(s string) (version, sha, unsigned string, ours bool) {
	n := normalised(s)
	rest, ok := strings.CutPrefix(n, "---\n")
	if !ok {
		return "", "", "", false
	}
	front, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return "", "", "", false
	}
	ls := strings.Split(front, "\n")
	if len(ls) < 7 || ls[0] != "name: mrw" || ls[1] != "description: >-" {
		return "", "", "", false
	}
	i := 2
	for i < len(ls) && strings.HasPrefix(ls[i], "  ") {
		i++
	}
	if i == 2 || len(ls) != i+4 || ls[i] != "metadata:" || ls[i+2] != `  generated-by: "mrw skill"` {
		return "", "", "", false
	}
	version, ok = quotedValue(ls[i+1], "  mrw-version: ")
	if !ok {
		return "", "", "", false
	}
	sha, ok = quotedValue(ls[i+3], "  content-sha256: ")
	if !ok {
		return "", "", "", false
	}
	return version, sha, strings.Replace(n, ls[i+3]+"\n", "", 1), true
}

// newerRelease reports whether stamp names a release newer than this mrw's, so
// a stale binary on PATH does not roll an installed skill back. Only two
// versions that both begin vX.Y.Z are compared; a dev build is never judged.
func newerRelease(stamp, this string) bool {
	a, okA := releaseOf(stamp)
	b, okB := releaseOf(this)
	if !okA || !okB {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

// releaseOf is the X, Y and Z of a version that begins vX.Y.Z.
func releaseOf(v string) ([3]int, bool) {
	var r [3]int
	m := releasePattern.FindStringSubmatch(v)
	if m == nil {
		return r, false
	}
	for i := range r {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return r, false
		}
		r[i] = n
	}
	return r, true
}

var releasePattern = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)`)

// quotedValue is line's double-quoted value after prefix, unquoted.
func quotedValue(line, prefix string) (string, bool) {
	v, ok := strings.CutPrefix(line, prefix)
	if !ok || !strings.HasPrefix(v, `"`) {
		return "", false
	}
	u, err := strconv.Unquote(v)
	return u, err == nil
}

// normalised is s with a leading byte order mark dropped, CRLF read as LF, and
// ending in a newline: the text a SKILL.md holds whatever a host's tools or an
// editor gave it.
func normalised(s string) string {
	s = strings.TrimPrefix(s, "\ufeff")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if s != "" && !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	return s
}

// agentsHelp opens `mrw --help` and a bare `mrw` (ADR-168 Decision 4), in
// capitals as asked for on 2026-10-22: an agent that reads one line of the help
// reads this one. Subcommand help keeps its own template.
const agentsHelp = "AGENTS: READ THE SKILL FIRST: mrw --skill\n" +
	"  install it: mrw --skill --dir DIR (a host's skills folder) · the rules alone: mrw instructions --core\n\n"
