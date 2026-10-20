package check

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// ADR-164 Decision 4. A {packages}-only scoped_check that can place some of the
// edited paths runs those and names the files it could not place; anything it
// cannot reason about still runs the whole check.
func TestAWriteThatPlacesSomeFilesRunsThoseAndNamesTheRest(t *testing.T) {
	needSh(t)
	root := t.TempDir()
	for path, body := range map[string]string{
		"go.mod":          "module x\n",
		"go.sum":          "\n",
		"a/a.go":          "package a\n",
		"b/b.go":          "package b\n",
		"docs/guide.md":   "# prose\n",
		"run.sh":          "true\n",
		"z.go/keep.txt":   "a directory named like a Go file\n",
		"c/testdata/x.go": "package testdata\n",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const whole = "echo WHOLE"
	pk := Config{Check: whole, ScopedCheck: "echo {packages}"}
	mixed := Config{Check: whole, ScopedCheck: "echo {packages} {files}"}
	files := Config{Check: whole, ScopedCheck: "echo {files}"}

	for name, tc := range map[string]struct {
		cfg     Config
		paths   []string
		opts    []Option
		command string
		outside []string
	}{
		"a file outside is named and its package runs":  {pk, []string{"a/a.go", "run.sh"}, nil, "echo ./a", []string{"run.sh"}},
		"two files outside, in the order named":         {pk, []string{"run.sh", "a/a.go", "docs/guide.md"}, nil, "echo ./a", []string{"run.sh", "docs/guide.md"}},
		"nothing placed runs everything":                {pk, []string{"run.sh"}, nil, whole, nil},
		"a module file runs everything":                 {pk, []string{"a/a.go", "go.mod"}, nil, whole, nil},
		"go.sum runs everything":                        {pk, []string{"a/a.go", "go.sum"}, nil, whole, nil},
		"an unplaceable .go runs everything":            {pk, []string{"a/a.go", "z.go"}, nil, whole, nil},
		"a directory holding no package runs all":       {pk, []string{"a/a.go", "docs"}, nil, whole, nil},
		"a present testdata .go maps and is no outlier": {pk, []string{"a/a.go", "c/testdata/x.go"}, nil, "echo ./a ./c/testdata", nil},
		"the writer's whole-check flag runs everything": {pk, []string{"a/a.go", "run.sh"}, []Option{Whole()}, whole, nil},
		"files and dirs templates report nothing":       {files, []string{"a/a.go", "run.sh"}, nil, "echo a/a.go run.sh", nil},
		"a mixed template names every path":             {mixed, []string{"a/a.go", "run.sh"}, nil, "echo ./a a/a.go run.sh", nil},
		"files alone, nothing placed, is unchanged":     {files, []string{"run.sh"}, nil, "echo run.sh", nil},
	} {
		res, err := Run(context.Background(), root, tc.cfg, tc.paths, tc.opts...)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if res.Command != tc.command {
			t.Errorf("%s: ran %q, want %q", name, res.Command, tc.command)
		}
		if !reflect.DeepEqual(res.OutsideScope, tc.outside) {
			t.Errorf("%s: outside %v, want %v", name, res.OutsideScope, tc.outside)
		}
	}

	// command() agrees with Run on the scoped command.
	if got, scoped := command(root, pk, []string{"a/a.go", "run.sh"}); got != "echo ./a" || !scoped {
		t.Errorf("command: %q %v", got, scoped)
	}

	// The line both surfaces print names the template and the files.
	res, err := Run(context.Background(), root, pk, []string{"a/a.go", "run.sh"})
	if err != nil {
		t.Fatal(err)
	}
	want := `outside the scoped check: run.sh — scoped_check "echo {packages}" runs {packages}, which names Go packages only`
	if got := res.OutsideLine(); got != want {
		t.Errorf("OutsideLine = %q, want %q", got, want)
	}
	if got := (Result{}).OutsideLine(); got != "" {
		t.Errorf("a result with nothing outside printed %q", got)
	}

	// The files left out are taken with the command, before it runs: a check
	// that removes one still names it (the Codex review of #467).
	if err := os.WriteFile(filepath.Join(root, "gone.sh"), []byte("true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rm := Config{Check: whole, ScopedCheck: "echo {packages} && rm gone.sh"}
	res, err = Run(context.Background(), root, rm, []string{"a/a.go", "gone.sh"})
	if err != nil || !reflect.DeepEqual(res.OutsideScope, []string{"gone.sh"}) {
		t.Errorf("a check that removed a file it ran without: outside %v, err %v", res.OutsideScope, err)
	}

	// ADR-042: a path that is not there is refused, with or without the flag,
	// and a module file that is there is never handed to Run when missing.
	for _, opts := range [][]Option{nil, {Whole()}} {
		_, err := Run(context.Background(), root, pk, []string{"a/a.go", "gone.sum"}, opts...)
		if err == nil || !strings.Contains(err.Error(), "is not there") {
			t.Errorf("a missing path was not refused: %v", err)
		}
	}
}

// command is scope without the files left out: the shape the scoping tests
// were written against.
func command(root string, cfg Config, paths []string) (cmdline string, scoped bool) {
	cmdline, scoped, _ = scope(root, cfg, paths)
	return cmdline, scoped
}
