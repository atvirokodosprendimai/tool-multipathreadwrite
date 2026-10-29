# Static analysis after every commit

`scripts/static.sh` is the static gate (ADR-088): `gofmt -l`, `go vet`, `golangci-lint` with
`.golangci.yml`, `deadcode`, staticcheck U1000 with tests excluded, `govulncheck`, and
`scripts/fence-prose.py`, which reruns every task fence's `grep` over a tracked file so a prose rewrite
cannot turn a done task red unseen. For what is not Go (ADR-147) it runs ruff over the Python files,
shellcheck over the shell scripts, actionlint over the workflows and knip over the opencode plugin,
then a self-test that each of those four still reports a defect planted for it. Exit 0 is clean;
anything else is a finding. CI runs it on every push and pull request.

**After a commit, the analysis comes to you.** In Claude Code, `.claude/hooks/static-after-commit.py`
runs the script whenever a `git commit` moved HEAD and puts the verdict in front of you. Without the
hook — hooks off, Codex, Cursor, a human — run `./scripts/static.sh` yourself after committing,
unpiped, and read its exit code.

**A finding is fixed in the next commit, or you say why not.** Never silence one to make the gate
green:
- errcheck is whole. Its one exclusion is `fmt.Fprint*`, whose error reaches a checked `Flush` or
  goes to a buffer or stderr. Any other error you mean to drop is written `_ = f.Close()` at its
  site, where a reader sees the choice — and only when dropping it cannot make mrw report something
  false. An unchecked output flush once licensed lines nobody saw (ADR-088 T4).
- A deliberate swallow of a non-nil error gets `//nolint:<linter> // <why>` on that line, and the
  why must be true.
- A new exclusion goes in `.golangci.yml` with a comment saying why, never as a blanket rule.

**No dead code, and nothing in production that only a test reaches.** `deadcode` and U1000 with
`-tests=false` enforce it: a helper only tests need lives in a `_test.go` file, and code nothing
calls is deleted, not kept for later.

**The tools.** Three binaries to install: golangci-lint v2.14.0 (`brew install golangci-lint`),
ruff 0.15.6 (`brew install ruff`, or `pipx install ruff==0.15.6`) and shellcheck 0.11.0
(`brew install shellcheck`); the script names each install when one is missing and notes a version
other than CI's. deadcode, staticcheck, govulncheck and actionlint run pinned through `go run`, and every Go
tool runs under the toolchain `go.mod` names (`GOTOOLCHAIN`, exported at the top of the script), the one CI
builds with, so a newer local `go` changes no finding. knip is
the plugin's exact devDependency: run `npm ci` in `cmd/opencode/mrw-plugin` once. The first run
downloads the Go tools; later runs use the module cache.
