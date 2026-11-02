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
	hookLockWait = time.Second
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
	return ok && json.Unmarshal(raw, v) == nil
}

// hookRead is the command. Claude Code 2.1.316 sends a Read as
// tool_response.file.content, the lines joined by a newline with no line
// numbers, from file.startLine: what the model received (S1,
// testdata/hookread191). It never fails and never prints but the one line a
// person would want (ADR-191 Decision 2).
func hookRead(in io.Reader, rootFlag string, rootSet bool) {
	b, err := io.ReadAll(io.LimitReader(in, hookMaxBytes))
	if err != nil {
		return
	}
	var top, input, resp, file map[string]json.RawMessage
	var tool, cwd, path, content string
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
	err = seen.RecordWithin(root, map[string]seen.Observation{key: obs}, hookLockWait)
	if err != nil {
		fmt.Fprintf(os.Stderr, `mrw hook read: nothing recorded: %v`+hookNewline, err)
	}
}

// hookRel is path relative to root, or false when it lies outside. A host sends
// an absolute path (a backslash one on Windows), so a rooted path is made
// relative to the root as spelled and to where it leads (a temporary directory
// that is a link); the boundary is then rooted.Resolve's, as a read's.
func hookRel(root, path string) (string, bool) {
	if !rooted.IsRooted(path) {
		return filepath.Clean(path), true
	}
	bases := []string{}
	if spelled, err := filepath.Abs(root); err == nil {
		bases = append(bases, spelled)
	}
	if real, err := rooted.Abs(root); err == nil {
		bases = append(bases, real)
	}
	for _, base := range bases {
		rel, err := filepath.Rel(base, path)
		if err == nil && rel != `..` && !strings.HasPrefix(rel, `..`+string(filepath.Separator)) {
			return rel, true
		}
	}
	return ``, false
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
	for i, shown := range strings.Split(content, hookNewline) {
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
