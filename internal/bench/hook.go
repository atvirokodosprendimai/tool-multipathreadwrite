package bench

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// Hook is configuration B's PreToolUse hook (ADR-156 Decision 1). It reads the
// event Claude Code sends on stdin. A Bash command that writes a file by any
// means but mrw is refused: exit 2, the reason on stderr, which Claude Code
// hands back to the model. A command that reads a file's contents through cat,
// grep, head, sed -n and their kin passes and is counted, not refused — the
// default Zy may overrule. Every refusal and count is appended to log.
// tree is the attempt's task tree: a redirection or a tee whose target lands
// outside it passes (outsideTree; Zy, 2026-10-14, after the pilot). With tree
// empty, every target is inside.
func Hook(stdin io.Reader, stderr io.Writer, log, tree string) int {
	var ev struct {
		Tool  string `json:"tool_name"`
		Input struct {
			Command string `json:"command"`
		} `json:"tool_input"`
	}
	if err := json.NewDecoder(stdin).Decode(&ev); err != nil || ev.Tool != "Bash" {
		return 0
	}
	refusal, reads := ClassifyBash(ev.Input.Command, tree)
	var lines []string
	if refusal != "" {
		lines = append(lines, "refused\t"+oneLine(ev.Input.Command))
	}
	for _, r := range reads {
		lines = append(lines, "read\t"+r)
	}
	if len(lines) > 0 {
		if f, err := os.OpenFile(log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			_, _ = fmt.Fprintln(f, strings.Join(lines, "\n"))
			_ = f.Close()
		}
	}
	if refusal != "" {
		_, _ = fmt.Fprintf(stderr, "refused: %s. In this run files are changed only with mrw (mrw write); read them with mrw read.\n", refusal)
		return 2
	}
	return 0
}

func oneLine(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\t", " "), "\n", " ⏎ ")
}

var (
	heredoc  = regexp.MustCompile(`<<-?\s*(['"]?)([A-Za-z_][A-Za-z0-9_]*)['"]?`)
	redirect = regexp.MustCompile(`(?:^|[^0-9&<>=\-|])(?:[0-9]?>>?|&>>?)\s*(&?[^\s;&|<>()]+)`)
	dupFD    = regexp.MustCompile(`^&([0-9]+|-)$`)
	// The first word of a segment that changes files by itself.
	writers = map[string]string{
		"cp": "cp copies onto a file", "mv": "mv moves a file", "rm": "rm removes a file", "touch": "touch creates a file",
		"tee": "tee writes a file", "dd": "dd writes a file", "truncate": "truncate changes a file", "patch": "patch edits files",
		"install": "install writes a file", "ln": "ln makes a link", "gofmt": "gofmt may rewrite files", "unlink": "unlink removes a file",
	}
	readers      = map[string]bool{"cat": true, "head": true, "tail": true, "grep": true, "egrep": true, "rg": true, "awk": true, "sed": true, "less": true, "more": true, "nl": true}
	interpreters = map[string]bool{"python": true, "python3": true, "node": true, "perl": true, "ruby": true}
	// A redirect inside an interpreter's text counts only as a quoted mode
	// string (perl's ">out"): a comparison (a > b) and an annotation (-> str)
	// are not writes (T3's pilot, ADR-160 T6).
	writeCall = regexp.MustCompile(`open\([^)]*,\s*['"][wa]|write_text|writeFile|\.write\(|File\.write|['"]>>?\s*[A-Za-z]`)
)

