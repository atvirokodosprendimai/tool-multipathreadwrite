package main

import "testing"

// ADR-112 T2. A write's check could change a file the write had just landed —
// or another writer could, while the check ran — and the receipt said nothing.
// The write names each such file under drift, and the exit code stays the
// check's; a check that changes nothing carries no drift.
func TestAWriteNamesAFileChangedWhileItsCheckRan(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"a.txt":                 "a\n",
		"p.mrw":                 "@@ a.txt 1 replace\nb\n",
		".quality-harness.json": `{"check":"printf x >> a.txt"}` + "\n",
	})
	if out, code := runIn(t, root, "read", "a.txt"); code != 0 {
		t.Fatalf("read: exit %d:\n%s", code, out)
	}
	t.Chdir(root) // a plan file resolves from the working directory
	stdout, err := runSplit(t, "-C", root, "write", "--check", "--json", "p.mrw")
	doc, ok := oneDocument(stdout)
	drift, _ := doc["drift"].([]any)
	if err != nil || !ok || len(drift) != 1 || drift[0] != "a.txt" {
		t.Errorf("a check that appended to a.txt: err %v, want exit 0 and drift [a.txt]:\n%s", err, stdout)
	}

	writeFiles(t, root, map[string]string{
		"p.mrw":                 "@@ a.txt 1 replace\nc\n",
		".quality-harness.json": `{"check":"exit 0"}` + "\n",
	})
	if out, code := runIn(t, root, "read", "a.txt"); code != 0 {
		t.Fatalf("read: exit %d:\n%s", code, out)
	}
	stdout, err = runSplit(t, "-C", root, "write", "--check", "--json", "p.mrw")
	doc, ok = oneDocument(stdout)
	if _, has := doc["drift"]; err != nil || !ok || has {
		t.Errorf("a check that changed nothing: err %v, want exit 0 and no drift:\n%s", err, stdout)
	}
}
