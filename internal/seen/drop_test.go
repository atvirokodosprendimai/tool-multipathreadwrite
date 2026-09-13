package seen

import "testing"

func TestDropRemovesThePath(t *testing.T) {
	root := t.TempDir()
	if err := Record(root, map[string]Observation{
		"gone.go": {SHA: "aaa"},
		"keep.go": {SHA: "bbb"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := Drop(root, []string{"gone.go"}); err != nil {
		t.Fatal(err)
	}
	l, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := l["gone.go"]; ok {
		t.Fatalf("Drop left gone.go in the ledger: %v", l)
	}
	if l["keep.go"].SHA != "bbb" {
		t.Fatalf("Drop disturbed keep.go: %v", l)
	}
}
