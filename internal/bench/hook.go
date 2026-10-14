package bench

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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
func Hook(stdin io.Reader, stderr io.Writer, log string) int {
	var ev struct {
		Tool  string `json:"tool_name"`
		Input struct {
			Command string `json:"command"`
		} `json:"tool_input"`
	}
	if err := json.NewDecoder(stdin).Decode(&ev); err != nil || ev.Tool != "Bash" {
		return 0
	}
	refusal, reads := ClassifyBash(ev.Input.Command)
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
	writeCall    = regexp.MustCompile(`open\([^)]*,\s*['"][wa]|write_text|writeFile|\.write\(|File\.write|>>?\s*[A-Za-z]`)
)

// splitHeredocs separates a command from its heredoc bodies. A heredoc
// operator counts only outside quotes, so a quoted "<<EOF" opens nothing. The
// body of a heredoc whose delimiter is quoted is literal and dropped (a plan's
// body); the body of one whose delimiter is not quoted is expanded by the
// shell, so it is returned for its command substitutions.
func splitHeredocs(cmd string) (rest string, expanded []string) {
	var out []string
	lines := strings.Split(cmd, "\n")
	for i := 0; i < len(lines); i++ {
		out = append(out, lines[i])
		m := realHeredoc(lines[i])
		if m == nil {
			continue
		}
		for i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != m[2] {
			i++
			if m[1] == "" {
				expanded = append(expanded, lines[i])
			}
		}
		i++ // the terminator
	}
	return strings.Join(out, "\n"), expanded
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
// since what the hook cannot see it cannot pass (the review of #411).
func ClassifyBash(cmd string) (string, []string) {
	return classify(cmd, 0)
}

func classify(cmd string, depth int) (string, []string) {
	rest, expanded := splitHeredocs(cmd)
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
		why, r := classify(inner, depth+1)
		reads = append(reads, r...)
		if why != "" {
			return "a command substitution: " + why, reads
		}
	}
	reads = append(reads, readsOf(bare)...)
	for _, m := range redirect.FindAllStringSubmatch(bare, -1) {
		if target := m[1]; target != "/dev/null" && !dupFD.MatchString(target) {
			return "a redirection writes " + strings.TrimPrefix(target, "&"), reads
		}
	}
	for _, seg := range pipelineParts(bare) {
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
			return why, reads
		}
		switch {
		case head == "sed" && hasFlag(w[1:], "i"):
			return "sed -i edits a file", reads
		case head == "perl" && hasFlag(w[1:], "i"):
			return "perl -i edits a file", reads
		case head == "git" && len(w) > 1 && (w[1] == "apply" || w[1] == "checkout" || w[1] == "restore" || w[1] == "stash" || w[1] == "reset" || w[1] == "am"):
			return "git " + w[1] + " changes files", reads
		case head == "go" && len(w) > 1 && (w[1] == "fmt" || w[1] == "fix" || w[1] == "generate"):
			return "go " + w[1] + " rewrites files", reads
		case interpreters[head]:
			if writeCall.MatchString(cmd) {
				return head + " writes a file", reads
			}
		}
	}
	return "", reads
}

// scanShell reads a command once, tracking quotes as the shell does. It
// returns the unquoted text (quoted text dropped, each substitution left as a
// placeholder) and the text of every command substitution — `$(…)` or `…` —
// found outside single quotes, each matched to its own closing paren with its
// own quote context, so `echo "$(printf "'"; touch f; printf "'")"` gives
// back `printf "'"; touch f; printf "'"` whole (the reviews of #411). ok is
// false when a substitution never closes. With inDouble, the text is read as
// the inside of double quotes — an unquoted heredoc's body.
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
		case c == '"':
			double = !double
			if double {
				b.WriteString(`""`)
			}
		case c == '$' && i+1 < len(s) && s[i+1] == '(':
			end := matchParen(s, i+1)
			if end < 0 {
				return b.String(), subs, false
			}
			if !(i+2 < len(s) && s[i+2] == '(') { // $(( … )) is arithmetic
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

// pipelineParts splits a command into its simple commands: segments, and the
// stages of each pipeline.
func pipelineParts(cmd string) []string {
	var parts []string
	for _, seg := range segments(cmd) {
		parts = append(parts, strings.Split(seg, "|")...)
	}
	return parts
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
func SlippedWrites(tree, state string) ([]string, error) {
	recorded := map[string]string{}
	err := filepath.Walk(state, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() || info.Name() != "seen" {
			return nil
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
				recorded[parts[2]] = parts[0]
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
		if sha, ok := recorded[c.file]; err != nil || !ok || seen.SHA(b) != sha {
			slipped = append(slipped, c.file)
		}
	}
	return slipped, nil
}
