package ingest

import (
	"fmt"
	"os"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/quote"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

const (
	searchHead = "<<<<<<< SEARCH"
	srDiv      = "======="
	srEnd      = ">>>>>>> REPLACE"
)

// CompileSearchReplace turns an Aider SEARCH/REPLACE document into native plan
// text. root is the checkout the paths are relative to. The function writes
// nothing. A unique exact SEARCH match is a quote of the lines it replaces
// (ADR-190): each hunk carries a mark apply checks, and a near miss is not
// fuzzy applied.
func CompileSearchReplace(root string, doc []byte) ([]byte, error) {
	doc, _ = DropLeadingMarks(doc)
	s := strings.ReplaceAll(string(doc), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}

	var out strings.Builder
	lastPath := ""
	for i := 0; i < len(lines); {
		line := lines[i]
		if isMarkdownFence(line) {
			i++
			continue
		}
		if strings.HasPrefix(line, searchHead) {
			path := lastPath
			if rest := pathAfter(line, searchHead); rest != "" {
				path = rest
			}
			if path == "" {
				return nil, fmt.Errorf("search_replace: SEARCH with no path")
			}
			if rooted.IsRooted(path) {
				return nil, fmt.Errorf("search_replace: %s is not relative to the root", path)
			}
			i++
			var search []string
			for i < len(lines) && lines[i] != srDiv {
				if strings.HasPrefix(lines[i], searchHead) || lines[i] == srEnd {
					return nil, fmt.Errorf("search_replace: %s: SEARCH is missing %s", path, srDiv)
				}
				search = append(search, lines[i])
				i++
			}
			if i >= len(lines) {
				return nil, fmt.Errorf("search_replace: %s: SEARCH is missing %s", path, srDiv)
			}
			i++
			var replace []string
			for i < len(lines) && lines[i] != srEnd {
				if strings.HasPrefix(lines[i], searchHead) || lines[i] == srDiv {
					return nil, fmt.Errorf("search_replace: %s: REPLACE is missing %s", path, srEnd)
				}
				replace = append(replace, lines[i])
				i++
			}
			if i >= len(lines) {
				return nil, fmt.Errorf("search_replace: %s: REPLACE is missing %s", path, srEnd)
			}
			i++
			text, err := compileSearchHunk(root, path, search, replace)
			if err != nil {
				return nil, err
			}
			out.WriteString(text)
			lastPath = path
			continue
		}
		if strings.TrimSpace(line) != "" {
			lastPath = line // as written; the blank test above trims a copy (ADR-069)
		}
		i++
	}
	if out.Len() == 0 {
		return nil, fmt.Errorf("search_replace: no hunks")
	}
	return []byte(out.String()), nil
}

func isMarkdownFence(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "```")
}

func compileSearchHunk(root, path string, search, replace []string) (string, error) {
	if len(search) == 0 {
		full, err := rooted.Resolve(root, path)
		if err != nil {
			return "", fmt.Errorf("search_replace: %w", err)
		}
		if _, err := os.Stat(full); err == nil {
			return "", fmt.Errorf("search_replace: %s: empty SEARCH on an existing file (give the old side; do not replace the whole file)", path)
		}
		return emit("create", path, 0, 0, replace, ""), nil
	}
	lines, err := searchFileLines(root, path)
	if err != nil {
		return "", err
	}
	start, end, n := findUnique(lines, search)
	switch n {
	case 0:
		return "", fmt.Errorf("search_replace: %s: SEARCH matched no lines", path)
	case 1:
		// The block quoted every line it replaces, so the hunk carries a mark
		// apply can check against the file under the write lock (ADR-190). A
		// closer SEARCH and REPLACE share is kept, not rewritten: it is the
		// line after the range, and the quote is what licenses it (ADR-052).
		tail := 0
		if len(search) >= 2 && len(replace) >= 2 && search[len(search)-1] == replace[len(replace)-1] {
			tail, end, replace = 1, end-1, replace[:len(replace)-1]
		}
		anchor := ""
		if end > start {
			anchor = search[0]
		}
		return emitMarked("replace", path, start, end, replace, anchor, quote.Mark(path, start, end, tail, search)), nil
	default:
		return "", fmt.Errorf("search_replace: %s: SEARCH matched %d times; refuse rather than guess", path, n)
	}
}

func searchFileLines(root, path string) ([]string, error) {
	full, err := rooted.Resolve(root, path)
	if err != nil {
		return nil, fmt.Errorf("search_replace: %w", err)
	}
	b, err := targetBytes(full)
	if err != nil {
		return nil, fmt.Errorf("search_replace: %s: %w", path, err)
	}
	// The target's lines as the write engine numbers them (ADR-065).
	ls, _, _ := lines.Split(string(b))
	return ls, nil
}
