package mcp

import (
	"fmt"
	"strings"
	"testing"
)

// TestAstGrepOnMcpIsTheSamePrimitiveAsTheCli: MCP ast_grep is CLI --ast-grep
// (ADR-016). A missing binary names ast-grep on this surface too, and is not
// "needs at least one spec".
func TestAstGrepOnMcpIsTheSamePrimitiveAsTheCli(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	root, _ := checkout(t, "a.go", "package a\n")
	res := call(t, root, "mrw_read", map[string]any{"ast_grep": "fmt.Println"})
	all := fmt.Sprint(res["content"])
	if strings.Contains(all, "needs at least one spec") {
		t.Errorf("ast_grep was ignored as an unknown field:\n%s", all)
	}
	if !strings.Contains(all, "ast-grep") {
		t.Errorf("the MCP refusal does not name ast-grep:\n%s", all)
	}
}

// TestGrepAndAstGrepTogetherOnMcpAreUsage: same sentence as the CLI.
func TestGrepAndAstGrepTogetherOnMcpAreUsage(t *testing.T) {
	root, _ := checkout(t, "a.go", "package a\n")
	res := call(t, root, "mrw_read", map[string]any{"grep": "package", "ast_grep": "fmt.Println"})
	all := fmt.Sprint(res["content"])
	if res["isError"] != true {
		t.Fatalf("both find fields should be refused, got %v", res)
	}
	if !strings.Contains(all, "two sources") {
		t.Errorf("the MCP refusal does not say two sources of specs:\n%s", all)
	}
}
