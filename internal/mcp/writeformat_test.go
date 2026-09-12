package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const demoFile = "package demo\n\nfunc A() int { return 1 }\nfunc B() int { return 2 }\nfunc C() int { return 3 }\n"

const twoHunkApplyPatch = "*** Begin Patch\n" +
	"*** Update File: a.go\n" +
	"@@\n" +
	"-func A() int { return 1 }\n" +
	"+func A() int { return 10 }\n" +
	"@@\n" +
	"-func C() int { return 3 }\n" +
	"+func C() int { return 30 }\n" +
	"*** End Patch\n"

// TestWriteFormatApplyPatchUnreadWritesNothing is ADR-051 F-27: mrw_write
// format=apply_patch compiles then Apply refuses the unread sibling. A mutant
// that ignores format fails this as a parse error instead of FAIL+skip.
func TestWriteFormatApplyPatchUnreadWritesNothing(t *testing.T) {
	root, _ := checkout(t, "a.go", demoFile)
	served := call(t, root, "mrw_read", map[string]any{"specs": []any{"a.go:3"}})
	acks := checkpointsIn(served0(t, served))
	if len(acks) == 0 {
		t.Fatal("the line-3 serve carried no checkpoints")
	}

	got := structured(t, call(t, root, "mrw_write", map[string]any{
		"plan":   twoHunkApplyPatch,
		"format": "apply_patch",
		"ack":    acks,
	}))
	if applied, _ := got["applied"].(bool); applied {
		t.Fatalf("an unread compiled sibling applied: %v", got)
	}
	var failed, skipped int
	hunks, _ := got["hunks"].([]any)
	for _, raw := range hunks {
		h, _ := raw.(map[string]any)
		switch h["status"] {
		case "failed":
			failed++
			if reason, _ := h["reason"].(string); !strings.Contains(reason, "has not been read") {
				t.Errorf("failed hunk is not the ledger refusal: %q", reason)
			}
		case "skipped":
			skipped++
		}
	}
	if failed != 1 || skipped != 1 {
		t.Errorf("failed=%d skipped=%d, want 1 and 1: %v", failed, skipped, got)
	}
	b, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != demoFile {
		t.Errorf("the file was written:\n%s", b)
	}
}

// TestAnApplyPatchBlobOnWritePlanIsABadNativePlan is UC4-S2: without format,
// the blob is a bad native plan. Nothing is written. No third tool appears.
func TestAnApplyPatchBlobOnWritePlanIsABadNativePlan(t *testing.T) {
	root, _ := checkout(t, "a.go", demoFile)
	res := call(t, root, "mrw_write", map[string]any{"plan": twoHunkApplyPatch})
	if res["isError"] != true {
		t.Fatalf("an apply_patch blob without format was not a parse refusal: %v", res)
	}
	b, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != demoFile {
		t.Errorf("the file was written:\n%s", b)
	}
	if n := len(tools()); n != 2 {
		t.Errorf("tools = %d, want 2 (ADR-044 cargo)", n)
	}
}

// TestWriteFormatGitIsRefused is F-22 on this surface: format=git is usage.
func TestWriteFormatGitIsRefused(t *testing.T) {
	root, _ := checkout(t, "a.go", demoFile)
	resp := rawResponse(t, root, "mrw_write", map[string]any{
		"plan":   twoHunkApplyPatch,
		"format": "git",
	})
	e, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("format=git was not a JSON-RPC error: %v", resp)
	}
	msg, _ := e["message"].(string)
	if !strings.Contains(msg, "git patch is not") {
		t.Errorf("git refuse does not name that a git patch is not an apply_patch: %q", msg)
	}
	b, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != demoFile {
		t.Errorf("the file was written:\n%s", b)
	}
}

// TestWriteDeclaresFormatOnTheExistingTool is F-24 / F-25 / F-27 shape:
// optional format on mrw_write, plan default, no third tool, 4096 stays.
func TestWriteDeclaresFormatOnTheExistingTool(t *testing.T) {
	if n := len(tools()); n != 2 {
		t.Fatalf("tools = %d, want 2", n)
	}
	var schema map[string]any
	for _, tl := range tools() {
		if tl.Name != "mrw_write" {
			continue
		}
		schema, _ = tl.InputSchema.(map[string]any)
	}
	if schema == nil {
		t.Fatal("mrw_write declares no input schema")
	}
	props, _ := schema["properties"].(map[string]any)
	f, ok := props["format"].(map[string]any)
	if !ok {
		t.Fatal("mrw_write does not declare format")
	}
	if f["type"] != "string" {
		t.Errorf("format type = %v, want string", f["type"])
	}
	desc := fmt.Sprint(f["description"], f["enum"])
	if !strings.Contains(desc, "apply_patch") {
		t.Errorf("format does not name apply_patch: %v", f)
	}
	if !strings.Contains(desc, "plan") {
		t.Errorf("format does not name plan: %v", f)
	}
	req, _ := schema["required"].([]string)
	for _, r := range req {
		if r == "format" {
			t.Error("format is required; it must default to plan")
		}
	}
	got := instructionsText()
	if !strings.Contains(got, "apply_patch") {
		t.Error("instructions do not name apply_patch, so a host reading only initialize is not told")
	}
	if maxInstructionsChars != 4096 {
		t.Errorf("maxInstructionsChars is %d; ADR-037 forbids raising the bound", maxInstructionsChars)
	}
	if len(got) > maxInstructionsChars {
		t.Errorf("instructions are %d bytes, over the %d-byte bound — shorten what is there rather than raising it", len(got), maxInstructionsChars)
	}
}
