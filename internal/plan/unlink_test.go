package plan

import (
	"strings"
	"testing"
)

func TestUnlinkParses(t *testing.T) {
	hunks, err := Parse(strings.NewReader("@@ gone.txt - unlink\n"))
	if err != nil {
		t.Fatalf("parse unlink: %v", err)
	}
	if len(hunks) != 1 {
		t.Fatalf("got %d hunks, want 1", len(hunks))
	}
	h := hunks[0]
	if h.Path != "gone.txt" || h.Op != OpUnlink {
		t.Fatalf("hunk = %+v, want path gone.txt op unlink", h)
	}
	if h.Addr != (Addr{Start: 0, End: 0}) {
		t.Errorf("addr = %+v, want -", h.Addr)
	}
	if len(h.Body) != 0 {
		t.Errorf("empty body must be legal on unlink, got %q", h.Body)
	}
}

func TestRenameParsesWithDestBody(t *testing.T) {
	hunks, err := Parse(strings.NewReader("@@ old.txt - rename\ndest.txt\n"))
	if err != nil {
		t.Fatalf("parse rename: %v", err)
	}
	if len(hunks) != 1 {
		t.Fatalf("got %d hunks, want 1", len(hunks))
	}
	h := hunks[0]
	if h.Path != "old.txt" || h.Op != OpRename {
		t.Fatalf("hunk = %+v, want path old.txt op rename", h)
	}
	if len(h.Body) != 1 || h.Body[0] != "dest.txt" {
		t.Fatalf("body = %q, want one dest path", h.Body)
	}
}
