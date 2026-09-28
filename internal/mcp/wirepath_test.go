package mcp

import (
	"reflect"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
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

// TestAWriteReceiptPathIsSpelledWithSlashes is ADR-091's T2.
//
// boundedReceipt serialized the engine's apply.Result, whose paths the engine
// cleaned — backslashed on Windows — while the plan that produced them said
// internal/store/store.go. The helper is driven with `\` for every path field,
// and must leave root (an absolute OS path) and the engine's own result alone;
// the dry run below is the wiring, told apart from doing nothing only on Windows.
func TestAWriteReceiptPathIsSpelledWithSlashes(t *testing.T) {
	in := apply.Result{
		Root:        `C:\work\repo`,
		Hunks:       []apply.HunkResult{{Path: `internal\store\store.go`, Addr: "42-44", Op: "replace", Status: apply.StatusOK}},
		Files:       []apply.FileResult{{Path: `cmd\app\main.go`, RenamedTo: `cmd\app\run.go`, Target: `real\main.go`, Removed: true}},
		DirsCreated: []string{`cmd`, `cmd\app`},
	}
	got := slashResult(in, '\\')
	for name, pair := range map[string][2]string{
		"hunks[0].path":        {got.Hunks[0].Path, "internal/store/store.go"},
		"files[0].path":        {got.Files[0].Path, "cmd/app/main.go"},
		"files[0].renamed_to":  {got.Files[0].RenamedTo, "cmd/app/run.go"},
		"files[0].target":      {got.Files[0].Target, "real/main.go"},
		"dirs_created[1]":      {got.DirsCreated[1], "cmd/app"},
		"root (an OS path)":    {got.Root, `C:\work\repo`},
		"hunks[0].addr (kept)": {got.Hunks[0].Addr, "42-44"},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s = %q, want %q", name, pair[0], pair[1])
		}
	}
	if in.Hunks[0].Path != `internal\store\store.go` || in.Files[0].Target != `real\main.go` || in.DirsCreated[1] != `cmd\app` {
		t.Errorf("slashResult changed the engine's result it was handed: %+v", in)
	}

	root := exampleTree(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"internal/store/store.go"}})))
	rec := structured(t, call(t, root, "mrw_write", map[string]any{
		"plan": "@@ internal/store/store.go 8 replace\nimport \"errors\" // ADR-091\n", "dry_run": true, "ack": acks}))
	hunks, _ := rec["hunks"].([]any)
	files, _ := rec["files"].([]any)
	if len(hunks) != 1 || len(files) != 1 {
		t.Fatalf("the dry run's receipt carries %d hunk(s) and %d file(s), want one of each: %v", len(hunks), len(files), rec)
	}
	for name, v := range map[string]any{"hunks[0].path": hunks[0].(map[string]any)["path"], "files[0].path": files[0].(map[string]any)["path"]} {
		if v != "internal/store/store.go" {
			t.Errorf("the write receipt's %s is %v; want internal/store/store.go, the path the plan named", name, v)
		}
	}
}
