package bench

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	quoted   = regexp.MustCompile(`'[^']*'|"(?:[^"\\]|\\.)*"`)
	redirect = regexp.MustCompile(`(?:^|[^0-9&<>=\-|])(?:[0-9]?>>?|&>>?)\s*([^\s;&|<>()]+)`)
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

// stripBodies removes heredoc bodies and quoted strings, so a plan's body or
// a quoted ">" is never read as a redirection.
func stripBodies(cmd string) string {
	var out []string
	lines := strings.Split(cmd, "\n")
	for i := 0; i < len(lines); i++ {
		out = append(out, lines[i])
		if m := heredoc.FindStringSubmatch(lines[i]); m != nil {
			for i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != m[2] {
				i++
			}
			i++ // the terminator
		}
	}
	return quoted.ReplaceAllString(strings.Join(out, "\n"), "''")
}

// ClassifyBash returns why cmd would write a file other than through mrw ("" if
// it would not), and the read tools it applies to file contents.
func ClassifyBash(cmd string) (string, []string) {
	bare := stripBodies(cmd)
	for _, m := range redirect.FindAllStringSubmatch(bare, -1) {
		if target := m[1]; target != "/dev/null" && !strings.HasPrefix(target, "&") {
			return "a redirection writes " + target, readsOf(bare)
		}
	}
	for _, seg := range pipelineParts(bare) {
		w := strings.Fields(seg)
		if len(w) == 0 {
			continue
		}
		head := filepath.Base(w[0])
		if why, ok := writers[head]; ok {
			return why, readsOf(bare)
		}
		switch {
		case head == "sed" && hasFlag(w[1:], "i"):
			return "sed -i edits a file", readsOf(bare)
		case head == "perl" && hasFlag(w[1:], "i"):
			return "perl -i edits a file", readsOf(bare)
		case head == "git" && len(w) > 1 && (w[1] == "apply" || w[1] == "checkout" || w[1] == "restore" || w[1] == "stash" || w[1] == "reset" || w[1] == "am"):
			return "git " + w[1] + " changes files", readsOf(bare)
		case head == "go" && len(w) > 1 && (w[1] == "fmt" || w[1] == "fix" || w[1] == "generate"):
			return "go " + w[1] + " rewrites files", readsOf(bare)
		case interpreters[head]:
			if writeCall.MatchString(cmd) {
				return head + " writes a file", readsOf(bare)
			}
		}
	}
	return "", readsOf(bare)
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

// SlippedWrites names the files the tree shows changed that the attempt's own
// mrw ledger (under the XDG_STATE_HOME state) does not record as written by mrw
// (ADR-118's `w`), or records with another content than the file now has: a
// write that went past configuration B's hook.
func SlippedWrites(tree, state string) ([]string, error) {
	written := map[string]string{}
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
			parts := strings.SplitN(sc.Text(), "  ", 3)
			if len(parts) == 3 && strings.HasPrefix(parts[1], "w") {
				written[parts[2]] = parts[0]
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
		sha, ok := written[c.file]
		if !ok {
			slipped = append(slipped, c.file)
			continue
		}
		if b, err := os.ReadFile(filepath.Join(tree, c.file)); err == nil {
			sum := sha256.Sum256(b)
			if got := hex.EncodeToString(sum[:]); len(sha) >= 8 && !strings.HasPrefix(got, sha) && !strings.HasPrefix(sha, got) {
				slipped = append(slipped, c.file)
			}
		}
	}
	return slipped, nil
}
