package seen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// header is the first line this package writes when nobody told it its writer:
// the older tests compare a saved ledger, and plant fixtures, against it.
const header = "#mrw-seen v6 unknown"

// plant writes body as the ledger of root, byte for byte.
func plant(t *testing.T, root, body string) string {
	t.Helper()
	p, err := ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// ADR-150 Decisions 1 and 2: the header names its writer; v4, v5 and v6 are
// current (ADR-161), anything older or of another shape is discarded as stale.
func TestALedgerNamesItsWriter(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if got := writerName(); got != "unknown" {
		t.Fatalf("a binary that was not told its version names %q, want unknown", got)
	}
	old := writerName()
	t.Cleanup(func() { SetWriter(old) })
	SetWriter("v9.0.0 (abc1234)")
	sha := strings.Repeat("a", 64)
	if err := Record(root, map[string]Observation{"a.go": {SHA: sha}}); err != nil {
		t.Fatal(err)
	}
	p, err := ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if first, _, _ := strings.Cut(string(b), "\n"); first != "#mrw-seen v6 v9.0.0 (abc1234)" {
		t.Errorf("the header is %q, want it to name its format and its writer", first)
	}

	for _, current := range []string{
		"#mrw-seen v4\n" + sha + "  -  k.go\n",
		"#mrw-seen v5 v1.66.0 (1234567)\n" + sha + "  -  k.go\n",
		"#mrw-seen v5\n" + sha + "  -  k.go\n",
		"#mrw-seen v6 v1.68.0 (1234567)\n" + sha + "  -  k.go\n",
	} {
		plant(t, root, current)
		if l, err := Load(root); err != nil || len(l) != 1 {
			t.Errorf("a current ledger was not trusted: %q gave %v %v", current, l, err)
		}
		if stale, _ := IsStale(root); stale {
			t.Errorf("a current ledger was reported stale: %q", current)
		}
		if _, newer, _ := Newer(root); newer {
			t.Errorf("a current ledger was reported newer: %q", current)
		}
	}
	for _, older := range []string{
		"#mrw-seen v3\n" + sha + "  -  k.go\n",
		"#mrw-seen v04\n" + sha + "  -  k.go\n",
		"#mrw-seen v4x\n" + sha + "  -  k.go\n",
		"#mrw-seen v+5\n" + sha + "  -  k.go\n",
		"#mrw-seen v-6\n" + sha + "  -  k.go\n",
		"#mrw-seen v5 v1.66.0\r\n" + sha + "  -  k.go\r\n",
		"#not-a-ledger\n" + sha + "  -  k.go\n",
	} {
		plant(t, root, older)
		if l, err := Load(root); err != nil || len(l) != 0 {
			t.Errorf("an older ledger was trusted: %q gave %v %v", older, l, err)
		}
		if stale, _ := IsStale(root); !stale {
			t.Errorf("an older ledger was not reported stale: %q", older)
		}
		if _, newer, _ := Newer(root); newer {
			t.Errorf("an older ledger was reported newer: %q", older)
		}
	}
}

// ADR-150 Decision 2: a format newer than this mrw's licenses nothing, is not
// stale, is reported with its writer, and is never written over.
func TestANewerLedgerIsNeitherUsedNorOverwritten(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	sha := strings.Repeat("b", 64)
	body := "#mrw-seen v7 v9.9.9 (fffffff)\n" + sha + "  -  a.go\nwhatever v7 means\n"
	p := plant(t, root, body)

	if l, err := Load(root); err != nil || len(l) != 0 {
		t.Errorf("a newer ledger licensed something: %v %v", l, err)
	}
	if stale, _ := IsStale(root); stale {
		t.Error("a newer ledger was reported stale, which tells the caller it is older")
	}
	if notice, _ := DamageNotice(root); notice != "" {
		t.Errorf("a newer ledger was reported damaged: %s", notice)
	}
	writer, newer, err := Newer(root)
	if err != nil || !newer || writer != "v9.9.9 (fffffff)" {
		t.Errorf("Newer = %q, %v, %v; want the writer v9.9.9 (fffffff)", writer, newer, err)
	}
	if n := NewerNotice(writer); !strings.Contains(n, "written by a newer mrw (v9.9.9 (fffffff))") || !strings.Contains(n, writerName()) {
		t.Errorf("the notice does not name both binaries: %s", n)
	}

	if err := Record(root, map[string]Observation{"b.go": {SHA: sha}}); err != nil {
		t.Errorf("a read against a newer ledger failed: %v", err)
	}
	if err := Drop(root, []string{"a.go"}); err != nil {
		t.Errorf("a drop against a newer ledger failed: %v", err)
	}
	if got, err := os.ReadFile(p); err != nil || string(got) != body {
		t.Errorf("a newer ledger was written over:\n%q\nwant\n%q", got, body)
	}

	// A writer name is shown on a terminal: what is not printable is dropped.
	plant(t, root, "#mrw-seen v7 evil\x1b[31m red\n")
	if writer, newer, _ := Newer(root); !newer || strings.ContainsRune(writer, 0x1b) {
		t.Errorf("Newer = %q, %v; want it newer with the escape dropped", writer, newer)
	}

	// A format number past int's range is still newer, and still kept (the
	// Codex review of #403).
	huge := "#mrw-seen v9223372036854775808 future\n" + sha + "  -  a.go\n"
	plant(t, root, huge)
	if _, newer, _ := Newer(root); !newer {
		t.Error("a format number past int's range was not reported newer")
	}
	_ = Record(root, map[string]Observation{"b.go": {SHA: sha}})
	_ = Drop(root, []string{"a.go"})
	if got, _ := os.ReadFile(p); string(got) != huge {
		t.Errorf("a ledger whose format number is past int's range was written over: %q", got)
	}
}
