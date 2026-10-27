package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

const project178Doc = "{\n  \"name\": \"demo\",\n  \"scripts\": {\n    \"build\": \"go build\"\n  },\n  \"list\": [1, 2]\n}\n"

// project178Tree is a temporary root holding f.json (project178Doc) and g.json.
func project178Tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range map[string]string{"f.json": project178Doc, "g.json": `{"name":"other"}` + "\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// project178Run runs `mrw -C root argv` in process with stdin as given and
// returns standard output (with the exit error's text, as runIn does),
// standard error and the exit code.
func project178Run(t *testing.T, root, stdin string, argv ...string) (string, string, int) {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	done := make(chan []byte, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()
	var out string
	var code int
	withStdin(t, stdin, func() { out, code = runIn(t, root, argv...) })
	os.Stderr = old
	_ = w.Close()
	errOut := string(<-done)
	_ = r.Close()
	return out, errOut, code
}

// ADR-178 Decision 3. A projection licenses no line and records nothing, for a
// named file and for standard input: a plan replacing the projected value's
// line is refused as unread, the ledger holds no entry, and the path address of
// the same value, read after it, still licenses the line.
func TestAProjectionLicensesNoLine(t *testing.T) {
	root := project178Tree(t)
	out, _, code := project178Run(t, root, "", "read", "--json-path", "$.name", "f.json")
	if code != 0 || !strings.Contains(out, "==> f.json  ") || !strings.HasSuffix(out, `$.name = "demo"`+"\n") || strings.Contains(out, "@@") {
		t.Fatalf("the projection exited %d:\n%s", code, out)
	}
	if l, err := seen.Snapshot(root); err != nil || len(l) != 0 {
		t.Errorf("the ledger holds %v (%v) after a projection, want nothing", l, err)
	}
	plan := planFile(t, "@@ f.json 2 replace\n  \"name\": \"x\",\n")
	if out, code = runIn(t, root, "write", "--no-check", plan); code != 1 || !strings.Contains(out, "has not been read") {
		t.Errorf("a write to the projected line exited %d, want 1 refused as unread:\n%s", code, out)
	}
	if out, code = runIn(t, root, "read", "f.json:$.name"); code != 0 || !strings.Contains(out, "@@ 2-3") {
		t.Fatalf("the path address exited %d:\n%s", code, out)
	}
	if out, code = runIn(t, root, "write", "--no-check", plan); code != 0 {
		t.Errorf("the same write after the path address exited %d, want 0:\n%s", code, out)
	}

	// Standard input: the same lines, and the ledger is as it was.
	before, _ := seen.Snapshot(root)
	out, _, code = project178Run(t, root, `{"a":{"b":[1, 2]}}`+"\n", "read", "--json-path", "$.a.b", "-")
	if code != 0 || !strings.HasPrefix(out, "==> -  ") || !strings.HasSuffix(out, "$.a.b = [1,2]\n") {
		t.Errorf("a projection of standard input exited %d:\n%s", code, out)
	}
	if after, _ := seen.Snapshot(root); !reflect.DeepEqual(before, after) {
		t.Errorf("the ledger moved by a projection of standard input: %v -> %v", before, after)
	}
}

// ADR-178 Decision 1. An expression must begin $. or $[ and parse: an empty
// one, x.a and $. are usage errors, exit 2, naming the expression and the
// forms a step takes, before anything is read; two uses of the flag keep
// command-line order; a comma inside a quoted step reaches the file as part of
// the name.
func TestAProjectionExpressionMustBeAPath(t *testing.T) {
	root := project178Tree(t)
	for _, c := range []struct{ expr, named string }{{"", `""`}, {"x.a", `"x.a"`}, {"a", `"a"`}, {"$.", `"$."`}, {"$.a,", `""`}} {
		out, _, code := project178Run(t, root, "", "read", "--json-path", c.expr, "f.json")
		if code != 2 || !strings.Contains(out, c.named) || !strings.Contains(out, `["name"]`) || strings.Contains(out, "==>") {
			t.Errorf("--json-path %q exited %d, want 2 naming %s and the step forms, nothing read:\n%s", c.expr, code, c.named, out)
		}
	}
	out, _, code := project178Run(t, root, "", "read", "--json-path", "$.scripts.build", "--json-path", "$.name,$.list[1]", "f.json")
	if want := "$.scripts.build = \"go build\"\n$.name = \"demo\"\n$.list[1] = 2\n"; code != 0 || !strings.HasSuffix(out, want) {
		t.Errorf("two uses of the flag exited %d, want %q last:\n%s", code, want, out)
	}
	quoted := filepath.Join(root, "q.json")
	if err := os.WriteFile(quoted, []byte(`{"a,b": 1, "x\"y": 2, "c": 3}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code = project178Run(t, root, "", "read", "--json-path", `$["a,b"],$["x\"y"],$.c`, "q.json")
	if want := "$[\"a,b\"] = 1\n$[\"x\\\"y\"] = 2\n$.c = 3\n"; code != 0 || !strings.HasSuffix(out, want) {
		t.Errorf("a quoted comma exited %d, want %q last:\n%s", code, want, out)
	}
}

// ADR-178 Decisions 4 and 5. A value the budget does not hold is withheld and
// listed on standard error with its expression, and the read exits 1; the value
// that fit is printed.
func TestAProjectionTheBudgetDropsIsListed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.json"), []byte(`{"a":"xxxxxxxxxx","b":"yyyyyyyyyy"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	whole, _, code := project178Run(t, root, "", "read", "--json-path", "$.a,$.b", "f.json")
	if code != 0 {
		t.Fatalf("the unbudgeted projection exited %d:\n%s", code, whole)
	}
	fits := len([]rune(whole)) - len([]rune(`$.b = "yyyyyyyyyy"`+"\n"))
	out, errOut, code := project178Run(t, root, "", "read", "--json-path", "$.a,$.b", "--max-chars", strconv.Itoa(fits), "f.json")
	if code != 1 || !strings.Contains(out, `$.a = "xxxxxxxxxx"`) || strings.Contains(out, "$.b") || !strings.Contains(out, "--max-chars") {
		t.Errorf("--max-chars %d exited %d, want 1 with only $.a printed:\n%s", fits, code, out)
	}
	if want := "f.json:$.b: WITHHELD: --max-chars reached"; !strings.Contains(errOut, want) || strings.Contains(errOut, "$.a") {
		t.Errorf("standard error is %q, want it to list %q alone", errOut, want)
	}
	// --max-cols cuts a long projected line to its window, as any line.
	out, _, code = project178Run(t, root, "", "read", "--json-path", "$.a", "--max-cols", "10", "f.json")
	if code != 0 || strings.Contains(out, "xxxxxxxxxx") || !strings.Contains(out, "$.a = \"xxx") {
		t.Errorf("--max-cols 10 exited %d, want the line cut:\n%s", code, out)
	}
}

// ADR-178 Decision 5. Every flag that serves, folds or licenses lines is
// refused beside a projection, exit 2, by name, with nothing read; a PATH
// carrying an address is refused naming it; the flags that act on a
// projection are accepted, several files included; and `-` with --json is
// refused as it is without a projection.
func TestAProjectionRefusesTheFlagsThatServeLines(t *testing.T) {
	root := project178Tree(t)
	for _, c := range []struct {
		name string
		argv []string
	}{
		{"--grep", []string{"--grep", "name", "f.json"}},
		{"--ast-grep", []string{"--ast-grep", "name", "f.json"}},
		{"--sub", []string{"--sub", "x", "f.json"}},
		{"--rewrite", []string{"--rewrite", "x", "f.json"}},
		{"--count", []string{"--count", "f.json"}},
		{"--stat", []string{"--stat", "f.json"}},
		{"--quiet", []string{"--quiet", "f.json"}},
		{"-C", []string{"-C", "1", "f.json"}},
		{"--max-lines", []string{"--max-lines", "3", "f.json"}},
		{"--if-changed", []string{"--if-changed", "f.json=00000000", "f.json"}},
		{"f.json:3-9", []string{"f.json:3-9"}},
		{"f.json:$.name", []string{"f.json:$.name"}},
		{"-:2", []string{"-:2"}},
	} {
		argv := append([]string{"read", "--json-path", "$.name"}, c.argv...)
		out, _, code := project178Run(t, root, "{}\n", argv...)
		if code != 2 || !strings.Contains(out, c.name) || strings.Contains(out, "==>") {
			t.Errorf("%v exited %d, want 2 naming %s, nothing read:\n%s", argv, code, c.name, out)
		}
	}
	out, _, code := project178Run(t, root, `{"a":1}`, "read", "--json-path", "$.a", "--json", "-")
	if code != 2 || !strings.Contains(out, "--json") {
		t.Errorf("- with --json exited %d, want 2 naming --json:\n%s", code, out)
	}
	out, _, code = project178Run(t, root, "", "read", "--json-path", "$.name", "--no-numbers", "--max-cols", "40", "--max-chars", "400", "f.json", "g.json")
	if code != 0 || !strings.Contains(out, `$.name = "demo"`) || !strings.Contains(out, `$.name = "other"`) {
		t.Errorf("two files with the flags that act on a projection exited %d:\n%s", code, out)
	}
}

// ADR-178 Decision 5. Under --json the projection is the document's content,
// exactly the plain read's output, and the receipt's keys are those of any
// read, its observed empty.
func TestAProjectionUnderJSONIsTheContent(t *testing.T) {
	root := project178Tree(t)
	plain, _, code := project178Run(t, root, "", "read", "--json-path", "$.name,$.list[1]", "f.json")
	if code != 0 {
		t.Fatalf("the plain projection exited %d:\n%s", code, plain)
	}
	out, _, code := project178Run(t, root, "", "read", "--json", "--json-path", "$.name,$.list[1]", "f.json")
	var doc struct {
		Content string         `json:"content"`
		Receipt map[string]any `json:"receipt"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil || code != 0 {
		t.Fatalf("--json exited %d: %v\n%s", code, err, out)
	}
	if doc.Content != plain {
		t.Errorf("content is %q, want the plain output %q", doc.Content, plain)
	}
	if obs, ok := doc.Receipt["observed"].(map[string]any); !ok || len(obs) != 0 {
		t.Errorf("observed is %v, want an empty object", doc.Receipt["observed"])
	}
	var ref struct {
		Receipt map[string]any `json:"receipt"`
	}
	out, _, _ = project178Run(t, root, "", "read", "--json", "f.json:1")
	if err := json.Unmarshal([]byte(out), &ref); err != nil {
		t.Fatal(err)
	}
	for k := range ref.Receipt {
		if _, ok := doc.Receipt[k]; !ok {
			t.Errorf("the receipt of a projection lacks %q, which any read carries", k)
		}
	}
	for k := range doc.Receipt {
		if _, ok := ref.Receipt[k]; !ok {
			t.Errorf("the receipt of a projection has %q, which no other read carries", k)
		}
	}
}
