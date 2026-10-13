package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// stillLicensed names the files a null-device read served whose every line
// served in full an earlier read or write still licenses in this version
// (ADR-148). Such a read records nothing (ADR-133), yet a write to those lines
// is judged on the older licence, and the caller should hear that from the read
// rather than learn it from the write. A line --max-cols cut is not in the
// observation (ADR-137), so it is not vouched for. It answers "" when no file
// qualifies, or when the ledger cannot be read: the write then says what it finds.
func stillLicensed(root string, observed map[string]seen.Observation) string {
	ledger, err := seen.Snapshot(root)
	if err != nil {
		return ""
	}
	entries := entryFinder(root, ledger)
	var named []string
	served := 0
	for path, o := range observed {
		if !o.Whole() && len(o.Spans) == 0 {
			continue // served no line, so nothing to license
		}
		served++
		if licensedBy(root, path, entries(path), o) {
			named = append(named, path)
		}
	}
	if len(named) == 0 {
		return ""
	}
	sort.Strings(named)
	count := len(named)
	more := ""
	if len(named) > 5 {
		more = fmt.Sprintf(" and %d more", len(named)-5)
		named = named[:5]
	}
	return fmt.Sprintf("mrw: still licensed in this version by an earlier read or write, every line served here in full: %s%s (%d of %d files served); a write to those lines is judged on that licence, with the usual checks (ADR-148)",
		strings.Join(named, ", "), more, count, served)
}

// entryFinder looks a path up in the ledger as a write does (ADR-029): by the
// spelling given, and on a miss by another spelling of the same file — an
// in-root link, or a case variant where the filesystem folds case — asking
// os.SameFile, as sameFileEntry in internal/apply does. Each ledger key is
// stat'ed at most once, on the first miss, so a read of many unread files does
// not stat the ledger once per file.
func entryFinder(root string, ledger seen.Ledger) func(string) seen.Observation {
	var infos map[string]os.FileInfo
	return func(path string) seen.Observation {
		if rec, ok := ledger[path]; ok {
			return rec
		}
		want, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return seen.Observation{}
		}
		if infos == nil {
			infos = map[string]os.FileInfo{}
			for key := range ledger {
				if fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(key))); err == nil {
					infos[key] = fi
				}
			}
		}
		for key, fi := range infos {
			if os.SameFile(want, fi) {
				return ledger[key]
			}
		}
		return seen.Observation{}
	}
}

// licensedBy reports whether rec, the ledger's entry for a file, licenses every
// line o served, by the rule a write applies: the same version, and a whole
// entry or one covering each served span. An entry mrw wrote is whole (ADR-118).
// A whole-file read is licensed by spans that reach its last line too, so for
// that one case the file is counted again (the Codex review of #397).
func licensedBy(root, path string, rec, o seen.Observation) bool {
	if rec.SHA == "" || rec.SHA != o.SHA {
		return false
	}
	if rec.Whole() {
		return true
	}
	if !o.Whole() {
		for _, s := range o.Spans {
			if !rec.Covers(s[0], s[1]) {
				return false
			}
		}
		return true
	}
	n, ok := lineCount(filepath.Join(root, filepath.FromSlash(path)), o.SHA)
	return ok && n > 0 && rec.Covers(1, n)
}

// recountCap bounds lineCount's second read of a file, as ADR-104 bounds the
// first (maxFileBytes in internal/read, 1 GiB): the file may have grown since
// the read served it (the second Codex review of #397). A variable so a test
// can lower it.
var recountCap int64 = 1 << 30

// lineCount numbers the file as the write engine does (lines.Split), and only
// when it still holds the version the read observed. It reads at most
// recountCap+1 bytes: a file over the cap is cut there, and a cut file cannot
// hash to a version the read served, since the read refuses a file that large.
func lineCount(full, sha string) (int, bool) {
	f, _, err := regular.Open(full)
	if err != nil {
		return 0, false
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, recountCap+1))
	if err != nil {
		return 0, false
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != sha {
		return 0, false
	}
	ls, _, _ := lines.Split(string(b))
	return len(ls), true
}
