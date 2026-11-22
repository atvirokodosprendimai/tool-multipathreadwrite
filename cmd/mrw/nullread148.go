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
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
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
	entries := seen.NewAliasIndex(root, ledger, os.Stat)
	var named []string
	served := 0
	for path, o := range observed {
		if !o.Whole() && len(o.Spans) == 0 {
			continue // served no line, so nothing to license
		}
		served++
		if licensedBy(root, path, entries, o) {
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
	return fmt.Sprintf("mrw: still licensed in this version by an earlier read or write, every line served here in full: %s%s (%d of %d files served); a write to those lines is judged on that licence, with the usual checks (ADR-148); in a file mrw wrote, a number alone still needs the line shown (ADR-203)",
		strings.Join(named, ", "), more, count, served)
}

// licensedBy asks whether every line observed by this read is licensed in the
// current version, including a licence split across several names. For a
// whole-file observation with partial licences, count the file once before
// judging whether the spans reach the end (ADR-148).
func licensedBy(root, path string, entries *seen.AliasIndex, o seen.Observation) bool {
	full := filepath.Join(root, filepath.FromSlash(path))
	evidence := entries.File(full, path, o.SHA)
	rec, ok := evidence.Primary()
	if !ok || rec.SHA != o.SHA {
		return false
	}
	if rec.Whole() {
		return true
	}
	if o.Whole() {
		n, counted := lineCount(root, full, o.SHA)
		if counted && n > 0 {
			_, licensed := evidence.Licence(1, n)
			return licensed
		}
		rec, ok = evidence.All()
		return ok && rec.SHA == o.SHA && rec.Whole()
	}
	for _, span := range o.Spans {
		if _, licensed := evidence.Licence(span[0], span[1]); !licensed {
			return false
		}
	}
	return true
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
func lineCount(root, full, sha string) (int, bool) {
	f, _, err := rooted.OpenRead(root, full)
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
