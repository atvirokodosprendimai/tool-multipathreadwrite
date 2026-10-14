package read

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
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

// jsonSpan resolves a path against a file's bytes to the 1-based lines its
// value spans, first byte to last. The file must be exactly one JSON document;
// a leading UTF-8 byte order mark is skipped. A name that appears more than
// once in an object the path goes through is refused with every line it is on,
// since the path cannot say which copy it means (ADR-155).
func jsonSpan(b []byte, steps []jsonStep) (int, int, error) {
	cr := !bytes.Contains(b, []byte("\n")) && bytes.Contains(b, []byte("\r"))
	line := func(off int) int {
		if cr {
			return 1 + bytes.Count(b[:off], []byte("\r"))
		}
		return 1 + bytes.Count(b[:off], []byte("\n"))
	}
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
			return 0, 0, fmt.Errorf("not one JSON document: %w, at line %d", err, line(base+int(se.Offset)-1))
		}
		return 0, 0, fmt.Errorf("not one JSON document: %w", err)
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
		var found [][2]int
		count := 0
		for dec.More() {
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
				found = append(found, [2]int{s + vs, s + int(dec.InputOffset())})
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
				ls[i] = strconv.Itoa(line(f[0]))
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
	return line(s), line(e - 1), nil
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
