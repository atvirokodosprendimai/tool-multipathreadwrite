package mcp

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

const goEdit113 = "@@ a.go 2 replace anchor=\"func A\"\nfunc A() { _ = 1 }\n"

// checkTree113 is a checkout holding a.go and notes.md, both wholly known, and
// the given harness.
func checkTree113(t *testing.T, harness string) string {
	t.Helper()
	files := map[string]string{"a.go": "package a\nfunc A() {}\n", "notes.md": "# notes\nline two\n"}
	if harness != "" {
		files[".quality-harness.json"] = harness
	}
	return licensed(t, files)
}

func tally113(t *testing.T, root string) authoring.Tally {
	t.Helper()
	tally, err := authoring.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return tally
}

// ADR-113 T2. mrw_write ran no check, so a caller with no shell learned what
// was written and nothing about whether it still built. It runs the check by
// ADR-054's rule, returns the verdict in the receipt, and counts it as the CLI
// does; check: false turns it off. A check that ran and failed is not
// isError — the write landed and the verdict is data — while one that could
// not run is.
func TestAnMCPWriteRunsTheCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the checks are POSIX shell lines")
	}
	t.Run("a code write runs the check and reports its failure", func(t *testing.T) {
		root := checkTree113(t, `{"check":"echo boom; exit 3"}`)
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113})
		sc := structured(t, res)
		chk, _ := sc["check"].(map[string]any)
		if chk == nil || chk["ran"] != true || chk["exit_code"] != float64(3) {
			t.Fatalf("check %v, want one that ran and exited 3:\n%v", chk, res)
		}
		if res["isError"] == true {
			t.Errorf("a check that ran and failed is isError: the write landed and the verdict is data")
		}
		if text := firstText(t, res); !strings.Contains(strings.SplitN(text, "\n", 2)[0], "check") {
			t.Errorf("the text does not lead with the check's verdict:\n%s", text)
		}
		if got := tally113(t, root); got["failed_check"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want failed_check 1", got)
		}
	})
	t.Run("check false runs none", func(t *testing.T) {
		root := checkTree113(t, `{"check":"exit 3"}`)
		sc := structured(t, call(t, root, "mrw_write", map[string]any{"plan": goEdit113, "check": false}))
		if _, has := sc["check"]; has || sc["applied"] != true {
			t.Fatalf("check: false still checked, or did not apply: %v", sc)
		}
		if got := tally113(t, root); got["applied"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want applied 1", got)
		}
	})
	t.Run("a prose write runs none", func(t *testing.T) {
		root := checkTree113(t, `{"check":"exit 3"}`)
		sc := structured(t, call(t, root, "mrw_write", map[string]any{"plan": "@@ notes.md 2 replace anchor=\"line two\"\nline 2\n"}))
		if _, has := sc["check"]; has {
			t.Fatalf("a markdown write ran the check: %v", sc)
		}
	})
	t.Run("a passing check is reported and priced as held", func(t *testing.T) {
		root := checkTree113(t, `{"check":"exit 0"}`)
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113})
		chk, _ := structured(t, res)["check"].(map[string]any)
		if chk == nil || chk["ran"] != true || chk["exit_code"] != float64(0) || res["isError"] == true {
			t.Fatalf("check %v isError %v, want a passing check and no error", chk, res["isError"])
		}
		if got := tally113(t, root); got["applied"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want applied 1", got)
		}
	})
	t.Run("a check that cannot start is isError with the receipt", func(t *testing.T) {
		root := checkTree113(t, `{"check":"exit 0"}`)
		t.Setenv("PATH", t.TempDir())
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113})
		chk, _ := structured(t, res)["check"].(map[string]any)
		if res["isError"] != true || chk == nil || chk["ran"] != false || structured(t, res)["applied"] != true {
			t.Fatalf("isError %v check %v, want isError with an applied receipt and a check that did not run", res["isError"], chk)
		}
		if got := tally113(t, root); got["check_not_run"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want check_not_run 1", got)
		}
	})
	t.Run("a check that cannot run is isError with the receipt", func(t *testing.T) {
		root := checkTree113(t, `{"check":"exit 0"}`)
		t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113})
		sc := structured(t, res)
		if res["isError"] != true || sc["applied"] != true || sc["error"] == nil {
			t.Fatalf("isError %v, want isError with an applied receipt naming the error: %v", res["isError"], sc)
		}
		if got := tally113(t, root); got["check_not_run"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want check_not_run 1", got)
		}
	})
	t.Run("a malformed harness refuses the write before anything lands", func(t *testing.T) {
		root := checkTree113(t, `{`)
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113})
		if res["isError"] != true || !strings.Contains(firstText(t, res), "nothing was written") {
			t.Fatalf("want a refusal saying nothing was written: %v", res)
		}
		if got := tally113(t, root); got["refused_apply"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want refused_apply 1", got)
		}
		root = checkTree113(t, `{`)
		if sc := structured(t, call(t, root, "mrw_write", map[string]any{"plan": goEdit113, "check": false})); sc["applied"] != true {
			t.Errorf("check: false read the harness anyway: %v", sc)
		}
	})
	t.Run("an oversized receipt drops the check's tail and keeps its verdict", func(t *testing.T) {
		restore := MaxResultChars
		MaxResultChars = 20000
		t.Cleanup(func() { MaxResultChars = restore })
		root := checkTree113(t, `{"check":"i=0; while [ $i -lt 30 ]; do printf '%01000d\\n' 0; i=$((i+1)); done; exit 3"}`)
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113})
		sc := structured(t, res)
		chk, _ := sc["check"].(map[string]any)
		if chk == nil || chk["ran"] != true || chk["exit_code"] != float64(3) {
			t.Fatalf("check %v, want the verdict kept: %v", chk, sc)
		}
		if _, has := chk["tail"]; has {
			t.Errorf("the tail survived a receipt over the ceiling")
		}
		if el, _ := sc["elided"].(string); !strings.Contains(el, "check") {
			t.Errorf("elided %q does not say the check's tail went", el)
		}
	})
	t.Run("a passing check's dropped tail is kept in a log the receipt names", func(t *testing.T) {
		restore := MaxResultChars
		MaxResultChars = 20000
		t.Cleanup(func() { MaxResultChars = restore })
		root := checkTree113(t, `{"check":"i=0; while [ $i -lt 30 ]; do printf '%01000d\\n' 0; i=$((i+1)); done; exit 0"}`)
		chk, _ := structured(t, call(t, root, "mrw_write", map[string]any{"plan": goEdit113}))["check"].(map[string]any)
		if chk == nil || chk["exit_code"] != float64(0) {
			t.Fatalf("check %v, want a passing verdict", chk)
		}
		if _, has := chk["tail"]; has {
			t.Fatalf("the tail survived a receipt over the ceiling")
		}
		log, _ := chk["output_file"].(string)
		if log != "" {
			t.Cleanup(func() { _ = os.Remove(log) })
		}
		b, err := os.ReadFile(log)
		if log == "" || err != nil || strings.Count(string(b), "\n") != 30 {
			t.Fatalf("output_file %q (%v) does not hold the dropped tail's 30 lines", log, err)
		}
	})
	t.Run("every verdict phrase fits the write floor", func(t *testing.T) {
		for _, c := range []struct {
			chk *check.Result
			err error
		}{
			{nil, nil},
			{nil, errors.New(strings.Repeat("e", 9999))},
			{&check.Result{Skipped: strings.Repeat("s", 9999)}, nil},
			{&check.Result{Ran: true}, nil},
			{&check.Result{Ran: true, ExitCode: math.MinInt}, nil},
			{&check.Result{Ran: true, ExitCode: math.MaxInt}, nil},
		} {
			msg := errorResult(appliedButUnreportable(math.MaxInt, math.MaxInt, math.MaxInt, true) + checkPhrase(c.chk, c.err) + leftNote(math.MaxInt))
			if n, floor := encodedSize(msg), writeFloor(); n > floor {
				t.Errorf("phrase %q makes the terminal answer %d bytes, over the floor %d", checkPhrase(c.chk, c.err), n, floor)
			}
		}
	})
	t.Run("at the smallest ceiling a landed write still names its check's verdict", func(t *testing.T) {
		restore := MaxResultChars
		MaxResultChars = minWriteCeiling()
		t.Cleanup(func() { MaxResultChars = restore })
		root := checkTree113(t, `{"check":"echo boom; exit 3"}`)
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113})
		if text := firstText(t, res); !strings.Contains(text, "exit 3") {
			t.Fatalf("at ceiling %d the answer does not name the check's verdict:\n%s", MaxResultChars, text)
		}
		if got := tally113(t, root); got["failed_check"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want failed_check 1", got)
		}
	})
}

// firstText is the first text block of a tool result.
func firstText(t *testing.T, res map[string]any) string {
	t.Helper()
	content, _ := res["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("no content: %v", res)
	}
	block, _ := content[0].(map[string]any)
	s, _ := block["text"].(string)
	return s
}
