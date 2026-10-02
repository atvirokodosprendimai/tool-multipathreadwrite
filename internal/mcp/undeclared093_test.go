package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
)

// ADR-093. A tools/call whose arguments carry a key the tool's advertised
// inputSchema does not declare was answered as if the key were absent, so a
// caller that asked for a check, a step or a line cap got an ordinary answer
// that did not do what it asked. The call is now refused by name, before
// anything is served, written or acknowledged.

// declaredArgs is the sorted property names of tool's inputSchema, read from
// tools(): the one list a host is shown and the server enforces.
func declaredArgs(t *testing.T, tool string) []string {
	t.Helper()
	for _, tl := range tools() {
		if tl.Name != tool {
			continue
		}
		schema, _ := tl.InputSchema.(map[string]any)
		props, _ := schema["properties"].(map[string]any)
		names := make([]string, 0, len(props))
		for k := range props {
			names = append(names, k)
		}
		sort.Strings(names)
		return names
	}
	t.Fatalf("tools() lists no %s", tool)
	return nil
}

// quotedList renders names the way the refusal does: each quoted, in the
// order given, comma-separated.
func quotedList(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = fmt.Sprintf("%q", n)
	}
	return strings.Join(q, ", ")
}

func TestAnUndeclaredArgumentIsRefusedByName(t *testing.T) {
	restore := Version
	Version = "v0.0.0-adr093"
	t.Cleanup(func() { Version = restore })

	const plan = `"@@ b.txt 0 create\nX\n"`
	for _, c := range []struct {
		name, tool, args, valid, meta string
		keys                          []string
	}{
		{"no_check", "mrw_write", `{"plan":` + plan + `,"no_check":true}`, `{"plan":` + plan + `}`, "", []string{"no_check"}},
		{"steps", "mrw_write", `{"plan":` + plan + `,"steps":["vet"]}`, `{"plan":` + plan + `}`, "", []string{"steps"}},
		{"then_sh", "mrw_write", `{"plan":` + plan + `,"then_sh":"true"}`, `{"plan":` + plan + `}`, "", []string{"then_sh"}},
		{"force", "mrw_write", `{"plan":` + plan + `,"force":true}`, `{"plan":` + plan + `}`, "", []string{"force"}},
		{"a case variant of plan", "mrw_write", `{"Plan":` + plan + `}`, `{"plan":` + plan + `}`, "", []string{"Plan"}},
		{"context", "mrw_read", `{"specs":["a.txt"],"context":3}`, `{"specs":["a.txt"]}`, "", []string{"context"}},
		{"max_lines", "mrw_read", `{"specs":["a.txt"],"max_lines":1}`, `{"specs":["a.txt"]}`, "", []string{"max_lines"}},
		{"stat", "mrw_read", `{"specs":["a.txt"],"stat":true}`, `{"specs":["a.txt"]}`, "", []string{"stat"}},
		{"_meta inside arguments", "mrw_read", `{"specs":["a.txt"],"_meta":{"progressToken":1}}`, `{"specs":["a.txt"]}`, "", []string{"_meta"}},
		{"two undeclared keys", "mrw_write", `{"plan":` + plan + `,"then_sh":"true","force":true}`, `{"plan":` + plan + `}`, "", []string{"force", "then_sh"}},
		{"the modern era", "mrw_write", `{"plan":` + plan + `,"no_check":true}`, `{"plan":` + plan + `}`, modernMeta, []string{"no_check"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			root, _ := checkout(t, "a.txt", "one\ntwo\n")
			before, err := authoring.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			resp := session(t, root, callReq(c.tool, c.args, c.meta))[0]
			if e, ok := resp["error"]; ok {
				t.Fatalf("answered a JSON-RPC error %v, want an isError result", e)
			}
			res, _ := resp["result"].(map[string]any)
			if res == nil || res["isError"] != true {
				t.Fatalf("an undeclared argument was not refused: %v", resp)
			}
			got := served0(t, res)
			for _, want := range []string{
				quotedList(c.keys),
				quotedList(declaredArgs(t, c.tool)),
				Version,
				"was not recorded",
			} {
				if !strings.Contains(got, want) {
					t.Errorf("the refusal does not name %s:\n%s", want, got)
				}
			}
			if c.meta != "" && res["resultType"] != "complete" {
				t.Errorf("a modern refusal is not decorated for its era: %v", res)
			}
			after, err := authoring.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			if fmt.Sprint(after) != fmt.Sprint(before) {
				t.Errorf("the refusal moved the tally: %v -> %v", before, after)
			}
			if _, err := os.Stat(filepath.Join(root, "b.txt")); err == nil {
				t.Error("a refused write created its file")
			}
			if strings.Contains(got, "-- ck ") {
				t.Errorf("a refused read served lines:\n%s", got)
			}

			// The valid sibling: the same call with the key deleted, or renamed
			// to its declared spelling, is applied or served.
			ok, _ := session(t, root, callReq(c.tool, c.valid, c.meta))[0]["result"].(map[string]any)
			if ok == nil || ok["isError"] == true {
				t.Fatalf("the same call with declared arguments only was refused: %v", ok)
			}
			if c.tool == "mrw_write" {
				if b, err := os.ReadFile(filepath.Join(root, "b.txt")); err != nil || string(b) != "X\n" {
					t.Errorf("the valid write did not create b.txt: %q, %v", b, err)
				}
			} else if !strings.Contains(served0(t, ok), "1| one") {
				t.Errorf("the valid read did not serve a.txt:\n%s", served0(t, ok))
			}
		})
	}

	// An unknown tool is a protocol error whatever keys it carries (ADR-067).
	got := rawCall(t, `{"name":"no_such_tool","arguments":{"x":1}}`)
	if e, ok := got["error"].(map[string]any); !ok || e["code"] != float64(codeInvalidParams) {
		t.Errorf("an unknown tool with an undeclared key answered %v, want -32602", got)
	}
}

