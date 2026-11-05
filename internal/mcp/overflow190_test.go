package mcp

import "testing"

func TestAQuoteRangeAtTheIntegerLimitIsRefusedOverMCP(t *testing.T) {
	root, _ := checkout(t, `a.go`, demoFile)
	nl := string(rune(10))
	plan := `@@ a.go 1-9223372036854775807 replace anchor=package quote=1.x` + nl + `x` + nl
	res := call(t, root, `mrw_write`, map[string]any{`plan`: plan, `check`: false})
	if res[`isError`] != true {
		t.Fatal(res)
	}
	again := call(t, root, `mrw_read`, map[string]any{`specs`: []any{`a.go:1`}})
	if again[`isError`] == true {
		t.Fatal(again)
	}
}
