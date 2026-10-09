package rooted

import (
	"strings"
	"syscall"
	"testing"
)

// The Codex review of #361. os.SameFile opens a file with share mode 0 and
// answers false when another process holds it, so a ledger held open (a
// concurrent mrw, an editor, a scanner) let its alias through. Identity is
// read from an attributes-only handle shared with every other opener.
func TestAHeldLedgerIsStillRefusedThroughItsAlias(t *testing.T) {
	root, ledger := hardLinkRoot(t)
	p, err := syscall.UTF16PtrFromString(ledger)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatalf("could not hold the ledger open: %v", err)
	}
	defer func() { _ = syscall.CloseHandle(h) }()
	r := NewResolver(root)
	for who, resolve := range map[string]func(string) (string, error){
		"Resolve":          func(p string) (string, error) { return Resolve(root, p) },
		"Resolver.Resolve": r.Resolve,
	} {
		if _, err := resolve("hl.txt"); err == nil || !strings.Contains(err.Error(), "own state") {
			t.Errorf("%s(hl.txt) while the ledger is held open = %v, want it refused as mrw's own state", who, err)
		}
	}
}
