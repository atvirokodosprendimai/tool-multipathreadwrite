// Package ingest compiles a foreign edit grammar into a native mrw plan.
//
// The first grammar is Codex apply_patch (ADR-051). The second is Aider
// SEARCH/REPLACE (--format=search_replace). Compile emits plan text;
// plan.Parse is the only door into Apply. Context matching locates an old
// side; it is not a license and it is not a target-syntax parse.
package ingest

import (
	"fmt"
	"os"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

const (
	beginPatch = "*** Begin Patch"
	endPatch   = "*** End Patch"
	updateFile = "*** Update File:"
	addFile    = "*** Add File:"
	deleteFile = "*** Delete File:"
	moveTo     = "*** Move to:"
)

// CompileApplyPatch turns a Codex apply_patch document into native plan text.
// root is the checkout the paths are relative to. The function writes nothing.
func CompileApplyPatch(root string, doc []byte) ([]byte, error) {
	s := strings.ReplaceAll(string(doc), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if looksLikeGit(s) {
		return nil, fmt.Errorf("a git patch is not an apply_patch; pass --format=apply_patch only for %s documents", beginPatch)
	}
	body, err := patchBody(s)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	var (
		path string
		kind string // update | add
		hunk []string
	)
	flush := func() error {
		if path == "" {
			hunk = nil
			return nil
		}
		if len(hunk) == 0 && kind != "add" {
			hunk = nil
			return nil
		}
		text, err := compileHunk(root, path, kind, hunk)
		if err != nil {
			return err
		}
		out.WriteString(text)
		hunk = nil
		if kind == "add" {
			path, kind = "", ""
		}
		return nil
	}
	for _, line := range strings.Split(body, "\n") {
		switch {
		case line == endPatch:
			if err := flush(); err != nil {
				return nil, err
			}
			path, kind = "", ""
		case strings.HasPrefix(line, deleteFile), strings.HasPrefix(line, moveTo):
			return nil, fmt.Errorf("apply_patch: %s is not compiled this slice (mrw has no unlink)", trimStar(line))
		case strings.HasPrefix(line, updateFile):
			if err := flush(); err != nil {
				return nil, err
			}
			path = strings.TrimSpace(strings.TrimPrefix(line, updateFile))
			kind = "update"
			hunk = []string{}
		case strings.HasPrefix(line, addFile):
			if err := flush(); err != nil {
				return nil, err
			}
			path = strings.TrimSpace(strings.TrimPrefix(line, addFile))
			kind = "add"
			hunk = []string{}
		case strings.HasPrefix(line, "@@"):
			if path == "" {
				return nil, fmt.Errorf("apply_patch: hunk with no file")
			}
			if kind == "update" && len(hunk) > 0 {
				if err := flush(); err != nil {
					return nil, err
				}
				hunk = []string{}
			}
		case line == "" && (kind == "update" || kind == "add"):
			if err := flush(); err != nil {
				return nil, err
			}
		case kind == "update" || kind == "add":
			hunk = append(hunk, line)
		case strings.TrimSpace(line) == "":
			// leading blank inside the envelope
		default:
			return nil, fmt.Errorf("apply_patch: unexpected line %q", line)
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	if out.Len() == 0 {
		return nil, fmt.Errorf("apply_patch: no hunks")
	}
	return []byte(out.String()), nil
}

func patchBody(s string) (string, error) {
	begin := strings.Index(s, beginPatch)
	if begin < 0 {
		return "", fmt.Errorf("apply_patch: missing %s", beginPatch)
	}
	if strings.TrimSpace(s[:begin]) != "" {
		return "", fmt.Errorf("apply_patch: text before %s", beginPatch)
	}
	end := strings.Index(s, endPatch)
	if end < 0 {
		return "", fmt.Errorf("apply_patch: missing %s", endPatch)
	}
	if end < begin {
		return "", fmt.Errorf("apply_patch: %s before %s", endPatch, beginPatch)
	}
	if strings.TrimSpace(s[end+len(endPatch):]) != "" {
		return "", fmt.Errorf("apply_patch: text after %s", endPatch)
	}
	rest := s[begin+len(beginPatch) : end]
	return strings.TrimPrefix(rest, "\n"), nil
}

func looksLikeGit(s string) bool {
	t := strings.TrimSpace(s)
	return strings.HasPrefix(t, "diff --git") || strings.HasPrefix(t, "--- a/") || strings.Contains(s, "\n--- a/")
}

func compileHunk(root, path, kind string, hunk []string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("apply_patch: empty path")
	}
	if rooted.IsRooted(path) {
		return "", fmt.Errorf("apply_patch: %s is not relative to the root", path)
	}
	if kind == "add" {
		var body []string
		for _, line := range hunk {
			text, ok := plusLine(line)
			if !ok {
				return "", fmt.Errorf("apply_patch: Add File %s: expected +line, got %q", path, line)
			}
			body = append(body, text)
		}
		return emit("create", path, 0, 0, body, ""), nil
	}
	old, neu, err := sides(hunk)
	if err != nil {
		return "", err
	}
	if len(old) == 0 {
		return "", fmt.Errorf("apply_patch: %s: update hunk has no old side (use Add File, or give context)", path)
	}
	lines, err := fileLines(root, path)
	if err != nil {
		return "", err
	}
	start, end, n := findUnique(lines, old)
	switch n {
	case 0:
		return "", fmt.Errorf("apply_patch: %s: old side matched no lines", path)
	case 1:
		anchor := ""
		if end > start {
			anchor = old[0]
		}
		return emit("replace", path, start, end, neu, anchor), nil
	default:
		return "", fmt.Errorf("apply_patch: %s: old side matched %d times; refuse rather than guess", path, n)
	}
}

func sides(hunk []string) (old, neu []string, err error) {
	for _, line := range hunk {
		if line == `\ No newline at end of file` || strings.HasPrefix(line, `\`) {
			continue
		}
		if line == "" {
			return nil, nil, fmt.Errorf("apply_patch: empty hunk line (use a leading space for blank context)")
		}
		switch line[0] {
		case ' ':
			old = append(old, line[1:])
			neu = append(neu, line[1:])
		case '-':
			old = append(old, line[1:])
		case '+':
			neu = append(neu, line[1:])
		default:
			return nil, nil, fmt.Errorf("apply_patch: hunk line must start with space, - or +: %q", line)
		}
	}
	return old, neu, nil
}

func plusLine(line string) (string, bool) {
	if line == "" || line[0] != '+' {
		return "", false
	}
	return line[1:], true
}

func fileLines(root, path string) ([]string, error) {
	full, err := rooted.Resolve(root, path)
	if err != nil {
		return nil, fmt.Errorf("apply_patch: %w", err)
	}
	b, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("apply_patch: %s: %w", path, err)
	}
	if len(b) == 0 {
		return nil, nil
	}
	s := string(b)
	if strings.HasSuffix(s, "\n") {
		s = s[:len(s)-1]
	}
	return strings.Split(s, "\n"), nil
}

func findUnique(lines, old []string) (start, end, n int) {
	if len(old) == 0 || len(old) > len(lines) {
		return 0, 0, 0
	}
	for i := 0; i+len(old) <= len(lines); i++ {
		ok := true
		for j := range old {
			if lines[i+j] != old[j] {
				ok = false
				break
			}
		}
		if ok {
			n++
			start, end = i+1, i+len(old)
		}
	}
	return start, end, n
}

func emit(op, path string, start, end int, body []string, anchor string) string {
	var b strings.Builder
	addr := fmt.Sprintf("%d", start)
	if op != "create" && end != start {
		addr = fmt.Sprintf("%d-%d", start, end)
	}
	fmt.Fprintf(&b, "@@ %s %s %s", path, addr, op)
	if anchor != "" {
		fmt.Fprintf(&b, " %s", quoteAnchor(anchor))
	}
	needCount := false
	for _, line := range body {
		if strings.HasPrefix(line, "@@") {
			needCount = true
			break
		}
	}
	if needCount || (op == "create" && len(body) == 0) {
		fmt.Fprintf(&b, " body=%d", len(body))
		if needCount {
			b.WriteString(" raw=true")
		}
	}
	b.WriteByte('\n')
	for _, line := range body {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func quoteAnchor(s string) string {
	if strings.Contains(s, `"`) && !strings.Contains(s, "'") {
		return "anchor='" + s + "'"
	}
	return `anchor="` + s + `"`
}

func trimStar(line string) string {
	line = strings.TrimSpace(line)
	if i := strings.IndexByte(line, ':'); i >= 0 {
		return strings.TrimSpace(line[:i])
	}
	return line
}