// TestARefusedUndeclaredArgumentPromotesNoAck attempts the write an ack would
// license, rather than inspecting the refusal: an ack sent beside an undeclared
// key is not recorded, so the write is refused as unread until the same ack
// arrives in a declared call.
func TestARefusedUndeclaredArgumentPromotesNoAck(t *testing.T) {
	root, _ := checkout(t, "a.txt", "one\ntwo\n")
	if err := os.WriteFile(filepath.Join(root, "c.txt"), []byte("c\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}})))
	if len(acks) == 0 {
		t.Fatal("the read carries no checkpoints")
	}
	// c.txt, not a.txt: were this read served, a fresh serve of a.txt would
	// not be what licenses the write below; only the ack can.
	if res := call(t, root, "mrw_read", map[string]any{"specs": []any{"c.txt"}, "ack": acks, "max_lines": 1}); res["isError"] != true {
		t.Errorf("a read with an undeclared key was not refused: %v", res)
	}
	read := func() string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, "a.txt"))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nONE\n"})
	if got := read(); got != "one\ntwo\n" {
		t.Fatalf("an ack sent beside an undeclared key licensed a write: a.txt is %q", got)
	}
	call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nONE\n", "ack": acks})
	if got := read(); got != "ONE\ntwo\n" {
		t.Errorf("the same ack in a declared call did not license the write: a.txt is %q", got)
	}
}

