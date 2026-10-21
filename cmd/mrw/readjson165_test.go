package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// ADR-165 T3. An adapter parsed mrw's report to learn what was served and at
// which sha; mrw_read already answers that over MCP, as content[1]. `mrw read
// --json` prints {"content", "receipt"}: content is the plain read's standard
// output byte for byte, and receipt is built by the function mrw_read uses, so
// for a read mrw_read answers whole the two receipts are equal. Through the
// built binary, since the comparison is between two processes' answers.
func TestReadJSONCarriesTheServedTextAndTheReceiptMrwReadSends(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "mrw.exe") // .exe: Windows exec will not launch an extensionless file
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	tree := t.TempDir()
	for name, body := range map[string]string{
		"a.txt":   "alpha\nbravo\ncharlie\n",
		"b.txt":   "delta\necho\n",
		"bin.dat": "a\x00binary\n", // a walk skips it: the receipt's skipped
	} {
		if err := os.WriteFile(filepath.Join(tree, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(stdin string, argv ...string) (stdout, stderr string, code int) {
		t.Helper()
		cmd := exec.Command(bin, argv...)
		cmd.Dir = tree
		cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+t.TempDir())
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
	type document struct {
		Content string         `json:"content"`
		Receipt map[string]any `json:"receipt"`
		Error   string         `json:"error"`
	}
	decode := func(s string) document {
		t.Helper()
		var d document
		if err := json.Unmarshal([]byte(s), &d); err != nil {
			t.Fatalf("not one JSON document: %v\n%s", err, s)
		}
		return d
	}
	// mrw_read's content[1], parsed, for the same arguments.
	mcpReceipt := func(args map[string]any) map[string]any {
		t.Helper()
		req, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call",
			"params": map[string]any{"name": "mrw_read", "arguments": args}})
		out, _, _ := run(string(req)+"\n", "mcp")
		var resp struct {
			Result struct {
				Content []struct{ Text string } `json:"content"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(strings.SplitN(out, "\n", 2)[0]), &resp); err != nil || len(resp.Result.Content) < 2 {
			t.Fatalf("mrw_read %v: %v\n%s", args, err, out)
		}
		var r map[string]any
		if err := json.Unmarshal([]byte(resp.Result.Content[1].Text), &r); err != nil {
			t.Fatalf("content[1] is not JSON: %v\n%s", err, resp.Result.Content[1].Text)
		}
		return r
	}

	for _, c := range []struct {
		argv []string
		args map[string]any
	}{
		{[]string{"a.txt:1-2", "b.txt"}, map[string]any{"specs": []string{"a.txt:1-2", "b.txt"}}},
		{[]string{"--grep", "a", "."}, map[string]any{"grep": "a", "specs": []string{"."}}},
		{[]string{"--grep", "zzz", "."}, map[string]any{"grep": "zzz", "specs": []string{"."}}},
	} {
		plain, _, plainCode := run("", append([]string{"read"}, c.argv...)...)
		out, _, code := run("", append([]string{"read", "--json"}, c.argv...)...)
		d := decode(out)
		if code != plainCode {
			t.Errorf("%q: --json exits %d, the plain read %d", c.argv, code, plainCode)
		}
		if d.Content != plain {
			t.Errorf("%q: content is not the plain read's standard output:\n got %q\nwant %q", c.argv, d.Content, plain)
		}
		if want := mcpReceipt(c.args); !reflect.DeepEqual(d.Receipt, want) {
			t.Errorf("%q: receipt differs from mrw_read's content[1]:\n got %v\nwant %v", c.argv, d.Receipt, want)
		}
		for _, k := range []string{"next_read", "index", "next_index"} {
			if _, ok := d.Receipt[k]; ok {
				t.Errorf("%q: the CLI receipt carries %s", c.argv, k)
			}
		}
	}

	// A budgeted read: its WITHHELD list stays on stderr, its count in problems.
	out, errOut, code := run("", "read", "--json", "--max-chars", "40", "a.txt", "b.txt")
	if d := decode(out); code != 1 || strings.Contains(d.Content, "WITHHELD") || !strings.Contains(errOut, "WITHHELD") || d.Receipt["problems"].(float64) < 1 {
		t.Errorf("--json --max-chars 40: exit %d, doc %+v, stderr %q", code, d, errOut)
	}
	// A refusal before any file is one error document, at the plain read's exit.
	// (A value the flag parser rejects, -C x, is refused before the action, plain,
	// as check --json's are: ADR-100.)
	plainOut, plainErr, plainCode := run("", "read", "-C", "1", "a.txt")
	out, _, code = run("", "read", "--json", "-C", "1", "a.txt")
	if d := decode(out); code != plainCode || d.Error == "" || !strings.Contains(plainErr, d.Error) || plainOut != "" {
		t.Errorf("a refusal: exit %d (plain %d), doc %+v, plain stderr %q", code, plainCode, d, plainErr)
	}
	if _, errOut, code := run("", "read", "--json", "--grep", "a", "--sub", "b", "."); code != 2 || !strings.Contains(errOut, "--json") {
		t.Errorf("--json --sub: exit %d, %q", code, errOut)
	}
	// --quiet: the quiet read's lines are the content, and it licenses as --quiet.
	quiet, _, _ := run("", "read", "--quiet", "a.txt")
	if out, _, _ := run("", "read", "--json", "--quiet", "a.txt"); decode(out).Content != quiet || !strings.Contains(quiet, "-- licensed") {
		t.Errorf("--json --quiet: content %q, want %q", decode(out).Content, quiet)
	}
}
