package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/statehint"
)

// ADR-161 Decision 12: the MCP surface takes the CLI's plan makers. The plan is
// built by read.BuildSub and read.BuildRewrite, the functions `mrw read --sub`
// and `mrw write --grep --sub` call, so the two surfaces cannot disagree about
// which lines change.

// presentKeys is the arguments a call actually sent, apart from their values:
// "grep": "" is a pattern that matches every line, as the CLI's empty --grep is,
// and not a missing argument. A null counts as missing.
func presentKeys(args json.RawMessage) map[string]bool {
	var raw map[string]json.RawMessage
	if json.Unmarshal(args, &raw) != nil {
		return nil
	}
	has := make(map[string]bool, len(raw))
	for k, v := range raw {
		has[k] = string(bytes.TrimSpace(v)) != "null"
	}
	return has
}

// hasGrep and hasAst say a finder was given. On mrw_read only a plan call
// reads presence: an empty grep beside no sub still means no finder, as before.
// Where presence is read, the raw keys decide alone: a duplicate key whose last
// value is null leaves the struct holding the earlier value, so the value cannot
// be asked as well (the Codex review of T9).
func (a readArgs) hasGrep() bool {
	if a.planned() && a.has != nil {
		return a.has["grep"]
	}
	return a.Grep != ""
}
func (a readArgs) hasAst() bool {
	if a.planned() && a.has != nil {
		return a.has["ast_grep"]
	}
	return a.AstGrep != ""
}
func (a writeArgs) hasGrep() bool {
	if a.has != nil {
		return a.has["grep"]
	}
	return a.Grep != ""
}
func (a writeArgs) hasAst() bool {
	if a.has != nil {
		return a.has["ast_grep"]
	}
	return a.AstGrep != ""
}

// invalidPlanText refuses a plan JSON would corrupt, naming the file whose
// planned line holds the byte.
func invalidPlanText(plan []byte) string {
	file, header := "a file", ""
	for _, l := range bytes.Split(plan, []byte("\n")) {
		if bytes.HasPrefix(l, []byte("@@ ")) {
			header = string(bytes.ToValidUTF8(l, nil))
		}
		if !utf8.Valid(l) {
			if f := strings.Fields(header); len(f) > 1 {
				file = f[1]
			}
			break
		}
	}
	return fmt.Sprintf("the plan for %s holds a byte that is not valid UTF-8, which the JSON answer would replace, so the plan would write a different line than it licenses: "+
		"nothing was returned or licensed. Use mrw_write with grep and sub (the plan stays in memory), or the CLI (`mrw read --sub`, `mrw write --grep --sub`)", file)
}

// refusalText is a builder's refusal as the CLI prints it: what the walk could
// not serve first, then the reason.
func refusalText(err error, diag *bytes.Buffer) string {
	if d := strings.TrimSpace(diag.String()); d != "" {
		return d + "\n" + err.Error()
	}
	return err.Error()
}

// fitsAsError reports whether msg, sent as an isError answer, is within the
// ceiling withinCeiling enforces.
func fitsAsError(msg string) bool {
	return encodedSize(decorateCall(errorResult(msg))) <= MaxResultChars
}

// hintedText is the state-directory hint's words when the isError answer they
// make still fits the ceiling, else the plain ones: a hint is only ever added
// to an answer that would have been sent, so it cannot turn an isError result
// into the JSON-RPC error withinCeiling gives what fits no way (ADR-164).
func hintedText(plain, hinted string) string {
	if fitsAsError(hinted) {
		return hinted
	}
	return plain
}

// planned reports whether the call asks for a plan made by a walk.
func (a readArgs) planned() bool { return a.Sub != nil || a.Rewrite != nil }

// planRefusal is the CLI's rule for `read --sub` and `read --rewrite`: a plan
// serves no line, so what shapes served lines has nothing to act on (IsSet, not
// non-empty: an empty replacement is a real one). Empty when the call is fine.
func (a readArgs) planRefusal() string {
	switch {
	case a.Rewrite != nil:
		if !a.hasAst() {
			return "rewrite is ast-grep's rewrite of the matches ast_grep finds: give ast_grep"
		}
		if f := a.shapesLines(true); len(f) > 0 {
			return fmt.Sprintf("rewrite returns a plan and serves no line, so %s has nothing to act on, or makes a plan of its own: drop %s", f[0], f[0])
		}
	case a.Sub != nil:
		if a.hasAst() {
			return "sub rewrites the lines a grep matches, and an ast_grep hit is a span of lines: use grep"
		}
		if !a.hasGrep() {
			return "sub rewrites the lines a grep matches: give grep"
		}
		if f := a.shapesLines(false); len(f) > 0 {
			return fmt.Sprintf("sub returns a plan and serves no line, so %s has nothing to act on: drop %s", f[0], f[0])
		}
	}
	return ""
}

// shapesLines names the arguments set that act on served lines, or on an index
// or a count, which a plan has none of. withSub adds sub, which a rewrite
// cannot take beside it.
func (a readArgs) shapesLines(withSub bool) []string {
	var set []string
	for _, f := range []struct {
		name string
		on   bool
	}{
		{"sub", withSub && a.Sub != nil},
		{"count", a.Count},
		{"stat", a.Stat},
		{"quiet", a.Quiet},
		{"max_cols", a.MaxCols > 0},
		{"max_lines", a.MaxLines != nil},
		{"files_from", a.FilesFrom != nil},
		{"after", a.After != ""},
	} {
		if f.on {
			set = append(set, f.name)
		}
	}
	return set
}

