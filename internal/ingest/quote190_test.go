package ingest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer"
)

var nl = string(rune(10))

const fileName190 = "fx"

func text190(rows []string) string {
	return strings.Join(rows, nl) + nl
}

func block190(old, nu []string) string {
	out := []string{fileName190, searchHead}
	out = append(out, old...)
	out = append(out, srDiv)
	out = append(out, nu...)
	out = append(out, srEnd)
	return text190(out)
}

func land190(t *testing.T, root string, planText []byte) apply.Result {
	t.Helper()
	hunks, err := plan.Parse(bytes.NewReader(planText))
	if err != nil {
		t.Fatal(err, string(planText))
	}
	in := make([]apply.Input, 0, len(hunks))
	for _, h := range hunks {
		in = append(in, apply.Input{
			Path: h.Path, Start: h.Addr.Start, End: h.Addr.End, Op: string(h.Op),
			StartPat: h.Addr.StartPat, EndPat: h.Addr.EndPat, RelEnd: h.Addr.RelEnd,
			CountedBody: h.CountedBody, Body: h.Body, SHA: h.SHA, Lines: h.Lines,
			Anchor: h.Anchor, Quote: h.Quote, SrcLine: h.SrcLine, Index: h.Index,
		})
	}
	prep, err := writer.Prepare(writer.Request{Root: root, In: in, Check: writer.CheckOff})
	if err != nil {
		t.Fatal(err)
	}
	landed, err := prep.Land()
	if err != nil {
		t.Fatal(err)
	}
	return landed.Res
}

func reasons190(res apply.Result) string {
	var b strings.Builder
	for _, h := range res.Hunks {
		b.WriteString(h.Reason + nl)
	}
	return b.String()
}

func tree190(t *testing.T, file string) string {
	t.Helper()
	return writeTree(t, map[string]string{fileName190: file})
}

func stateDir190(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
}

func compile190(t *testing.T, root, doc string) string {
	t.Helper()
	planText, err := CompileSearchReplace(root, []byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	return string(planText)
}

func content190(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, fileName190))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func rewrite190(t *testing.T, root, body string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(root, fileName190), []byte(body), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}

func refused190(t *testing.T, root string, res apply.Result, want, wantFile string) {
	t.Helper()
	if res.Applied || res.Failed == 0 {
		t.Fatal(res.Hunks)
	}
	if !strings.Contains(reasons190(res), want) {
		t.Error(want, reasons190(res))
	}
	if got := content190(t, root); got != wantFile {
		t.Error(got)
	}
}

func applied190(t *testing.T, root string, res apply.Result, planText, wantFile string) {
	t.Helper()
	if !res.Applied || res.Failed != 0 {
		t.Fatal(reasons190(res), planText)
	}
	if got := content190(t, root); got != wantFile {
		t.Error(got)
	}
}

func rows190(s string) []string {
	return strings.Fields(s)
}

func caseApplies190(t *testing.T) {
	stateDir190(t)
	four := text190(rows190("a b c d"))
	root := tree190(t, four)
	planText := compile190(t, root, block190(rows190("c d"), rows190("C D")))
	res := land190(t, root, []byte(planText))
	applied190(t, root, res, planText, text190(rows190("a b C D")))
}
func changedLine190(t *testing.T) {
	stateDir190(t)
	root := tree190(t, text190(rows190(`a b c d`)))
	planText := compile190(t, root, block190(rows190(`c d`), rows190(`C D`)))
	changed := text190(rows190(`a b c dd`))
	rewrite190(t, root, changed)
	refused190(t, root, land190(t, root, []byte(planText)), `quote`, changed)
}

func twiceBlock190(t *testing.T) {
	stateDir190(t)
	root := tree190(t, text190(rows190(`x y x y`)))
	_, err := CompileSearchReplace(root, []byte(block190(rows190(`x y`), rows190(`X`))))
	if err == nil || !strings.Contains(err.Error(), `matched 2 times`) {
		t.Fatal(err)
	}
}