// TestTheRefusalRoutesOnlyToFlagsTheCLIHas applies ADR-016's two checks to the
// refusal's routing: every flag it names is in that subcommand's --help, and
// none is an argument the tool declares.
func TestTheRefusalRoutesOnlyToFlagsTheCLIHas(t *testing.T) {
	root, _ := checkout(t, "a.txt", "one\n")
	for _, c := range []struct {
		tool, sub string
		args      map[string]any
	}{
		{"mrw_read", "read", map[string]any{"specs": []any{"a.txt"}, "max_lines": 1}},
		// `then_sh` alone (ADR-115 declared `then`), so `check` in the text comes from the routing.
		{"mrw_write", "write", map[string]any{"plan": "@@ b.txt 0 create\nX\n", "then_sh": "true"}},
	} {
		res := call(t, root, c.tool, c.args)
		got := served0(t, res)
		if res["isError"] != true {
			t.Errorf("%s: an undeclared argument was not refused:\n%s", c.tool, got)
			continue
		}
		help := cliHelp(t, c.sub)
		declared := declaredArgs(t, c.tool)
		flags := regexp.MustCompile(`--[a-z][a-z-]+`).FindAllString(got, -1)
		if len(flags) == 0 {
			t.Errorf("%s: the refusal routes to no CLI flag:\n%s", c.tool, got)
		}
		for _, f := range flags {
			if !strings.Contains(help, f) {
				t.Errorf("%s: the refusal names %s, which `mrw %s --help` does not list", c.tool, f, c.sub)
			}
			if arg := strings.ReplaceAll(strings.TrimPrefix(f, "--"), "-", "_"); slices.Contains(declared, arg) {
				t.Errorf("%s: the refusal routes %s to the CLI, but the tool declares %q", c.tool, f, arg)
			}
		}
		if c.tool == "mrw_write" {
			for _, want := range []string{"mrw write", "check", "--then-sh"} {
				if !strings.Contains(got, want) {
					t.Errorf("the mrw_write refusal does not say %q:\n%s", want, got)
				}
			}
		}
	}
}

// decodeTypes maps every tool tools() lists to the type its handler decodes
// its arguments into. It lives in a test file because only this test reads it
// (ADR-088); a tool without an entry fails the test below.
var decodeTypes = map[string]any{
	"mrw_read":  readArgs{},
	"mrw_write": writeArgs{},
}

// TestEveryToolDecodesExactlyTheArgumentsItDeclares binds the two lists ADR-093
// found unbound: a decode type's json names and its tool's schema properties.
// A check keyed on either would silently drop a key the other lacks.
func TestEveryToolDecodesExactlyTheArgumentsItDeclares(t *testing.T) {
	var listed, tabled []string
	for _, tl := range tools() {
		listed = append(listed, tl.Name)
	}
	for name := range decodeTypes {
		tabled = append(tabled, name)
	}
	sort.Strings(listed)
	sort.Strings(tabled)
	if fmt.Sprint(listed) != fmt.Sprint(tabled) {
		t.Errorf("tools() lists %v, and the decode table names %v", listed, tabled)
	}
	for _, name := range listed {
		v, ok := decodeTypes[name]
		if !ok {
			continue
		}
		rt := reflect.TypeOf(v)
		var fields []string
		for i := 0; i < rt.NumField(); i++ {
			tag, _, _ := strings.Cut(rt.Field(i).Tag.Get("json"), ",")
			if tag == "" {
				tag = rt.Field(i).Name // what encoding/json decodes an untagged field from
			}
			fields = append(fields, tag)
		}
		sort.Strings(fields)
		if got, want := fmt.Sprint(fields), fmt.Sprint(declaredArgs(t, name)); got != want {
			t.Errorf("%s decodes %s and declares %s", name, got, want)
		}
	}
}

// TestEveryInputSchemaIsClosed reads tools/list through the wire a host reads:
// every tool's inputSchema says additionalProperties false, so the advertised
// schema describes what the server enforces (ADR-093 T2).
func TestEveryInputSchemaIsClosed(t *testing.T) {
	lines := serve(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	res, _ := decode(t, lines[0])["result"].(map[string]any)
	listed, _ := res["tools"].([]any)
	checked := 0
	for _, raw := range listed {
		tl, _ := raw.(map[string]any)
		schema, _ := tl["inputSchema"].(map[string]any)
		if v, ok := schema["additionalProperties"]; !ok || v != false {
			t.Errorf("%v's inputSchema is not closed: additionalProperties = %v", tl["name"], v)
		}
		checked++
	}
	if checked < 2 {
		t.Errorf("tools/list checked %d tools, want at least 2", checked)
	}
}
