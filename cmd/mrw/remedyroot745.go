package main

import (
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

// remedyRoot is the checkout as the command line named it, when it is not the
// working directory: the read a refusal tells the caller to make runs from where
// the caller is, so it must say where the root is (review of #745).
func remedyRoot(cmd *cli.Command) string {
	if !cmd.Root().IsSet("root") {
		return ""
	}
	r := cmd.Root().String("root")
	abs, err := filepath.Abs(r)
	if wd, werr := os.Getwd(); err == nil && werr == nil && abs == wd {
		return ""
	}
	return r
}
