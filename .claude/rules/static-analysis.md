# Static analysis after every commit

`scripts/static.sh` is the static gate (ADR-088): `gofmt -l`, `go vet`, `golangci-lint` with
`.golangci.yml`, `deadcode`, staticcheck U1000 with tests excluded, and `govulncheck`. Exit 0 is
clean; anything else is a finding. CI runs it on every push and pull request.

**After a commit, the analysis comes to you.** In Claude Code, `.claude/hooks/static-after-commit.py`
runs the script whenever a `git commit` moved HEAD and puts the verdict in front of you. Without the
hook — hooks off, Codex, Cursor, a human — run `./scripts/static.sh` yourself after committing,
unpiped, and read its exit code.

**A finding is fixed in the next commit, or you say why not.** Never silence one to make the gate
green:
- An unchecked error on a best-effort path (`Close`, `Flush`, `Remove`, `fmt.Fprint`) is already
  excluded by the `std-error-handling` preset. Anything else errcheck names is real.
- A deliberate swallow gets `//nolint:<linter> // <why>` on that line, and the why must be true.
- A new exclusion goes in `.golangci.yml` with a comment saying why, never as a blanket rule.

**No dead code, and nothing in production that only a test reaches.** `deadcode` and U1000 with
`-tests=false` enforce it: a helper only tests need lives in a `_test.go` file, and code nothing
calls is deleted, not kept for later.

**The tools.** golangci-lint is the one binary to install (v2.14.0: `brew install golangci-lint`);
the script names the install if it is missing. deadcode, staticcheck and govulncheck run pinned
through `go run`, so there is nothing else to install. The first run downloads them; later runs use
the module cache.