// splitHeredocs separates a command from its heredoc bodies. A heredoc
// operator counts only outside quotes, so a quoted "<<EOF" opens nothing. The
// body of a heredoc whose delimiter is quoted is literal and dropped (a plan's
// body); the body of one whose delimiter is not quoted is expanded by the
// shell, so it is returned for its command substitutions. bodies is every
// body, quoted or not, for an interpreter that reads one.
func splitHeredocs(cmd string) (rest string, expanded []string, bodies string) {
	var out, all []string
	lines := strings.Split(cmd, "\n")
	for i := 0; i < len(lines); i++ {
		out = append(out, lines[i])
		m := realHeredoc(lines[i])
		if m == nil {
			continue
		}
		for i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != m[2] {
			i++
			all = append(all, lines[i])
			if m[1] == "" {
				expanded = append(expanded, lines[i])
			}
		}
		i++ // the terminator
	}
	return strings.Join(out, "\n"), expanded, strings.Join(all, "\n")
}

// realHeredoc finds a heredoc operator on a line that is not inside a quoted
// string: `<<'PLAN'` opens a heredoc, `echo "<<EOF"` does not.
func realHeredoc(line string) []string {
	quotedSpans := regexp.MustCompile(`'[^']*'|"(?:[^"\\]|\\.)*"`).FindAllStringIndex(line, -1)
	for _, loc := range heredoc.FindAllStringSubmatchIndex(line, -1) {
		inside := false
		for _, q := range quotedSpans {
			if loc[0] > q[0] && loc[0] < q[1] {
				inside = true
			}
		}
		if !inside {
			return []string{line[loc[0]:loc[1]], line[loc[2]:loc[3]], line[loc[4]:loc[5]]}
		}
	}
	return nil
}

// ClassifyBash returns why cmd would write a file other than through mrw ("" if
// it would not), and the read tools it applies to file contents. A command
// substitution, in double quotes or an unquoted heredoc as much as bare, is
// classified as a command of its own; one nested too deep to read is refused,
// since what the hook cannot see it cannot pass (the review of #411). A write
// whose target lands outside tree is not one (outsideTree).
func ClassifyBash(cmd, tree string) (string, []string) {
	return classify(cmd, tree, 0)
}

func classify(cmd, tree string, depth int) (string, []string) {
	rest, expanded, bodies := splitHeredocs(cmd)
	rest = resolveTargets(rest, os.Getenv("TMPDIR"))
	bare, subs, ok := scanShell(rest, false)
	for _, body := range expanded {
		_, s, bodyOK := scanShell(body, true)
		subs, ok = append(subs, s...), ok && bodyOK
	}
	var reads []string
	if !ok {
		return "a command substitution the hook cannot inspect", reads
	}
	for _, inner := range subs {
		if depth >= 4 {
			return "a command substitution nested too deep to inspect", reads
		}
		why, r := classify(inner, tree, depth+1)
		reads = append(reads, r...)
		if why != "" {
			return "a command substitution: " + why, reads
		}
	}
	reads = append(reads, readsOf(bare)...)
	for _, m := range redirect.FindAllStringSubmatch(bare, -1) {
		target := strings.TrimPrefix(m[1], "&")
		if m[1] != "/dev/null" && !dupFD.MatchString(m[1]) && !outsideTree(target, tree) {
			// An expansion or a quote is judged inside, as it must be, but an
			// agent building a plan in $(mktemp) cannot tell what would pass, and
			// paid a round to find out in three of four MRW cells (CORE-RESULTS.md).
			if strings.ContainsAny(target, "$`~*?[{\"'\\") {
				return "a redirection writes " + target + ", a path the hook cannot resolve: put a plan file at a literal absolute path outside the task tree, or pipe the plan to mrw write -", reads
			}
			return "a redirection writes " + target, reads
		}
	}
	for _, line := range segments(bare) {
		for _, seg := range strings.Split(line, "|") {
			w := strings.Fields(seg)
			if len(w) == 0 {
				continue
			}
			w = commandWords(w)
			if len(w) == 0 {
				continue
			}
			head := filepath.Base(w[0])
			if why, ok := writers[head]; ok {
				if head == "tee" && operandsOutside(w[1:], tree) {
					continue
				}
				return why, reads
			}
			switch {
			// sed -i and perl -i on files outside the tree pass, as a redirect
			// or tee there does: B fixing its own plan in /tmp is not a write
			// to the task (seven Stage 3 attempts, T3's pilot; ADR-160 T6).
			case head == "sed" && hasFlag(w[1:], "i") && !inPlaceOutside(w[1:], tree):
				return "sed -i edits a file", reads
			case head == "perl" && hasFlag(w[1:], "i") && !inPlaceOutside(w[1:], tree):
				return "perl -i edits a file", reads
			case head == "git" && len(w) > 1 && (w[1] == "apply" || w[1] == "checkout" || w[1] == "restore" || w[1] == "stash" || w[1] == "reset" || w[1] == "am"):
				return "git " + w[1] + " changes files", reads
			case head == "go" && len(w) > 1 && (w[1] == "fmt" || w[1] == "fix" || w[1] == "generate"):
				return "go " + w[1] + " rewrites files", reads
			case interpreters[head]:
				// The interpreter's text is the command without its heredoc
				// bodies, and those bodies only when its pipeline reads one: an
				// mrw plan fed to another command is not python's (T3's pilot
				// found B refused for a plan holding `-> str`, ADR-160 T6), and
				// `cat <<PY | python3` is (the review of #434).
				text := rest
				if strings.Contains(line, "<<") {
					text += "\n" + bodies
				}
				if writeCall.MatchString(text) {
					return head + " writes a file", reads
				}
			}
		}
	}
	return "", reads
}

