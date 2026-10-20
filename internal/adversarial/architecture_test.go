package adversarial

import (
	"os/exec"
	"sort"
	"strings"
	"testing"
)

// The import graph docs/architecture.md rests on. Each rule is read from
// `go list` over the module, so a new package is judged the day it lands,
// without anyone remembering to list it here.

const module = "github.com/atvirokodosprendimai/tool-multipathreadwrite/"

// imports maps each in-module package, relative to the module, to everything
// it imports (standard library included), production files only.
func imports(t *testing.T) map[string][]string {
	t.Helper()
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}|{{join .Imports \",\"}}", "./...")
	cmd.Dir = "../.."
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	g := map[string][]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		pkg, deps, _ := strings.Cut(line, "|")
		var ds []string
		for _, d := range strings.Split(deps, ",") {
			if d != "" {
				ds = append(ds, strings.TrimPrefix(d, module))
			}
		}
		g[strings.TrimPrefix(pkg, module)] = ds
	}
	return g
}

// reach is every package p reaches through the module's own packages.
func reach(g map[string][]string, p string) map[string]bool {
	seen := map[string]bool{}
	var walk func(string)
	walk = func(q string) {
		for _, d := range g[q] {
			if !seen[d] {
				seen[d] = true
				walk(d)
			}
		}
	}
	walk(p)
	return seen
}

// D1: the engine answers to every surface, so it may import none of them.
func TestTheEngineImportsNoSurface(t *testing.T) {
	g := imports(t)
	surfaces := []string{"internal/mcp", "internal/writer", "internal/bench", "internal/curve"}
	for _, e := range []string{"internal/read", "internal/apply", "internal/plan", "internal/seen", "internal/check", "internal/state"} {
		if _, ok := g[e]; !ok {
			t.Fatalf("engine package %s is gone: update this test and docs/architecture.md together", e)
		}
		r := reach(g, e)
		for d := range r {
			if strings.HasPrefix(d, "cmd/") {
				t.Errorf("%s reaches %s, a command", e, d)
			}
		}
		for _, s := range surfaces {
			if r[s] {
				t.Errorf("%s reaches %s", e, s)
			}
		}
	}
}

// D2: each surface package has one composition root, so nothing else can build it.
func TestEachSurfaceHasOneRoot(t *testing.T) {
	g := imports(t)
	for pkg, root := range map[string]string{
		"internal/mcp":   "cmd/mrw",
		"internal/bench": "cmd/bench",
		"internal/curve": "cmd/curve",
	} {
		var by []string
		for p, ds := range g {
			for _, d := range ds {
				if d == pkg {
					by = append(by, p)
				}
			}
		}
		sort.Strings(by)
		if strings.Join(by, " ") != root {
			t.Errorf("%s is imported by %v, want only %s", pkg, by, root)
		}
	}
	if r := reach(g, "cmd/mrw"); r["internal/bench"] || r["internal/curve"] {
		t.Error("the mrw binary reaches a benchmark package")
	}
}

// D3: inside the mrw binary only these start a process: check runs the
// project's command, read finds ast-grep, subproc owns the process group.
func TestOnlyTheseStartAProcess(t *testing.T) {
	g := imports(t)
	allowed := map[string]bool{"internal/check": true, "internal/read": true, "internal/subproc": true}
	r := reach(g, "cmd/mrw")
	r["cmd/mrw"] = true
	for p := range r {
		if strings.Contains(p, ".") || !(strings.HasPrefix(p, "internal/") || strings.HasPrefix(p, "cmd/")) {
			continue
		}
		for _, d := range g[p] {
			if d == "os/exec" && !allowed[p] {
				t.Errorf("%s imports os/exec", p)
			}
		}
	}
}
