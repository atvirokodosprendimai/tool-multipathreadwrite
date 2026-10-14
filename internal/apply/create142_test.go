package apply

import (
	"strings"
	"testing"
)

// ADR-142. A create of a path that exists says that it exists, whether or not
// the file was read: a create never overwrites, so "read it first, or --force"
// sent the caller down a path that cannot work. An edit of the same unread
// file keeps the read-before-modify refusal.
func TestACreateOfAnUnreadExistingPathSaysItExists(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "one\n")
	res, err := Apply(root, []Input{{Path: "a.txt", Op: "create", Body: []string{"x"}, Lines: -1, Index: 0}}, Options{Seen: map[string]Seen{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied || len(res.Hunks) != 1 {
		t.Fatalf("the create of an existing path applied or lost its verdict: %+v", res.Hunks)
	}
	if r := res.Hunks[0].Reason; !strings.Contains(r, "already exists") || strings.Contains(r, "has not been read") || strings.Contains(r, "--force") {
		t.Errorf("reason = %q, want a message that says the path exists and offers no read or --force", r)
	}
	res, err = Apply(root, []Input{{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"x"}, Lines: -1, Index: 0}}, Options{Seen: map[string]Seen{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied || len(res.Hunks) != 1 || !strings.Contains(res.Hunks[0].Reason, "has not been read") {
		t.Errorf("an edit of the unread file lost its read-before-modify refusal: %+v", res.Hunks)
	}
	if read(t, root, "a.txt") != "one\n" {
		t.Error("a refused create wrote the tree")
	}
}
