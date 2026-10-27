package seen

import (
	"os"
	"strings"
	"testing"
)

// ADR-175 Decision 1: Forget removes every observation of the checkout, read
// and written alike, keeps the header, and says how many files it held. A
// newer mrw's ledger is not touched and the refusal is ADR-150's notice.
func TestForgetDropsEveryLicenceInTheCheckout(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	sha := strings.Repeat("c", 64)
	if err := Record(root, map[string]Observation{
		"read.go":  {SHA: sha},
		"wrote.go": {SHA: sha, Written: true},
	}); err != nil {
		t.Fatal(err)
	}
	n, err := Forget(root)
	if err != nil || n != 2 {
		t.Fatalf("Forget = %d, %v; want 2 files forgotten", n, err)
	}
	if l, err := Load(root); err != nil || len(l) != 0 {
		t.Errorf("the ledger still licenses %v after Forget (%v)", l, err)
	}
	p, err := ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "#mrw-seen v6 ") {
		t.Errorf("the emptied ledger lost its header: %q", b)
	}
	if n, err := Forget(root); err != nil || n != 0 {
		t.Errorf("Forget on an empty ledger = %d, %v; want 0, nil", n, err)
	}

	body := "#mrw-seen v7 v9.9.9 (fffffff)\n" + sha + "  -  a.go\nwhatever v7 means\n"
	p = plant(t, root, body)
	n, err = Forget(root)
	if err == nil || n != 0 || !strings.Contains(err.Error(), "written by a newer mrw (v9.9.9 (fffffff))") {
		t.Errorf("Forget over a newer ledger = %d, %v; want 0 and ADR-150's notice", n, err)
	}
	if got, _ := os.ReadFile(p); string(got) != body {
		t.Errorf("a newer ledger was written over:\n%q\nwant\n%q", got, body)
	}
}
