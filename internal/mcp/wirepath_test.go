package mcp

import (
	"reflect"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// TestAReceiptKeyIsSpelledWithSlashes is ADR-091's Enforced-by.
//
// read.Run keys an observation by filepath.Clean, which on Windows is
// internal\store\store.go — while a plan, hunks.path and a grep index all say
// internal/store/store.go. The helper is driven with `\` here so the conversion
// is proved on every platform; the served read below is the wiring, which only
// a Windows run can tell apart from doing nothing (windows-shard 1, #261).
func TestAReceiptKeyIsSpelledWithSlashes(t *testing.T) {
	in := map[string]seen.Observation{
		`internal\store\store.go`: {SHA: "abc", Spans: [][2]int{{40, 60}}},
		`main.go`:                 {SHA: "def"},
	}
	got := slashKeys(in, '\\')
	want := map[string]seen.Observation{
		"internal/store/store.go": in[`internal\store\store.go`],
		"main.go":                 in["main.go"],
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("slashKeys(%v, '\\\\') = %v, want %v", in, got, want)
	}

	root := exampleTree(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{"internal/store/store.go:40-44"}})
	observed, ok := receipt(t, res)["observed"].(map[string]any)
	if !ok {
		t.Fatalf("the read receipt carries no observed map: %v", receipt(t, res))
	}
	if _, ok := observed["internal/store/store.go"]; !ok || len(observed) != 1 {
		t.Errorf("the receipt keys the served file as %v; want exactly internal/store/store.go, the path the spec named", observed)
	}
}
