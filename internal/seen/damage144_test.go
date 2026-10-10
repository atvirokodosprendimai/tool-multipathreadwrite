package seen

import (
	"os"
	"strings"
	"testing"
)

// ADR-144. A ledger line mrw cannot parse is skipped and a line past the record
// bound discards the ledger; the CLI says so, from this notice. A ledger mrw
// wrote, a missing one, an empty one and a stale one say nothing here (the
// stale one is IsStale's to tell).
func TestADamagedLedgerIsCountedAndTold(t *testing.T) {
	sha := strings.Repeat("0", 64)
	ledger := func(t *testing.T, extra string) string {
		t.Helper()
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		if err := Record(root, map[string]Observation{"a.go": {SHA: sha}}); err != nil {
			t.Fatal(err)
		}
		p, err := ReadPath(root)
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString(extra); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		return root
	}

	t.Run("a ledger mrw wrote says nothing", func(t *testing.T) {
		if got, err := DamageNotice(ledger(t, "")); err != nil || got != "" {
			t.Errorf("DamageNotice = %q, %v; want none", got, err)
		}
	})
	t.Run("garbage, an empty line and NULs are counted", func(t *testing.T) {
		root := ledger(t, "garbage with no separator\n\n\x00\x00\x00\n"+sha+"  -  b.go\n")
		got, err := DamageNotice(root)
		if err != nil || !strings.Contains(got, "3 line(s) of the read ledger could not be understood") || !strings.Contains(got, "read the files you mean to edit again") {
			t.Errorf("DamageNotice = %q, %v; want the three bad lines counted and the remedy named", got, err)
		}
		if l, _ := Load(root); l["b.go"].SHA != sha {
			t.Errorf("the good line after the garbage was lost: %v", keys(l))
		}
	})
	t.Run("a line past the record bound is told as discarded", func(t *testing.T) {
		old := maxRecordBytes
		maxRecordBytes = 1 << 10
		t.Cleanup(func() { maxRecordBytes = old })
		root := ledger(t, strings.Repeat("x", 4<<10)+"\n")
		got, err := DamageNotice(root)
		if err != nil || !strings.Contains(got, "longer than mrw writes") || !strings.Contains(got, "discarded") {
			t.Errorf("DamageNotice = %q, %v; want the discarded ledger named", got, err)
		}
	})
	t.Run("a stale, an empty and a missing ledger say nothing", func(t *testing.T) {
		root := ledger(t, "")
		p, _ := ReadPath(root)
		if err := os.WriteFile(p, []byte("#not-a-ledger\ngarbage\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if got, err := DamageNotice(root); err != nil || got != "" {
			t.Errorf("stale: DamageNotice = %q, %v; want none (IsStale tells it)", got, err)
		}
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if got, err := DamageNotice(root); err != nil || got != "" {
			t.Errorf("empty: DamageNotice = %q, %v; want none", got, err)
		}
		if got, err := DamageNotice(t.TempDir()); err != nil || got != "" {
			t.Errorf("missing: DamageNotice = %q, %v; want none", got, err)
		}
	})
}

// ADR-144 T3 (the second Codex review of #390). The scan reads the ledger whole
// so it can close the file before it parses, and that read is bounded: a ledger
// past maxScanBytes is not scanned and says nothing, where an unbounded read
// of a hostile ledger could exhaust memory that Load's streaming never would.
func TestAHugeLedgerIsNotReadWhole(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := Record(root, map[string]Observation{"a.go": {SHA: strings.Repeat("0", 64)}}); err != nil {
		t.Fatal(err)
	}
	p, err := ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(strings.Repeat("g\n", 2000)); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	old := maxScanBytes
	maxScanBytes = 1 << 10
	got, err := DamageNotice(root)
	maxScanBytes = old
	if err != nil || got != "" {
		t.Errorf("a ledger past the scan bound: DamageNotice = %q, %v; want none and no read past the bound", got, err)
	}
	if got, err := DamageNotice(root); err != nil || !strings.Contains(got, "2000 line(s)") {
		t.Errorf("within the bound: DamageNotice = %q, %v; want the 2000 bad lines counted", got, err)
	}
}