// quietRefusal: quiet licenses the lines a read serves, and count and stat
// serve no line while a plan licenses its own (the CLI's --quiet rule).
func (a readArgs) quietRefusal() string {
	if !a.Quiet {
		return ""
	}
	for _, f := range []struct {
		name string
		on   bool
	}{{"count", a.Count}, {"stat", a.Stat}, {"sub", a.Sub != nil}, {"rewrite", a.Rewrite != nil}} {
		if f.on {
			return fmt.Sprintf("quiet licenses the lines a read serves, and %s serves or licenses differently: drop one", f.name)
		}
	}
	return ""
}

// readPlan answers mrw_read with sub or rewrite: the plan as the text, exactly,
// and the planned lines recorded as read, licensed and not shown (ADR-118). No
// ack is asked for: an ack certifies that lines SHOWN arrived, and a plan shows
// none of the lines it licenses; the plan carries the file's sha, so it applies
// to that version or to none. A plan that does not fit one answer is refused
// whole, with nothing recorded, since half a plan is a partial edit.
func readPlan(root string, a readArgs) (callToolResult, *rpcError) {
	var diag bytes.Buffer
	var b *read.Built
	var err error
	if a.Rewrite != nil {
		b, err = read.BuildRewrite(root, a.Specs, a.AstGrep, *a.Rewrite, a.Exclude, a.NoIgnore, &diag)
	} else {
		b, err = read.BuildSub(root, a.Specs, a.Grep, *a.Sub, a.Exclude, a.NoIgnore, &diag)
	}
	if err != nil {
		return errorResult(hintedText(refusalText(err, &diag), refusalText(statehint.Hint(err), &diag))), nil
	}
	if b.Problems > 0 {
		return errorResult(refusalText(fmt.Errorf("incomplete: %d path(s) or range(s) could not be served, so no plan is returned and nothing is licensed", b.Problems), &diag)), nil
	}
	// JSON replaces a byte that is not valid UTF-8, so a plan holding one would
	// reach the host as a different plan while the original lines stay licensed
	// (ADR-108). It is refused before anything is recorded; mrw_write, whose plan
	// never crosses JSON, applies the same plan byte for byte.
	if !utf8.Valid(b.Plan) {
		return errorResult(invalidPlanText(b.Plan)), nil
	}
	res, rpcErr := readResult(withSkipped(map[string]any{
		"observed": slashKeys(b.Observed, filepath.Separator),
		"problems": 0,
	}, b.Skipped), string(b.Plan), false)
	if rpcErr != nil {
		return callToolResult{}, rpcErr
	}
	if n := encodedSize(res); n > ceiling() {
		return errorResult(fmt.Sprintf("that plan is %d bytes encoded, over the %d-byte limit of one answer, and half a plan is a partial edit: it was not returned and nothing was licensed. "+
			"Apply it in one call with mrw_write (grep and sub, or ast_grep and rewrite), or narrow specs and exclude", n, ceiling())), nil
	}
	if err := seen.Record(root, b.Observed); err != nil {
		return callToolResult{}, &rpcError{Code: codeInternal, Message: "recording the planned lines: " + statehint.Hint(err).Error()}
	}
	return res, nil
}

// planSource builds the plan an mrw_write with grep and sub, or ast_grep and
// rewrite, applies in place of a plan argument, with the CLI's refusals. The
// planned lines are recorded as read before the plan is applied, as a read
// records what it printed; a partial plan (a path the walk refused) applies
// nothing. The string is a refusal, empty when the plan was built.
func planSource(root string, a writeArgs) ([]byte, string) {
	switch {
	case a.hasGrep() != (a.Sub != nil):
		return nil, "grep and sub go together: sub is the replacement for the lines grep matches"
	case (a.Rewrite != nil) != a.hasAst():
		return nil, "ast_grep and rewrite go together: rewrite is the replacement for the matches ast_grep finds"
	case a.hasGrep() && a.hasAst():
		return nil, "ast_grep with rewrite and grep with sub are two ways to make a plan: use one"
	case strings.TrimSpace(a.Plan) != "" || strings.TrimSpace(a.Format) != "":
		return nil, "grep with sub, or ast_grep with rewrite, makes its own plan: drop plan and format"
	}
	if err := read.CheckExclude(a.Exclude); err != nil {
		return nil, "exclude " + err.Error()
	}
	var diag bytes.Buffer
	var b *read.Built
	var err error
	if a.Rewrite != nil {
		b, err = read.BuildRewrite(root, a.Paths, a.AstGrep, *a.Rewrite, a.Exclude, a.NoIgnore, &diag)
	} else {
		b, err = read.BuildSub(root, a.Paths, a.Grep, *a.Sub, a.Exclude, a.NoIgnore, &diag)
	}
	if err != nil {
		return nil, hintedText(refusalText(err, &diag), refusalText(statehint.Hint(err), &diag))
	}
	if b.Problems > 0 {
		return nil, refusalText(fmt.Errorf("incomplete: %d path(s) or range(s) could not be served, so nothing was written", b.Problems), &diag)
	}
	if err := seen.Record(root, b.Observed); err != nil {
		return nil, hintedText("recording the planned lines: "+err.Error(), "recording the planned lines: "+statehint.Hint(err).Error())
	}
	return b.Plan, ""
}
