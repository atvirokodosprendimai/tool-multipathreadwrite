package bench

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// stage9Classes are the four classes of Stage 9's task list, in the order the
// list names them (ADR-206 Decision 8): a task's shape is its class.
var stage9Classes = []string{ShapeSingleSite, ShapeBatch, ShapeFollowOn, ShapeControl}

// stage9Sites are a task's change regions as LoadTask parsed the reference
// patch: every file's hunks in base order, and how many files hold one.
func stage9Sites(t Task) (sites [][]hunk, files int) {
	for _, f := range t.Files() {
		hs := append([]hunk(nil), t.regions[f]...)
		if len(hs) == 0 {
			continue
		}
		sort.Slice(hs, func(i, j int) bool { return hs[i].Start < hs[j].Start })
		sites = append(sites, hs)
		files++
	}
	return sites, files
}

// stage9Problems holds the class list under benchDir (a bench/ directory:
// stage-9/classes.json, tasks/, repos.json) to ADR-206 Decision 8 and returns
// each way it fails. The rules that read only the task files always run; with
// reference set, each task's require is also held against its reference
// result built from the pinned commit, and unreached counts the tasks whose
// repository or commit is not on this machine, which that half cannot judge.
func stage9Problems(benchDir string, reference bool) (problems []string, unreached int, err error) {
	b, err := os.ReadFile(filepath.Join(benchDir, "stage-9", "classes.json"))
	if err != nil {
		return []string{"no class list: " + err.Error()}, 0, nil
	}
	var list map[string][]string
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, 0, err
	}
	bad := func(format string, a ...any) { problems = append(problems, fmt.Sprintf(format, a...)) }
	for class := range list {
		if !stdSliceHas(stage9Classes, class) {
			bad("class %q is not one of %v", class, stage9Classes)
		}
	}
	owner := map[string]string{}
	for _, class := range stage9Classes {
		ids := list[class]
		if len(ids) != 12 {
			bad("class %s holds %d tasks, want 12", class, len(ids))
		}
		for _, id := range ids {
			if prev, dup := owner[id]; dup {
				bad("task %s is listed twice (in %s and %s)", id, prev, class)
				continue
			}
			owner[id] = class
			t, err := LoadTask(filepath.Join(benchDir, "tasks", id))
			if err != nil {
				bad("task %s: %v", id, err)
				continue
			}
			if t.Shape != class {
				bad("task %s has shape %q, want its class %q", id, t.Shape, class)
			}
			if !t.Exact {
				bad("task %s is not exact", id)
			}
			if t.SupersededBy != "" {
				bad("task %s is superseded by %s", id, t.SupersededBy)
			}
			for f, made := range t.created {
				if made {
					bad("task %s creates %s", id, f)
				}
			}
			if len(t.Require) == 0 {
				bad("task %s has no require", id)
			}
			sites, files := stage9Sites(t)
			n := 0
			for _, hs := range sites {
				n += len(hs)
			}
			switch class {
			case ShapeSingleSite:
				if n != 1 {
					bad("task %s has %d sites, want 1", id, n)
				}
			case ShapeBatch:
				if n < 5 || files < 3 {
					bad("task %s has %d sites in %d files, want 5 or more sites in 3 or more files", id, n, files)
				}
			case ShapeFollowOn, ShapeControl:
				if n != 2 || files != 1 {
					bad("task %s has %d sites in %d files, want 2 sites in 1 file", id, n, files)
					continue
				}
				first := sites[0][0]
				if changes := first.Count != first.Added; changes != (class == ShapeFollowOn) {
					bad("task %s: the first site %s the line count (%d lines become %d), which a %s task's first site %s",
						id, map[bool]string{true: "changes", false: "keeps"}[changes], first.Count, first.Added, class,
						map[bool]string{true: "changes", false: "keeps"}[class == ShapeFollowOn])
				}
			}
			if !reference {
				continue
			}
			held, err := stage9ReferenceHolds(t)
			switch {
			case errors.Is(err, errUnreachable):
				unreached++
			case err != nil:
				bad("task %s: %v", id, err)
			case held != "":
				bad("task %s: %s", id, held)
			}
		}
	}
	if len(owner) != 48 {
		bad("%d distinct tasks listed, want 48", len(owner))
	}
	return problems, unreached, nil
}

