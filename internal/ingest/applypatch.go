// Package ingest compiles a foreign edit grammar into a native mrw plan.
//
// The first grammar is Codex apply_patch (ADR-051). The second is Aider
// SEARCH/REPLACE (--format=search_replace). Compile emits plan text;
// plan.Parse is the only door into Apply. Context matching locates an old
// side; it is not a license and it is not a target-syntax parse.
package ingest

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
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
		case strings.HasPrefix(line, deleteFile):
			if err := flush(); err != nil {
				return nil, err
			}
			p := pathAfter(line, deleteFile)
			text, err := compilePathOp("unlink", p, nil)
			if err != nil {
				return nil, err
			}
			out.WriteString(text)
			path, kind = "", ""
			hunk = nil
		case strings.HasPrefix(line, moveTo):
			if kind != "update" || path == "" {
				return nil, fmt.Errorf("apply_patch: %s with no Update File", trimStar(line))
			}
			if len(hunk) > 0 {
				return nil, fmt.Errorf("apply_patch: Move to with hunks is not compiled this slice")
			}
			dest := pathAfter(line, moveTo)
			text, err := compilePathOp("rename", path, []string{dest})
			if err != nil {
				return nil, err
			}
			out.WriteString(text)
			kind = "moved"
			hunk = nil
		case strings.HasPrefix(line, updateFile):
			if err := flush(); err != nil {
				return nil, err
			}
			path = pathAfter(line, updateFile)
			kind = "update"
			hunk = []string{}
		case strings.HasPrefix(line, addFile):
			if err := flush(); err != nil {
				return nil, err
			}
			path = pathAfter(line, addFile)
			kind = "add"
			hunk = []string{}
		case strings.HasPrefix(line, "@@"):
			if path == "" {
				return nil, fmt.Errorf("apply_patch: hunk with no file")
			}
			if kind == "moved" {
				return nil, fmt.Errorf("apply_patch: Move to with hunks is not compiled this slice")
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
			// leading blank inside the envelope; trailing blank after Move to
		case kind == "moved":
			return nil, fmt.Errorf("apply_patch: Move to with hunks is not compiled this slice")
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

// pathAfter returns the path that follows a format marker. Exactly one
// separator space is removed, never the path's own edge spaces, so a patch
// naming "x " compiles to a hunk on "x " and not on x (ADR-069). A marker
// followed only by whitespace names no path.
func pathAfter(line, marker string) string {
	rest := strings.TrimPrefix(strings.TrimPrefix(line, marker), " ")
	if strings.TrimSpace(rest) == "" {
		return ""
	}
	return rest
}

func compilePathOp(op, path string, body []string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("apply_patch: empty path")
	}
	if rooted.IsRooted(path) {
		return "", fmt.Errorf("apply_patch: %s is not relative to the root", path)
	}
	if op == "rename" {
		if len(body) != 1 || strings.TrimSpace(body[0]) == "" {
			return "", fmt.Errorf("apply_patch: Move to needs a dest path")
		}
		dest := body[0] // as written: the emptiness guard above uses a trimmed copy (ADR-069)
		if rooted.IsRooted(dest) {
			return "", fmt.Errorf("apply_patch: %s is not relative to the root", dest)
		}
		body = []string{dest}
	}
	return emitPath(op, path, body), nil
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
	b, err := targetBytes(full)
	if err != nil {
		return nil, fmt.Errorf("apply_patch: %s: %w", path, err)
	}
	// The target's lines as the write engine numbers them (ADR-065): an LF
	// patch then meets "two", not "two\r", in a CRLF or CR-only file.
	ls, _, _ := lines.Split(string(b))
	return ls, nil
}

// targetBytes reads the file a foreign document edits, to locate its old side.
// It looks before it opens: the compilers read the target before apply's own
// regular-file check (ADR-073), so a FIFO named in a document blocked the write
// until something wrote to the pipe (review of #230). A directory keeps
// ReadFile's own error.
func targetBytes(full string) ([]byte, error) {
	if fi, err := os.Stat(full); err == nil && !fi.Mode().IsRegular() && !fi.IsDir() {
		return nil, errors.New(lines.NotRegular)
	}
	return os.ReadFile(full)
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

func emitPath(op, path string, body []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "@@ %s - %s\n", quotePlanPath(path), op)
	for _, line := range body {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func emit(op, path string, start, end int, body []string, anchor string) string {
	var b strings.Builder
	addr := fmt.Sprintf("%d", start)
	if op != "create" && end != start {
		addr = fmt.Sprintf("%d-%d", start, end)
	}
	fmt.Fprintf(&b, "@@ %s %s %s", quotePlanPath(path), addr, op)
	if anchor != "" {
		fmt.Fprintf(&b, " %s", quoteAnchor(anchor))
	}
	// A body with an @@ line needs a count so the parser knows where it ends;
	// one whose first line begins body= needs one so the parser does not refuse
	// it as an option written under the header (ADR-070).
	needCount := len(body) > 0 && strings.HasPrefix(strings.TrimSpace(body[0]), "body=")
	raw := false
	for _, line := range body {
		if strings.HasPrefix(line, "@@") {
			needCount, raw = true, true
			break
		}
	}
	if needCount || (op == "create" && len(body) == 0) {
		fmt.Fprintf(&b, " body=%d", len(body))
		if raw {
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

func quotePlanPath(p string) string {
	if !strings.ContainsAny(p, " \t\"") {
		return p
	}
	esc := strings.ReplaceAll(p, `\`, `\\`)
	esc = strings.ReplaceAll(esc, `"`, `\"`)
	return `"` + esc + `"`
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
