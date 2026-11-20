package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Deep review of #745, through real reads and the built command. One file is one
// observation whatever the plan calls it (ADR-029), and the read a refusal names
// must be one that lifts it.

// remedyOf is the argv of the read a refusal ends with, without mrw.
func remedyOf(t *testing.T, out string) []string {
	t.Helper()
	m := regexp.MustCompile(`mrw ((?:--root \S+ )?read --max-cols 0 \S+)`).FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no read to make in:\n%s", out)
	}
	return strings.Fields(m[1])
}

// aliasRoot is a root with f.go written by mrw and an alias of it: link.go, a
// symlink, or F.go where the filesystem folds case.
func aliasRoot(t *testing.T, alias string) string {
	t.Helper()
	needShell(t)
	if !regexp.MustCompile(`^[A-Za-z0-9._/-]+$`).MatchString(t.TempDir()) {
		t.Skip("a root that needs quoting is described, not commanded: the remedy is not a command to run")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ f.go 0 create\n"+strings.TrimSuffix(fGo208, "\n")+"\n")); code != 0 {
		t.Fatalf("create: %d\n%s", code, out)
	}
	switch alias {
	case "link.go":
		if err := os.Symlink("f.go", filepath.Join(root, alias)); err != nil {
			t.Skipf("no symlinks: %v", err)
		}
	case "F.go":
		if _, err := os.Stat(filepath.Join(root, alias)); err != nil {
			t.Skip("this filesystem is case-sensitive")
		}
	}
	return root
}

// A quiet read through an alias shows nothing, and recorded a partial record under
// the alias that covered the line after: the exact-path record then hid f.go's
// whole written one, and the rule did not apply.
func TestAQuietReadThroughAnAliasDoesNotLiftTheLineAfterRule(t *testing.T) {
	for _, alias := range []string{"link.go", "F.go"} {
		t.Run(alias, func(t *testing.T) {
			root := aliasRoot(t, alias)
			plan := strings.Replace(rep208, "@@ f.go", "@@ "+alias, 1)
			if out, code := runIn(t, root, "read", "--quiet", alias+":1-4"); code != 0 {
				t.Fatalf("quiet read: %d\n%s", code, out)
			}
			out, code := writeIn(t, root, "--no-check", planFile(t, plan))
			if code != 1 || !strings.Contains(out, "needs a served line after 3") {
				t.Fatalf("after a quiet read through %s: exit %d, want 1:\n%s", alias, code, out)
			}
			if got, _ := os.ReadFile(filepath.Join(root, "f.go")); string(got) != fGo208 {
				t.Errorf("a refused plan changed f.go: %q", got)
			}
			// The pair: a read that shows the line lifts it, through the same alias.
			if out, code := runIn(t, root, remedyOf(t, out)[len(remedyOf(t, out))-4:]...); code != 0 {
				t.Fatalf("the remedy read: %d\n%s", code, out)
			}
			if out, code := writeIn(t, root, "--no-check", planFile(t, plan)); code != 0 {
				t.Errorf("after the read the refusal named: exit %d\n%s", code, out)
			}
		})
	}
}

// Following the remedy made through an alias must not break an occurrence= check
// the plan had satisfied: the read recorded under the alias held none of the
// earlier showings, and the exact-path record hid the rest.
func TestTheAliasRemedyKeepsAnEarlierOccurrenceShowing(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	body := "start\none\nstart\ntwo\ncloser\nrest\n"
	if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ f.txt 0 create\n"+strings.TrimSuffix(body, "\n")+"\n")); code != 0 {
		t.Fatalf("create: %d\n%s", code, out)
	}
	if err := os.Symlink("f.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("no symlinks: %v", err)
	}
	if out, code := runIn(t, root, "read", "f.txt:1"); code != 0 {
		t.Fatalf("read: %d\n%s", code, out)
	}
	plan := planFile(t, "@@ link.txt /^start$/,+1 replace anchor=\"start\" occurrence=2\nnew\n")
	out, code := writeIn(t, root, "--no-check", plan)
	if code != 1 || !strings.Contains(out, "needs a served line after 4") {
		t.Fatalf("first write: exit %d, want the line-after refusal:\n%s", code, out)
	}
	argv := remedyOf(t, out)
	if out, code := runIn(t, root, argv[len(argv)-4:]...); code != 0 {
		t.Fatalf("the remedy read: %d\n%s", code, out)
	}
	if out, code := writeIn(t, root, "--no-check", plan); code != 0 {
		t.Errorf("after the read the refusal named: exit %d\n%s", code, out)
	}
}

// The read a refusal names runs from where the caller is, and the caller may be
// outside the root: it names the root when the command line gave one that is not
// the working directory, for the line-after rule and for ADR-203's stale number.
func TestTheRemedyNamesTheRootItWasGiven(t *testing.T) {
	root := aliasRoot(t, "f.go")
	if !regexp.MustCompile(`^[A-Za-z0-9._/-]+$`).MatchString(root) {
		t.Skipf("a root that needs quoting is described, not commanded: %q", root)
	}
	out, code := writeIn(t, root, "--no-check", planFile(t, rep208))
	want := "mrw --root " + root + " read --max-cols 0 f.go:1-4"
	if code != 1 || !strings.Contains(out, want) {
		t.Errorf("line-after refusal: exit %d, want %q in:\n%s", code, want, out)
	}
	// ADR-203's refusal, the same remedy: a number alone on a line no read has shown.
	out, code = writeIn(t, root, "--no-check", planFile(t, "@@ f.go 2 replace\n\tz()\n"))
	want = "mrw --root " + root + " read --max-cols 0 f.go:2"
	if code != 1 || !strings.Contains(out, want) {
		t.Errorf("stale-number refusal: exit %d, want %q in:\n%s", code, want, out)
	}
}
