package mcp

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ADR-214. Over MCP the read's note is report text in content[0], outside the
// checkpoint spans, and a write keeps the mark and reports the landed lines.
// M is the UTF-8 byte order mark, written as bytes.
func TestMrwReadAndWriteKeepAMarkLikeTheCLI(t *testing.T) {
	const (
		m      = "\xEF\xBB\xBF"
		note   = "-- note: a UTF-8 byte order mark begins this file"
		marked = m + "one\ntwo\nthree\n"
	)
	text := func(t *testing.T, res map[string]any) string {
		t.Helper()
		c, _ := res["content"].([]any)
		if len(c) == 0 {
			t.Fatalf("no content: %v", res)
		}
		first, _ := c[0].(map[string]any)
		s, _ := first["text"].(string)
		return s
	}

	t.Run("the_note_is_outside_every_checkpoint", func(t *testing.T) {
		root, path := checkout(t, "m.txt", marked)
		out := text(t, call(t, root, "mrw_read", map[string]any{"specs": []any{path}}))
		if strings.Count(out, note) != 1 {
			t.Fatalf("the note is printed %d times:\n%s", strings.Count(out, note), out)
		}
		lines := strings.Split(out, "\n")
		in, rows, open := false, 0, ""
		for _, l := range lines {
			switch {
			case strings.HasPrefix(l, "-- ck ") && strings.Contains(l, " open "):
				in, open = true, l
			case strings.HasPrefix(l, "-- ck ") && strings.HasSuffix(l, " close"):
				in = false
			case in && strings.HasPrefix(l, note):
				t.Errorf("the note sits inside a checkpoint span:\n%s", out)
			case in:
				rows++
			}
		}
		if rows != 3 || !strings.Contains(open, "lines 1-3 (3 lines follow)") {
			t.Errorf("the checkpoint covers %d rows, opens %q: want the three numbered rows only", rows, open)
		}
	})

	t.Run("a_write_keeps_the_mark_and_reports_the_landed_lines", func(t *testing.T) {
		root, path := checkout(t, "m.txt", marked)
		out := text(t, call(t, root, "mrw_read", map[string]any{"specs": []any{path}}))
		var acks []any
		for _, mt := range regexp.MustCompile(`-- ck ([0-9a-f]{16}) open`).FindAllStringSubmatch(out, -1) {
			acks = append(acks, mt[1])
		}
		res := call(t, root, "mrw_write", map[string]any{"plan": "@@ m.txt 1 replace\nONE\n", "ack": acks})
		if res["isError"] == true {
			t.Fatalf("write refused: %v", res)
		}
		b, err := os.ReadFile(filepath.Join(root, "m.txt"))
		if err != nil || string(b) != m+"ONE\ntwo\nthree\n" {
			t.Errorf("file = % x, err %v: want the mark at byte zero", b, err)
		}
		files, _ := structured(t, res)["files"].([]any)
		f, _ := files[0].(map[string]any)
		if n, _ := f["lines_after"].(float64); n != 3 {
			t.Errorf("lines_after = %v, want 3", f["lines_after"])
		}
	})

	// At every ceiling a read either answers within it or refuses: a served
	// line 1 carries the note, and the note is never served without line 1.
	t.Run("at_the_ceiling_the_note_and_line_1_travel_together", func(t *testing.T) {
		restore := MaxResultChars
		t.Cleanup(func() { MaxResultChars = restore })
		root, path := checkout(t, "m.txt", marked)
		served := 0
		for budget := 300; budget <= 1400; budget += 10 {
			MaxResultChars = budget
			for _, specs := range [][]any{{path}, {path + ":1"}} {
				raw := rawResult(t, root, "mrw_read", map[string]any{"specs": specs})
				if len(raw) > budget {
					t.Fatalf("budget %d, specs %v: the answer is %d bytes", budget, specs, len(raw))
				}
				res := call(t, root, "mrw_read", map[string]any{"specs": specs})
				if res["isError"] == true {
					continue
				}
				out := text(t, res)
				hasNote, hasLine1 := strings.Contains(out, note), strings.Contains(out, "| "+m+"one")
				if hasNote != hasLine1 {
					t.Fatalf("budget %d, specs %v: note %v, line 1 %v:\n%s", budget, specs, hasNote, hasLine1, out)
				}
				if hasNote {
					served++
				}
			}
		}
		if served == 0 {
			t.Error("no budget served line 1 with its note: the sweep reached nothing")
		}
	})
}