var (
	// assigned is NAME=VALUE whose VALUE ADR-186 can know: $TMPDIR, a bare
	// mktemp, or a literal absolute path, with a literal suffix, quoted whole or
	// not at all. Where it may stand is checked by hand: Go has no lookaround.
	assigned  = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)=("?)(\$\{TMPDIR\}|\$TMPDIR|\$\(mktemp(?: -d)?\)|/)([A-Za-z0-9._/-]*)("?)`)
	expansion = regexp.MustCompile(`("?)\$(?:\{([A-Za-z_][A-Za-z0-9_]*)\}|([A-Za-z_][A-Za-z0-9_]*))([A-Za-z0-9._/-]*)("?)`)
	plainPath = regexp.MustCompile(`^/[A-Za-z0-9._/-]*$`)
	rebinders = map[string]bool{"read": true, "for": true, "declare": true, "local": true, "typeset": true, "unset": true, "readonly": true}
	anyWord   = regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])(?:eval|source)(?:$|[^A-Za-z0-9_-])`)
)

// binding is a value ADR-186 knows for a name, and where the statement that
// binds it ends: only an expansion after that point reads it.
type binding struct {
	value string
	from  int
}

// resolveTargets replaces each expansion ADR-186 can know with its value, so
// that classify judges a plan file at $TMPDIR/plan.txt, in $(mktemp -d), or at
// a literal path held in a variable, as it judges any absolute target: by
// outsideTree. tmp is the hook's own TMPDIR, used as spelled. A name is known
// only from a statement of its own at the top level — outside every quote and
// parenthesis, not a prefix to a command — that is the command's one binding
// of it, and only after that statement; a command holding eval, source, a
// backquote or a backslash outside single quotes knows none, and TMPDIR is
// known only when the command neither assigns nor rebinds it. Every other
// expansion is left as written, so it stays judged inside (ADR-156). The
// Codex review of ADR-186 found each of these limits as a way a resolved path
// could differ from what bash writes.
func resolveTargets(rest, tmp string) string {
	top := topLevel(rest)
	if top == nil || anyWord.MatchString(rest) {
		return rest
	}
	known := map[string]binding{}
	if tmp != "" && plainPath.MatchString(tmp) && !rebound(rest, "TMPDIR") && countAssigned(rest, "TMPDIR") == 0 {
		known["TMPDIR"] = binding{tmp, 0}
	}
	for _, m := range assigned.FindAllStringSubmatchIndex(rest, -1) {
		start, end := m[0], m[1]
		name, open, base, suffix, closing := rest[m[2]:m[3]], rest[m[4]:m[5]], rest[m[6]:m[7]], rest[m[8]:m[9]], rest[m[10]:m[11]]
		if open != closing || name == "TMPDIR" || !top[start] || !statement(rest, start, end) {
			continue
		}
		if countAssigned(rest, name) != 1 || rebound(rest, name) {
			continue
		}
		dir, ok := known["TMPDIR"]
		var value string
		switch {
		case base == "/":
			value = "/" + suffix
		case !ok:
			continue
		case strings.HasPrefix(base, "$(mktemp"):
			value = strings.TrimSuffix(dir.value, "/") + "/tmp.mktemp" + suffix
		case base == "$TMPDIR" && suffix != "" && isNameByte(suffix[0]):
			continue // $TMPDIR_X is the variable TMPDIR_X, not TMPDIR and _X
		default:
			value = dir.value + suffix
		}
		known[name] = binding{value, end}
	}
	return expansionsAfter(rest, known)
}

