package read

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// jsonStep is one step of a JSON path address (ADR-155): a name in an object,
// or an index into an array.
type jsonStep struct {
	name    string
	index   int
	isIndex bool
}

func (st jsonStep) String() string {
	if st.isIndex {
		return "[" + strconv.Itoa(st.index) + "]"
	}
	if strings.ContainsAny(st.name, ".[] ") {
		q, _ := json.Marshal(st.name)
		return "[" + string(q) + "]"
	}
	return "." + st.name
}

// isJSONPath reports whether a range is written as a JSON path. A bare `$`,
// `$-N` and `N-$` are line addresses and stay so (ADR-063).
func isJSONPath(s string) bool {
	return strings.HasPrefix(s, "$.") || strings.HasPrefix(s, "$[")
}

// parseJSONPath reads `$` and its steps: `.name`, `[N]` and `["name"]`.
// Everything else JSONPath has — `..`, `*`, filters, single quotes — is refused
// by name, since one address serves one span.
func parseJSONPath(s string) ([]jsonStep, error) {
	bad := func(why string) error {
		return fmt.Errorf("JSON path %q: %s; a step is .name, [N] or [\"name\"]", s, why)
	}
	var steps []jsonStep
	rest := s[1:]
	for rest != "" {
		switch rest[0] {
		case '.':
			rest = rest[1:]
			end := strings.IndexAny(rest, ".[]")
			if end < 0 {
				end = len(rest)
			}
			name := rest[:end]
			switch {
			case strings.HasPrefix(rest, "."):
				return nil, bad("`..` (recursive descent) is not supported")
			case name == "":
				return nil, bad("a `.` is followed by no name")
			case name == "*":
				return nil, bad("`*` (a wildcard) is not supported")
			}
			steps = append(steps, jsonStep{name: name})
			rest = rest[end:]
		case '[':
			if strings.HasPrefix(rest, `["`) {
				dec := json.NewDecoder(strings.NewReader(rest[1:]))
				var name string
				if err := dec.Decode(&name); err != nil {
					return nil, bad("the quoted name is not a JSON string")
				}
				after := 1 + int(dec.InputOffset())
				if after >= len(rest) || rest[after] != ']' {
					return nil, bad("the quoted name is not closed with ]")
				}
				steps = append(steps, jsonStep{name: name})
				rest = rest[after+1:]
				continue
			}
			end := strings.IndexByte(rest, ']')
			if end < 0 {
				return nil, bad("a `[` is never closed")
			}
			in := rest[1:end]
			switch {
			case strings.HasPrefix(in, "'"):
				return nil, bad(`a single-quoted name: write ["name"]`)
			case in == "*":
				return nil, bad("`*` (a wildcard) is not supported")
			case in == "" || strings.Trim(in, "0123456789") != "" || (len(in) > 1 && in[0] == '0'):
				return nil, bad(fmt.Sprintf("[%s] is not an index (a decimal from 0)", in))
			}
			n, err := strconv.Atoi(in)
			if err != nil {
				return nil, bad(fmt.Sprintf("[%s] is not an index (a decimal from 0)", in))
			}
			steps = append(steps, jsonStep{index: n, isIndex: true})
			rest = rest[end+1:]
		default:
			return nil, bad(fmt.Sprintf("%q follows the last step", rest))
		}
	}
	return steps, nil
}

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

const jsonSpace = " \t\r\n"

// errNotJSON is what jsonValue wraps when the file is not exactly one JSON
// document, so a projection can say the document is at fault and not the path.
var errNotJSON = errors.New("not one JSON document")

// lineOf is the 1-based line of a byte offset in b, counted the way the file's
// reader counts: by LF, or by CR when the file has no LF.
func lineOf(b []byte) func(int) int {
	cr := !bytes.Contains(b, []byte("\n")) && bytes.Contains(b, []byte("\r"))
	return func(off int) int {
		if cr {
			return 1 + bytes.Count(b[:off], []byte("\r"))
		}
		return 1 + bytes.Count(b[:off], []byte("\n"))
	}
}

