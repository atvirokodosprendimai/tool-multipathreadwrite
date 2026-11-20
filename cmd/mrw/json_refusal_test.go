package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// dirReadCause is the error text a read of a directory ends with: Windows
// answers ReadFile on a directory "Incorrect function", unix "is a directory".
var dirReadCause = map[bool]string{true: "Incorrect function", false: "is a directory"}[runtime.GOOS == "windows"]

// jsonRefusal parses a --json refusal and checks the shape every one shares:
// exit 2, applied false, files and hunks empty arrays, an error naming want.
func jsonRefusal(t *testing.T, out string, code int, want string) {
	t.Helper()
	if code != exitUsage {
		t.Fatalf("exit %d, want %d:\n%s", code, exitUsage, out)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("--json printed something that is not one JSON document: %v\n%s", err, out)
	}
	if doc["applied"] != false {
		t.Errorf("applied = %v, want false", doc["applied"])
	}
	for _, k := range []string{"files", "hunks"} {
		if a, ok := doc[k].([]any); !ok || len(a) != 0 {
			t.Errorf("%s = %#v, want an empty array (null breaks .%s[] in jq)", k, doc[k], k)
		}
	}
	if e, _ := doc["error"].(string); !strings.Contains(e, want) {
		t.Errorf("error = %q, want it to name %q", e, want)
	}
}

// ADR-072 T3. --json printed text on a plan that did not parse, and on every
// other refusal between opening the plan and applying it.
func TestAPlanThatDoesNotParseIsAJSONDocumentUnderJSON(t *testing.T) {
	root := checkTree(t)
	out, code := writeIn(t, root, "--json", planFile(t, "garbage\n"))
	jsonRefusal(t, out, code, "p.mrw")
}

func TestAMissingBodyFileIsAJSONDocumentUnderJSON(t *testing.T) {
	root := checkTree(t)
	out, code := writeIn(t, root, "--json", planFile(t, "@@ a.go 2 replace anchor=\"func A\" body=@nope.txt\n"))
	jsonRefusal(t, out, code, "nope.txt")
}

func TestAMalformedHarnessIsAJSONDocumentUnderJSON(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := checkTree(t)
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, "--json", "--check", planFile(t, goPlan))
	jsonRefusal(t, out, code, ".quality-harness.json")
}

// A plan naming a directory printed nothing under --json: only a result that
// already had hunks was rendered (review of #229, B2). Since #703 the load
// failure is carried on its hunk, so the document is a receipt with that
// verdict, exit 2, applied false.
func TestAFilesystemFailureBeforeAnyHunkIsAJSONDocumentUnderJSON(t *testing.T) {
	root := checkTree(t)
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, "--json", planFile(t, "@@ sub 1 replace\nX\n"))
	if code != exitUsage {
		t.Fatalf("exit %d, want %d:\n%s", code, exitUsage, out)
	}
	var doc struct {
		Applied bool   `json:"applied"`
		Error   string `json:"error"`
		Hunks   []struct {
			Path, Status, Reason string
		} `json:"hunks"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("--json printed something that is not one JSON document: %v\n%s", err, out)
	}
	if doc.Applied || !strings.Contains(doc.Error, "sub") {
		t.Errorf("applied %v, error %q, want false and an error naming sub", doc.Applied, doc.Error)
	}
	if len(doc.Hunks) != 1 || doc.Hunks[0].Path != "sub" || doc.Hunks[0].Status != "failed" || !strings.Contains(doc.Hunks[0].Reason, dirReadCause) {
		t.Errorf("hunks %+v, want sub failed with its load error", doc.Hunks)
	}
}
