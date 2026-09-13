package mcp

import (
	"strings"
	"testing"
)

func writeEchoPad(t *testing.T, content string, echoPad any) map[string]any {
	t.Helper()
	root, path := checkout(t, "f.txt", content)
	acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{path + ":2-4"}})))
	args := map[string]any{
		"plan": "@@ f.txt 2-3 replace anchor=\"2\"\nX\nY\n",
		"ack":  acks,
	}
	if echoPad != nil {
		args["echo_pad"] = echoPad
	}
	return structured(t, call(t, root, "mrw_write", args))
}

func hunkEcho(t *testing.T, got map[string]any) []string {
	t.Helper()
	hunks, _ := got["hunks"].([]any)
	if len(hunks) == 0 {
		t.Fatalf("no hunks: %v", got)
	}
	h, _ := hunks[0].(map[string]any)
	if status, _ := h["status"].(string); status != "ok" {
		t.Fatalf("status=%v, want ok: %v", h["status"], h)
	}
	raw, ok := h["echo"]
	if !ok || raw == nil {
		return nil
	}
	arr, _ := raw.([]any)
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		s, _ := v.(string)
		out = append(out, s)
	}
	return out
}

// TestWriteToolDeclaresEchoPad is ADR-052 T2: echo_pad lives on the existing
// mrw_write tool; cargo stays two tools.
func TestWriteToolDeclaresEchoPad(t *testing.T) {
	if n := len(tools()); n != 2 {
		t.Fatalf("tools = %d, want 2", n)
	}
	var schema map[string]any
	for _, tl := range tools() {
		if tl.Name != "mrw_write" {
			continue
		}
		schema, _ = tl.InputSchema.(map[string]any)
	}
	if schema == nil {
		t.Fatal("mrw_write declares no input schema")
	}
	props, _ := schema["properties"].(map[string]any)
	p, ok := props["echo_pad"].(map[string]any)
	if !ok {
		t.Fatal("mrw_write does not declare echo_pad")
	}
	if p["type"] != "integer" {
		t.Errorf("echo_pad type = %v, want integer", p["type"])
	}
	desc := strings.ToLower(strings.TrimSpace(stringOf(p["description"])))
	if !strings.Contains(desc, "not a checker") {
		t.Errorf("echo_pad does not say it is not a checker: %v", p["description"])
	}
	req, _ := schema["required"].([]string)
	for _, r := range req {
		if r == "echo_pad" {
			t.Error("echo_pad is required; it must default to 0")
		}
	}
}

// echo_pad N on mrw_write is the same pad as --echo-pad: N numbered lines,
// a closer in them stays ok. Schema-only coverage would survive a dead field.
func TestAnMCPWriteEchoPadPrintsNNumberedLines(t *testing.T) {
	got := writeEchoPad(t, "1\n2\n3\na\nb\nc\n", 3)
	echo := hunkEcho(t, got)
	if len(echo) != 3 {
		t.Fatalf("echo=%v, want 3 pad lines", echo)
	}
	for i, want := range []string{"4|", "5|", "6|"} {
		if !strings.Contains(echo[i], want) {
			t.Errorf("echo[%d]=%q, want numbered %s", i, echo[i], want)
		}
	}
}

func TestAnMCPWriteOmitsEchoAtDefaultZero(t *testing.T) {
	got := writeEchoPad(t, "1\n2\n3\n</div>\n5\n", nil)
	if echo := hunkEcho(t, got); len(echo) != 0 {
		t.Errorf("default echo_pad printed %v", echo)
	}
	if applied, _ := got["applied"].(bool); !applied {
		t.Errorf("default write did not apply: %v", got)
	}
}

func TestAnMCPWriteEchoPadClampsAtEOF(t *testing.T) {
	got := writeEchoPad(t, "1\n2\n3\n4\n", 10)
	echo := hunkEcho(t, got)
	if len(echo) != 1 {
		t.Fatalf("echo=%v, want the one line that remains", echo)
	}
	if !strings.Contains(echo[0], "4|") {
		t.Errorf("clamped pad is not the last line: %q", echo[0])
	}
}

func TestAnMCPWriteEchoPadKeepsACloserOk(t *testing.T) {
	got := writeEchoPad(t, "1\n2\n3\n</div>\n5\n", 1)
	echo := hunkEcho(t, got)
	if len(echo) != 1 {
		t.Fatalf("echo=%v, want one pad line", echo)
	}
	if !strings.Contains(echo[0], "</div>") {
		t.Errorf("pad does not show the closer: %q", echo[0])
	}
}

func stringOf(v any) string {
	s, _ := v.(string)
	return s
}
