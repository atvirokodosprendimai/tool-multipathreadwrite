package apply

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// Slashed returns res with every root-relative path spelled with "/" where
// sep separated it: hunk and file paths, a rename's destination, a symlink's
// target, the directories a plan made and what it left behind. ADR-091 decided
// a receipt names a path the way a plan does; the engine, the ledger and the
// drift baseline keep the platform's spelling, so only the copy a receipt is
// built from is converted, and the slices are cloned. Root is not touched: it
// is absolute, and a caller hands it to its own filesystem. Both the CLI and
// mrw_write build their receipts through it (ADR-132). sep is a parameter so a
// test can drive `\` on any platform.
func Slashed(res Result, sep rune) Result {
	res.Hunks = slices.Clone(res.Hunks)
	for i := range res.Hunks {
		h := &res.Hunks[i]
		// A reason names its own file in the plan's words too: that spelling
		// follows the path, so one hunk does not show both (the Codex review
		// of #347). An absolute path inside the reason is the system's, kept.
		h.Reason = SlashIn(h.Reason, h.Path, sep)
		h.Reason = SlashIn(h.Reason, h.srcPath, sep)
		h.Path = Slash(h.Path, sep)
		h.BodyFrom, h.srcPath = Slash(h.BodyFrom, sep), Slash(h.srcPath, sep)
	}
	res.Files = slices.Clone(res.Files)
	for i := range res.Files {
		f := &res.Files[i]
		f.Path, f.RenamedTo, f.Target = Slash(f.Path, sep), Slash(f.RenamedTo, sep), Slash(f.Target, sep)
	}
	res.DirsCreated = slices.Clone(res.DirsCreated)
	for i, d := range res.DirsCreated {
		res.DirsCreated[i] = Slash(d, sep)
	}
	res.LeftBehind = slices.Clone(res.LeftBehind)
	for i, p := range res.LeftBehind {
		res.LeftBehind[i] = Slash(p, sep)
	}
	return res
}

// Slash spells p's sep separators as "/".
func Slash(p string, sep rune) string {
	return strings.ReplaceAll(p, string(sep), "/")
}

// reasonSep is the separator asPlan converts a path from: the platform's, and
// a variable so a test can drive `\` on any platform (ADR-198).
var reasonSep = filepath.Separator

// asPlan spells a cleaned root-relative path with "/" as a plan and its row do.
// A refusal site that puts a plan's path into a reason formats it through
// here, where mrw knows which argument is a path; text it did not format, a
// path as the caller typed it and the `attrib -r` command are never passed
// (ADR-198). A rooted path is returned as it is: a site that runs before the
// plan's absolute paths are refused (foldClashes) can hold one, and it is the
// caller's spelling of a place outside the plan (the Codex review of #643).
func asPlan(p string) string {
	if rooted.IsRooted(p) {
		return p
	}
	return Slash(p, reasonSep)
}

// SlashIn spells the root-relative path p with "/" where sep separated it, in
// the two places mrw's own words put it: at the start of text, followed by a
// space or a colon ("d\f.txt has not been read", "d\f.txt: …"), and between
// backticks ("Run `mrw read d\f.txt` first"). Anywhere else the path is part
// of something the system printed — an absolute path, a link target — and a
// filename may hold any character, so no boundary test can tell where it
// starts: it is kept as written (ADR-132; the Codex re-reviews of #347).
func SlashIn(text, p string, sep rune) string {
	if p == "" || !strings.ContainsRune(p, sep) {
		return text
	}
	q := Slash(p, sep)
	if rest, ok := strings.CutPrefix(text, p); ok && (strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, ":")) {
		text = q + rest
	}
	return strings.ReplaceAll(text, "`mrw read "+p+"`", "`mrw read "+q+"`")
}

// SlashLead spells the root-relative path p with "/" where sep separated it,
// when text begins with exactly p followed by a space or a colon, and returns
// text as it was otherwise. It is SlashIn's first place and no other: a refusal
// `rooted` words, or the error of a stopped commit, begins with the path as the
// platform spells it, and the rest of the text is the system's (ADR-198
// Decision 6).
func SlashLead(text, p string, sep rune) string {
	if p == "" || !strings.ContainsRune(p, sep) {
		return text
	}
	if rest, ok := strings.CutPrefix(text, p); ok && (strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, ":")) {
		return Slash(p, sep) + rest
	}
	return text
}

// SlashErr is err with the start of its text spelled by SlashLead against the
// first path of res, as Apply returned it, that the text begins with; it still
// unwraps to err. An error that begins with none of them (a directory named in
// a plan, a lock) comes back as it was. A caller converts before the receipt's
// own conversion has spelled those paths with "/" (ADR-198 Decision 7).
func SlashErr(res Result, err error, sep rune) error {
	if err == nil {
		return nil
	}
	text := err.Error()
	// The longest path that fits: a\b and "a\b c\d.txt" both begin the text
	// "a\b c\d.txt appeared …", and the shorter would convert half of it (the
	// go-reviewer pass on #645). Every error Apply returns for a stopped write
	// begins with a hunk's path.
	best := ""
	for _, h := range res.Hunks {
		if len(h.Path) > len(best) && SlashLead(text, h.Path, sep) != text {
			best = h.Path
		}
	}
	if best == "" {
		return err
	}
	return &slashedError{err: err, text: SlashLead(text, best, sep)}
}

// slashedError is an error whose text has been spelled for a receipt and that
// unwraps to the error it came from, so errors.As still finds its cause.
type slashedError struct {
	err  error
	text string
}

func (e *slashedError) Error() string { return e.text }
func (e *slashedError) Unwrap() error { return e.err }

// reasonOf is err's text as a reason holds it: the start spelled as the row
// spells p, the path `rooted` was given. A rooted p is the caller's own
// spelling of a place and is kept, as asPlan keeps it (ADR-198 Decision 6).
func reasonOf(err error, p string) string {
	if rooted.IsRooted(p) {
		return err.Error()
	}
	return SlashLead(err.Error(), p, reasonSep)
}
