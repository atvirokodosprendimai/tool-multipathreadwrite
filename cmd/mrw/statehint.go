package main

import "github.com/urfave/cli/v3"

// exitWith is cli.Exit with exit 2 (exitUsage) for an error that stays unwrappable.
// urfave/cli's exit error has no Unwrap, so errors.As could not reach the
// *fs.PathError under a command that failed in the state directory, and
// statehint.Hint could not see it. Every site it replaced exited 2.
func exitWith(err error) cli.ExitCoder {
	return &exitError{err: err, code: exitUsage}
}

// exitSaying is exitWith with the words a site already printed, so a failure
// keeps its text and its type: the landing's exitTwoLine points at the FAIL
// line instead of repeating the reason, and still unwraps to the error.
func exitSaying(msg string, err error) cli.ExitCoder {
	return &exitError{err: err, msg: msg, code: exitUsage}
}

// exitError is a failure that exits with its own code and unwraps to its error:
// 2 for a usage-or-filesystem failure, and a plan builder's refusal (exitOf) 1
// or 2.
type exitError struct {
	err  error
	msg  string // what to print, when it is not err's own text
	code int
}

func (e *exitError) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return e.err.Error()
}
func (e *exitError) ExitCode() int { return e.code }
func (e *exitError) Unwrap() error { return e.err }
