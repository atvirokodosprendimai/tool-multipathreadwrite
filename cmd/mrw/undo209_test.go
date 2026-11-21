package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-209 T1. A commit that fails after a content file was renamed in puts the file
// back: a.go replaced and d/x.txt unlinked in a read-only directory exits 2,
// NOTHING WRITTEN, a.go as it was, the write counted refused_apply, and the next
// write to a.go lands without a new read, since the read made before still licenses it.
func TestTheCLIPutsTheContentFilesBackWhenACommitFails(t *testing.T) {
	root := partialTree(t)
	out, code := writeIn(t, root, "--no-check", planFile(t, partialPlan))
	if code != exitUsage || !strings.Contains(out, "NOTHING WRITTEN") || strings.Contains(out, "PARTIALLY APPLIED") {
		t.Fatalf("exit %d, want %d and a NOTHING WRITTEN receipt:\n%s", code, exitUsage, out)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "a.go")); string(b) != "package a\n" {
		t.Fatalf("a.go was not put back: %q", b)
	}
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".mrw-") {
			t.Errorf("the failed commit left %s behind", e.Name())
		}
	}
	js, code := runIn(t, root, "stats", "--json")
	var s struct {
		Counts map[string]int `json:"counts"`
		Landed int            `json:"landed"`
	}
	if err := json.Unmarshal([]byte(js), &s); code != 0 || err != nil {
		t.Fatalf("stats --json: exit %d, %v:\n%s", code, err, js)
	}
	if s.Counts["refused_apply"] != 1 || s.Counts["partially_applied"] != 0 || s.Landed != 0 {
		t.Errorf("counts %v landed %d, want refused_apply 1, partially_applied 0, landed 0", s.Counts, s.Landed)
	}
	if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ a.go 1 replace\npackage c\n")); code != 0 {
		t.Fatalf("the next write to a.go needed a read it should not: exit %d:\n%s", code, out)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "a.go")); string(b) != "package c\n" {
		t.Errorf("a.go = %q", b)
	}
}