// stage9ReferenceHolds builds the task's base tree and its reference result.
// It returns what is wrong when every require already matches the base, or the
// reference result misses one; "" when at least one require misses the base and
// every one holds on the reference result.
func stage9ReferenceHolds(t Task) (string, error) {
	tree, err := os.MkdirTemp("", "stage9-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tree)
	if _, err := Materialize(t, tree, t.GradedFiles()); err != nil {
		return "", err
	}
	atBase := 0
	for _, m := range t.Require {
		ok, err := matches(tree, m, false)
		if err != nil {
			return "", err
		}
		if ok {
			atBase++
		}
	}
	if atBase == len(t.Require) {
		return "every require already matches the base, so none of them tells the fix from no fix", nil
	}
	if err := ApplyPatch(tree, filepath.Join(t.Dir, "reference.patch")); err != nil {
		return "", err
	}
	for _, m := range t.Require {
		ok, err := matches(tree, m, false)
		if err != nil {
			return "", err
		}
		if !ok {
			return fmt.Sprintf("the reference result does not satisfy require %s: %s", m.File, m.Match), nil
		}
	}
	return "", nil
}

func stdSliceHas(s []string, x string) bool {
	for _, e := range s {
		if e == x {
			return true
		}
	}
	return false
}

func stage9Bench(t *testing.T) string {
	t.Helper()
	return filepath.Dir(tasksDir(t))
}

// TestStage9ClassesHoldTheirRules holds Stage 9's committed class list
// (bench/stage-9/classes.json) to ADR-206 Decision 8: 48 distinct tasks, 12 a
// class; each exact, creating no file, with a require its reference result
// satisfies; and a reference patch whose change regions, as LoadTask parses
// them, have the class's shape: one site; five or more in three or more files;
// two in one file, the first changing the file's line count; two in one file,
// the first keeping it.
func TestStage9ClassesHoldTheirRules(t *testing.T) {
	needGit(t)
	problems, unreached, err := stage9Problems(stage9Bench(t), true)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		t.Error(p)
	}
	if unreached > 0 {
		t.Skipf("%d task(s) have a repository or commit that is not on this machine, so their require was held against no reference result", unreached)
	}
}