// jsonSpan resolves a path against a file's bytes to the 1-based lines its
// value spans, first byte to last (ADR-155).
func jsonSpan(b []byte, steps []jsonStep) (int, int, error) {
	s, e, err := jsonValue(b, steps)
	if err != nil {
		return 0, 0, err
	}
	line := lineOf(b)
	return line(s), line(e - 1), nil
}

// jsonValue resolves a path against a file's bytes to the offsets [s, e) of
// its value. The file must be exactly one JSON document; a leading UTF-8 byte
// order mark is skipped. A name that appears more than once in an object the
// path goes through is refused with every line it is on, since the path cannot
// say which copy it means (ADR-155).
func jsonValue(b []byte, steps []jsonStep) (int, int, error) {
	line := lineOf(b)
	base := 0
	if bytes.HasPrefix(b, utf8BOM) {
		base = len(utf8BOM)
	}
	doc := b[base:]
	if !json.Valid(doc) {
		var raw json.RawMessage
		err := json.Unmarshal(doc, &raw)
		var se *json.SyntaxError
		if errors.As(err, &se) && se.Offset > 0 {
			return 0, 0, fmt.Errorf("%w: %w, at line %d", errNotJSON, err, line(base+int(se.Offset)-1))
		}
		return 0, 0, fmt.Errorf("%w: %w", errNotJSON, err)
	}
	// [s, e) is the current value, as offsets into b.
	s := base + len(doc) - len(bytes.TrimLeft(doc, jsonSpace))
	e := base + len(bytes.TrimRight(doc, jsonSpace))
	at := "$"
	for _, st := range steps {
		sub := b[s:e]
		want, kind := byte('{'), "an object"
		if st.isIndex {
			want, kind = '[', "an array"
		}
		if sub[0] != want {
			return 0, 0, fmt.Errorf("the value at %s, line %d, is %s, not %s", at, line(s), jsonKind(sub[0]), kind)
		}
		dec := json.NewDecoder(bytes.NewReader(sub))
		if _, err := dec.Token(); err != nil {
			return 0, 0, err
		}
		// found holds, per match, the value's first byte, the byte after its
		// last, and where its name starts: a duplicate is reported by the lines
		// of its names, which is where a reader of the file looks.
		var found [][3]int
		count := 0
		for dec.More() {
			at0 := int(dec.InputOffset())
			nameAt := at0 + len(sub[at0:]) - len(bytes.TrimLeft(sub[at0:], jsonSpace+","))
			if !st.isIndex {
				tok, err := dec.Token()
				if err != nil {
					return 0, 0, err
				}
				key, _ := tok.(string)
				if key != st.name {
					var skip json.RawMessage
					if err := dec.Decode(&skip); err != nil {
						return 0, 0, err
					}
					continue
				}
			}
			off := int(dec.InputOffset())
			vs := off + len(sub[off:]) - len(bytes.TrimLeft(sub[off:], jsonSpace+":,"))
			var v json.RawMessage
			if err := dec.Decode(&v); err != nil {
				return 0, 0, err
			}
			if !st.isIndex || count == st.index {
				found = append(found, [3]int{s + vs, s + int(dec.InputOffset()), s + nameAt})
			}
			count++
		}
		switch {
		case st.isIndex && len(found) == 0:
			return 0, 0, fmt.Errorf("index %d is past the end of the %d-element array at %s, line %d", st.index, count, at, line(s))
		case len(found) == 0:
			return 0, 0, fmt.Errorf("no name %q in the object at %s, line %d", st.name, at, line(s))
		case len(found) > 1:
			ls := make([]string, len(found))
			for i, f := range found {
				ls[i] = strconv.Itoa(line(f[2]))
			}
			times := "twice"
			if len(found) > 2 {
				times = fmt.Sprintf("%d times", len(found))
			}
			return 0, 0, fmt.Errorf("the name %q appears %s in the object at %s, at lines %s and %s: a path cannot say which",
				st.name, times, at, strings.Join(ls[:len(ls)-1], ", "), ls[len(ls)-1])
		}
		s, e = found[0][0], found[0][1]
		at += st.String()
	}
	return s, e, nil
}

