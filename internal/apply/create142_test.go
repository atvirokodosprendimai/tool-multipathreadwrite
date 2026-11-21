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

// #754. A create of a path the ledger holds a record for, at another version
// than the file has now, said "changed since mrw last saw it ... pass --force to
// overwrite blind": a create never overwrites, so --force cannot do what that
// says. It says what ADR-142 says, on either surface.
func TestACreateOfAPathChangedSinceItsRecordSaysItExistsAndOffersNoForce(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "now\n")
	seen := map[string]Seen{"a.txt": {SHA: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}
	for _, noForce := range []bool{false, true} {
		res, err := Apply(root, []Input{{Path: "a.txt", Op: "create", Body: []string{"x"}, Lines: -1, Index: 0}}, Options{Seen: seen, NoForce: noForce})
		if err != nil {
			t.Fatal(err)
		}
		if res.Applied || len(res.Hunks) != 1 {
			t.Fatalf("noForce=%v: the create of an existing path applied or lost its verdict: %+v", noForce, res.Hunks)
		}
		r := res.Hunks[0].Reason
		if !strings.Contains(r, "already exists") || !strings.Contains(r, "never overwrites") || strings.Contains(r, "--force") || strings.Contains(r, "changed since") {
			t.Errorf("noForce=%v: reason = %q, want the path exists, a create never overwrites, and no --force", noForce, r)
		}
	}
	if read(t, root, "a.txt") != "now\n" {
		t.Error("a refused create wrote the tree")
	}
}

// #754. A create beside another hunk on the same changed path is still a create
// of a path that exists: the refusal says that, and offers no --force, which
// cannot make it overwrite. Both a second create and a replace beside it.
func TestACreateBesideAnotherHunkOnAChangedPathOffersNoForce(t *testing.T) {
	stale := map[string]Seen{"a.txt": {SHA: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}
	for name, in := range map[string][]Input{
		"two creates": {
			{Path: "a.txt", Op: "create", Body: []string{"z"}, Lines: -1, Index: 0},
			{Path: "a.txt", Op: "create", Body: []string{"y"}, Lines: -1, Index: 1},
		},
		"a replace, then a create": {
			{Path: "a.txt", Op: "replace", Start: 1, End: 1, Body: []string{"u"}, Lines: -1, Index: 0},
			{Path: "a.txt", Op: "create", Body: []string{"y"}, Lines: -1, Index: 1},
		},
	} {
		root := t.TempDir()
		write(t, root, "a.txt", "now\n")
		res, err := Apply(root, in, Options{Seen: stale})
		if err != nil {
			t.Fatal(err)
		}
		if res.Applied {
			t.Fatalf("%s: applied", name)
		}
		said := false
		for _, h := range res.Hunks {
			if strings.Contains(h.Reason, "--force") {
				t.Errorf("%s: a hunk offers --force: %q", name, h.Reason)
			}
			said = said || strings.Contains(h.Reason, "never overwrites")
		}
		if !said {
			t.Errorf("%s: no hunk says a create never overwrites: %+v", name, res.Hunks)
		}
		if read(t, root, "a.txt") != "now\n" {
			t.Errorf("%s: a refused plan wrote the tree", name)
		}
	}
}