// expansionsAfter rewrites each expansion of a known name that stands after
// the statement binding it, quoted whole or not at all.
func expansionsAfter(rest string, known map[string]binding) string {
	var b strings.Builder
	last := 0
	for _, m := range expansion.FindAllStringSubmatchIndex(rest, -1) {
		s := rest[m[0]:m[1]]
		sub := expansion.FindStringSubmatch(s)
		bind, ok := known[sub[2]+sub[3]]
		if !ok || sub[1] != sub[5] || m[0] < bind.from {
			continue
		}
		b.WriteString(rest[last:m[0]])
		b.WriteString(bind.value + sub[4])
		last = m[1]
	}
	b.WriteString(rest[last:])
	return b.String()
}

// topLevel reports, for each byte of s, whether it stands outside every quote
// and every parenthesis, or nil when s holds a backslash or a backquote outside
// single quotes, which this coarse reading does not follow.
func topLevel(s string) []bool {
	top := make([]bool, len(s)+1)
	depth := 0
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		top[i] = depth == 0 && quote == 0
		switch {
		case quote == '\'':
			if c == '\'' {
				quote = 0
			}
		case c == '\\' || c == '`':
			return nil
		case quote == '"':
			if c == '"' {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '(':
			depth++
		case c == ')' && depth > 0:
			depth--
		}
	}
	top[len(s)] = depth == 0 && quote == 0
	return top
}

// statement says whether the assignment s[start:end] is a statement of its
// own: after nothing, a separator or export, and followed by nothing or a
// separator, so that P=x cmd, whose binding the command's own redirection does
// not see, is not one.
func statement(s string, start, end int) bool {
	before := strings.TrimRight(s[:start], " \t")
	before = strings.TrimSuffix(before, "export")
	before = strings.TrimRight(before, " \t")
	after := strings.TrimLeft(s[end:], " \t")
	return (before == "" || strings.ContainsRune(";&|\n", rune(before[len(before)-1]))) &&
		(after == "" || strings.ContainsRune(";&|\n", rune(after[0])))
}

