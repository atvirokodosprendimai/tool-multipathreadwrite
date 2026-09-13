package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadHonoursFenceTimeout is the Zeus shape: quality-harness writes
// fenceTimeout and no timeout_seconds. Dropping the unknown key left
// TimeoutSeconds 0 and Run used the five-minute default.
func TestLoadHonoursFenceTimeout(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"),
		[]byte(`{"check":"true","fenceTimeout":1800}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TimeoutSeconds != 1800 {
		t.Errorf("TimeoutSeconds = %d, want 1800 (fenceTimeout alias)", cfg.TimeoutSeconds)
	}
}

func TestLoadTimeoutSecondsAloneStillApplies(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"),
		[]byte(`{"check":"true","timeout_seconds":42}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TimeoutSeconds != 42 {
		t.Errorf("TimeoutSeconds = %d, want 42", cfg.TimeoutSeconds)
	}
}

func TestLoadEqualTimeoutKeysAreAccepted(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"),
		[]byte(`{"check":"true","timeout_seconds":30,"fenceTimeout":30}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TimeoutSeconds != 30 {
		t.Errorf("TimeoutSeconds = %d, want 30", cfg.TimeoutSeconds)
	}
}

func TestLoadRefusesWhenTimeoutKeysDisagree(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"),
		[]byte(`{"check":"true","timeout_seconds":10,"fenceTimeout":1800}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root)
	if err == nil {
		t.Fatal("disagreeing timeout keys loaded")
	}
	msg := err.Error()
	if !strings.Contains(msg, "10") || !strings.Contains(msg, "1800") {
		t.Errorf("error does not name both values: %v", err)
	}
}

func TestAZeusShapedHarnessTimesOutUsingFenceTimeout(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"),
		[]byte(`{"check":"sleep 5","fenceTimeout":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Run(context.Background(), root, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK() {
		t.Error("a fenceTimeout-bounded sleep reported OK")
	}
	if !strings.Contains(res.Skipped, "timed out") {
		t.Errorf("Skipped = %q", res.Skipped)
	}
	os.Remove(res.OutputFile)
}
