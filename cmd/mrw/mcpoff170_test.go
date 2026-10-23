package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// ADR-170. `mrw mcp` put two tools in front of every model whose host had it
// registered, and the model took them over the CLI it could have shelled out
// to. Started without --enable or MRW_MCP=on it now validates its startup,
// speaks both protocol eras, lists no tools and refuses every call, naming the
// switch. Through the built binary, since the switch is resolved where the
// subcommand starts the server and an in-process Serve never meets it.
func TestMrwMcpServesNoToolsUntilEnabled(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "mrw.exe") // .exe: Windows exec will not launch an extensionless file
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	tree := t.TempDir()
	if err := os.WriteFile(filepath.Join(tree, "f.txt"), []byte("alpha\nbravo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	state := t.TempDir()

	// start runs the binary with MRW_MCP and CLAUDE_PROJECT_DIR cleared, then
	// MRW_MCP set to mcpEnv when it is non-nil, so an MRW_MCP the test's own
	// environment inherited decides nothing. Windows folds the case of a name.
	drop := func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		for _, d := range []string{"MRW_MCP", "CLAUDE_PROJECT_DIR"} {
			if name == d || (runtime.GOOS == "windows" && strings.EqualFold(name, d)) {
				return true
			}
		}
		return false
	}
	start := func(dir string, mcpEnv *string, stdin string, argv ...string) (stdout, stderr string, code int) {
		t.Helper()
		cmd := exec.Command(bin, argv...)
		cmd.Dir = dir
		for _, kv := range os.Environ() {
			if !drop(kv) {
				cmd.Env = append(cmd.Env, kv)
			}
		}
		cmd.Env = append(cmd.Env, "XDG_STATE_HOME="+state)
		if mcpEnv != nil {
			cmd.Env = append(cmd.Env, "MRW_MCP="+*mcpEnv)
		}
		cmd.Stdin = strings.NewReader(stdin)
		var o, e bytes.Buffer
		cmd.Stdout, cmd.Stderr = &o, &e
		err := cmd.Run()
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		return o.String(), e.String(), code
	}
	// serve sends the requests to `mrw --root tree mcp args...` and returns the
	// answers by id.
	serve := func(mcpEnv *string, args []string, reqs ...string) (map[int]map[string]any, string, string) {
		t.Helper()
		argv := append([]string{"--root", tree, "mcp"}, args...)
		out, errOut, code := start(tree, mcpEnv, strings.Join(reqs, "\n")+"\n", argv...)
		if code != 0 {
			t.Fatalf("mrw %v exits %d\n%s", argv, code, errOut)
		}
		answers := map[int]map[string]any{}
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			var m map[string]any
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				t.Fatalf("not JSON: %q", line)
			}
			answers[int(m["id"].(float64))] = m
		}
		return answers, out, errOut
	}
	on, off, yes, empty := "on", "off", "yes", ""
	padded, newline, blank, upper, offPadded := " on ", "on\n", "  ", "ON", "off "

	const modernMeta = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}`
	initialize := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`
	list := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	ping := `{"jsonrpc":"2.0","id":3,"method":"ping"}`
	call := func(id, name, args string) string {
		return `{"jsonrpc":"2.0","id":` + id + `,"method":"tools/call","params":{"name":"` + name + `","arguments":` + args + `}}`
	}
	toolNames := func(m map[string]any) []any {
		t.Helper()
		result, _ := m["result"].(map[string]any)
		tools, ok := result["tools"].([]any)
		if !ok {
			t.Fatalf("tools/list result.tools is not an array: %v", m)
		}
		return tools
	}
	wantSwitchNamed := func(label string, text string) {
		t.Helper()
		for _, w := range []string{"--enable", "MRW_MCP=on"} {
			if !strings.Contains(text, w) {
				t.Errorf("%s does not name %q: %s", label, w, text)
			}
		}
	}

	t.Run("off, legacy era", func(t *testing.T) {
		seenBefore, _, _ := start(tree, nil, "", "--root", tree, "seen")
		statsBefore, _, _ := start(tree, nil, "", "--root", tree, "stats", "--json")
		a, raw, _ := serve(nil, nil, initialize, list, ping,
			call("4", "mrw_read", `{"specs":["f.txt"]}`),
			call("5", "mrw_write", `{"plan":"@@ new.txt 0 create\nx\n"}`),
			call("6", "mrw_nothing", `{}`))
		res, _ := a[1]["result"].(map[string]any)
		if _, ok := res["capabilities"].(map[string]any)["tools"]; !ok {
			t.Errorf("initialize dropped capabilities.tools: %v", a[1])
		}
		wantSwitchNamed("initialize instructions", res["instructions"].(string))
		if got := toolNames(a[2]); len(got) != 0 {
			t.Errorf("tools/list lists %v", got)
		}
		if !strings.Contains(raw, `"tools":[]`) {
			t.Errorf("tools/list is not an empty JSON array: %s", raw)
		}
		if a[3]["error"] != nil || a[3]["result"] == nil {
			t.Errorf("ping: %v", a[3])
		}
		for _, id := range []int{4, 5, 6} {
			e, _ := a[id]["error"].(map[string]any)
			if e == nil || a[id]["result"] != nil {
				t.Errorf("call %d is not a JSON-RPC error: %v", id, a[id])
				continue
			}
			wantSwitchNamed("the refusal of call "+string(rune('0'+id)), e["message"].(string))
		}
		if _, err := os.Stat(filepath.Join(tree, "new.txt")); err == nil {
			t.Error("a refused mrw_write created new.txt")
		}
		seenAfter, _, _ := start(tree, nil, "", "--root", tree, "seen")
		statsAfter, _, _ := start(tree, nil, "", "--root", tree, "stats", "--json")
		if seenAfter != seenBefore || statsAfter != statsBefore {
			t.Errorf("a refused call changed state:\nseen %q -> %q\nstats %q -> %q", seenBefore, seenAfter, statsBefore, statsAfter)
		}
	})

	t.Run("off, modern era", func(t *testing.T) {
		a, _, _ := serve(nil, nil,
			`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{`+modernMeta+`}}`,
			`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{`+modernMeta+`}}`,
			`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{`+modernMeta+`,"name":"mrw_write","arguments":{"plan":"@@ new.txt 0 create\nx\n"}}}`)
		res, _ := a[1]["result"].(map[string]any)
		if _, ok := res["capabilities"].(map[string]any)["tools"]; !ok {
			t.Errorf("server/discover dropped capabilities.tools: %v", a[1])
		}
		wantSwitchNamed("server/discover instructions", res["instructions"].(string))
		if got := toolNames(a[2]); len(got) != 0 {
			t.Errorf("modern tools/list lists %v", got)
		}
		if e, _ := a[3]["error"].(map[string]any); e == nil {
			t.Errorf("modern mrw_write was not refused: %v", a[3])
		}
		if _, err := os.Stat(filepath.Join(tree, "new.txt")); err == nil {
			t.Error("a refused modern mrw_write created new.txt")
		}
	})

	// snapshot is every file under the state directory with its bytes, so a
	// ledger, a tally or a pending-checkpoint file a call touched all show.
	snapshot := func() map[string]string {
		t.Helper()
		files := map[string]string{}
		err := filepath.WalkDir(state, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, rerr := os.ReadFile(p)
			files[p] = string(b)
			return rerr
		})
		if err != nil {
			t.Fatal(err)
		}
		return files
	}
	t.Run("a refused call changes no state, in either era", func(t *testing.T) {
		// The pair first: an enabled read holds pending spans, so the snapshot
		// is able to see the store a refused read must not touch.
		serve(nil, []string{"--enable"}, call("1", "mrw_read", `{"specs":["f.txt"]}`))
		held := false
		for p := range snapshot() {
			held = held || strings.Contains(filepath.Base(p), "pending")
		}
		if !held {
			t.Fatal("an enabled mrw_read left no pending store for the snapshot to see")
		}
		seenBefore, _, _ := start(tree, nil, "", "--root", tree, "seen")
		statsBefore, _, _ := start(tree, nil, "", "--root", tree, "stats", "--json")
		for _, c := range []struct{ name, args string }{
			{"mrw_read", `{"specs":["f.txt"]}`},
			{"mrw_write", `{"plan":"@@ new.txt 0 create\nx\n"}`},
			{"mrw_nothing", `{}`},
		} {
			for era, req := range map[string]string{
				"legacy": call("1", c.name, c.args),
				"modern": `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{` + modernMeta + `,"name":"` + c.name + `","arguments":` + c.args + `}}`,
			} {
				before := snapshot()
				a, _, _ := serve(nil, nil, req)
				if e, _ := a[1]["error"].(map[string]any); e == nil {
					t.Errorf("%s %s was not refused: %v", era, c.name, a[1])
				}
				if after := snapshot(); !reflect.DeepEqual(before, after) {
					t.Errorf("a refused %s %s changed the state directory", era, c.name)
				}
			}
		}
		seenAfter, _, _ := start(tree, nil, "", "--root", tree, "seen")
		statsAfter, _, _ := start(tree, nil, "", "--root", tree, "stats", "--json")
		if seenAfter != seenBefore || statsAfter != statsBefore {
			t.Errorf("refused calls changed state:\nseen %q -> %q\nstats %q -> %q", seenBefore, seenAfter, statsBefore, statsAfter)
		}
	})

	// Decision 2's table: the flag decides when given, the variable only when not.
	for _, c := range []struct {
		name      string
		args      []string
		env       *string
		wantOn    bool
		wantNamed bool // stderr names MRW_MCP, with its value quoted verbatim
	}{
		{"--enable", []string{"--enable"}, nil, true, false},
		{"--enable beside off", []string{"--enable"}, &off, true, false},
		{"--enable=true beside off", []string{"--enable=true"}, &off, true, false},
		{"--enable=false beside on", []string{"--enable=false"}, &on, false, false},
		{"--enable beside yes names nothing", []string{"--enable"}, &yes, true, false},
		{"--enable=false beside yes names nothing", []string{"--enable=false"}, &yes, false, false},
		{"on", nil, &on, true, false},
		{"off", nil, &off, false, false},
		{"empty", nil, &empty, false, false},
		{"unset", nil, nil, false, false},
		{"yes is off and named", nil, &yes, false, true},
		{"a padded on is off and named", nil, &padded, false, true},
		{"on and a newline is off and named", nil, &newline, false, true},
		{"only blanks is off and named", nil, &blank, false, true},
		{"ON is off and named", nil, &upper, false, true},
		{"a padded off is named", nil, &offPadded, false, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			a, _, errOut := serve(c.env, c.args, list)
			if got := len(toolNames(a[2])) > 0; got != c.wantOn {
				t.Errorf("tools listed: %v, want %v", got, c.wantOn)
			}
			if named := strings.Contains(errOut, "MRW_MCP"); named != c.wantNamed {
				t.Errorf("stderr names MRW_MCP: %v, want %v\n%s", named, c.wantNamed, errOut)
			}
			if c.wantNamed {
				for _, w := range []string{"MRW_MCP=" + strconv.Quote(*c.env), "on", "off"} {
					if !strings.Contains(errOut, w) {
						t.Errorf("stderr lacks %q: %s", w, errOut)
					}
				}
			}
		})
	}

	t.Run("on, both tools and a read", func(t *testing.T) {
		a, _, _ := serve(nil, []string{"--enable"}, list, call("4", "mrw_read", `{"specs":["f.txt"]}`))
		names := ""
		for _, tl := range toolNames(a[2]) {
			names += tl.(map[string]any)["name"].(string) + " "
		}
		if names != "mrw_read mrw_write " {
			t.Errorf("tools listed: %q", names)
		}
		res, _ := a[4]["result"].(map[string]any)
		content, _ := res["content"].([]any)
		if len(content) == 0 || !strings.Contains(content[0].(map[string]any)["text"].(string), "alpha") {
			t.Errorf("the read did not answer: %v", a[4])
		}
	})

	t.Run("startup validation holds with the switch or without", func(t *testing.T) {
		for _, args := range [][]string{{"--root", tree, "mcp", "--max-result-chars", "-1"}, {"--root", tree, "mcp", "--enable", "--max-result-chars", "-1"}} {
			if _, errOut, code := start(tree, nil, "", args...); code != exitUsage {
				t.Errorf("%v: exit %d, want %d\n%s", args, code, exitUsage, errOut)
			}
		}
		// A root nobody named that is the filesystem root is refused (ADR-018).
		fsRoot := filepath.VolumeName(tree) + string(filepath.Separator)
		for _, args := range [][]string{{"mcp"}, {"mcp", "--enable"}} {
			if _, errOut, code := start(fsRoot, nil, "", args...); code != exitUsage {
				t.Errorf("%v from %s: exit %d, want %d\n%s", args, fsRoot, code, exitUsage, errOut)
			}
		}
	})
}
