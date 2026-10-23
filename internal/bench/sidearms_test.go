package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// The tests of ADR-163 T6: the side arms G and C, read against P.

const (
	sideAsset  = "qh-darwin-arm64"
	sideStage  = "qh/1.0.0/qh" // where the staged binary sits in the plugin's data directory
	sideMarker = "qh-go-hooks"
)

// sideRig is the stage6 rig with A, P, W, G and C: the registration a frozen
// tag would give, a stub qh binary standing for the release asset, and the
// SHA256SUMS of that release. The stub CLI logs what each arm's hooks and data
// directory look like when it starts, and keeps a copy of its tree.
type sideRig struct {
	*stage6Rig
	dir, reg, binary, sums string
	sha                    string
	log, snap, staged      string
	n                      int
}

func newSideRig(t *testing.T) *sideRig {
	t.Helper()
	g := checkRig(t)
	// Claude Code reads the config directory by its real path; so does the stage.
	if err := os.MkdirAll(g.opts.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if real, err := filepath.EvalSymlinks(g.opts.ConfigDir); err == nil {
		g.opts.ConfigDir = real
	}
	dir := t.TempDir()
	body := "#!/bin/sh\necho runtime=qh " + sideMarker + "\n"
	s := &sideRig{stage6Rig: g, dir: dir, reg: filepath.Join(dir, "side-arms.json"), binary: filepath.Join(dir, "download", "asset"),
		sums: filepath.Join(dir, "download", "SHA256SUMS.txt"), sha: hexSHA([]byte(body)),
		log: filepath.Join(g.work, "plugin.log"), snap: filepath.Join(g.work, "snap"), staged: filepath.Join(g.work, "staged")}
	put(t, s.binary, body, 0o755)
	put(t, s.sums, strings.Repeat("0", 64)+"  qh-linux-amd64\n"+s.sha+"  "+sideAsset+"\n", 0o644)
	s.register(t, nil)
	s.opts.Configs = []string{"A", "P", "W", "G", "C"}
	s.opts.SideArms, s.opts.QhBinary, s.opts.QhSums = s.reg, s.binary, s.sums
	if err := resolveProfile(&s.opts); err != nil {
		t.Fatal(err)
	}
	s.setStub(true)
	return s
}

// setStub installs the stand-in CLI. Its plugin data directory is the one
// Claude Code derives for a hook, <config dir>/plugins/data/<plugin id>, and
// it ignores whatever CLAUDE_PLUGIN_DATA the environment holds, as the real one
// overwrites it; write says whether its hooks leave state there.
func (s *sideRig) setStub(write bool) {
	pre := "D=\"$CLAUDE_CONFIG_DIR/plugins/data/quality-harness-inline\"; { echo \"data=$D\"; echo \"inherited=${CLAUDE_PLUGIN_DATA-unset}\"; " +
		"echo \"listing=$(cd \"$D\" && find . -type f | sort | tr '\\n' ' ')\"; " +
		"echo \"env=$(env | grep -E '^(QH_|QUALITY_HARNESS_)' | sort | tr '\\n' ' ')\"; " +
		"echo \"base=$(git show HEAD:.quality-harness.json)\"; echo \"status=$(git status --porcelain)\"; " +
		"if [ \"$QH_HOOK_RUNTIME\" = go ]; then \"$D/" + sideStage + "\"; cp \"$D/" + sideStage + "\" " + shellQuote(s.staged) + "; else node -c 'echo runtime=node'; fi; "
	if write {
		pre += "echo state > \"$D/state.txt\"; "
	}
	pre += "rm -rf " + shellQuote(s.snap) + "; cp -R . " + shellQuote(s.snap) + "; } >> " + shellQuote(s.log) + " 2>&1"
	s.claude(pre)
}

// asides are the data directories moved out of Claude Code's way, kept.
func (s *sideRig) asides() []string {
	m, _ := filepath.Glob(filepath.Join(s.opts.Out, "plugin-data-aside", "*"))
	return m
}

// register writes the registration the frozen tag would be given in BACKLOG,
// with edit applied to it.
func (s *sideRig) register(t *testing.T, edit func(m map[string]any)) {
	t.Helper()
	m := map[string]any{
		"tag": "v1.0.0",
		"c":   map[string]any{"key": "turnEnd", "value": "notify"},
		"g": map[string]any{"asset": sideAsset, "sha256": s.sha, "stage_as": sideStage, "marker": sideMarker,
			"runtime": map[string]any{"QH_HOOK_RUNTIME": "go"}},
	}
	if edit != nil {
		edit(m)
	}
	put(t, s.reg, mustJSON(t, m), 0o644)
}

// runArm runs one attempt of the configuration and returns its receipt, what
// the stub CLI logged as it started, and a copy of the tree it started in.
func (s *sideRig) runArm(name string) (r Receipt, log, tree string) {
	t := s.t
	t.Helper()
	for _, p := range []string{s.log, s.snap, s.staged} {
		if err := os.RemoveAll(p); err != nil {
			t.Fatal(err)
		}
	}
	s.n++
	a := attempt{ID: fmt.Sprintf("arm.%s.%d", name, s.n), Task: s.opts.Tasks[0], Model: "stub-model", Config: Configs[name], Repeat: 1}
	r = runAttempt(s.opts, a, 0, "9.9.9", "")
	if r.Exclusion != "" {
		t.Fatalf("%s: %s %s", name, r.Exclusion, r.Note)
	}
	b, _ := os.ReadFile(s.log)
	tree = filepath.Join(t.TempDir(), "tree")
	if err := os.Rename(s.snap, tree); err != nil {
		t.Fatal(err)
	}
	return r, string(b), tree
}

// refuses says the options are refused, by gates or by what resolving reads,
// with a message holding want.
func (s *sideRig) refuses(name, want string, mutate func(*RunOptions)) {
	s.t.Helper()
	x := s.opts
	mutate(&x)
	_, err := gates(x)
	if err == nil {
		err = resolveProfile(&x)
	}
	switch {
	case err == nil:
		s.t.Errorf("%s was accepted", name)
	case !strings.Contains(err.Error(), want):
		s.t.Errorf("%s: %q, want a refusal holding %q", name, err, want)
	}
}

// editing replaces a file's body and returns what puts it back.
func editing(t *testing.T, path, body string) (restore func()) {
	t.Helper()
	old, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	put(t, path, body, 0o755)
	return func() { put(t, path, string(old), 0o755) }
}

// treeFiles are the files of a tree, by relative path, without .git.
func treeFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.Type().IsRegular() {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, p)
			out[rel] = string(b)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// invoked is the arguments, environment and settings file an arm's attempt
// gets, with its scratch directory scrubbed out.
func (s *sideRig) invoked(name string) (args, env []string, settings, scratch string) {
	t := s.t
	t.Helper()
	scratch = t.TempDir()
	a, e, err := invocation(s.opts, attempt{ID: "x." + name, Task: s.opts.Tasks[0], Model: "m", Config: Configs[name], Repeat: 1}, scratch)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(scratch, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	return scrub(a, scratch), scrub(e, scratch), strings.ReplaceAll(string(b), scratch, "<scratch>"), scratch
}

func envKeys(env []string) map[string]string {
	m := map[string]string{}
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		m[k] = v
	}
	return m
}

// hookTranscript is a session with one hook_response, as a hook that succeeded
// prints it; fields override its keys, and a nil value removes one.
func hookTranscript(fields map[string]any) []byte {
	m := map[string]any{"type": "system", "subtype": "hook_response", "hook_event": "SessionStart", "hook_name": "SessionStart:startup", "hook_id": "h1", "outcome": "success", "exit_code": 0}
	for k, v := range fields {
		if v == nil {
			delete(m, k)
		} else {
			m[k] = v
		}
	}
	return []byte(`{"type":"system","subtype":"init","tools":["Bash"],"mcp_servers":[]}` + "\n" + mustJSONLine(m))
}

func mustJSONLine(v any) string {
	b, _ := json.Marshal(v)
	return string(b) + "\n"
}

// TestConfigurationGRunsTheHooksThroughTheBinaryAndPThroughNode: G's
// arguments, settings and environment are P's plus the registered runtime
// setting and the fetch switch, and its data directory holds the staged
// binary; P's hooks run under Node and G's under that binary; every plugin
// arm's data directory is fresh; a binary that is not its SHA256SUMS line, G
// without P or without its registration is refused (ADR-163 Decision 9).
func TestConfigurationGRunsTheHooksThroughTheBinaryAndPThroughNode(t *testing.T) {
	s := newSideRig(t)

	t.Run("G is P plus the runtime setting, the fetch switch and the binary", func(t *testing.T) {
		pArgs, pEnv, pSettings, _ := s.invoked("P")
		// Claude Code owns CLAUDE_PLUGIN_DATA: <config dir>/plugins/data/<plugin id>. P's is fresh and empty.
		data := pluginDataDir(s.opts)
		if ents, err := os.ReadDir(data); err != nil || len(ents) != 0 {
			t.Errorf("P's plugin data directory %s: %v %v, want empty", data, ents, err)
		}
		put(t, filepath.Join(data, "leak.txt"), "a hook's state\n", 0o644)
		asides := len(s.asides())
		gArgs, gEnv, gSettings, _ := s.invoked("G")
		if !slices.Equal(gArgs, pArgs) || gSettings != pSettings {
			t.Errorf("G's arguments or settings are not P's:\nP %v %s\nG %v %s", pArgs, pSettings, gArgs, gSettings)
		}
		p, g := envKeys(pEnv), envKeys(gEnv)
		var extra []string
		for k, v := range g {
			if pv, ok := p[k]; !ok {
				extra = append(extra, k+"="+v)
			} else if pv != v {
				t.Errorf("G's %s is %q, P's is %q", k, v, pv)
			}
		}
		slices.Sort(extra)
		if want := []string{"QH_HOOK_RUNTIME=go", "QUALITY_HARNESS_QH_FETCH=off"}; !slices.Equal(extra, want) {
			t.Errorf("G's environment adds %v to P's, want %v", extra, want)
		}
		for k := range p {
			if _, ok := g[k]; !ok {
				t.Errorf("P's %s is missing from G's environment", k)
			}
		}
		if _, ok := p["QUALITY_HARNESS_QH_FETCH"]; ok {
			t.Error("P has the fetch switch: it is G's alone")
		}
		// G's data directory is fresh too: P's state is moved aside and kept, and the verified bytes alone are in it.
		if _, err := os.Stat(filepath.Join(data, "leak.txt")); err == nil {
			t.Error("the earlier attempt's state is in G's plugin data directory")
		}
		if kept, _ := filepath.Glob(filepath.Join(s.opts.Out, "plugin-data-aside", "*", "data", "leak.txt")); len(kept) != 1 || len(s.asides()) != asides+1 {
			t.Errorf("the earlier attempt's data directory was not moved aside and kept: %v", s.asides())
		}
		staged, err := os.ReadFile(filepath.Join(data, filepath.FromSlash(sideStage)))
		if err != nil || hexSHA(staged) != s.sha {
			t.Errorf("G's staged binary: %v, sha %s, want %s", err, hexSHA(staged), s.sha)
		}
		if fi, err := os.Stat(filepath.Join(data, filepath.FromSlash(sideStage))); err != nil || fi.Mode()&0o111 == 0 {
			t.Errorf("G's staged binary is not executable: %v %v", fi, err)
		}
		for _, env := range [][]string{pEnv, gEnv} {
			if _, ok := envKeys(env)["CLAUDE_PLUGIN_DATA"]; ok {
				t.Error("bench sets CLAUDE_PLUGIN_DATA, which Claude Code overwrites for a hook")
			}
		}
	})

	t.Run("P's hooks ran under Node and G's under the staged binary", func(t *testing.T) {
		_, pLog, _ := s.runArm("P")
		if !strings.Contains(pLog, "runtime=node") || strings.Contains(pLog, sideMarker) || strings.Contains(pLog, "QUALITY_HARNESS_QH_FETCH") {
			t.Errorf("P's hooks:\n%s", pLog)
		}
		r, gLog, _ := s.runArm("G")
		if !strings.Contains(gLog, "runtime=qh "+sideMarker) || strings.Contains(gLog, "runtime=node") ||
			!strings.Contains(gLog, "QH_HOOK_RUNTIME=go") || !strings.Contains(gLog, "QUALITY_HARNESS_QH_FETCH=off") {
			t.Errorf("G's hooks:\n%s", gLog)
		}
		staged, err := os.ReadFile(s.staged)
		if err != nil || hexSHA(staged) != s.sha {
			t.Errorf("the binary G's hook ran: %v, sha %s, want the verified %s", err, hexSHA(staged), s.sha)
		}
		if r.Plugin == nil || *r.Plugin != *s.opts.plugin || r.Side == nil || r.Side.Tag != "v1.0.0" || r.Side.AssetSHA256 != s.sha || r.Side.Setting["QH_HOOK_RUNTIME"] != "go" {
			t.Errorf("G's receipt names plugin %v and side %+v", r.Plugin, r.Side)
		}
		if pr, _, _ := s.runArm("P"); pr.Side != nil {
			t.Errorf("P's receipt carries side %+v", pr.Side)
		}
	})

	t.Run("every plugin arm gets a fresh data directory where Claude Code puts it", func(t *testing.T) {
		want := pluginDataDir(s.opts)
		if !strings.HasPrefix(want, filepath.Join(s.opts.ConfigDir, "plugins", "data")+string(filepath.Separator)) {
			t.Fatalf("the data directory %s is not under the config directory's plugins/data", want)
		}
		for _, name := range []string{"P", "P", "W", "C", "G", "G"} {
			before := len(s.asides())
			_, log, _ := s.runArm(name)
			if !strings.Contains(log, "data="+want+"\n") || !strings.Contains(log, "inherited=unset\n") {
				t.Errorf("%s's hooks saw another data directory, or bench passed one:\n%s", name, log)
			}
			wantList := "listing=\n"
			if name == "G" {
				wantList = "listing=./" + sideStage + " \n"
			}
			if !strings.Contains(log, wantList) {
				t.Errorf("%s started with data directory content other than %q (the last attempt's state.txt leaked, or the binary is missing):\n%s", name, wantList, log)
			}
			if len(s.asides()) != before+1 {
				t.Errorf("%s: the earlier attempt's data directory was not moved aside and kept: %v", name, s.asides())
			}
		}
		if kept, _ := filepath.Glob(filepath.Join(s.opts.Out, "plugin-data-aside", "*", "data", "state.txt")); len(kept) < 6 {
			t.Errorf("the state the hooks wrote is not kept: %v", kept)
		}
	})

	t.Run("the canary attests the runtime only from the output of a hook that succeeded", func(t *testing.T) {
		m := sideMarker
		for _, c := range []struct {
			name, config string
			fields       map[string]any
			clean        bool
		}{
			{"marker in output", "G", map[string]any{"output": "ok " + m}, true},
			{"marker in stdout", "G", map[string]any{"stdout": m}, true},
			{"no marker", "G", map[string]any{"output": "ok"}, false},
			{"marker only in the hook's name", "G", map[string]any{"output": "ok", "hook_name": m}, false},
			{"marker only in the hook's id", "G", map[string]any{"output": "ok", "hook_id": m}, false},
			{"marker only in stderr", "G", map[string]any{"output": "ok", "stderr": m}, false},
			{"marker only in a diagnostic", "G", map[string]any{"output": "ok", "message": m}, false},
			{"a hook that failed", "G", map[string]any{"output": m, "outcome": "error"}, false},
			{"a hook that exited non-zero", "G", map[string]any{"output": m, "exit_code": 1}, false},
			{"a hook with no outcome", "G", map[string]any{"output": m, "outcome": nil}, false},
			{"P without the marker", "P", map[string]any{"output": "ok"}, true},
			{"P, marker in output", "P", map[string]any{"output": m}, false},
			{"P, marker in a failed hook's output", "P", map[string]any{"output": m, "outcome": "error"}, false},
			{"P, marker only in a name", "P", map[string]any{"output": "ok", "hook_name": m}, true},
			{"W, marker in output", "W", map[string]any{"output": m}, false},
			{"C, marker in output", "C", map[string]any{"output": m}, false},
			{"C without the marker", "C", map[string]any{"output": "ok"}, true},
			{"A", "A", map[string]any{"output": "ok"}, true},
		} {
			if p := CheckHookRuntime(Configs[c.config], hookTranscript(c.fields), sideMarker); (len(p) == 0) != c.clean {
				t.Errorf("%s: %v", c.name, p)
			}
		}
		if p := CheckHookRuntime(Configs["G"], []byte(`{"type":"system","subtype":"init"}`+"\n"), sideMarker); len(p) == 0 {
			t.Error("G with no hook output passed the runtime check")
		}
	})

	t.Run("the canary records the side arms and asks the hooks' runtime", func(t *testing.T) {
		o := s.opts
		o.Out = filepath.Join(s.work, "canary-out")
		c, err := RunCanary(o, o.Tasks[0], "")
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(c.Configs, []string{"A", "P", "W"}) || !slices.Equal(c.SideConfigs, []string{"G", "C"}) {
			t.Errorf("the canary names configs %v and side arms %v, want A, P, W and G, C apart", c.Configs, c.SideConfigs)
		}
		has := func(arm, sub string) bool {
			return slices.ContainsFunc(c.Problems[arm], func(p string) bool { return strings.Contains(p, sub) })
		}
		// The stub CLI prints no hook output: G's hooks cannot have run under the binary, and P's, W's and C's carry no marker.
		if !has("G", "staged binary") {
			t.Errorf("G's canary problems: %v, want the runtime question asked", c.Problems["G"])
		}
		for _, arm := range []string{"P", "W", "C"} {
			if has(arm, sideMarker) || has(arm, "did not run cleanly") {
				t.Errorf("%s's canary problems: %v", arm, c.Problems[arm])
			}
		}
		var onDisk struct {
			Configs     []string `json:"configs"`
			SideConfigs []string `json:"side_configs"`
		}
		if err := readJSON(filepath.Join(o.Out, "canary.json"), &onDisk); err != nil || !slices.Equal(onDisk.SideConfigs, []string{"G", "C"}) || slices.Contains(onDisk.Configs, "G") {
			t.Errorf("canary.json: %v %+v", err, onDisk)
		}
		// Its attempts are in side/, as a stage's are.
		if _, err := os.Stat(filepath.Join(o.Out, sideDir, "canary.G.transcript.jsonl")); err != nil {
			t.Errorf("G's canary transcript is not beside the core's, in side/: %v", err)
		}
		if c.PluginData != pluginDataDir(o) || c.SideRegistration == nil || !reflect.DeepEqual(*c.SideRegistration, *s.opts.side) || c.SideAsset != s.sha {
			t.Errorf("the canary records data directory %q, side registration %+v and asset %q, want %s, %+v and %s", c.PluginData, c.SideRegistration, c.SideAsset, pluginDataDir(o), *s.opts.side, s.sha)
		}
		for _, arm := range []string{"A", "P", "W", "G", "C"} {
			if has(arm, "plugin data") {
				t.Errorf("%s's canary problems: %v", arm, c.Problems[arm])
			}
		}
		// Hooks that leave nothing in the directory Claude Code owns have not been seen to use it.
		s.setStub(false)
		o.Out = filepath.Join(s.work, "canary-out-silent")
		c, err = RunCanary(o, o.Tasks[0], "")
		s.setStub(true)
		if err != nil {
			t.Fatal(err)
		}
		for _, arm := range []string{"P", "W", "G", "C"} {
			if !has(arm, "plugin data") {
				t.Errorf("%s left nothing in the plugin data directory and the canary did not say so: %v", arm, c.Problems[arm])
			}
		}
	})

	t.Run("refusals", func(t *testing.T) {
		if _, err := gates(s.opts); err != nil {
			t.Fatalf("the rig's own options were refused: %v", err)
		}
		s.refuses("G without P", "beside P", func(x *RunOptions) { x.Configs = []string{"A", "G"} })
		s.refuses("G beside W alone", "beside P", func(x *RunOptions) { x.Configs = []string{"A", "W", "G"}; x.SideArms = s.reg })
		// W is stage6-only too, so the rig's W is dropped: the refusal must be the side arm's own.
		s.refuses("G under stage5", "side arm(s) [G] run only under --profile stage6", func(x *RunOptions) { x.Profile = ProfileStage5; x.Configs = []string{"A", "P", "G"} })
		s.refuses("G under no profile", "side arm(s) [G] run only under --profile stage6", func(x *RunOptions) { x.Profile = ""; x.Configs = []string{"A", "P", "G"} })
		s.refuses("G without --side-arms", "side-arms", func(x *RunOptions) { x.SideArms = "" })
		s.refuses("G without --qh-binary", "qh-binary", func(x *RunOptions) { x.QhBinary = "" })
		s.refuses("G without --qh-sums", "qh-sums", func(x *RunOptions) { x.QhSums = "" })
		s.refuses("--qh-binary with no G", "--qh-binary", func(x *RunOptions) { x.Configs = []string{"A", "P", "C"} })
		s.refuses("G named in --configs-for", "configs-for", func(x *RunOptions) { x.ConfigsFor = map[string][]string{"stub-model": {"A", "P", "G"}} })
		s.refuses("a registration without G", "g", func(x *RunOptions) {
			s.register(t, func(m map[string]any) { delete(m, "g") })
			x.SideArms = s.reg
		})
		s.register(t, nil)
		for name, edit := range map[string]func(map[string]any){
			"a registration at another tag": func(m map[string]any) { m["tag"] = "v9.9.9" },
			"a registration with no tag":    func(m map[string]any) { delete(m, "tag") },
			"a registered sha that is not the release's": func(m map[string]any) {
				m["g"].(map[string]any)["sha256"] = strings.Repeat("1", 64)
			},
			"a stage path above the data directory": func(m map[string]any) { m["g"].(map[string]any)["stage_as"] = "../qh" },
			"an absolute stage path":                func(m map[string]any) { m["g"].(map[string]any)["stage_as"] = "/tmp/qh" },
			"no runtime setting":                    func(m map[string]any) { m["g"].(map[string]any)["runtime"] = map[string]any{} },
			"a runtime setting that is PATH":        func(m map[string]any) { m["g"].(map[string]any)["runtime"] = map[string]any{"PATH": "x"} },
			"no marker":                             func(m map[string]any) { m["g"].(map[string]any)["marker"] = "" },
			"an unknown key":                        func(m map[string]any) { m["g"].(map[string]any)["fetch"] = "on" },
		} {
			s.register(t, edit)
			s.refuses(name, "", func(*RunOptions) {})
			s.register(t, nil)
		}

		// The registration is exactly one JSON document, read the same way by run and canary.
		goodReg, _ := os.ReadFile(s.reg)
		for name, body := range map[string]string{"trailing garbage": string(goodReg) + "garbage\n", "a second document": string(goodReg) + string(goodReg), "a stray brace": string(goodReg) + "}"} {
			restore := editing(t, s.reg, body)
			s.refuses(name, "", func(*RunOptions) {})
			if _, err := SideNamed(s.reg); err == nil {
				t.Errorf("SideNamed read a registration with %s", name)
			}
			restore()
		}
		s.register(t, func(m map[string]any) { m["unknown"] = 1 })
		if _, err := SideNamed(s.reg); err == nil {
			t.Error("SideNamed read a registration with an unknown field")
		}
		s.register(t, nil)
		restoreReg := editing(t, s.reg, string(goodReg)+"\n\n  \n")
		xr := s.opts
		if err := resolveProfile(&xr); err != nil {
			t.Errorf("trailing white space was refused: %v", err)
		}
		if arms, err := SideNamed(s.reg); err != nil || !slices.Equal(arms, []string{"C", "G"}) {
			t.Errorf("SideNamed over a registration with trailing white space: %v %v", arms, err)
		}
		restoreReg()
		// The asset is the release's: its bytes, its SHA256SUMS line, its registered sha.
		bin, sums := s.binary, s.sums
		goodSums, _ := os.ReadFile(sums)
		for name, c := range map[string]struct {
			path, body, want string
		}{
			"a binary that is not its SHA256SUMS line": {bin, "#!/bin/sh\necho tampered\n", "SHA256SUMS"},
			"a SHA256SUMS line not the registered sha": {sums, strings.Repeat("3", 64) + "  " + sideAsset + "\n", "not its SHA256SUMS line"},
			"an asset missing from SHA256SUMS":         {sums, strings.Repeat("0", 64) + "  qh-linux-amd64\n", sideAsset},
			"a malformed SHA256SUMS":                   {sums, string(goodSums) + "this is not a sum\n", "malformed"},
			"a short hash":                             {sums, "abc  " + sideAsset + "\n", "malformed"},
			"an empty SHA256SUMS":                      {sums, "", sideAsset},
			"two lines for the asset":                  {sums, string(goodSums) + strings.Repeat("2", 64) + "  " + sideAsset + "\n", "twice"},
			"a line that is another asset's":           {sums, s.sha + "  qh-linux-amd64\n", sideAsset},
		} {
			restore := editing(t, c.path, c.body)
			s.refuses(name, c.want, func(*RunOptions) {})
			restore()
		}
		// The passing sibling of every refusal above: the untouched rig resolves.
		x := s.opts
		if err := resolveProfile(&x); err != nil {
			t.Errorf("the untouched release was refused: %v", err)
		}
		// A binary-mode line (` *name`) names the asset too.
		restore := editing(t, sums, s.sha+" *"+sideAsset+"\n")
		x = s.opts
		if err := resolveProfile(&x); err != nil {
			t.Errorf("a binary-mode SHA256SUMS line was refused: %v", err)
		}
		restore()
	})
}

// TestConfigurationCDiffersFromPOnlyInTurnEnd: C's tree is P's but for the
// turnEnd key of .quality-harness.json, in the working tree and in the base
// commit; its arguments, settings and environment are P's; A, P and W trees
// are unchanged by C; C without P, without its registration or outside stage6
// is refused (ADR-163 Decision 9).
func TestConfigurationCDiffersFromPOnlyInTurnEnd(t *testing.T) {
	s := newSideRig(t)
	trees := map[string]map[string]string{}
	logs := map[string]string{}
	for _, name := range []string{"A", "P", "W", "G", "C"} {
		_, log, tree := s.runArm(name)
		trees[name], logs[name] = treeFiles(t, tree), log
	}
	const declared = `{"check":"pytest"}` + "\n"
	for _, name := range []string{"A", "P", "W", "G"} {
		if trees[name][".quality-harness.json"] != declared {
			t.Errorf("%s's check file is %q, want exactly the declared check", name, trees[name][".quality-harness.json"])
		}
		if !maps.Equal(trees[name], trees["P"]) {
			t.Errorf("%s's tree differs from P's", name)
		}
		if !strings.Contains(logs[name], "base="+strings.TrimSpace(declared)+"\n") || !strings.Contains(logs[name], "status=\n") {
			t.Errorf("%s started with a base commit or status other than the declared check and a clean tree:\n%s", name, logs[name])
		}
	}
	var differs []string
	for f, c := range trees["C"] {
		if trees["P"][f] != c {
			differs = append(differs, f)
		}
	}
	for f := range trees["P"] {
		if _, ok := trees["C"][f]; !ok {
			differs = append(differs, f)
		}
	}
	if !slices.Equal(differs, []string{".quality-harness.json"}) {
		t.Errorf("C's tree differs from P's in %v, want the check file alone", differs)
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(trees["C"][".quality-harness.json"]), &got); err != nil || len(got) != 2 || got["check"] != "pytest" || got["turnEnd"] != "notify" {
		t.Errorf("C's check file is %q: %v, want P's check and turnEnd notify", trees["C"][".quality-harness.json"], err)
	}
	if !strings.Contains(logs["C"], `"turnEnd":"notify"`) || !strings.Contains(logs["C"], "status=\n") {
		t.Errorf("C's base commit does not hold turnEnd, or its tree is dirty:\n%s", logs["C"])
	}

	pArgs, pEnv, pSettings, _ := s.invoked("P")
	cArgs, cEnv, cSettings, _ := s.invoked("C")
	if !slices.Equal(cArgs, pArgs) || !slices.Equal(cEnv, pEnv) || cSettings != pSettings {
		t.Errorf("C's arguments, environment or settings are not P's:\nP %v\nC %v", pEnv, cEnv)
	}

	// A task with no declared check still carries C's setting; no other arm gets a file from this.
	for _, c := range []struct {
		extra map[string]string
		want  string
	}{{map[string]string{"turnEnd": "notify"}, `{"turnEnd":"notify"}` + "\n"}, {nil, ""}} {
		dir := t.TempDir()
		if _, err := materializeArm(s.opts.Tasks[0], dir, nil, "", c.extra); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(dir, ".quality-harness.json"))
		if (err == nil) != (c.want != "") || string(b) != c.want {
			t.Errorf("a task with no declared check and extra keys %v gets the file %q (%v), want %q", c.extra, b, err, c.want)
		}
	}

	// The canary's tree check, with the setting and without it.
	reg := s.opts.side
	for _, c := range []struct {
		config string
		extra  map[string]string
		clean  bool
	}{{"C", map[string]string{"turnEnd": "notify"}, true}, {"C", nil, false}, {"P", nil, true}, {"P", map[string]string{"turnEnd": "notify"}, false},
		{"W", map[string]string{"turnEnd": "notify"}, false}, {"G", map[string]string{"turnEnd": "notify"}, false}, {"A", map[string]string{"turnEnd": "notify"}, false},
		{"C", map[string]string{"turnEnd": "continue"}, false}} {
		dir := t.TempDir()
		if err := writeCheckFile(dir, "pytest", c.extra); err != nil {
			t.Fatal(err)
		}
		if p := CheckSideTree(Configs[c.config], dir, reg); (len(p) == 0) != c.clean {
			t.Errorf("%s on a check file with %v: %v", c.config, c.extra, p)
		}
	}

	t.Run("refusals", func(t *testing.T) {
		s.refuses("C without P", "beside P", func(x *RunOptions) { x.Configs = []string{"A", "C"} })
		s.refuses("C under stage5", "side arm(s) [C] run only under --profile stage6", func(x *RunOptions) { x.Profile = ProfileStage5; x.Configs = []string{"A", "P", "C"} })
		s.refuses("C under no profile", "side arm(s) [C] run only under --profile stage6", func(x *RunOptions) { x.Profile = ""; x.Configs = []string{"A", "P", "C"} })
		s.refuses("C without --side-arms", "side-arms", func(x *RunOptions) {
			x.Configs = []string{"A", "P", "C"}
			x.SideArms = ""
			x.QhBinary, x.QhSums = "", ""
		})
		s.refuses("C named in --configs-for", "configs-for", func(x *RunOptions) { x.ConfigsFor = map[string][]string{"stub-model": {"A", "P", "C"}} })
		for name, edit := range map[string]func(map[string]any){
			"a registration without C":             func(m map[string]any) { delete(m, "c") },
			"a registration at another tag":        func(m map[string]any) { m["tag"] = "v9.9.9" },
			"a setting with no value":              func(m map[string]any) { m["c"].(map[string]any)["value"] = "" },
			"a setting with no key":                func(m map[string]any) { m["c"].(map[string]any)["key"] = "" },
			"a setting that is the declared check": func(m map[string]any) { m["c"].(map[string]any)["key"] = "check" },
		} {
			s.register(t, edit)
			s.refuses(name, "", func(x *RunOptions) { x.Configs = []string{"A", "P", "C"}; x.QhBinary, x.QhSums = "", "" })
			s.register(t, nil)
		}
		// A tag's own spelling is the registration's: C writes it, whatever it is.
		s.register(t, func(m map[string]any) { m["c"] = map[string]any{"key": "turn_end", "value": "quiet"} })
		x := s.opts
		if err := resolveProfile(&x); err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		if err := writeCheckFile(dir, "pytest", map[string]string{x.side.C.Key: x.side.C.Value}); err != nil {
			t.Fatal(err)
		}
		if b, _ := os.ReadFile(filepath.Join(dir, ".quality-harness.json")); string(b) != `{"check":"pytest","turn_end":"quiet"}`+"\n" {
			t.Errorf("C writes %q", b)
		}
		s.register(t, nil)
	})
}

