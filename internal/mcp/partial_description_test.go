package mcp

import (
	"strings"
	"testing"
)

// ADR-066 T3 (Codex review of #207). A commit failure can leave some files
// written with `failed` non-zero, so the receipt's own descriptions must not
// tell a host that a non-zero `failed` means nothing was written, or that
// `written` is false for every file whenever a hunk failed.
func TestTheReceiptDescriptionsAllowAPartialCommit(t *testing.T) {
	failed := writeDescriptions["failed"]
	if strings.Contains(failed, "Non-zero means NOTHING was written") {
		t.Errorf("`failed` still says any failure means nothing was written: %s", failed)
	}
	if !strings.Contains(failed, "files[].written") {
		t.Errorf("`failed` does not point at files[].written for a commit failure: %s", failed)
	}
	if w := writeDescriptions["files.written"]; strings.Contains(w, "false for every file when any hunk failed") {
		t.Errorf("`files.written` still says every file is unwritten when a hunk failed: %s", w)
	}
}
