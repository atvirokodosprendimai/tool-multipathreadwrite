// Command content-edit applies exact whole-line edits in a disposable benchmark checkout.
package main

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer"
)

type edit struct {
	Target           string
	Old, Replacement lineList
}

// UnmarshalJSON rejects duplicate fields before one can overwrite another edit target.
func (e *edit) UnmarshalJSON(raw []byte) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	if token, err := d.Token(); err != nil || token != json.Delim('{') {
		return errors.New("expected an edit object")
	}
	fields := map[string]any{"target": &e.Target, "old": &e.Old, "replacement": &e.Replacement}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return err
		}
		key, _ := token.(string)
		dest, ok := fields[key]
		if !ok {
			return fmt.Errorf("unknown or repeated edit field %q", key)
		}
		if err := d.Decode(dest); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		delete(fields, key)
	}
	return nil
}

type lineList []string

// UnmarshalJSON keeps null distinct from an empty string or an explicit deletion.
func (l *lineList) UnmarshalJSON(raw []byte) error {
	var values []*string
	if err := json.Unmarshal(raw, &values); err != nil {
		return err
	}
	if values == nil {
		return errors.New("expected an array of lines")
	}
	*l = make(lineList, len(values))
	for i, v := range values {
		if v == nil || strings.ContainsAny(*v, "\x00\r\n") {
			return fmt.Errorf("element %d must be a string containing one line", i+1)
		}
		(*l)[i] = *v
	}
	return nil
}

func bounded(r io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("exceeds %d bytes", limit)
	}
	return b, err
}

func readEdits(r io.Reader) ([]edit, error) {
	b, err := bounded(r, 8<<20)
	if err != nil {
		return nil, err
	}
	if !jsontext.Value(b).IsValid(jsontext.AllowDuplicateNames(true)) {
		return nil, errors.New("expected one UTF-8 JSON document")
	}
	var edits []edit
	if err := json.Unmarshal(b, &edits); err != nil {
		return nil, err
	}
	if len(edits) == 0 {
		return nil, errors.New("expected a nonempty array of edits")
	}
	for i, e := range edits {
		e.Target = filepath.FromSlash(e.Target)
		edits[i].Target = e.Target
		if rooted.IsRooted(e.Target) || !filepath.IsLocal(e.Target) || filepath.Clean(e.Target) != e.Target || e.Target == "." {
			return nil, fmt.Errorf("edit %d: expected a clean root-relative target", i+1)
		}
		if len(e.Old) == 0 || e.Replacement == nil {
			return nil, fmt.Errorf("edit %d: expected nonempty old lines and an explicit replacement array", i+1)
		}
	}
	return edits, nil
}

func loadTarget(root, path string) ([]string, string, error) {
	if err := rooted.GitDir(root, path); err != nil {
		return nil, "", err
	}
	full, err := rooted.Resolve(root, path)
	if err != nil {
		return nil, "", err
	}
	f, _, err := rooted.OpenRead(root, full)
	if err != nil {
		return nil, "", err
	}
	b, err := bounded(f, 16<<20)
	if err := errors.Join(err, f.Close()); err != nil {
		return nil, "", err
	}
	if !utf8.Valid(b) || lines.Unsplittable(b) != "" {
		return nil, "", errors.New("expected a UTF-8 text file")
	}
	text, _, _ := lines.Split(string(b))
	for _, line := range text {
		if strings.ContainsRune(line, '\r') {
			return nil, "", errors.New("expected consistent line endings with no embedded carriage return")
		}
	}
	return text, seen.SHA(b), nil
}

func match(text, old []string, positions map[string][]int, remaining *int) (int, error) {
	// A blank line or brace commonly starts many blocks. Try their rarest line.
	anchor, cost := 0, 0
	for i, line := range old {
		cost += 1 + len(line)
		if len(positions[line]) < len(positions[old[anchor]]) {
			anchor = i
		}
	}
	found := -1
	for _, pos := range positions[old[anchor]] {
		if cost > *remaining {
			return 0, errors.New("batch exceeds the exact-match comparison budget")
		}
		*remaining -= cost
		i := pos - anchor
		if i >= 0 && len(old) <= len(text)-i && slices.Equal(text[i:i+len(old)], old) {
			if found >= 0 {
				return 0, errors.New("ambiguous match")
			}
			found = i
		}
	}
	if found < 0 {
		return 0, errors.New("no match")
	}
	return found + 1, nil
}

func execute(root string, r io.Reader) (int, error) {
	edits, err := readEdits(r)
	if err != nil {
		return 0, err
	}
	unlock, err := seen.LockWrites(root)
	if err != nil {
		return 0, err
	}
	defer unlock()

	var paths []string
	byPath := make(map[string][]int)
	for i, e := range edits {
		if _, ok := byPath[e.Target]; !ok {
			paths = append(paths, e.Target)
		}
		byPath[e.Target] = append(byPath[e.Target], i)
	}

	var inputs []apply.Input
	var failures []error
	remaining := 64 << 20
	observations := make(map[string]seen.Observation)
	for _, path := range paths {
		text, sha, readErr := loadTarget(root, path)
		positions := make(map[string][]int)
		for _, i := range byPath[path] {
			for _, line := range edits[i].Old {
				positions[line] = nil
			}
		}
		for i, line := range text {
			if _, needed := positions[line]; needed {
				positions[line] = append(positions[line], i)
			}
		}
		obs := seen.Observation{SHA: sha}
		for _, i := range byPath[path] {
			e := edits[i]
			start, err := 0, readErr
			if err == nil {
				start, err = match(text, e.Old, positions, &remaining)
			}
			if err != nil {
				failures = append(failures, fmt.Errorf("edit %d (%q): %w", i+1, path, err))
				continue
			}
			end := start + len(e.Old) - 1
			obs.Spans = append(obs.Spans, [2]int{start, end})
			// A guarded delete can verify even a block starting with a blank line.
			// The engine orders an insertion at its start before consuming that block.
			h := apply.Input{Path: path, Start: start, End: start, SHA: sha, Lines: -1, SrcLine: i + 1}
			if len(e.Replacement) > 0 {
				h.Op, h.Body, h.Index = "insert-before", e.Replacement, len(inputs)
				inputs = append(inputs, h)
			}
			h.Op, h.Body, h.End, h.Index = "delete", e.Old, end, len(inputs)
			inputs = append(inputs, h)
		}
		observations[path] = obs
	}
	if len(failures) > 0 {
		return 0, errors.Join(failures...)
	}
	res, err := apply.Apply(root, inputs, apply.Options{Seen: observations, NoForce: true})
	if err != nil {
		return 0, fmt.Errorf("write outcome %s: %w", writer.MutationOf(res), err)
	}
	for _, h := range res.Hunks {
		if h.Status == apply.StatusFailed {
			failures = append(failures, fmt.Errorf("edit %d (%q): %s", h.SrcLine, h.Path, h.Reason))
		}
	}
	if len(failures) > 0 {
		return 0, errors.Join(failures...)
	}
	return len(edits), nil
}

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: content-edit < edits.json")
		os.Exit(2)
	}
	root, err := os.Getwd()
	if err == nil {
		var n int
		n, err = execute(root, os.Stdin)
		if err == nil {
			fmt.Printf("applied %d edit(s)\n", n)
			return
		}
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