func isNameByte(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

// countAssigned counts NAME= and NAME+= in s, each a separate binding.
func countAssigned(s, name string) int {
	return len(regexp.MustCompile(`(?:^|[^A-Za-z0-9_$])`+regexp.QuoteMeta(name)+`\+?=`).FindAllStringIndex(s, -1))
}

// rebound says whether a segment that holds read, for, declare, local,
// typeset, unset or readonly as a word also names NAME, so that IFS= read -r P
// and (read P) count, as a binding no plain assignment makes.
func rebound(s, name string) bool {
	for _, line := range strings.FieldsFunc(s, func(r rune) bool { return strings.ContainsRune(";&|\n", r) }) {
		words := strings.FieldsFunc(line, func(r rune) bool { return strings.ContainsRune(" \t(){}", r) })
		binds, names := false, false
		for _, w := range words {
			binds = binds || rebinders[w]
			names = names || w == name || strings.HasPrefix(w, name+"=")
		}
		if binds && names {
			return true
		}
	}
	return false
}

// outsideTree says whether a write's target is a path that lands outside the
// task tree, which the hook then passes (Zy, 2026-10-14: the pilot refused a
// plan file in /tmp, the input of mrw write PLAN, not a way around it). Only an
// absolute path the hook can resolve is judged: a relative one would be judged
// against a working directory the Bash tool keeps across calls, which the
// hook cannot see; a path holding a quote, an expansion or a glob is not
// resolved at all; and a dot-dot is refused, since the shell resolves it where
// a link before it leads while Clean resolves it on the spelling. Each of those
// is inside. Links on the deepest part that exists are followed, a dangling one
// judged inside, and the tree is recognised by identity, not by spelling, so a
// case-folded spelling of it is the tree. SlippedWrites is the backstop for what
// this misjudges.
func outsideTree(target, tree string) bool {
	if tree == "" || !filepath.IsAbs(target) || strings.ContainsAny(target, "$`~*?[{\"'\\") {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(target), "/") {
		if part == ".." {
			return false
		}
	}
	root, err := os.Stat(tree)
	if err != nil {
		return false
	}
	p := filepath.Clean(target)
	for q := p; ; q = filepath.Dir(q) {
		if _, err := os.Lstat(q); err == nil {
			r, err := filepath.EvalSymlinks(q)
			if err != nil {
				return false
			}
			rest, err := filepath.Rel(q, p)
			if err != nil {
				return false
			}
			p = filepath.Join(r, rest)
			break
		}
		if filepath.Dir(q) == q {
			break
		}
	}
	for q := p; ; q = filepath.Dir(q) {
		if info, err := os.Stat(q); err == nil && os.SameFile(info, root) {
			return false
		}
		if filepath.Dir(q) == q {
			break
		}
	}
	// A file that already exists outside the tree may be another name for one in
	// it, a hard link: it is inside when it is the same file as one in the tree
	// (the review of #413). A path that does not exist yet is a new file.
	info, err := os.Stat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	if err != nil {
		return false
	}
	alias := false
	walkErr := filepath.WalkDir(tree, func(q string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			if qi, err := os.Stat(q); err == nil && os.SameFile(qi, info) {
				alias = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return walkErr == nil && !alias
}

// operandsOutside says whether every file operand of a tee lands outside tree.
func operandsOutside(args []string, tree string) bool {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") && !outsideTree(a, tree) {
			return false
		}
	}
	return true
}

// inPlaceOutside says whether a sed -i or perl -i edits files outside tree
// only: its last word is a path outside tree, and so is every other unquoted
// operand. A quoted word — the script, or BSD sed's empty -i suffix — reaches here as
// the placeholder scanShell leaves, and is not judged; an unquoted script is
// judged as a path, which only refuses more. A quoted file operand set before
// an outside one passes: a way past recorded in ADR-156 (ADR-160 T6).
func inPlaceOutside(args []string, tree string) bool {
	if len(args) == 0 || strings.HasPrefix(args[len(args)-1], "-") || !outsideTree(args[len(args)-1], tree) {
		return false
	}
	for _, a := range args {
		if !strings.HasPrefix(a, "-") && a != "''" && a != `""` && !outsideTree(a, tree) {
			return false
		}
	}
	return true
}

// scanShell reads a command once, tracking quotes as the shell does. It
// returns the unquoted text (quoted text dropped, each substitution left as a
// placeholder) and the text of every command substitution — `$(…)` or `…` —
// found outside single quotes, each matched to its own closing paren with its
// own quote context, so `echo "$(printf "'"; touch f; printf "'")"` gives
// back `printf "'"; touch f; printf "'"` whole (the reviews of #411); an
// arithmetic `$((…))` is not one, but the substitutions inside it are. ok is
// false when a substitution never closes. With inDouble, the text is read as
// an unquoted heredoc's body or an arithmetic expression: as inside double
// quotes, except that a quote is a character of the text, never the end of
// one (the third and fourth re-reviews).
func scanShell(s string, inDouble bool) (bare string, subs []string, ok bool) {
	var b strings.Builder
	double := inDouble
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s):
			if !double {
				b.WriteString(s[i : i+2])
			}
			i++
		case c == '\'' && !double:
			end := strings.IndexByte(s[i+1:], '\'')
			if end < 0 {
				return b.String(), subs, false
			}
			b.WriteString("''")
			i += end + 1
		case c == '"' && !inDouble:
			double = !double
			if double {
				b.WriteString(`""`)
			}
		case c == '$' && i+1 < len(s) && s[i+1] == '(':
			end := matchParen(s, i+1)
			if end < 0 {
				return b.String(), subs, false
			}
			if i+2 < len(s) && s[i+2] == '(' { // $(( … )) is arithmetic
				_, inner, innerOK := scanShell(s[i+3:end-1], true)
				if !innerOK {
					return b.String(), subs, false
				}
				subs = append(subs, inner...)
			} else {
				subs = append(subs, s[i+2:end])
			}
			if !double {
				b.WriteString(" $S ")
			}
			i = end
		case c == '`':
			end := strings.IndexByte(s[i+1:], '`')
			if end < 0 {
				return b.String(), subs, false
			}
			subs = append(subs, s[i+1:i+1+end])
			if !double {
				b.WriteString(" $S ")
			}
			i += end + 1
		default:
			if !double {
				b.WriteByte(c)
			}
		}
	}
	return b.String(), subs, true
}

