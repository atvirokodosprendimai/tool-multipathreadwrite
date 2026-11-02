package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

const (
	hookNewline  = "\n"
	hookDeadline = time.Second // the whole command: standard input and the ledger lock
	// hookMaxBytes bounds what the command reads of standard input and of the
	// file; a host's Read shows at most a few thousand lines.
	hookMaxBytes = 67108864
)

// hookCmd is `mrw hook`: the commands a host runs from its own hooks (ADR-191).
func hookCmd() *cli.Command {
	return &cli.Command{
		Name:     `hook`,
		Usage:    `commands a host's hook runs`,
		Commands: []*cli.Command{hookReadCmd()},
		// A host's hook runs it, not a person or an agent, and the root help is held to 30 lines (ADR-168).
		CommandNotFound: func(_ context.Context, cmd *cli.Command, name string) {
			// A typo in a hook block is a usage error, exit 2; the framework exit 3 means a check failed.
			cmd.Root().Metadata = map[string]any{`notFound`: cli.Exit(fmt.Sprintf(`unknown command %q (want read)`, name), exitUsage)}
		},
		Hidden: true,
	}
}

func hookReadCmd() *cli.Command {
	return &cli.Command{
		Name:  `read`,
		Usage: `record in the ledger the lines a host's Read showed (PostToolUse payload on standard input)`,
		Description: `Reads one PostToolUse JSON object for the host's Read tool on standard input and
records as served the lines the Read returned whose text equals the file on disk now,
as a mrw read of them would (ADR-191). Anything else records nothing. It prints nothing
on standard output, exits 0 whatever it is given, and gives up on the ledger lock after
one second. The root is --root when given, else the payload's cwd. docs/agents.md has
the Claude Code hook block.`,
		Action: func(_ context.Context, cmd *cli.Command) error {
			root := cmd.Root()
			hookRead(os.Stdin, root.String(`root`), root.IsSet(`root`))
			return nil
		},
	}
}

// hookField decodes obj[key] into v, and says whether the key was there with
// that type: a payload of any other shape licenses nothing.
func hookField(obj map[string]json.RawMessage, key string, v any) bool {
	raw, ok := obj[key]
	// A null decodes into any Go value without error, as an empty one: it is no
	// text, no object and no number, so it is refused here.
	return ok && strings.TrimSpace(string(raw)) != `null` && json.Unmarshal(raw, v) == nil
}

// hookStdin reads standard input to its end, at most hookMaxBytes, but not past
// deadline: a host that never closes the pipe must not hold the command. The
// reader is left behind when the deadline passes; the process is about to exit.
func hookStdin(in io.Reader, deadline time.Time) ([]byte, bool) {
	type result struct {
		b   []byte
		err error
	}
	done := make(chan result, 1)
	go func() {
		b, err := io.ReadAll(io.LimitReader(in, hookMaxBytes))
		done <- result{b, err}
	}()
	select {
	case r := <-done:
		return r.b, r.err == nil
	case <-time.After(time.Until(deadline)):
		return nil, false
	}
}

