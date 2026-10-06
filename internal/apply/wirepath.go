package apply

import (
	"slices"
	"strings"
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
		h.Path = Slash(h.Path, sep)
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