// matchParen returns the index of the paren that closes the one at s[open],
// reading quotes, nested substitutions and backquotes inside as the shell
// does, or -1.
func matchParen(s string, open int) int {
	depth := 0
	for k := open; k < len(s); k++ {
		switch s[k] {
		case '\\':
			k++
		case '\'':
			end := strings.IndexByte(s[k+1:], '\'')
			if end < 0 {
				return -1
			}
			k += end + 1
		case '`':
			end := strings.IndexByte(s[k+1:], '`')
			if end < 0 {
				return -1
			}
			k += end + 1
		case '"':
			for k++; k < len(s) && s[k] != '"'; k++ {
				switch {
				case s[k] == '\\':
					k++
				case s[k] == '$' && k+1 < len(s) && s[k+1] == '(':
					if k = matchParen(s, k+1); k < 0 {
						return -1
					}
				}
			}
			if k >= len(s) {
				return -1
			}
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				return k
			}
		}
	}
	return -1
}

// wrappers run the command that follows them; the hook judges that command.
var wrappers = map[string]bool{"env": true, "command": true, "builtin": true, "exec": true, "nohup": true, "time": true, "nice": true, "stdbuf": true, "xargs": true, "sudo": true, "timeout": true}

// commandWords drops what stands before the command a simple command runs:
// variable assignments, and wrappers such as env, with their options and, for
// timeout, its duration — so `env cp a b` is judged as cp.
func commandWords(w []string) []string {
	for len(w) > 0 {
		switch {
		case regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`).MatchString(w[0]):
			w = w[1:]
		case filepath.Base(w[0]) == "command" && len(w) > 1 && (w[1] == "-v" || w[1] == "-V"):
			return nil // command -v asks where a command is; it runs nothing
		case wrappers[filepath.Base(w[0])]:
			timeout := filepath.Base(w[0]) == "timeout"
			w = w[1:]
			for len(w) > 0 && strings.HasPrefix(w[0], "-") {
				w = w[1:]
			}
			if timeout && len(w) > 0 {
				w = w[1:]
			}
		default:
			return w
		}
	}
	return w
}

// segments splits a command at ;, &&, || and newlines, keeping pipelines whole.
func segments(cmd string) []string {
	return regexp.MustCompile(`;|&&|\|\||\n`).Split(cmd, -1)
}

// readsOf counts the read tools given a path: cat, head and tail with one
// operand, grep, rg, sed and awk with a pattern or script and then a path. A
// tool reading a pipe, as in `mrw read f | grep x`, reads mrw's answer, not a
// file, and is not counted.
func readsOf(cmd string) []string {
	var reads []string
	for _, seg := range segments(cmd) {
		for _, part := range strings.Split(seg, "|") {
			w := strings.Fields(part)
			if len(w) == 0 || !readers[filepath.Base(w[0])] {
				continue
			}
			if w = commandWords(w); len(w) == 0 {
				continue
			}
			head := filepath.Base(w[0])
			operands := 0
			for _, a := range w[1:] {
				if !strings.HasPrefix(a, "-") {
					operands++
				}
			}
			need := 1
			switch head {
			case "grep", "egrep", "rg", "sed", "awk":
				need = 2
			}
			if operands >= need {
				reads = append(reads, head)
			}
		}
	}
	return reads
}

func hasFlag(args []string, f string) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.Contains(a[1:], f) {
			return true
		}
		if a == "--in-place" || strings.HasPrefix(a, "--in-place=") {
			return true
		}
	}
	return false
}

// hookCounts reads a hook log: how many commands were refused, how many reads
// were counted.
func hookCounts(log string) (refused, reads int) {
	b, err := os.ReadFile(log)
	if err != nil {
		return 0, 0
	}
	for _, l := range strings.Split(string(b), "\n") {
		switch {
		case strings.HasPrefix(l, "refused\t"):
			refused++
		case strings.HasPrefix(l, "read\t"):
			reads++
		}
	}
	return refused, reads
}

// SlippedWrites names the files the tree shows changed whose content the
// attempt's own mrw ledger (under the XDG_STATE_HOME state) has not recorded:
// no entry for the file, an entry with another sha than the file now has, or a
// file gone from the tree. mrw records the sha of every file it writes or
// reads, so a file it changed, or read after a change, matches; a change made
// by other means and never read back does not. A change made by other means
// and then read back with mrw is not caught — a heuristic, as the hook is
// (the review of #411: the ledger's `w` mark does not survive a whole read).
// A ledger's paths are relative to the root its `root` marker names, so each
// is read against that root; one with no marker is an error, not a guess (the
// final review of #411).
func SlippedWrites(tree, state string) ([]string, error) {
	realTree, err := filepath.EvalSymlinks(tree)
	if err != nil {
		return nil, err
	}
	recorded := map[string]map[string]bool{} // tree path → every sha a ledger holds for it
	err = filepath.Walk(state, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() || info.Name() != "seen" {
			return nil
		}
		marker, err := os.ReadFile(filepath.Join(filepath.Dir(p), "root"))
		if err != nil {
			return fmt.Errorf("ledger %s names no root: %w", p, err)
		}
		rel, err := filepath.Rel(realTree, strings.TrimSuffix(string(marker), "\n"))
		if err != nil {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<24)
		for sc.Scan() {
			if parts := strings.SplitN(sc.Text(), "  ", 3); len(parts) == 3 {
				file := filepath.ToSlash(filepath.Join(rel, parts[2]))
				if recorded[file] == nil {
					recorded[file] = map[string]bool{}
				}
				recorded[file][parts[0]] = true
			}
		}
		return sc.Err()
	})
	if err != nil {
		return nil, err
	}
	changed, err := changes(tree)
	if err != nil {
		return nil, err
	}
	var slipped []string
	for _, c := range changed {
		b, err := os.ReadFile(filepath.Join(tree, c.file))
		shas := recorded[c.file]
		if err != nil || !shas[seen.SHA(b)] {
			slipped = append(slipped, c.file)
		} else if len(shas) > 1 {
			// Ledgers of two roots that disagree cannot say which mrw saw
			// last, and a stale one would vouch for a later change by other
			// means (the narrow review of #411).
			return nil, fmt.Errorf("the ledgers disagree on %s: which content mrw saw last cannot be told", c.file)
		}
	}
	return slipped, nil
}
