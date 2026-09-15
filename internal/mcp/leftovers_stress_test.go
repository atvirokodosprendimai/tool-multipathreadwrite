package mcp

import (
	"fmt"
	"strings"
	"testing"
)

func TestMcpEnglishWordSpecsNameQuoting(t *testing.T) {
	root, _ := checkout(t, "keep.go", "package keep\n")
	res := call(t, root, "mrw_read", map[string]any{"specs": []string{"rules", "that", "will"}})
	all := fmt.Sprint(res["content"])
	if strings.Count(all, "UNREADABLE") < 2 {
		t.Fatalf("want UNREADABLE specs, got:\n%s", all)
	}
	if !strings.Contains(all, "shell-split") {
		t.Fatalf("MCP specs of English-word fragments must carry the same hint as the CLI:\n%s", all)
	}
	if !strings.Contains(all, "quot") || !strings.Contains(all, "--grep") {
		t.Fatalf("the hint must name quoting and --grep:\n%s", all)
	}
}

func TestMcpOneDottedMissStaysSilent(t *testing.T) {
	root, _ := checkout(t, "keep.go", "package keep\n")
	res := call(t, root, "mrw_read", map[string]any{"specs": []string{"nope.go"}})
	all := fmt.Sprint(res["content"])
	if !strings.Contains(all, "UNREADABLE") {
		t.Fatalf("expected UNREADABLE, got:\n%s", all)
	}
	if strings.Contains(all, "shell-split") {
		t.Fatalf("one dotted miss picked up the English-word hint:\n%s", all)
	}
}

func TestMcpGrepAndAstGrepDisagreeOnNeitherSurface(t *testing.T) {
	root, _ := checkout(t, "a.go", "package a\n")
	res := call(t, root, "mrw_read", map[string]any{"grep": "package", "ast_grep": "fmt.Println"})
	all := fmt.Sprint(res["content"])
	if res["isError"] != true {
		t.Fatalf("both find fields should be refused, got %v", res)
	}
	if !strings.Contains(all, "two sources") {
		t.Errorf("MCP two-sources wording drifted from the CLI:\n%s", all)
	}
}

func TestMcpRangeAndAstGrepAreTwoAnswers(t *testing.T) {
	root, _ := checkout(t, "a.go", "package a\n")
	res := call(t, root, "mrw_read", map[string]any{
		"ast_grep": "fmt.Println",
		"specs":    []string{"a.go:1-2"},
	})
	all := fmt.Sprint(res["content"])
	if res["isError"] != true {
		t.Fatalf("a range plus ast_grep should be refused, got %v", res)
	}
	if !strings.Contains(all, "two answers") {
		t.Errorf("MCP range+ast_grep wording drifted from the CLI:\n%s", all)
	}
}

func TestMcpExcludeWithAstGrepIsNotExcludeWithoutGrep(t *testing.T) {
	root, _ := checkout(t, "a.go", "package a\n")
	t.Setenv("PATH", t.TempDir())
	res := call(t, root, "mrw_read", map[string]any{
		"ast_grep": "fmt.Println",
		"exclude":  []string{"*.go"},
	})
	all := fmt.Sprint(res["content"])
	if strings.Contains(all, "exclude without grep") {
		t.Fatalf("exclude with ast_grep was refused as exclude-without-grep:\n%s", all)
	}
}

func TestInstructionsNameAstGrepWithoutTheCliFlag(t *testing.T) {
	got := instructionsText()
	if strings.Contains(got, "--ast-grep") {
		t.Fatal("instructions mention the CLI flag --ast-grep; that was dropped to keep the 4096-byte bound")
	}
	if !strings.Contains(got, "ast_grep") {
		t.Fatal("instructions never name the MCP ast_grep argument")
	}
	if !strings.Contains(got, "ast-grep") {
		t.Fatal("instructions never name the missing-binary")
	}
	if len(got) > maxInstructionsChars {
		t.Fatalf("instructions are %d bytes, over %d", len(got), maxInstructionsChars)
	}
}
