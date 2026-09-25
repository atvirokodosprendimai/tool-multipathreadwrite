package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// markupCheckout writes a file of the shape the v1.25.1 round measured: 153,600
// bytes of markup, whose `<`, `>` and `&` cost six bytes each once the answer is
// JSON-encoded, so it renders inside the ceiling and encodes far past it.
func markupCheckout(t *testing.T) (root, path string, lines int) {
	t.Helper()
	root = t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	const line = `<div class="a">&amp;</div>`
	lines = 153600 / (len(line) + 1)
	path = "m.tsx"
	body := strings.Repeat(line+"\n", lines)
	if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, path, lines
}

// TestAMarkupFilePagesByItsEncodedSize is ADR-074's Enforced-by. The first page
// was sized from the RAW bytes while the ceiling measures the ENCODED answer, so
// a markup file was refused whole, with no next_read, where a plain file twice
// its size paged. Every page must fit, and following next_read must give back
// the whole file.
func TestAMarkupFilePagesByItsEncodedSize(t *testing.T) {
	root, path, lines := markupCheckout(t)
	var got []string
	spec := path
	for page := 0; ; page++ {
		if page > 20 {
			t.Fatalf("still paging after %d pages", page)
		}
		res := call(t, root, "mrw_read", map[string]any{"specs": []any{spec}})
		content, _ := res["content"].([]any)
		if len(content) == 0 {
			t.Fatalf("page %d carried no content", page)
		}
		body, _ := content[0].(map[string]any)["text"].(string)
		if isErr, _ := res["isError"].(bool); isErr {
			t.Fatalf("page %d was refused, not paged:\n%s", page, body)
		}
		raw, err := json.Marshal(res)
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) > MaxResultChars {
			t.Fatalf("page %d encodes to %d, over the %d ceiling", page, len(raw), MaxResultChars)
		}
		got = append(got, numberedLines(t, body)...)
		next := nextOf(t, res)
		if page == 0 && next == "" {
			t.Fatalf("the first answer is not a page: it names no next_read:\n%.400s", body)
		}
		if next == "" {
			break
		}
		if next == spec {
			t.Fatalf("page %d hands back its own spec %q", page, next)
		}
		spec = next
	}
	if len(got) != lines {
		t.Fatalf("reassembled %d lines, want %d", len(got), lines)
	}
	for i, l := range got {
		if l != `<div class="a">&amp;</div>` {
			t.Fatalf("line %d of the reassembly is %q", i+1, l)
		}
	}
}

// A CLOSED range is not paged (ADR-014), so it is refused — and the refusal
// must name what overflowed. It blamed the per-file receipt, whose remedy,
// "name fewer files", does nothing for one file.
func TestAClosedMarkupRangeRefusalNamesTheEncoding(t *testing.T) {
	root, path, _ := markupCheckout(t)
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{path + ":1-5000"}})
	content, _ := res["content"].([]any)
	if len(content) == 0 {
		t.Fatal("the refusal carried no text")
	}
	txt, _ := content[0].(map[string]any)["text"].(string)
	if isErr, _ := res["isError"].(bool); !isErr {
		t.Fatalf("a closed range over the ceiling was not refused:\n%.400s", txt)
	}
	if !strings.Contains(txt, "encoded") || !strings.Contains(txt, "narrower range") {
		t.Errorf("the refusal does not name the encoding and its remedy:\n%s", txt)
	}
	if strings.Contains(txt, "per-file receipt") {
		t.Errorf("the refusal blames the receipt for an encoding overflow:\n%s", txt)
	}
}

// Review of #232. A line that alone encodes past the ceiling fits no range, as
// mrw serves whole lines. The refusal advised "around 1 lines", and following
// it was refused the same way, for ever. It sends the caller to the CLI,
// whether the spec names the file or that one line.
func TestALineThatEncodesPastTheCeilingIsSentToTheCLI(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	body := strings.Repeat("<", 150000) + "\nsecond\n"
	if err := os.WriteFile(filepath.Join(root, "f.svg"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, spec := range []string{"f.svg", "f.svg:1-1"} {
		res := call(t, root, "mrw_read", map[string]any{"specs": []any{spec}})
		content, _ := res["content"].([]any)
		if len(content) == 0 {
			t.Fatalf("%s: the answer carried no text", spec)
		}
		txt, _ := content[0].(map[string]any)["text"].(string)
		if isErr, _ := res["isError"].(bool); !isErr {
			t.Fatalf("%s: a line that encodes past the ceiling was served:\n%.300s", spec, txt)
		}
		if strings.Contains(txt, "around 1 lines") || strings.Contains(txt, ":1-1") {
			t.Errorf("%s: the refusal sends the caller back to the same unservable line:\n%s", spec, txt)
		}
		if !strings.Contains(txt, "serves whole lines") || !strings.Contains(txt, "mrw read") {
			t.Errorf("%s: the refusal does not say why no range helps, nor name a reader that can:\n%s", spec, txt)
		}
	}
}

// Review of #232. A page is sized from a sample's average, and a file whose
// heavily escaped half comes first paged once, then refused next_read in the
// middle of the file: the sample mixed both halves, and a page of the escaped
// half alone encoded past the ceiling. Following next_read must reach the end.
func TestAFileWhoseEscapedHalfComesFirstPagesToItsEnd(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var want []string
	for i := 0; i < 2500; i++ {
		want = append(want, strings.Repeat("<", 60))
	}
	for i := 0; i < 2500; i++ {
		want = append(want, strings.Repeat("a", 60))
	}
	if err := os.WriteFile(filepath.Join(root, "mixed.html"), []byte(strings.Join(want, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var got []string
	spec := "mixed.html"
	for page := 0; ; page++ {
		if page > 40 {
			t.Fatalf("still paging after %d pages", page)
		}
		res := call(t, root, "mrw_read", map[string]any{"specs": []any{spec}})
		content, _ := res["content"].([]any)
		if len(content) == 0 {
			t.Fatalf("page %d carried no content", page)
		}
		txt, _ := content[0].(map[string]any)["text"].(string)
		if isErr, _ := res["isError"].(bool); isErr {
			t.Fatalf("page %d (%s) was refused, so next_read breaks mid-file:\n%.400s", page, spec, txt)
		}
		raw, err := json.Marshal(res)
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) > MaxResultChars {
			t.Fatalf("page %d encodes to %d, over the %d ceiling", page, len(raw), MaxResultChars)
		}
		got = append(got, numberedLines(t, txt)...)
		next := nextOf(t, res)
		if next == "" {
			break
		}
		if next == spec {
			t.Fatalf("page %d hands back its own spec %q", page, next)
		}
		spec = next
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("the pages reassemble %d lines, want %d, or not in order", len(got), len(want))
	}
}
