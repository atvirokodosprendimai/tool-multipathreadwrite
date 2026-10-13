package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// stillLicensed names the files a null-device read served whose every served
// line an earlier read or write still licenses in this version (ADR-148). Such
// a read records nothing (ADR-133), yet a write to those lines applies on the
// older licence, and the caller should hear that from the read rather than
// learn it from the write. It answers "" when no file qualifies, or when the
// ledger cannot be read: the write then says what it finds.
func stillLicensed(root string, observed map[string]seen.Observation) string {
	ledger, err := seen.Snapshot(root)
	if err != nil {
		return ""
	}
	var named []string
	served := 0
	for path, o := range observed {
		if !o.Whole() && len(o.Spans) == 0 {
			continue // served no line, so nothing to license
		}
		served++
		if licensedBy(ledger[path], o) {
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
	return fmt.Sprintf("mrw: still licensed in this version by an earlier read or write, so a write to the lines served here applies all the same: %s%s (%d of %d files served) (ADR-148)",
		strings.Join(named, ", "), more, count, served)
}

// licensedBy reports whether rec, the ledger's entry for a file, licenses every
// line o served, by the rule a write applies: the same version, and a whole
// entry or one covering each served span. An entry mrw wrote is whole (ADR-118).
func licensedBy(rec, o seen.Observation) bool {
	if rec.SHA == "" || rec.SHA != o.SHA {
		return false
	}
	if o.Whole() {
		return rec.Whole()
	}
	for _, s := range o.Spans {
		if !rec.Covers(s[0], s[1]) {
			return false
		}
	}
	return true
}
