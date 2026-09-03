# AGENTS.md — working in this repository

For any coding agent (Codex, Cursor, Claude Code, or a human who wants the short
version). Nothing here requires a plugin, an MCP server, or a private skill.

## What this is

`mrw` is one Go binary at `./cmd/mrw`. It reads many ranges across many files,
and applies many edits across many files, in **one** invocation — and it reports
a verdict for **every** edit. Read `README.md` for the full interface; this file
is only about changing the code.

## Build and check

```sh
go build -o bin/mrw ./cmd/mrw          # bin/mrw.exe on Windows
go test ./...                          # the project's declared check
go test -race ./...
gofmt -l .                             # must print nothing
go vet ./...
./scripts/contract.sh                  # bash; WSL or Git Bash on Windows
```

`.quality-harness.json` names `go test ./...` as the check. A run that is piped
or ends in `|| true` is not evidence: the exit code is the whole verdict, and a
pipeline reports the *last* command's status, not the script's.

## The rules this codebase is built around

These are decided, recorded in `docs/adr/`, and asserted by
`scripts/contract.sh`. Do not relax one without retiring the ADR.

1. **A plan applies whole or not at all.** Any failing hunk writes nothing.
   Siblings report `skip`, never `ok`. — ADR-001
2. **mrw will not edit a file it has not read**, and the guard is per *line*,
   not per file. — ADR-002
3. **A check's verdict comes from the process, never its output.** A check that
   prints `PASS` and exits 1 is a failure. — ADR-003
4. **Nothing is left in the working tree** by a failed run. — ADR-004
5. **mrw finds the files it serves**, and says which path it looked for when it
   cannot. — ADR-007
6. **A delete says what it removed.** — ADR-008

The failure the whole design exists to prevent: *a read that returns nothing is
visible; a write that changes nothing is not.*

## Exit codes are the contract

`0` fine · `1` a hunk failed and nothing was written · `2` usage or filesystem
failure · `3` the write applied but `--check` failed, so the tree is changed and
unverified. Tests assert these; changing one is a breaking change.

## Adding behaviour

- A new promise needs a row in `scripts/contract.sh` that makes it go wrong on
  purpose. A test that cannot fail is worse than no test.
- A durable decision — a public contract, an exit code, a trust boundary — needs
  an ADR in `docs/adr/`, not just a commit message.
- Exported identifiers carry doc comments starting with the name. Packages carry
  a package comment.

## Using mrw on this repository

Building and using `./bin/mrw` here is fine — it is the project. `mrw read` the
lines first; a write to lines it has not served you is refused, per rule 2.

See `CONTRIBUTING.md` for the full gate list and the release process.
