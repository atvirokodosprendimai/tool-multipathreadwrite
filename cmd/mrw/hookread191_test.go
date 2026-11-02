package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

const (
	nl       = "\n"
	dq       = "\""
	cr       = "\r"
	badBytes = "\xff\xfe\x00{"
)

// hookRoot is a checkout holding the files the captured payloads name (S1,
// testdata/hookread191): a.txt of 12 lines, long.txt with a 2,500-character
// line, and pic.png. The state home is a fresh temporary directory.
func hookRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv(`XDG_STATE_HOME`, t.TempDir())
	var a strings.Builder
	for i := 1; i <= 12; i++ {
		a.WriteString(`line ` + strconv.Itoa(i) + ` of a.txt` + nl)
	}
	files := map[string]string{
		`a.txt`:    a.String(),
		`long.txt`: `short one` + nl + strings.Repeat(`x`, 2500) + nl + `short three` + nl,
		`pic.png`:  strings.Repeat(`z`, 72),
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// capturedPayloads are the four PostToolUse payloads S1 captured from Claude
// Code 2.1.316, as objects, with {ROOT} replaced: 0 a whole file, 1 offset 4
// limit 3, 2 a 2,500-character line, 3 an image.
func capturedPayloads(t *testing.T, root string) []map[string]any {
	t.Helper()
	f, err := os.Open(filepath.Join(`testdata`, `hookread191`, `payloads.jsonl`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	esc, _ := json.Marshal(root)
	quoted := strings.Trim(string(esc), dq)
	var out []map[string]any
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1048576)
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal([]byte(strings.ReplaceAll(sc.Text(), `{ROOT}`, quoted)), &m); err != nil {
			t.Fatal(err)
		}
		out = append(out, m)
	}
	if len(out) != 4 {
		t.Fatalf(`testdata/hookread191/payloads.jsonl holds %d payloads, want 4`, len(out))
	}
	return out
}

// file digs tool_response.file out of a payload, for a case to alter.
func file(p map[string]any) map[string]any {
	return p[`tool_response`].(map[string]any)[`file`].(map[string]any)
}

func jsonOf(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// runHook runs `mrw [-C root] hook read` in-process with payload on standard
// input, and returns what it printed on standard output and standard error and
// its exit status. An empty root leaves the root to the payload's cwd.
func runHook(t *testing.T, root string, payload []byte) (stdout, stderr string, code int) {
	t.Helper()
	in := filepath.Join(t.TempDir(), `payload.json`)
	if err := os.WriteFile(in, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(in)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	old := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = old }()
	argv := []string{`mrw`}
	if root != `` {
		argv = append(argv, `-C`, root)
	}
	argv = append(argv, `hook`, `read`)
	stderr = stderrOf(t, func() {
		var runErr error
		stdout, runErr = captureStdout(t, func() error {
			cmd := rootCommand()
			var sink strings.Builder
			cmd.Writer, cmd.ErrWriter = &sink, &sink
			return cmd.Run(t.Context(), argv)
		})
		if runErr != nil {
			code = exitCode(runErr)
			stdout += runErr.Error()
		}
	})
	return stdout, stderr, code
}

// servedOf is the ledger's observation of one file.
func servedOf(t *testing.T, root, path string) (seen.Observation, bool) {
	t.Helper()
	l, err := seen.Snapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	o, ok := l[path]
	return o, ok
}

// wantLines asserts which of lines 1..n the ledger licenses for path: exactly
// the ones in want. A file the ledger does not hold licenses none.
func wantLines(t *testing.T, label, root, path string, n int, want ...int) {
	t.Helper()
	o, ok := servedOf(t, root, path)
	for line := 1; line <= n; line++ {
		expect := false
		for _, w := range want {
			expect = expect || w == line
		}
		if got := ok && o.ServedLine(line); got != expect {
			t.Errorf(`%s: %s line %d licensed = %v, want %v (ledger: %v)`, label, path, line, got, expect, o.Served())
		}
	}
}

func quiet(t *testing.T, label, out, errOut string, code int) {
	t.Helper()
	if code != 0 || out != `` || errOut != `` {
		t.Errorf(`%s: exit %d, stdout %q, stderr %q; want exit 0 and nothing printed`, label, code, out, errOut)
	}
}

func ledgerEmpty(t *testing.T, root string) {
	t.Helper()
	if l, err := seen.Snapshot(root); err != nil || len(l) != 0 {
		t.Errorf(`the ledger holds %v (%v), want nothing`, l, err)
	}
}

// ADR-191 Decision 1 and 2 through the command, on payloads Claude Code 2.1.316
// really sent: the lines the Read returned are licensed as they are on disk,
// and no other line is; anything else the command is given records nothing,
// prints nothing and exits 0.
func TestAHostReadLicensesOnlyTheLinesItShowedAsTheyAre(t *testing.T) {
	t.Run(`a whole file licenses its lines and not the empty one after the last newline`, func(t *testing.T) {
		root := hookRoot(t)
		out, errOut, code := runHook(t, root, jsonOf(t, capturedPayloads(t, root)[0]))
		quiet(t, `whole`, out, errOut, code)
		// The captured content ends in an empty element: a file of 12 lines.
		wantLines(t, `whole`, root, `a.txt`, 14, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12)
	})

	t.Run(`an offset and limit license those lines, and a plan to one of them applies`, func(t *testing.T) {
		root := hookRoot(t)
		out, errOut, code := runHook(t, root, jsonOf(t, capturedPayloads(t, root)[1]))
		quiet(t, `offset`, out, errOut, code)
		wantLines(t, `offset`, root, `a.txt`, 12, 4, 5, 6)
		plan := filepath.Join(t.TempDir(), `p.plan`)
		// The refused plan first: a file mrw has just written is wholly licensed.
		if err := os.WriteFile(plan, []byte(`@@ a.txt 9 replace`+nl+`NINE`+nl), 0o644); err != nil {
			t.Fatal(err)
		}
		if o, c := runIn(t, root, `write`, `--no-check`, plan); c != 1 {
			t.Errorf(`the pair: a plan to a line outside the read exited %d, want 1: %s`, c, o)
		}
		if err := os.WriteFile(plan, []byte(`@@ a.txt 5 replace`+nl+`FIVE`+nl), 0o644); err != nil {
			t.Fatal(err)
		}
		if o, c := runIn(t, root, `write`, `--no-check`, plan); c != 0 {
			t.Errorf(`a plan to a licensed line exited %d: %s`, c, o)
		}
	})

	t.Run(`the lines returned are licensed, not the lines asked for`, func(t *testing.T) {
		root := hookRoot(t)
		p := capturedPayloads(t, root)[1]
		p[`tool_input`].(map[string]any)[`limit`] = 9 // asks for 4-12, was given 4-6
		out, errOut, code := runHook(t, root, jsonOf(t, p))
		quiet(t, `asked for more`, out, errOut, code)
		wantLines(t, `asked for more`, root, `a.txt`, 12, 4, 5, 6)
	})

	t.Run(`a 2,500-character line the host showed whole is licensed`, func(t *testing.T) {
		root := hookRoot(t)
		out, errOut, code := runHook(t, root, jsonOf(t, capturedPayloads(t, root)[2]))
		quiet(t, `long`, out, errOut, code)
		wantLines(t, `long`, root, `long.txt`, 4, 1, 2, 3)
	})

	t.Run(`a line the host cut is not licensed`, func(t *testing.T) {
		root := hookRoot(t)
		p := capturedPayloads(t, root)[2]
		f := file(p)
		f[`content`] = strings.Replace(f[`content`].(string), strings.Repeat(`x`, 2500), strings.Repeat(`x`, 2000), 1)
		out, errOut, code := runHook(t, root, jsonOf(t, p))
		quiet(t, `cut`, out, errOut, code)
		wantLines(t, `cut`, root, `long.txt`, 4, 1, 3)
	})

	t.Run(`a line that changed after the read is not licensed`, func(t *testing.T) {
		root := hookRoot(t)
		p := capturedPayloads(t, root)[0]
		b, _ := os.ReadFile(filepath.Join(root, `a.txt`))
		changed := strings.Replace(string(b), `line 5 of a.txt`, `line 5 CHANGED`, 1)
		if err := os.WriteFile(filepath.Join(root, `a.txt`), []byte(changed), 0o644); err != nil {
			t.Fatal(err)
		}
		out, errOut, code := runHook(t, root, jsonOf(t, p))
		quiet(t, `changed`, out, errOut, code)
		wantLines(t, `changed`, root, `a.txt`, 12, 1, 2, 3, 4, 6, 7, 8, 9, 10, 11, 12)
	})

	t.Run(`a payload with no returned lines records nothing`, func(t *testing.T) {
		root := hookRoot(t)
		image := capturedPayloads(t, root)[3]
		noStart := capturedPayloads(t, root)[0]
		delete(file(noStart), `startLine`)
		zeroStart := capturedPayloads(t, root)[0]
		file(zeroStart)[`startLine`] = 0
		textStart := capturedPayloads(t, root)[0]
		file(textStart)[`startLine`] = `1`
		numContent := capturedPayloads(t, root)[0]
		file(numContent)[`content`] = 7
		asError := capturedPayloads(t, root)[0]
		asError[`tool_response`] = `File does not exist.`
		isErr := capturedPayloads(t, root)[0]
		isErr[`tool_response`].(map[string]any)[`is_error`] = true
		delete(file(isErr), `content`)
		cases := map[string]map[string]any{
			`an image`:                image,
			`no startLine`:            noStart,
			`startLine 0`:             zeroStart,
			`startLine as text`:       textStart,
			`content that is no text`: numContent,
			`an error as a string`:    asError,
			`an error object`:         isErr,
		}
		for name, p := range cases {
			out, errOut, code := runHook(t, root, jsonOf(t, p))
			quiet(t, name, out, errOut, code)
		}
		ledgerEmpty(t, root)
	})

	t.Run(`another tool records nothing`, func(t *testing.T) {
		root := hookRoot(t)
		for _, tool := range []string{`Grep`, `Edit`, `Bash`, `read`, ``} {
			p := capturedPayloads(t, root)[0]
			p[`tool_name`] = tool
			out, errOut, code := runHook(t, root, jsonOf(t, p))
			quiet(t, `tool `+tool, out, errOut, code)
		}
		ledgerEmpty(t, root)
	})

	t.Run(`a path outside the root, in the version-control directory or in mrw's state records nothing`, func(t *testing.T) {
		root := hookRoot(t)
		outside := t.TempDir()
		vcs := `.g` + `it`
		if err := os.MkdirAll(filepath.Join(root, vcs), 0o755); err != nil {
			t.Fatal(err)
		}
		dir, err := state.Dir(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		paths := map[string]string{
			`outside`: filepath.Join(outside, `o.txt`),
			`vcs`:     filepath.Join(root, vcs, `config`),
			`state`:   filepath.Join(dir, `notes.txt`),
		}
		for name, path := range paths {
			if err := os.WriteFile(path, []byte(`line 1 of a.txt`+nl), 0o644); err != nil {
				t.Fatal(err)
			}
			p := capturedPayloads(t, root)[0]
			p[`tool_input`].(map[string]any)[`file_path`] = path
			f := file(p)
			f[`filePath`] = path
			f[`content`] = `line 1 of a.txt`
			out, errOut, code := runHook(t, root, jsonOf(t, p))
			quiet(t, name, out, errOut, code)
		}
		ledgerEmpty(t, root)
	})

	t.Run(`a malformed payload records nothing and exits 0`, func(t *testing.T) {
		root := hookRoot(t)
		whole := jsonOf(t, capturedPayloads(t, root)[0])
		inputs := map[string][]byte{
			`empty`:     nil,
			`not json`:  []byte(`not json at all`),
			`truncated`: whole[:len(whole)/2],
			`an array`:  []byte(`[]`),
			`null`:      []byte(`null`),
			`a number`:  []byte(`7`),
			`bad bytes`: []byte(badBytes),
		}
		for name, in := range inputs {
			out, errOut, code := runHook(t, root, in)
			quiet(t, name, out, errOut, code)
		}
		ledgerEmpty(t, root)
	})

	t.Run(`the payload's cwd is the root when none is given`, func(t *testing.T) {
		root := hookRoot(t)
		out, errOut, code := runHook(t, ``, jsonOf(t, capturedPayloads(t, root)[1]))
		quiet(t, `cwd`, out, errOut, code)
		wantLines(t, `cwd`, root, `a.txt`, 12, 4, 5, 6)
	})

	t.Run(`a held ledger lock returns within the deadline and records nothing`, func(t *testing.T) {
		root := hookRoot(t)
		top, _ := state.Checkout(root)
		release, err := state.Hold(top, seen.Name+`.lock`)
		if err != nil {
			t.Fatal(err)
		}
		type result struct {
			out, errOut string
			code        int
		}
		done := make(chan result, 1)
		go func() {
			out, errOut, code := runHook(t, root, jsonOf(t, capturedPayloads(t, root)[0]))
			done <- result{out, errOut, code}
		}()
		var r result
		select {
		case r = <-done:
		case <-time.After(10 * time.Second):
			release()
			<-done
			t.Fatal(`the command was still waiting for a held ledger lock after 10 s`)
		}
		release()
		if r.code != 0 || r.out != `` {
			t.Errorf(`exit %d, stdout %q; want exit 0 and nothing on stdout`, r.code, r.out)
		}
		if n := len(strings.Split(strings.TrimSpace(r.errOut), nl)); n != 1 || !strings.Contains(r.errOut, seen.Name+`.lock`) {
			t.Errorf(`stderr %q; want one line naming the lock`, r.errOut)
		}
		ledgerEmpty(t, root)
	})

	t.Run(`a legacy ./.mrw whose destination lock is held does not hold the command`, func(t *testing.T) {
		root := hookRoot(t)
		cwd := t.TempDir()
		if err := os.MkdirAll(filepath.Join(cwd, state.LegacyDir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cwd, state.LegacyDir, seen.Name), []byte(`legacy`+nl), 0o644); err != nil {
			t.Fatal(err)
		}
		payload := jsonOf(t, capturedPayloads(t, root)[1]) // read before the chdir: its path is relative
		t.Chdir(cwd)
		release, err := state.Hold(`.`, seen.Name+`.lock`)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan int, 1)
		go func() {
			_, _, code := runHook(t, root, payload)
			done <- code
		}()
		select {
		case code := <-done:
			if code != 0 {
				t.Errorf(`exit %d, want 0`, code)
			}
		case <-time.After(10 * time.Second):
			release()
			<-done
			t.Fatal(`the command waited on the legacy migration's lock`)
		}
		release()
		wantLines(t, `legacy`, root, `a.txt`, 12, 4, 5, 6)
	})

	t.Run(`a newer mrw's ledger is left alone and said so once`, func(t *testing.T) {
		root := newerRoot(t)
		if err := os.WriteFile(filepath.Join(root, `a.txt`), []byte(`line 1 of a.txt`+nl), 0o644); err != nil {
			t.Fatal(err)
		}
		ledger, _ := seen.ReadPath(root)
		before, _ := os.ReadFile(ledger)
		p := capturedPayloads(t, root)[0]
		file(p)[`content`] = `line 1 of a.txt`
		out, errOut, code := runHook(t, root, jsonOf(t, p))
		if code != 0 || out != `` || strings.Count(errOut, `written by a newer mrw`) != 1 || strings.Count(errOut, nl) != 1 {
			t.Errorf(`exit %d, stdout %q, stderr %q; want exit 0 and the newer-ledger notice once`, code, out, errOut)
		}
		if after, _ := os.ReadFile(ledger); string(after) != string(before) {
			t.Errorf(`the newer ledger was changed: %q`, after)
		}
	})

	t.Run(`a CRLF file's lines are licensed as the host showed them, without the CR`, func(t *testing.T) {
		root := hookRoot(t)
		crlf := `crlf one` + cr + nl + `crlf two` + cr + nl + `crlf three` + cr + nl
		path := filepath.Join(root, `crlf.txt`)
		if err := os.WriteFile(path, []byte(crlf), 0o644); err != nil {
			t.Fatal(err)
		}
		p := capturedPayloads(t, root)[0]
		p[`tool_input`].(map[string]any)[`file_path`] = path
		f := file(p)
		f[`filePath`] = path
		f[`content`] = `crlf one` + nl + `crlf two` + nl + `crlf three` + nl
		out, errOut, code := runHook(t, root, jsonOf(t, p))
		quiet(t, `crlf`, out, errOut, code)
		// Three lines; the empty element after the last newline is no line.
		wantLines(t, `crlf`, root, `crlf.txt`, 5, 1, 2, 3)
	})

	t.Run(`a stray CR the host did not show keeps its line unlicensed`, func(t *testing.T) {
		root := hookRoot(t)
		mixed := `one` + nl + `two` + cr + `x` + nl + `three` + nl
		path := filepath.Join(root, `mixed.txt`)
		if err := os.WriteFile(path, []byte(mixed), 0o644); err != nil {
			t.Fatal(err)
		}
		p := capturedPayloads(t, root)[0]
		p[`tool_input`].(map[string]any)[`file_path`] = path
		f := file(p)
		f[`filePath`] = path
		f[`content`] = `one` + nl + `twox` + nl + `three` + nl
		out, errOut, code := runHook(t, root, jsonOf(t, p))
		quiet(t, `mixed`, out, errOut, code)
		wantLines(t, `mixed`, root, `mixed.txt`, 4, 1, 3)
	})
}