func duplicateLine190(t *testing.T) {
	for _, after := range []string{`p q x r x`, `p x q x r`} {
		stateDir190(t)
		root := tree190(t, text190(rows190(`p q x r`)))
		planText := compile190(t, root, block190(rows190(`x`), rows190(`X`)))
		moved := text190(rows190(after))
		rewrite190(t, root, moved)
		refused190(t, root, land190(t, root, []byte(planText)), `quote`, moved)
	}
}

func nativeUnread190(t *testing.T) {
	stateDir190(t)
	file := text190(rows190(`a b c d`))
	root := tree190(t, file)
	native := text190([]string{`@@ fx 3-4 replace anchor=c`, `C`, `D`})
	refused190(t, root, land190(t, root, []byte(native)), `has not been read`, file)
}

func keptLine190(t *testing.T) {
	stateDir190(t)
	root := tree190(t, text190(rows190(`a b c d e`)))
	planText := compile190(t, root, block190(rows190(`b c d`), rows190(`B C d`)))
	res := land190(t, root, []byte(planText))
	applied190(t, root, res, planText, text190(rows190(`a B C d e`)))
	if !strings.Contains(planText, ` 2-3 replace`) {
		t.Error(planText)
	}
}

func noKeptLine190(t *testing.T) {
	file := text190(rows190(`a b c d e`))
	for _, old := range []string{`b c d`, `b c`} {
		stateDir190(t)
		root := tree190(t, file)
		planText := compile190(t, root, block190(rows190(old), rows190(`B`)))
		if !strings.Contains(planText, ` 2-`+fmt.Sprint(1+len(rows190(old)))+` replace`) {
			t.Error(planText)
		}
		refused190(t, root, land190(t, root, []byte(planText)), `needs a served line after`, file)
	}
}

func unquotedSibling190(t *testing.T) {
	stateDir190(t)
	file := text190(rows190(`a b c d e`))
	root := tree190(t, file)
	planText := compile190(t, root, block190(rows190(`b c d`), rows190(`B C d`)))
	both := planText + text190([]string{`@@ fx 5 replace`, `E`})
	res := land190(t, root, []byte(both))
	refused190(t, root, res, `has not been read`, file)
	for _, h := range res.Hunks {
		if h.Status == apply.StatusOK {
			t.Error(h)
		}
	}
}

func typedQuote190(t *testing.T) {
	stateDir190(t)
	file := text190(rows190(`a b c d`))
	root := tree190(t, file)
	forged := `0.` + strings.Repeat(`0`, 64)
	native := text190([]string{`@@ fx 3-4 replace anchor=c quote=` + forged, `C`, `D`})
	refused190(t, root, land190(t, root, []byte(native)), `quote`, file)
}

func movedOp190(t *testing.T) {
	stateDir190(t)
	file := text190(rows190(`a b c d`))
	root := tree190(t, file)
	planText := compile190(t, root, block190(rows190(`c`), rows190(`C`)))
	edited := strings.Replace(planText, ` replace `, ` delete `, 1)
	refused190(t, root, land190(t, root, []byte(edited)), `quote`, file)
}

func TestAnExactSearchBlockLicensesTheLinesItQuotes(t *testing.T) {
	t.Run(`an exact block matching once applies with no read`, caseApplies190)
	t.Run(`a quoted line changed after the compile is refused`, changedLine190)
	t.Run(`a block that matches twice is refused`, twiceBlock190)
	t.Run(`a duplicate of a quoted line, anywhere, refuses the block`, duplicateLine190)
	t.Run(`a native plan of the same lines is refused as unread`, nativeUnread190)
	t.Run(`a block ending in a kept line applies`, keptLine190)
	t.Run(`a block without a kept line is refused for the line after`, noKeptLine190)
	t.Run(`an unquoted sibling stays unread`, unquotedSibling190)
	t.Run(`a quote typed into a native plan is refused`, typedQuote190)
	t.Run(`a compiled mark moved to another op licenses nothing`, movedOp190)
}
