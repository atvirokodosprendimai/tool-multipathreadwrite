package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ADR-133. Only the null device counts as one: a pipe and a regular file are
// places an answer can be read from, so a read to either still records. A
// console cannot be put on a CI runner's stdout, so the Windows console branch
// is not reached here (the reviews of #350).
func TestOnlyTheNullDeviceCountsAsNull(t *testing.T) {
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = null.Close() }()
	if !toNullDevice(null) {
		t.Error("the null device does not count as the null device")
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close(); _ = w.Close() }()
	if toNullDevice(w) {
		t.Error("a pipe counts as the null device")
	}
	f, err := os.Create(filepath.Join(t.TempDir(), "answer"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if toNullDevice(f) {
		t.Error("a regular file counts as the null device")
	}
}
