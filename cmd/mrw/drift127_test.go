package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-127, the in-process review of #342. The human receipt names the other
// writes that landed while the check ran, and says nothing when none did.
func TestTheCLIReceiptNamesOtherWritersDuringTheCheck(t *testing.T) {
	render := func(others int) string {
		t.Helper()
		f, err := os.Create(filepath.Join(t.TempDir(), "out"))
		if err != nil {
			t.Fatal(err)
		}
		reportDrift(f, nil, others)
		_ = f.Close()
		b, err := os.ReadFile(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if got := render(3); !strings.Contains(got, "drift: 3 other write(s) landed in this checkout while the check ran") {
		t.Errorf("three other writes were not named: %q", got)
	}
	if got := render(0); got != "" {
		t.Errorf("no other write, and the receipt said %q", got)
	}
}
