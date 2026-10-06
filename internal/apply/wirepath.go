package apply

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
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

// SlashIn spells each occurrence of the root-relative path p in text with "/"
// where sep separated it — only where it stands whole: not the tail of a
// longer path (an absolute one the system printed, `C:\root\old\f.txt` for
// `d\f.txt`) nor the head of one (`d\f.txt\child`), which are kept as written
// (ADR-132; the Codex re-review of #347).
func SlashIn(text, p string, sep rune) string {
	if p == "" || !strings.ContainsRune(p, sep) {
		return text
	}
	inName := func(r rune) bool {
		return r == sep || r == '/' || r == '.' || r == '_' || r == '-' || unicode.IsLetter(r) || unicode.IsDigit(r)
	}
	var b strings.Builder
	from := 0
	for {
		i := strings.Index(text[from:], p)
		if i < 0 {
			b.WriteString(text[from:])
			return b.String()
		}
		i += from
		end := i + len(p)
		b.WriteString(text[from:i])
		before, _ := utf8.DecodeLastRuneInString(text[:i])
		after, _ := utf8.DecodeRuneInString(text[end:])
		whole := (i == 0 || !inName(before)) && (end == len(text) || !inName(after))
		if whole {
			b.WriteString(Slash(p, sep))
		} else {
			b.WriteString(p)
		}
		from = end
	}
}
