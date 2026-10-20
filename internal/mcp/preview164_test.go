package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// previewRows counts the `~`, `-` and `+` rows of a receipt's text.
func previewRows(text string) int {
	n := 0
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(l, "   ~ ") || strings.HasPrefix(l, "   - ") || strings.HasPrefix(l, "   + ") {
			n++
		}
	}
	return n
}

// hunksOf is the structured receipt's hunks.
func hunksOf(t *testing.T, res map[string]any) []map[string]any {
	t.Helper()
	var out []map[string]any
	hs, _ := structured(t, res)["hunks"].([]any)
	for _, h := range hs {
		m, _ := h.(map[string]any)
		out = append(out, m)
	}
	return out
}

// ADR-164 Decision 2. mrw_write with grep+sub or ast_grep+rewrite and dry_run
// shows each change in the text and in hunks[].before and after; a real write
// and a hand-written plan's dry run show none; and under a ceiling that
// elides, the preview goes first and `elided` names it.
func TestAnMCPSubDryRunShowsEachChange(t *testing.T) {
	t.Run("grep and sub", func(t *testing.T) {
		root := tree161(t, map[string]string{"a.css": ".a { color: red; }\n.b { color: red and red; }\n"})
		res := call(t, root, "mrw_write", map[string]any{"grep": `\bred\b`, "sub": "blue", "dry_run": true, "check": false})
		text := firstText(t, res)
		for _, want := range []string{"   ~ 1| .a { color: [-red-]{+blue+}; }", "   ~ 2| .b { color: [-red and red-]{+blue and blue+}; }"} {
			if !strings.Contains(text, want) {
				t.Errorf("the text lacks %q:\n%s", want, text)
			}
		}
		hs := hunksOf(t, res)
		if len(hs) != 2 || fmt.Sprint(hs[0]["before"]) != "[.a { color: red; }]" || fmt.Sprint(hs[0]["after"]) != "[.a { color: blue; }]" {
			t.Errorf("hunks[].before/after: %v", hs)
		}
		if got := file161(t, root, "a.css"); got != ".a { color: red; }\n.b { color: red and red; }\n" {
			t.Errorf("a dry run changed a.css to %q", got)
		}

		// the real write shows nothing
		res = call(t, root, "mrw_write", map[string]any{"grep": `\bred\b`, "sub": "blue", "check": false})
		for _, h := range hunksOf(t, res) {
			if _, has := h["before"]; has {
				t.Errorf("a real write carries before: %v", h)
			}
		}
		if previewRows(firstText(t, res)) != 0 {
			t.Errorf("a real write prints preview rows:\n%s", firstText(t, res))
		}
	})

	t.Run("a plain plan's dry run shows none", func(t *testing.T) {
		root := tree161(t, map[string]string{"a.css": ".a { color: red; }\n"})
		call(t, root, "mrw_read", map[string]any{"specs": []string{"a.css"}, "quiet": true})
		res := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.css 1 replace\n.a { color: blue; }\n", "dry_run": true, "check": false})
		if res["isError"] == true || previewRows(firstText(t, res)) != 0 {
			t.Errorf("a plain dry run: %v", res)
		}
		for _, h := range hunksOf(t, res) {
			if _, has := h["before"]; has {
				t.Errorf("a plain dry run carries before: %v", h)
			}
		}
	})

	t.Run("ast_grep and rewrite", func(t *testing.T) {
		const src = "package a\n\nfunc f() {\n\tx := foo(\n\t\t3,\n\t\t4)\n}\n"
		root := tree161(t, map[string]string{"a.go": src})
		installRecordingAstGrep(t, stream161(t, "a.go", src, [2]string{"foo(\n\t\t3,\n\t\t4)", "bar(4, 3)"}))
		res := call(t, root, "mrw_write", map[string]any{"ast_grep": "foo($A, $B)", "rewrite": "bar($B, $A)", "dry_run": true, "check": false})
		text := firstText(t, res)
		for _, want := range []string{"   - 4| \tx := [-foo(-]\n", "   - 5| \t\t3,\n", "   - 6| \t\t4)\n", "   + 4| \tx := {+bar(4, 3)+}\n"} {
			if !strings.Contains(text, want) {
				t.Errorf("the text lacks %q:\n%s", want, text)
			}
		}
		hs := hunksOf(t, res)
		if len(hs) != 1 {
			t.Fatalf("hunks: %v", hs)
		}
		before, _ := hs[0]["before"].([]any)
		after, _ := hs[0]["after"].([]any)
		if len(before) != 3 || len(after) != 1 {
			t.Errorf("hunks[].before/after: %v", hs)
		}
	})

	t.Run("under a ceiling the preview goes first and elided names it", func(t *testing.T) {
		const n = 300
		var body, plan strings.Builder
		for i := 0; i < n; i++ {
			fmt.Fprintf(&body, "line %03d Old keep keep keep keep keep keep\n", i)
			fmt.Fprintf(&plan, "@@ big.txt %d replace\nline %03d New keep keep keep keep keep keep\n", i+1, i)
		}
		root := tree161(t, map[string]string{"big.txt": body.String()})
		call(t, root, "mrw_read", map[string]any{"specs": []string{"big.txt"}, "quiet": true})

		// the size of the same receipt with no preview: a plain plan's dry run
		plain := call(t, root, "mrw_write", map[string]any{"plan": plan.String(), "dry_run": true, "check": false})
		b, err := json.Marshal(plain)
		if err != nil {
			t.Fatal(err)
		}
		restore := MaxResultChars
		t.Cleanup(func() { MaxResultChars = restore })

		// a ceiling the preview does not fit but the plain receipt does
		MaxResultChars = len(b) + 1500
		res := call(t, root, "mrw_write", map[string]any{"grep": `\bOld\b`, "sub": "New", "dry_run": true, "check": false})
		sc := structured(t, res)
		elided, _ := sc["elided"].(string)
		if !strings.Contains(elided, "preview") || !strings.Contains(elided, "before") || !strings.Contains(elided, "after") || !strings.Contains(elided, "300 hunk(s)") {
			t.Errorf("elided does not name the preview: %q", elided)
		}
		hs := hunksOf(t, res)
		if len(hs) != n {
			t.Fatalf("%d hunk verdicts, want all %d: the preview goes before any verdict", len(hs), n)
		}
		for _, h := range hs {
			if _, has := h["before"]; has {
				t.Fatalf("a hunk kept its before: %v", h)
			}
		}
		if previewRows(firstText(t, res)) != 0 || !strings.Contains(firstText(t, res), "elided") && !strings.Contains(firstText(t, res), "preview") {
			t.Errorf("the text keeps preview rows or does not say they went:\n%.300s", firstText(t, res))
		}

		// a ceiling that fits everything elides nothing
		MaxResultChars = 4_000_000
		res = call(t, root, "mrw_write", map[string]any{"grep": `\bOld\b`, "sub": "New", "dry_run": true, "check": false})
		if e, _ := structured(t, res)["elided"].(string); e != "" || previewRows(firstText(t, res)) != n {
			t.Errorf("a roomy ceiling: elided %q, %d preview rows, want none elided and %d", e, previewRows(firstText(t, res)), n)
		}
	})
}
