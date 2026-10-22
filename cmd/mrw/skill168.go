package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
// when the file is absent, a whole-file replace when it is mrw's own. The engine
// keeps every guard a write has — a link that leaves DIR, a reserved name, .git,
// mrw's state, a held file — and --dir adds the ones a skill needs: DIR must
// already exist, a SKILL.md a person wrote is never replaced, and a link inside
// DIR (which the engine would follow) is refused at mrw and at SKILL.md.
func installSkill(dir string, out io.Writer) error {
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return cli.Exit(fmt.Sprintf("--dir %s: not an existing directory; mrw does not create the folder a host reads skills from", dir), exitUsage)
	}
	// A --dir install is a write, so a legacy ./.mrw/ moves first, as it does
	// before any write (ADR-004); the Before hook skips it for plain skill.
	migrateLegacyState()
	text := guide.Skill(versionString())
	target := filepath.Join(dir, "mrw", "SKILL.md")
	in := apply.Input{Path: "mrw/SKILL.md", Op: "create", Body: strings.Split(strings.TrimSuffix(text, "\n"), "\n"), CountedBody: true, Lines: -1}
	verb, old := "created", ""
	if fi, err := os.Lstat(filepath.Join(dir, "mrw")); err == nil && (fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir()) {
		return cli.Exit(fmt.Sprintf("%s is not a directory (a file, a link or a junction); --dir writes only into a folder it can name", filepath.Join(dir, "mrw")), exitNotApplied)
	}
	switch fi, err := os.Lstat(target); {
	case err == nil && !fi.Mode().IsRegular():
		return cli.Exit(fmt.Sprintf("%s is not a regular file (a link, a junction or a directory); --dir replaces only a SKILL.md it wrote", target), exitNotApplied)
	case err == nil:
		b, err := readSkillFile(target)
		if err != nil {
			return err
		}
		version, ours := skillStamp(string(b))
		if !ours || len(b) > maxSkillBytes {
			return cli.Exit(fmt.Sprintf("%s was not written by `mrw skill` (its frontmatter is not the shape mrw skill prints, with generated-by: \"mrw skill\" in its metadata), so it is left as it is; move it aside to install here", target), exitNotApplied)
		}
		// Line endings and a final newline are the engine's to keep — a replace
		// is written in the file's own style — so text equal up to them is
		// unchanged; otherwise a CRLF copy would be replaced on every run.
		if normalised(string(b)) == text {
			_, err := fmt.Fprintf(out, "unchanged %s (mrw-version %s)\n", target, version)
			return err
		}
		// The bytes judged above are the bytes replaced: sha= refuses the write
		// if the file changed since, and Force stands for the read this was.
		verb, old = "replaced", version+" → "
		in = apply.Input{Path: "mrw/SKILL.md", Start: 1, End: apply.EOF, Op: "replace", Body: in.Body, CountedBody: true,
			SHA: seen.SHA(b), Anchor: "---", Lines: -1}
	case !errors.Is(err, fs.ErrNotExist):
		return targetErr(target, err)
	}
	prep, err := writer.Prepare(writer.Request{Root: dir, In: []apply.Input{in}, Check: writer.CheckOff, Opts: apply.Options{Force: in.Op == "replace"}})
	if err != nil {
		return exitWith(gateRefusal(err))
	}
	land, err := prep.Land()
	if err != nil {
		return exitWith(gateRefusal(err))
	}
	res := shown(land.Res)
	if land.Err != nil {
		report(os.Stdout, res, false)
		return exitSaying(exitTwoLine(land.Res, land.Err), land.Err)
	}
	if res.Failed > 0 {
		report(os.Stdout, res, false)
		return cli.Exit("the skill was refused — nothing was written", exitNotApplied)
	}
	if notice := writer.NewerUntold(dir, land.Res); notice != "" {
		fmt.Fprintln(os.Stderr, notice)
	}
	// The file changed whatever the ledger says, so the line that says so comes
	// first; a ledger that could not record it is then the exit's reason.
	_, err = fmt.Fprintf(out, "%s %s (mrw-version %s%s)\n", verb, target, old, versionString())
	if land.LedgerErr != nil {
		return exitWith(land.LedgerErr)
	}
	return err
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

// targetErr is a SKILL.md --dir could not read: a permission or a file that is
// not regular is the target's cause, exit 1; anything else, an I/O error, is
// the environment's, exit 2 — the split a write's staging makes (ADR-132).
func targetErr(path string, err error) error {
	code := exitUsage
	if errors.Is(err, fs.ErrPermission) || errors.Is(err, regular.ErrNotRegular) {
		code = exitNotApplied
	}
	return cli.Exit(fmt.Sprintf("%s: %v", path, err), code)
}

// skillStamp reports a SKILL.md's mrw-version and whether `mrw skill` wrote it,
// by the exact frontmatter shape Skill prints: name: mrw, a folded description
// of indented lines, then metadata holding mrw-version and generated-by and
// nothing else. Any other shape is a person's file: a generated-by line inside a
// quoted description, the body or a comment does not make it mrw's. CRLF and a
// missing final newline are read as the engine keeps them.
func skillStamp(s string) (version string, ours bool) {
	rest, ok := strings.CutPrefix(normalised(s), "---\n")
	if !ok {
		return "", false
	}
	front, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return "", false
	}
	ls := strings.Split(front, "\n")
	if len(ls) < 6 || ls[0] != "name: mrw" || ls[1] != "description: >-" {
		return "", false
	}
	i := 2
	for i < len(ls) && strings.HasPrefix(ls[i], "  ") {
		i++
	}
	if i == 2 || len(ls) != i+3 || ls[i] != "metadata:" || ls[i+2] != `  generated-by: "mrw skill"` {
		return "", false
	}
	v, ok := strings.CutPrefix(ls[i+1], "  mrw-version: ")
	if !ok || !strings.HasPrefix(v, `"`) {
		return "", false
	}
	u, err := strconv.Unquote(v)
	if err != nil {
		return "", false
	}
	return u, true
}

// normalised is s with CRLF read as LF and ending in a newline: the text a
// SKILL.md holds whatever line endings a host's tools gave it.
func normalised(s string) string {
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