// TestCoreExportsAreIdenticalWithOrWithoutSideArms: a stage that ran G and C
// beside A, P and W writes a run.json whose configs and configs_for are the
// core's and whose side arms are in their own fields, keeps their receipts out
// of the core receipts, and exports --pair A,P and --pair A,W byte for byte
// as the same stage without them; blocks with and without side arms export as
// one; --pair with a side arm is refused; a side arm cannot join after block 1
// (ADR-163 Decision 9).
func TestCoreExportsAreIdenticalWithOrWithoutSideArms(t *testing.T) {
	s := newSideRig(t)
	canaryFor := func(sideToo bool) string {
		c := map[string]any{"pass": true, "cli_version": "9.9.9", "profile": "stage6", "configs": []string{"A", "P", "W"}, "plugin": s.opts.plugin,
			"plugin_data": pluginDataDir(s.opts), "total_cost_usd": map[string]any{"A": 0.1, "P": 0.2, "W": 0.3}}
		if sideToo {
			c["side_configs"] = []string{"G", "C"}
			c["side_registration"], c["side_asset_sha256"] = s.opts.side, s.sha
			c["total_cost_usd"] = map[string]any{"A": 0.1, "P": 0.2, "W": 0.3, "G": 0.4, "C": 0.5}
		}
		return mustJSON(t, c)
	}
	exportCanary := func(sideToo bool) string {
		c := map[string]any{"pass": true, "configs": []string{"A", "P", "W"}, "plugin_data": pluginDataDir(s.opts), "total_cost_usd": map[string]any{"A": 0.1, "P": 0.2, "W": 0.3}}
		if sideToo { // a canary file as it stood before the core projection: side arms inline
			c["configs"] = []string{"A", "P", "W", "G", "C"}
			c["side_configs"] = []string{"G", "C"}
			c["side_registration"], c["side_asset_sha256"] = s.opts.side, s.sha
			c["total_cost_usd"] = map[string]any{"A": 0.1, "P": 0.2, "W": 0.3, "G": 0.4, "C": 0.5}
		}
		p := filepath.Join(t.TempDir(), "canary.json")
		put(t, p, mustJSON(t, c), 0o644)
		return p
	}
	selection := filepath.Join(t.TempDir(), "selection.json")
	put(t, selection, `{"rule":"the 1 ids with the lowest sha256(\"adr-163:\" + instance_id)","dataset":"SWE-bench/SWE-bench_Verified","revision":"78f4","split":"test","selected":["`+s.opts.Tasks[0].ID+`"]}`, 0o644)
	runStage := func(out string, configs []string, sideToo bool) {
		o := s.opts
		o.Out, o.Configs, o.StatusLine = out, configs, true
		if !sideToo {
			o.SideArms, o.QhBinary, o.QhSums = "", "", ""
		}
		put(t, o.Canary, canaryFor(sideToo), 0o644)
		if err := Run(o); err != nil {
			t.Fatal(err)
		}
	}
	bare, withSide := filepath.Join(s.work, "bare"), filepath.Join(s.work, "with-side")
	runStage(bare, []string{"A", "P", "W"}, false)
	runStage(withSide, []string{"A", "P", "W", "G", "C"}, true)

	export := func(pair [2]string, canary string, dirs ...string) (string, error) {
		var out bytes.Buffer
		err := ExportPaired(ExportOptions{Dirs: dirs, X: pair[0], Y: pair[1], Model: "stub-model", ScorerSHA: strings.Repeat("c", 40), Selection: selection, Canary: canary}, &out)
		return out.String(), err
	}
	for _, pair := range [][2]string{{"A", "P"}, {"A", "W"}} {
		want, err := export(pair, exportCanary(false), bare)
		if err != nil {
			t.Fatal(err)
		}
		for _, canarySide := range []bool{false, true} {
			got, err := export(pair, exportCanary(canarySide), withSide)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("--pair %v (canary with side arms %v): the export of a stage with side arms differs from the stage without:\nwithout %s\nwith    %s", pair, canarySide, want, got)
			}
		}
		if strings.Count(want, "\n") != 3 || strings.Contains(want, `"arm":"G"`) || strings.Contains(want, `"arm":"C"`) || !strings.Contains(want, `"configs":["A","P","W"]`) {
			t.Errorf("--pair %v: the export is not the config line and two rows of the core arms:\n%s", pair, want)
		}
	}

	// run.json: ADR-112's keys as they were, the side arms in their own.
	var p plan
	raw, err := os.ReadFile(filepath.Join(withSide, "run.json"))
	if err != nil || json.Unmarshal(raw, &p) != nil {
		t.Fatalf("run.json: %v %s", err, raw)
	}
	if !slices.Equal(p.Configs, []string{"A", "P", "W"}) || p.ConfigsFor != nil || !slices.Equal(p.SideAdmitted, []string{"C", "G"}) ||
		!slices.Equal(p.SideConfigsFor["stub-model"], []string{"C", "G"}) || p.Side["G"].AssetSHA256 != s.sha || p.Side["G"].Tag != "v1.0.0" ||
		p.Side["G"].Setting["QH_HOOK_RUNTIME"] != "go" || p.Side["C"].Setting["turnEnd"] != "notify" || p.Side["C"].AssetSHA256 != "" {
		t.Errorf("run.json of a stage with side arms: %+v", p)
	}
	bareRaw, _ := os.ReadFile(filepath.Join(bare, "run.json"))
	if strings.Contains(string(bareRaw), "side") {
		t.Errorf("a stage without side arms writes side keys:\n%s", bareRaw)
	}
	for _, arm := range []string{"G", "C"} {
		task := s.opts.Tasks[0].ID
		if _, err := os.Stat(filepath.Join(withSide, "side", task+"."+arm+".stub-model.1.receipt.json")); err != nil {
			t.Errorf("%s's receipt is not beside the core's, in side/: %v", arm, err)
		}
		if _, err := os.Stat(filepath.Join(withSide, task+"."+arm+".stub-model.1.receipt.json")); err == nil {
			t.Errorf("%s's receipt is among the core receipts", arm)
		}
	}
	if got := receipts(t, withSide); len(got) != 3 {
		t.Errorf("%d core receipts, want 3 (A, P, W)", len(got))
	}
	// The core's attempts run in the order they would without the side arms, and
	// the side arms run for the models that run P alone.
	many := func(o RunOptions) RunOptions {
		o.Models = []string{"m1", "m2"}
		o.ConfigsFor = map[string][]string{"m2": {"A"}}
		o.Tasks = nil
		for i := range 8 {
			task := s.opts.Tasks[0]
			task.ID = fmt.Sprintf("x%d", i)
			o.Tasks = append(o.Tasks, task)
		}
		return o
	}
	coreIDs := func(o RunOptions) (core, side []string) {
		for _, a := range planAttempts(o) {
			if a.Config.Side() {
				side = append(side, a.ID)
			} else {
				core = append(core, a.ID)
			}
		}
		return core, side
	}
	bareOpts, sideOpts := many(s.opts), many(s.opts)
	bareOpts.Configs = []string{"A", "P", "W"}
	coreBare, sideBare := coreIDs(bareOpts)
	coreSide, sideArms := coreIDs(sideOpts)
	if len(sideBare) != 0 || len(sideArms) != 16 || !slices.Equal(coreBare, coreSide) {
		t.Errorf("planned attempts: %d side attempts without side arms and %d with (want 0 and 16, G and C for 8 tasks of the model that runs P); the core's order moved: %v", len(sideBare), len(sideArms), !slices.Equal(coreBare, coreSide))
	}
	if slices.ContainsFunc(sideArms, func(id string) bool { return strings.Contains(id, ".m2.") }) {
		t.Errorf("a model that does not run P was given side arms: %v", sideArms)
	}
	// bench verify reads the side arms' receipts beside P's, and what they record.
	if err := Verify(withSide, "test", 1, 1); err != nil {
		t.Errorf("a complete stage with side arms does not verify: %v", err)
	}
	gReceipt := filepath.Join(withSide, sideDir, s.opts.Tasks[0].ID+".G.stub-model.1.receipt.json")
	good, err := os.ReadFile(gReceipt)
	if err != nil {
		t.Fatal(err)
	}
	var gr Receipt
	if err := json.Unmarshal(good, &gr); err != nil {
		t.Fatal(err)
	}
	gr.Side.AssetSHA256 = strings.Repeat("9", 64)
	for name, body := range map[string][]byte{"a side receipt that records another asset": []byte(mustJSON(t, gr)), "a side receipt that is missing": nil} {
		if body == nil {
			if err := os.Remove(gReceipt); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(gReceipt, body, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := Verify(withSide, "test", 1, 1); err == nil || !strings.Contains(err.Error(), ".G.") {
			t.Errorf("%s verified: %v", name, err)
		}
	}
	if err := os.WriteFile(gReceipt, good, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("a canary authorizes only the data directory and side registration it ran", func(t *testing.T) {
		n := 0
		try := func(edit func(c map[string]any)) error {
			var c map[string]any
			if err := json.Unmarshal([]byte(canaryFor(true)), &c); err != nil {
				t.Fatal(err)
			}
			edit(c)
			n++
			o := s.opts
			o.Out, o.StatusLine = filepath.Join(s.work, fmt.Sprintf("refused-%d", n)), true
			put(t, o.Canary, mustJSON(t, c), 0o644)
			return Run(o)
		}
		reg := func(c map[string]any) map[string]any { return c["side_registration"].(map[string]any) }
		for name, edit := range map[string]func(map[string]any){
			"another plugin data directory": func(c map[string]any) { c["plugin_data"] = "/elsewhere/plugins/data/quality-harness-inline" },
			"no plugin data directory":      func(c map[string]any) { delete(c, "plugin_data") },
			"no side registration":          func(c map[string]any) { delete(c, "side_registration") },
			"another C value":               func(c map[string]any) { reg(c)["c"].(map[string]any)["value"] = "quiet" },
			"another C key":                 func(c map[string]any) { reg(c)["c"].(map[string]any)["key"] = "turn_end" },
			"another G runtime": func(c map[string]any) {
				reg(c)["g"].(map[string]any)["runtime"] = map[string]any{"QH_HOOK_RUNTIME": "node"}
			},
			"another G marker":     func(c map[string]any) { reg(c)["g"].(map[string]any)["marker"] = "other" },
			"another G stage path": func(c map[string]any) { reg(c)["g"].(map[string]any)["stage_as"] = "qh/2/qh" },
			"another tag":          func(c map[string]any) { reg(c)["tag"] = "v9.9.9" },
			"another asset sha256": func(c map[string]any) { c["side_asset_sha256"] = strings.Repeat("9", 64) },
			"no asset sha256":      func(c map[string]any) { delete(c, "side_asset_sha256") },
		} {
			if err := try(edit); err == nil || !strings.Contains(err.Error(), "canary") {
				t.Errorf("%s was accepted: %v", name, err)
			}
		}
	})

	t.Run("blocks with and without side arms export as one", func(t *testing.T) {
		dir := t.TempDir()
		plugin := &PluginRef{Tag: "v3.8.39", Commit: strings.Repeat("a", 40)}
		block := func(name string, edit func(*plan)) string {
			p := plan{Stage: "stage6", Tasks: []string{"i1"}, Models: []string{"m1"}, Configs: []string{"A", "P"}, Repeats: 1, Seed: 163,
				CLIVersion: "9.9.9 (Claude Code)", Containment: "bypass", Profile: ProfileStage6, Plugin: plugin,
				StatusLineSHA256: strings.Repeat("b", 64), PromptSHA256: strings.Repeat("d", 64), BudgetSeconds: 1200, MaxTurns: 60, ChecksSHA256: strings.Repeat("c", 64)}
			edit(&p)
			d := filepath.Join(dir, name)
			put(t, filepath.Join(d, "run.json"), mustJSON(t, p), 0o644)
			return d
		}
		can := filepath.Join(dir, "canary.json")
		put(t, can, `{"pass":true}`, 0o644)
		sel := filepath.Join(dir, "selection.json")
		put(t, sel, `{"rule":"the 1 ids with the lowest sha256(\"adr-163:\" + instance_id)","dataset":"SWE-bench/SWE-bench_Verified","revision":"78f4","split":"test","selected":["i1"]}`, 0o644)
		line := func(dirs ...string) string {
			var out bytes.Buffer
			if err := ExportPaired(ExportOptions{Dirs: dirs, X: "A", Y: "P", Model: "m1", ScorerSHA: strings.Repeat("c", 40), Selection: sel, Canary: can}, &out); err != nil {
				t.Fatalf("%v", err)
			}
			return out.String()
		}
		withArms := func(p *plan) {
			p.SideAdmitted = []string{"C", "G"}
			p.SideConfigsFor = map[string][]string{"m1": {"C", "G"}}
			p.Side = map[string]SideMeta{"G": {Tag: "v3.8.39", AssetSHA256: strings.Repeat("e", 64), Setting: map[string]string{"X": "y"}}, "C": {Tag: "v3.8.39", Setting: map[string]string{"turnEnd": "notify"}}}
		}
		plain := line(block("plain-one", func(*plan) {}), block("plain-two", func(*plan) {}))
		if mixed := line(block("mixed-one", func(*plan) {}), block("mixed-two", withArms)); mixed != plain {
			t.Errorf("a block with side arms changes the export:\n%s\n%s", plain, mixed)
		}
		if mixed := line(block("mixed-three", withArms), block("mixed-four", func(*plan) {})); mixed != plain {
			t.Errorf("the block order changes the export:\n%s\n%s", plain, mixed)
		}
		// A run.json that lists a side arm among its configs still exports the core's.
		if hostile := line(block("hostile", func(p *plan) { p.Configs = []string{"A", "P", "G", "C"} })); !strings.Contains(hostile, `"configs":["A","P"]`) {
			t.Errorf("a side arm in run.json's configs reaches the scaffold:\n%s", hostile)
		}
	})

	t.Run("a side arm is not a pair", func(t *testing.T) {
		for _, pair := range [][2]string{{"A", "G"}, {"A", "C"}, {"G", "P"}, {"P", "C"}, {"P", "G"}} {
			if _, err := export(pair, exportCanary(false), withSide); err == nil || !strings.Contains(err.Error(), "bench side") {
				t.Errorf("--pair %v: %v, want a refusal pointing at bench side", pair, err)
			}
		}
	})

	t.Run("a side arm cannot join or leave after block 1", func(t *testing.T) {
		stage := filepath.Join(s.work, "stage")
		put(t, filepath.Join(stage, "block-1", "run.json"), string(raw), 0o644)
		runBlock := func(edit func(*RunOptions)) error {
			o := s.opts
			o.Mrw, o.Block, o.Out, o.StatusLine = "", 2, filepath.Join(stage, "block-2"), true
			o.Configs = []string{"A", "P", "W", "G", "C"}
			edit(&o)
			put(t, o.Canary, canaryFor(true), 0o644)
			return Run(o)
		}
		withoutBlock1Side := func(edit func(p *plan)) {
			var q plan
			if err := json.Unmarshal(raw, &q); err != nil {
				t.Fatal(err)
			}
			edit(&q)
			put(t, filepath.Join(stage, "block-1", "run.json"), mustJSON(t, q), 0o644)
		}
		withoutBlock1Side(func(q *plan) { q.SideAdmitted, q.SideConfigsFor, q.Side = nil, nil, nil })
		if err := runBlock(func(*RunOptions) {}); err == nil || !strings.Contains(err.Error(), "never joins or leaves after block 1") {
			t.Errorf("a side arm joining at block 2: %v", err)
		}
		withoutBlock1Side(func(q *plan) { q.SideAdmitted = []string{"C", "G"}; q.Side = nil })
		if err := runBlock(func(*RunOptions) {}); err == nil || !strings.Contains(err.Error(), "one tag, asset and setting") {
			t.Errorf("block 2 with another asset or setting than block 1: %v", err)
		}
		withoutBlock1Side(func(q *plan) {})
		if err := runBlock(func(o *RunOptions) { o.Configs, o.QhBinary, o.QhSums = []string{"A", "P", "W", "C"}, "", "" }); err == nil || !strings.Contains(err.Error(), "side") {
			t.Errorf("a side arm leaving at block 2: %v", err)
		}
		if err := runBlock(func(*RunOptions) {}); err != nil {
			t.Errorf("block 2 with block 1's side arms was refused: %v", err)
		}
	})
}

// sideBlock writes one model's receipts into a block directory: A and P, and
// the arm's in side/. P costs 2.0, takes 10 turns and 100 s, the arm costs
// ratio of that in 8 turns and 80 s, and the arm fails the first lost tasks
// P solved.
func sideBlock(t *testing.T, block, model, arm string, from, to int, ratio float64, lost int) {
	t.Helper()
	write := func(dir string, r Receipt) {
		r.Attempt = fmt.Sprintf("%s.%s.%s.1", r.Task, r.Config, safe(model))
		put(t, filepath.Join(dir, r.Attempt+".receipt.json"), mustJSON(t, r), 0o644)
	}
	for i := from; i < to; i++ {
		task := fmt.Sprintf("t%03d", i)
		mk := func(cfg string, cost float64, turns int, wallMS int64, solved bool) Receipt {
			c := cost
			return Receipt{Stage: "stage6", Task: task, Source: "swe-bench", Model: model, Config: cfg, Repeat: 1, CostUSD: &c, NumTurns: turns, WallMS: wallMS, Oracle: Verdict{Solved: solved}}
		}
		write(block, mk("A", 1.0, 5, 50000, true))
		write(block, mk("P", 2.0, 10, 100000, true))
		write(filepath.Join(block, "side"), mk(arm, 2.0*ratio, 8, 80000, i-from >= lost))
	}
	sideRunJSON(t, block, "stage6", arm, nil)
}

// sideRunJSON writes the run.json of a block directory that ran the arm at the
// plugin's tag, with edit applied.
func sideRunJSON(t *testing.T, block, stage, arm string, edit func(*plan)) {
	t.Helper()
	meta := SideMeta{Tag: "v1.0.0", Setting: map[string]string{"turnEnd": "notify"}}
	if arm == "G" {
		meta = SideMeta{Tag: "v1.0.0", AssetSHA256: strings.Repeat("a", 64), Setting: map[string]string{"QH_HOOK_RUNTIME": "go", "QUALITY_HARNESS_QH_FETCH": "off"}}
	}
	p := plan{Stage: stage, Models: []string{"m1", "m2"}, Repeats: 1, Plugin: &PluginRef{Tag: "v1.0.0", Commit: strings.Repeat("c", 40)},
		SideAdmitted: []string{arm}, Side: map[string]SideMeta{arm: meta}}
	if edit != nil {
		edit(&p)
	}
	put(t, filepath.Join(block, "run.json"), mustJSON(t, p), 0o644)
}

func readSide(t *testing.T, arm string, dirs ...string) []SideRow {
	t.Helper()
	rows, err := Side(dirs, "P", arm)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// TestTheSideReaderUsesPAsControlAndTheRegisteredBar: the arm is read against
// P, never A, paired by task; an unpaired task is dropped and named, an
// exclusion counted, and the registered thresholds hold at each boundary on
// both sides; fewer than 60 paired tasks per model are descriptive only;
// a pilot is never counted (ADR-163 Decision 9, BACKLOG "Pre-registered for
// ADR-163").
func TestTheSideReaderUsesPAsControlAndTheRegisteredBar(t *testing.T) {
	stage := func() (string, string) {
		d := t.TempDir()
		return d, filepath.Join(d, "block-1")
	}

	t.Run("the bar at each boundary, both sides", func(t *testing.T) {
		for _, c := range []struct {
			arm              string
			relHi, solvedLo  float64
			pairs, costPairs int
			want             string
		}{
			// C: the upper bound of cost below 0, the lower bound of solved above -0.10.
			{"C", -1e-6, -0.0999, 60, 60, SideHolds},
			{"C", -1e-6, -0.0999, 59, 59, SideDescriptive},
			{"C", 0, -0.0999, 60, 60, SideFails},
			{"C", 1e-6, -0.0999, 60, 60, SideFails},
			{"C", -1e-6, -0.10, 60, 60, SideFails},
			{"C", -1e-6, -0.1001, 60, 60, SideFails},
			{"C", -0.5, 0.2, 60, 60, SideHolds},
			{"C", -0.5, 0.2, 60, 0, SideFails},
			{"C", -0.5, 0.2, 61, 61, SideHolds},
			{"C", 0.04, 0.2, 100, 100, SideFails},
			// G: the upper bound of cost below +0.05, the lower bound of solved above -0.10.
			{"G", 0.0499, -0.0999, 60, 60, SideHolds},
			{"G", 0.0499, -0.0999, 59, 59, SideDescriptive},
			{"G", 0.05, -0.0999, 60, 60, SideFails},
			{"G", 0.0501, -0.0999, 60, 60, SideFails},
			{"G", 0.0499, -0.10, 60, 60, SideFails},
			{"G", 0.0499, -0.1001, 60, 60, SideFails},
			{"G", 0.0001, 0.2, 60, 60, SideHolds},
			{"G", 0.0001, 0.2, 60, 0, SideFails},
			{"G", 0.0001, 0.2, 0, 0, SideDescriptive},
		} {
			if got := sideVerdict(c.arm, c.relHi, c.solvedLo, c.pairs, c.costPairs); got != c.want {
				t.Errorf("%s with cost upper %v, solved lower %v, %d pairs (%d with a cost): %s, want %s", c.arm, c.relHi, c.solvedLo, c.pairs, c.costPairs, got, c.want)
			}
		}
	})

	t.Run("P is the control, never A", func(t *testing.T) {
		_, block := stage()
		// The arm costs half of P and the same as A: against P it saves a half, against A it saves nothing.
		sideBlock(t, block, "m1", "C", 0, 60, 0.5, 0)
		rows := readSide(t, "C", block)
		if len(rows) != 1 {
			t.Fatalf("%d rows, want one per model: %+v", len(rows), rows)
		}
		r := rows[0]
		if r.Verdict != SideHolds || r.Control != "P" || r.Arm != "C" || r.Model != "m1" || r.Result.Tasks != 60 ||
			r.Result.RelCost > -0.49 || r.Result.RelCost < -0.51 || r.Result.RelHi >= 0 {
			t.Errorf("C at half P's cost: %+v", r)
		}
		if r.Result.Level != lookLevel || r.Result.Comparison != "C" {
			t.Errorf("the interval is at level %v for %q, want the look's %v for C", r.Result.Level, r.Result.Comparison, lookLevel)
		}
		if r.TurnsControl != 10 || r.TurnsArm != 8 || r.WallControl != 100 || r.WallArm != 80 {
			t.Errorf("turns %v/%v and wall %v/%v s beside the verdict, want 10/8 and 100/80", r.TurnsControl, r.TurnsArm, r.WallControl, r.WallArm)
		}
		if r.Claim == "" {
			t.Error("a row without the claim it tests")
		}
		for _, bad := range []struct{ control, arm string }{{"A", "C"}, {"B", "G"}, {"W", "C"}, {"G", "C"}, {"P", "A"}, {"P", "P"}, {"P", "W"}, {"P", ""}, {"", "G"}} {
			if _, err := Side([]string{block}, bad.control, bad.arm); err == nil {
				t.Errorf("--control %q --arm %q was read", bad.control, bad.arm)
			}
		}
	})

	t.Run("pairing, exclusions and the 60-pair floor", func(t *testing.T) {
		_, block := stage()
		sideBlock(t, block, "m1", "G", 0, 60, 1.04, 0)
		if r := readSide(t, "G", block)[0]; r.Verdict != SideHolds || r.Result.Tasks != 60 || len(r.Result.Unpaired) != 0 {
			t.Errorf("60 paired tasks at 4%% over P: %+v", r)
		}
		_, block = stage()
		sideBlock(t, block, "m1", "G", 0, 59, 0.96, 0)
		if r := readSide(t, "G", block)[0]; r.Verdict != SideDescriptive || r.Result.Tasks != 59 || r.Result.RelHi >= 0.05 {
			t.Errorf("59 paired tasks: %+v, want descriptive only with the numbers still there", r)
		}
		// 61 tasks of P, one without the arm: 60 pairs, a verdict; the unpaired one is named.
		_, block = stage()
		sideBlock(t, block, "m1", "G", 0, 61, 0.96, 0)
		if err := os.Remove(filepath.Join(block, "side", "t030.G.m1.1.receipt.json")); err != nil {
			t.Fatal(err)
		}
		if r := readSide(t, "G", block)[0]; r.Verdict != SideHolds || r.Result.Tasks != 60 || !slices.Equal(r.Result.Unpaired, []string{"t030"}) {
			t.Errorf("61 tasks, one unpaired: %+v", r)
		}
		// 60 tasks, one without the arm: 59 pairs, no verdict.
		_, block = stage()
		sideBlock(t, block, "m1", "G", 0, 60, 0.96, 0)
		if err := os.Remove(filepath.Join(block, "side", "t007.G.m1.1.receipt.json")); err != nil {
			t.Fatal(err)
		}
		if r := readSide(t, "G", block)[0]; r.Verdict != SideDescriptive || r.Result.Tasks != 59 || !slices.Equal(r.Result.Unpaired, []string{"t007"}) {
			t.Errorf("60 tasks, one unpaired: %+v", r)
		}
		// An excluded arm attempt is counted, leaves its task unpaired, and does not make a pair.
		_, block = stage()
		sideBlock(t, block, "m1", "G", 0, 60, 0.96, 0)
		path := filepath.Join(block, "side", "t009.G.m1.1.receipt.json")
		var ex Receipt
		if err := readJSON(path, &ex); err != nil {
			t.Fatal(err)
		}
		ex.Exclusion, ex.Note = ExclusionStart, "rate limit"
		if err := os.WriteFile(path, []byte(mustJSON(t, ex)), 0o644); err != nil {
			t.Fatal(err)
		}
		if r := readSide(t, "G", block)[0]; r.Verdict != SideDescriptive || r.Result.Tasks != 59 || r.Result.Excluded != 1 || !slices.Equal(r.Result.Unpaired, []string{"t009"}) {
			t.Errorf("one excluded attempt: %+v, want 59 pairs, 1 excluded, t009 named", r)
		}
		// An exclusion the registration does not admit is refused, not read.
		ex.Exclusion = "tired"
		if err := os.WriteFile(path, []byte(mustJSON(t, ex)), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Side([]string{block}, "P", "G"); err == nil {
			t.Error("an exclusion the registration does not admit was read")
		}
	})

	t.Run("each model is its own count, and blocks are cumulated", func(t *testing.T) {
		_, block := stage()
		sideBlock(t, block, "m1", "C", 0, 60, 0.5, 0)
		sideBlock(t, block, "m2", "C", 0, 59, 0.5, 0)
		rows := readSide(t, "C", block)
		if len(rows) != 2 || rows[0].Model != "m1" || rows[0].Verdict != SideHolds || rows[1].Model != "m2" || rows[1].Verdict != SideDescriptive {
			t.Errorf("60 pairs for one model and 59 for the other: %+v", rows)
		}
		d, b1 := stage()
		b2 := filepath.Join(d, "block-2")
		sideBlock(t, b1, "m1", "C", 0, 30, 0.5, 0)
		sideBlock(t, b2, "m1", "C", 30, 60, 0.5, 0)
		if r := readSide(t, "C", b1, b2)[0]; r.Verdict != SideHolds || r.Result.Tasks != 60 {
			t.Errorf("two blocks of 30: %+v, want the 60 pairs cumulated", r)
		}
		if r := readSide(t, "C", b1)[0]; r.Verdict != SideDescriptive || r.Result.Tasks != 30 {
			t.Errorf("one block of 30: %+v", r)
		}
	})

	t.Run("blocks that ran another plugin, asset or setting are not pooled", func(t *testing.T) {
		for name, c := range map[string]struct {
			arm  string
			edit func(*plan)
		}{
			"another plugin tag":    {"C", func(p *plan) { p.Plugin.Tag = "v2.0.0" }},
			"another plugin commit": {"C", func(p *plan) { p.Plugin.Commit = strings.Repeat("d", 40) }},
			"no plugin":             {"C", func(p *plan) { p.Plugin = nil }},
			"another C setting":     {"C", func(p *plan) { p.Side["C"] = SideMeta{Tag: "v1.0.0", Setting: map[string]string{"turnEnd": "quiet"}} }},
			"another side tag":      {"C", func(p *plan) { p.Side["C"] = SideMeta{Tag: "v2.0.0", Setting: map[string]string{"turnEnd": "notify"}} }},
			"another G asset":       {"G", func(p *plan) { m := p.Side["G"]; m.AssetSHA256 = strings.Repeat("b", 64); p.Side["G"] = m }},
			"another G runtime": {"G", func(p *plan) {
				m := p.Side["G"]
				m.Setting = map[string]string{"QH_HOOK_RUNTIME": "node"}
				p.Side["G"] = m
			}},
			"an arm it did not run": {"C", func(p *plan) { p.SideAdmitted = nil }},
		} {
			d, b1 := stage()
			b2 := filepath.Join(d, "block-2")
			sideBlock(t, b1, "m1", c.arm, 0, 30, 0.5, 0)
			sideBlock(t, b2, "m1", c.arm, 30, 60, 0.5, 0)
			sideRunJSON(t, b2, "stage6", c.arm, c.edit)
			if _, err := Side([]string{b1, b2}, "P", c.arm); err == nil || !strings.Contains(err.Error(), "another") {
				t.Errorf("%s: %v, want the blocks refused as not one stage", name, err)
			}
		}
		d, b1 := stage()
		b2 := filepath.Join(d, "block-2")
		sideBlock(t, b1, "m1", "C", 0, 30, 0.5, 0)
		sideBlock(t, b2, "m1", "C", 30, 60, 0.5, 0)
		if err := os.Remove(filepath.Join(b2, "run.json")); err != nil {
			t.Fatal(err)
		}
		if _, err := Side([]string{b1, b2}, "P", "C"); err == nil || !strings.Contains(err.Error(), "run.json") {
			t.Errorf("a block with no run.json: %v", err)
		}
	})

	t.Run("solved is Tango's lower bound against the margin", func(t *testing.T) {
		// At 60 pairs with every outcome concordant the 99% lower bound is -0.0996 (BACKLOG); find the
		// fewest tasks the arm may lose before it is below -0.10, with the interval the reader uses.
		if lo, _ := tangoInterval(0, 0, 60, lookLevel); lo < -0.0997 || lo > -0.0995 {
			t.Fatalf("the all-concordant lower bound at 60 pairs is %v, the registration says -0.0996", lo)
		}
		lose := 1
		for ; ; lose++ {
			if lo, _ := tangoInterval(0, lose, 60, lookLevel); lo <= -0.10 {
				break
			}
		}
		_, block := stage()
		sideBlock(t, block, "m1", "C", 0, 60, 0.5, lose-1)
		if r := readSide(t, "C", block)[0]; r.Verdict != SideHolds {
			t.Errorf("losing %d of 60: %+v", lose-1, r)
		}
		_, block = stage()
		sideBlock(t, block, "m1", "C", 0, 60, 0.5, lose)
		if r := readSide(t, "C", block)[0]; r.Verdict != SideFails || r.Result.SolvedLo > -0.10 {
			t.Errorf("losing %d of 60: %+v, want solved's lower bound at or below -0.10", lose, r)
		}
	})

	t.Run("cost at each boundary, end to end", func(t *testing.T) {
		for _, c := range []struct {
			arm   string
			ratio float64
			want  string
		}{{"C", 0.99, SideHolds}, {"C", 1.0, SideFails}, {"C", 1.01, SideFails},
			{"G", 1.049, SideHolds}, {"G", 1.05, SideFails}, {"G", 1.051, SideFails}} {
			_, block := stage()
			sideBlock(t, block, "m1", c.arm, 0, 60, c.ratio, 0)
			if r := readSide(t, c.arm, block)[0]; r.Verdict != c.want {
				t.Errorf("%s at %v of P's cost: %s (cost upper %v), want %s", c.arm, c.ratio, r.Verdict, r.Result.RelHi, c.want)
			}
		}
	})

	t.Run("a pilot is never counted", func(t *testing.T) {
		_, block := stage()
		sideBlock(t, block, "m1", "C", 0, 60, 0.5, 0)
		// Pilot receipts for 5 tasks beside them, the arm absurdly dear: counted, they would end the verdict.
		pilot := filepath.Join(filepath.Dir(block), "pilot", "block-0")
		sideBlock(t, pilot, "m1", "C", 0, 5, 50, 5)
		for _, dir := range []string{pilot, filepath.Join(pilot, "side")} {
			ents, _ := os.ReadDir(dir)
			for _, e := range ents {
				if !strings.HasSuffix(e.Name(), ".receipt.json") {
					continue
				}
				var r Receipt
				if err := readJSON(filepath.Join(dir, e.Name()), &r); err != nil {
					t.Fatal(err)
				}
				r.Stage = "pilot-agent"
				put(t, filepath.Join(dir, e.Name()), mustJSON(t, r), 0o644)
			}
		}
		// The pilot ran at another tag, and is not compared with the stage's blocks.
		sideRunJSON(t, pilot, "pilot-agent", "C", func(p *plan) { p.Plugin.Tag = "v0.0.1" })
		if r := readSide(t, "C", block, pilot)[0]; r.Verdict != SideHolds || r.Result.Tasks != 60 {
			t.Errorf("with the pilot's receipts beside the block's: %+v", r)
		}
	})

	t.Run("no receipts of the arm is an error, not a verdict", func(t *testing.T) {
		_, block := stage()
		sideBlock(t, block, "m1", "C", 0, 60, 0.5, 0)
		if _, err := Side([]string{block}, "P", "G"); err == nil {
			t.Error("a read of G over a stage that ran only C gave rows")
		}
	})
}