// Projection is one --json-path expression (ADR-178): the text the caller
// wrote, which is printed beside its value, and the steps it parsed to.
type Projection struct {
	Expr  string
	steps []jsonStep
}

// ParseProjections reads the values of --json-path: each is cut at its commas
// outside a quoted step, every expression must begin `$.` or `$[` (which
// parseJSONPath assumes) and parse as ADR-155's grammar, and they come back in
// the order the values and their parts were given.
func ParseProjections(values []string) ([]Projection, error) {
	var out []Projection
	for _, v := range values {
		for _, expr := range splitProjection(v) {
			if !isJSONPath(expr) {
				return nil, fmt.Errorf("%q is not a JSON path: it begins $. or $[, then steps .name, [N] or [\"name\"]", expr)
			}
			steps, err := parseJSONPath(expr)
			if err != nil {
				return nil, err
			}
			out = append(out, Projection{Expr: expr, steps: steps})
		}
	}
	return out, nil
}

// splitProjection cuts v at its commas outside a quoted step: after `["` the
// text is a JSON string, so `\"` and `\\` are escapes and a comma is part of
// the name.
func splitProjection(v string) []string {
	var parts []string
	start, quoted := 0, false
	for i := 0; i < len(v); i++ {
		switch c := v[i]; {
		case quoted && c == '\\':
			i++
		case quoted && c == '"':
			quoted = false
		case !quoted && c == '"' && i > 0 && v[i-1] == '[':
			quoted = true
		case !quoted && c == ',':
			parts = append(parts, v[start:i])
			start = i + 1
		}
	}
	return append(parts, v[start:])
}

// project writes a projection of one file's bytes (ADR-178): a line
// `EXPR = VALUE` per expression, the value with its insignificant whitespace
// removed and every literal's bytes kept, `EXPR  NOT FOUND: why` for one the
// document does not hold, and, when the file is not exactly one JSON document,
// `NOT JSON: why` once. A line the budget dropped is listed on withheld with
// its expression. It returns the problems it found; it records and licenses
// nothing.
func project(w, withheld io.Writer, bud *Budget, sp Spec, b []byte, opt Options) (problems int) {
	for _, p := range opt.Project {
		s, e, err := jsonValue(b, p.steps)
		var value bytes.Buffer
		if err == nil {
			err = json.Compact(&value, b[s:e])
		}
		name, line, stop := sp.Path+":"+p.Expr, p.Expr+" = "+value.String(), false
		switch {
		case errors.Is(err, errNotJSON):
			name, line, stop = sp.Path, "NOT JSON: "+strings.TrimPrefix(err.Error(), errNotJSON.Error()+": "), true
		case err != nil:
			line = p.Expr + "  NOT FOUND: " + err.Error()
		}
		if opt.MaxCols > 0 && utf8.RuneCountInString(line) > opt.MaxCols {
			line = window(line, opt.MaxCols, nil)
		}
		fmt.Fprintln(w, line)
		// One problem per expression that did not reach the caller whole: a
		// miss, or a line the budget dropped.
		if bud.Over() {
			fmt.Fprintf(withheld, "%s: WITHHELD: --max-chars reached\n", name)
			problems++
		} else if err != nil {
			problems++
		}
		if stop {
			break
		}
	}
	return problems
}

// jsonKind names the kind of JSON value that begins with c.
func jsonKind(c byte) string {
	switch c {
	case '{':
		return "an object"
	case '[':
		return "an array"
	case '"':
		return "a string"
	case 't', 'f':
		return "a boolean"
	case 'n':
		return "null"
	}
	return "a number"
}