// hookRead is the command. Claude Code 2.1.316 sends a Read as
// tool_response.file.content, the lines joined by a newline with no line
// numbers, from file.startLine: what the model received (S1,
// testdata/hookread191). It never fails and never prints but the one line a
// person would want (ADR-191 Decision 2).
func hookRead(in io.Reader, rootFlag string, rootSet bool) {
	deadline := time.Now().Add(hookDeadline)
	b, ok := hookStdin(in, deadline)
	if !ok {
		return
	}
	var top, input, resp, file map[string]json.RawMessage
	var kind, filePath, tool, cwd, path, content string
	var start int
	var failed bool
	if json.Unmarshal(b, &top) != nil || !hookField(top, `tool_name`, &tool) || tool != `Read` {
		return
	}
	if !hookField(top, `tool_input`, &input) || !hookField(input, `file_path`, &path) ||
		!hookField(top, `tool_response`, &resp) || !hookField(resp, `file`, &file) ||
		!hookField(file, `content`, &content) || !hookField(file, `startLine`, &start) || start < 1 {
		return
	}
	// Only the shape S1 measured: a text Read. A notebook or an image is not lines the model was shown.
	// filePath is what the host wrote of the file it read: a disagreement with the input licenses nothing.
	if !hookField(resp, `type`, &kind) || kind != `text` || (hookField(file, `filePath`, &filePath) && filePath != path) {
		return
	}
	if hookField(resp, `is_error`, &failed) && failed {
		return
	}
	root := rootFlag
	if !rootSet && hookField(top, `cwd`, &cwd) && cwd != `` {
		root = cwd
	}
	key, obs, ok := hookLicensed(root, path, content, start)
	if !ok {
		return
	}
	// ADR-150: a newer mrw's ledger is left alone, and the person is told.
	if writer, newer, err := seen.Newer(root); err == nil && newer {
		fmt.Fprintln(os.Stderr, seen.NewerNotice(writer))
		return
	}
	left := time.Until(deadline)
	if left <= 0 {
		fmt.Fprintf(os.Stderr, `mrw hook read: nothing recorded: the %s deadline passed before the ledger lock`+hookNewline, hookDeadline)
		return
	}
	if err := seen.RecordWithin(root, map[string]seen.Observation{key: obs}, left); err != nil {
		fmt.Fprintf(os.Stderr, `mrw hook read: nothing recorded: %v`+hookNewline, err)
	}
}

// hookRel is path relative to root, or false when it lies outside. A host sends
// an absolute path (a backslash one on Windows), made relative the way mrw read
// makes one, so a ledger key is the same for both; the boundary is then
// rooted.Resolve's, as a read's.
func hookRel(root, path string) (string, bool) {
	if !rooted.IsRooted(path) {
		return filepath.Clean(path), true
	}
	// Exactly what mrw read does with an absolute path (internal/read, the
	// argPath block): the link is followed, so the key is the file's own name.
	absRoot, err := rooted.Abs(root)
	if err != nil {
		return ``, false
	}
	cleaned := rooted.Real(path)
	if !rooted.Contains(absRoot, cleaned) {
		return ``, false
	}
	rel, err := filepath.Rel(absRoot, cleaned)
	return rel, err == nil
}

// hookLicensed is the ledger key and observation for the lines of content,
// numbered from start, whose text equals the same line of the file on disk. The
// file is read once and hashed once, so the lines and the sha are one version.
// Lines are mrw's own split (internal/lines): a CRLF line has no CR, as the host
// showed it, and the empty element after a final newline is no line.
func hookLicensed(root, path, content string, start int) (string, seen.Observation, bool) {
	none := seen.Observation{}
	rel, ok := hookRel(root, path)
	if !ok || rooted.GitDir(root, rel) != nil {
		return ``, none, false
	}
	full, err := rooted.Resolve(root, rel)
	if err != nil {
		return ``, none, false
	}
	f, fi, err := regular.Open(full)
	if err != nil {
		return ``, none, false
	}
	defer func() { _ = f.Close() }()
	if fi.IsDir() || fi.Size() > hookMaxBytes {
		return ``, none, false
	}
	raw, err := io.ReadAll(io.LimitReader(f, hookMaxBytes))
	if err != nil {
		return ``, none, false
	}
	disk, _, _ := lines.Split(string(raw))
	var spans [][2]int
	if start > len(disk) {
		return ``, none, false
	}
	// Lines start..len(disk), then one element for the rest: the content is not split further than the file can use.
	for i, shown := range strings.SplitN(content, hookNewline, len(disk)-start+2) {
		n := start + i
		if n > len(disk) {
			break
		}
		if disk[n-1] != shown {
			continue
		}
		if k := len(spans) - 1; k >= 0 && spans[k][1] == n-1 {
			spans[k][1] = n
		} else {
			spans = append(spans, [2]int{n, n})
		}
	}
	if len(spans) == 0 {
		return ``, none, false
	}
	return filepath.Clean(rel), seen.Observation{SHA: seen.SHA(raw), Spans: spans}, true
}
