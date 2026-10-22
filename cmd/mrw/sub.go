package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// exitOf turns a plan builder's refusal into the exit it names: 2 for a usage
// mistake, 1 when nothing was planned or a path cannot be carried. The builders
// live in internal/read, which the MCP surface calls too (ADR-161).
func exitOf(err error) error {
	var be *read.BuildError
	if !errors.As(err, &be) {
		return err
	}
	// The BuildError itself is kept, not its text, so a state-directory failure
	// under it still reaches statehint.Hint (ADR-164).
	if be.Usage {
		return &exitError{err: be, code: exitUsage}
	}
	return &exitError{err: be, code: exitNotApplied}
}

// subRead is `read --grep P --sub R`: the plan on stdout and nothing else,
// the count on stderr, and the planned lines recorded as read.
func subRead(cmd *cli.Command, pattern string, excludes, posArgs []string) error {
	root := cmd.Root().String("root")
	b, err := read.BuildSub(root, posArgs, pattern, cmd.String("sub"), excludes, cmd.Bool("no-ignore"), os.Stderr)
	if err != nil {
		return exitOf(err)
	}
	return servePlan(root, b)
}

// servePlan prints a built plan on stdout and nothing else, the count on
// stderr, and records the planned lines as read: the tail of every read that
// makes a plan (--sub, --rewrite).
func servePlan(root string, b *read.Built) error {
	out := bufio.NewWriter(os.Stdout)
	_, _ = out.Write(b.Plan) // a failed write surfaces at Flush
	// The plan reaches the caller BEFORE anything is recorded (ADR-088).
	if err := out.Flush(); err != nil {
		return cli.Exit(fmt.Sprintf("the answer could not be written, so nothing was recorded: %v", err), exitUsage)
	}
	fmt.Fprintf(os.Stderr, "-- planned: %d site(s) in %d file(s)\n", b.Sites, b.Files)
	if toNullDevice(os.Stdout) {
		fmt.Fprintln(os.Stderr, "mrw: the answer went to the null device, so nothing was recorded: a write to these lines needs a read whose answer you see (ADR-133); read again without redirecting, or license lines you already know with `mrw read --quiet SPEC`")
	} else if err := seen.Record(root, b.Observed); err != nil {
		return exitWith(err)
	}
	if b.Problems > 0 {
		return cli.Exit(fmt.Sprintf("incomplete: %d range(s) could not be served", b.Problems), exitNotApplied)
	}
	return nil
}
