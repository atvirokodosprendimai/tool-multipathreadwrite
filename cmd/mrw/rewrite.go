package main

import (
	"os"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

// rewriteRead is `read --ast-grep P --rewrite R`: the plan on stdout and
// nothing else, the count on stderr, and the planned spans recorded as read.
func rewriteRead(cmd *cli.Command, pattern string, excludes, posArgs []string) error {
	root := cmd.Root().String("root")
	b, err := read.BuildRewrite(root, posArgs, pattern, cmd.String("rewrite"), excludes, cmd.Bool("no-ignore"), os.Stderr)
	if err != nil {
		return exitOf(err)
	}
	return servePlan(root, b)
}