// TestTheStage9CheckRefusesEachBrokenList drives the rules above over a copy of
// the list and its task directories with one defect each, so the check that
// passes the real list is shown to fail otherwise (ADR-206 T3 S2). The copy
// leaves bench/ untouched.
func TestTheStage9CheckRefusesEachBrokenList(t *testing.T) {
	needGit(t)
	src := stage9Bench(t)
	b, err := os.ReadFile(filepath.Join(src, "stage-9", "classes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var list map[string][]string
	if err := json.Unmarshal(b, &list); err != nil {
		t.Fatal(err)
	}
	repos, err := os.ReadFile(filepath.Join(src, "repos.json"))
	if err != nil {
		t.Fatal(err)
	}
	// copyBench writes the list, repos.json and every listed task directory
	// into a new bench root and returns it; mutate runs on the root before the
	// list is written back.
	copyBench := func(t *testing.T, mutate func(root string, list map[string][]string)) string {
		root := t.TempDir()
		own := map[string][]string{}
		for k, v := range list {
			own[k] = append([]string(nil), v...)
		}
		must := func(err error) {
			t.Helper()
			if err != nil {
				t.Fatal(err)
			}
		}
		must(os.MkdirAll(filepath.Join(root, "stage-9"), 0o755))
		must(os.WriteFile(filepath.Join(root, "repos.json"), repos, 0o644))
		for _, ids := range list {
			for _, id := range ids {
				must(os.MkdirAll(filepath.Join(root, "tasks", id), 0o755))
				for _, f := range []string{"task.json", "reference.patch"} {
					body, err := os.ReadFile(filepath.Join(src, "tasks", id, f))
					must(err)
					must(os.WriteFile(filepath.Join(root, "tasks", id, f), body, 0o644))
				}
			}
		}
		if mutate != nil {
			mutate(root, own)
		}
		out, err := json.Marshal(own)
		must(err)
		must(os.WriteFile(filepath.Join(root, "stage-9", "classes.json"), out, 0o644))
		return root
	}
	rewrite := func(t *testing.T, path string, edit func(string) string) {
		t.Helper()
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(edit(string(b))), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// swap exchanges the first id of two classes.
	swap := func(l map[string][]string, a, b string) {
		l[a][0], l[b][0] = l[b][0], l[a][0]
	}
	var fiveSites string // a batch task with exactly five sites
	for _, id := range list["batch"] {
		task, err := LoadTask(filepath.Join(src, "tasks", id))
		if err != nil {
			t.Fatal(err)
		}
		if sites, _ := stage9Sites(task); len(sites) > 0 {
			n := 0
			for _, hs := range sites {
				n += len(hs)
			}
			if n == 5 {
				fiveSites = id
				break
			}
		}
	}
	if fiveSites == "" {
		t.Fatal("no batch task with exactly five sites: the batch-of-four defect has nothing to start from")
	}
	// A batch task of three files whose reference keeps five or more sites once
	// one file's changes are dropped: five sites in two files.
	var twoFiles, dropped string
find:
	for _, id := range list["batch"] {
		task, err := LoadTask(filepath.Join(src, "tasks", id))
		if err != nil {
			t.Fatal(err)
		}
		var files []string
		total := 0
		for _, f := range task.Files() {
			if n := len(task.regions[f]); n > 0 {
				files = append(files, f)
				total += n
			}
		}
		if len(files) != 3 {
			continue
		}
		for _, f := range files {
			if total-len(task.regions[f]) >= 5 {
				twoFiles, dropped = id, f
				break find
			}
		}
	}
	if twoFiles == "" {
		t.Fatal("no batch task of three files keeps five sites in two: the two-file defect has nothing to start from")
	}
	if problems, _, err := stage9Problems(copyBench(t, nil), false); err != nil || len(problems) != 0 {
		t.Fatalf("the unbroken copy has problems: %v %v", problems, err)
	}
	cases := []struct {
		name   string
		mutate func(t *testing.T, root string, l map[string][]string)
		want   string
	}{
		{"a class of eleven", func(t *testing.T, _ string, l map[string][]string) { l["control"] = l["control"][1:] }, "class control holds 11 tasks, want 12"},
		{"a repeated id", func(t *testing.T, _ string, l map[string][]string) { l["batch"][1] = l["batch"][0] }, "is listed twice"},
		{"a task that is not exact", func(t *testing.T, root string, l map[string][]string) {
			rewrite(t, filepath.Join(root, "tasks", l["follow-on"][0], "task.json"), func(s string) string {
				return strings.Replace(s, "\"exact\": true,\n", "", 1)
			})
		}, "is not exact"},
		{"a follow-on whose first site keeps the line count", func(t *testing.T, _ string, l map[string][]string) { swap(l, "follow-on", "control") }, "the first site keeps the line count"},
		{"a control whose first site changes the line count", func(t *testing.T, _ string, l map[string][]string) { swap(l, "follow-on", "control") }, "the first site changes the line count"},
		{"a batch of four sites", func(t *testing.T, root string, l map[string][]string) {
			l["batch"][0] = fiveSites
			rewrite(t, filepath.Join(root, "tasks", fiveSites, "reference.patch"), func(s string) string {
				i := strings.LastIndex(s, "\n@@ ")
				if i < 0 {
					t.Fatal("no hunk to drop")
				}
				return s[:i+1]
			})
		}, "want 5 or more sites in 3 or more files"},
		{"a single-site task with two sites", func(t *testing.T, _ string, l map[string][]string) { swap(l, "single-site", "control") }, "has 2 sites, want 1"},
		{"five sites in two files", func(t *testing.T, root string, l map[string][]string) {
			l["batch"][0] = twoFiles
			rewrite(t, filepath.Join(root, "tasks", twoFiles, "reference.patch"), func(s string) string {
				var kept []string
				for _, block := range strings.Split(s, "diff --git ") {
					if !strings.HasPrefix(block, "a/"+dropped+" ") {
						kept = append(kept, block)
					}
				}
				return strings.Join(kept, "diff --git ")
			})
		}, "in 2 files, want 5 or more sites in 3 or more files"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := copyBench(t, func(root string, l map[string][]string) { tc.mutate(t, root, l) })
			problems, _, err := stage9Problems(root, false)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range problems {
				if strings.Contains(p, tc.want) {
					return
				}
			}
			t.Fatalf("no problem says %q: %v", tc.want, problems)
		})
	}
	// The two defects below read the pinned repository, so each runs where the
	// clone of its own task is and skips where it is not. target is that task.
	target := list["follow-on"][0]
	heldOf := func(t *testing.T, mutate func(path string)) string {
		t.Helper()
		root := copyBench(t, func(root string, l map[string][]string) { mutate(filepath.Join(root, "tasks", target, "task.json")) })
		task, err := LoadTask(filepath.Join(root, "tasks", target))
		if err != nil {
			t.Fatal(err)
		}
		held, err := stage9ReferenceHolds(task)
		if errors.Is(err, errUnreachable) {
			t.Skipf("task %s: %v", target, err)
		}
		if err != nil {
			t.Fatal(err)
		}
		return held
	}
	t.Run("a require the reference result misses", func(t *testing.T) {
		held := heldOf(t, func(path string) {
			rewrite(t, path, func(s string) string {
				return strings.Replace(s, "\"match\": \"", "\"match\": \"zz-never-there-", 1)
			})
		})
		if !strings.Contains(held, "does not satisfy require") {
			t.Fatalf("task %s: no problem names the require the reference misses: %q", target, held)
		}
	})
	t.Run("a task whose every require already matches the base", func(t *testing.T) {
		held := heldOf(t, func(path string) {
			var doc map[string]any
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(b, &doc); err != nil {
				t.Fatal(err)
			}
			for _, r := range doc["require"].([]any) {
				r.(map[string]any)["match"] = "(?s)."
			}
			if b, err = json.Marshal(doc); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, b, 0o644); err != nil {
				t.Fatal(err)
			}
		})
		if !strings.Contains(held, "already matches the base") {
			t.Fatalf("task %s: no problem says every require matches the base: %q", target, held)
		}
	})
}
