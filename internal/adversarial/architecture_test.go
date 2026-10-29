package adversarial

import (
	"os"
	"os/exec"
	"regexp"
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
	for _, p := range startsAProcess(imports(t), "cmd/mrw") {
		t.Errorf("%s imports os/exec", p)
	}
}

// startsAProcess lists the module packages root reaches, itself included, that
// import os/exec without being allowed to. A package is the module's when g
// lists it: judged by membership, never by how its path is spelled.
func startsAProcess(g map[string][]string, root string) []string {
	allowed := map[string]bool{"internal/check": true, "internal/read": true, "internal/subproc": true}
	r := reach(g, root)
	r[root] = true
	var bad []string
	for p := range r {
		if _, ours := g[p]; !ours || allowed[p] {
			continue
		}
		for _, d := range g[p] {
			if d == "os/exec" {
				bad = append(bad, p)
			}
		}
	}
	sort.Strings(bad)
	return bad
}

// The D3 judge itself: a module package is judged wherever it lives and however
// its path is spelled, and a standard-library or third-party import is not one.
func TestStartsAProcessJudgesEveryModulePackage(t *testing.T) {
	g := map[string][]string{
		"cmd/mrw":             {"pkg/process", "internal/process.v1", "internal/check", "github.com/x/y"},
		"pkg/process":         {"os/exec"},
		"internal/process.v1": {"os/exec"},
		"internal/check":      {"os/exec"},
	}
	got := strings.Join(startsAProcess(g, "cmd/mrw"), " ")
	if got != "internal/process.v1 pkg/process" {
		t.Errorf("startsAProcess = %q, want the two unallowed module packages", got)
	}
}

// The Module Map in docs/architecture.md claims every package: a package that
// lands without a row is invisible to the doc's reader (cmd/nativebench did,
// 2026-10-29), and a row for a package that is gone describes nothing.
func TestEveryModulePackageIsInTheModuleMap(t *testing.T) {
	doc, err := os.ReadFile("../../docs/architecture.md")
	if err != nil {
		t.Fatal(err)
	}
	_, after, ok := strings.Cut(string(doc), "\n## Module Map\n")
	if !ok {
		t.Fatal("docs/architecture.md has no \"## Module Map\" section")
	}
	section, _, _ := strings.Cut(after, "\n## ")
	named := map[string]bool{}
	for _, m := range regexp.MustCompile("`([a-z][a-z0-9/._-]*)`").FindAllStringSubmatch(section, -1) {
		named[m[1]] = true
	}
	for pkg := range imports(t) {
		if !named[pkg] {
			t.Errorf("%s has no Module Map row in docs/architecture.md", pkg)
		}
	}
	for name := range named {
		if !strings.HasPrefix(name, "cmd/") && !strings.HasPrefix(name, "internal/") {
			continue
		}
		if _, err := os.Stat("../../" + name); err != nil {
			t.Errorf("the Module Map names %s, which is not in the tree", name)
		}
	}
}
