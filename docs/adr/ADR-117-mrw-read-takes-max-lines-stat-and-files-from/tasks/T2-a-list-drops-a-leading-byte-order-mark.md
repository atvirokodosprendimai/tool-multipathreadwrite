# Task ADR-117-T2: A spec list drops a leading byte order mark and says so

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `speclist.Parse(r, label, notes)`: a leading byte order mark dropped from the first line, one note naming the list; `TestAListThatStartsWithAByteOrderMarkDropsItAndSaysSo`; a contract section
**Consumes:** `specList` (`cmd/mrw/main.go`), `filesFrom` (`internal/mcp/tools.go`)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the mark is dropped from the first line only`, `the note names the list`, `the CLI hands Parse its standard error`

## Goal

A `--files-from` list that starts with a UTF-8 byte order mark serves its first spec. Measured on
v1.86.5 (babbf03), 2026-11-11: `printf '\xef\xbb\xbfgo.mod:1\n' | mrw read --files-from -` printed the
path with the mark in front of it as `UNREADABLE  open …: no such file or directory`, exit 1, and a list
whose first line was the mark followed by `# c` read the comment as a path. Notepad, `Out-File -Encoding
utf8` in Windows PowerShell 5.1, and a pipe under a UTF-8 `$OutputEncoding` with a preamble write that
mark (the same source as ADR-142).

Decided by Zy, 2026-11-11: *"ADR-117 task, strip and note on stderr, exempt?"*. The code and the test
were first written as the qh-core pilot task, proven there by qh's native Go evidence and left on a
local branch. This file is that task in this repository's form: its evidence is `adr-verify`'s, and
it adds the contract section the pilot left out.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/speclist/speclist_test.go` | new | the red test |
| `internal/speclist/speclist.go` | edit | drop the mark from the first line; write the note to `notes` |
| `cmd/mrw/main.go` | edit | `specList` passes `os.Stderr` |
| `internal/mcp/tools.go` | edit | `filesFrom` passes `nil`: an MCP caller has no standard error, and the server's own log is not the caller's |
| `scripts/contract.sh` | edit | the next free section, through the built binary |
| `docs/adr/BACKLOG.md` | edit | the deferral of a note on `mrw_read`'s answer |

## Ordered Steps

1. [S1] Write the failing test `TestAListThatStartsWithAByteOrderMarkDropsItAndSaysSo` and see it RED: `Parse` takes two arguments, so the package does not compile. Its cases: the mark before a first spec and before a first comment is dropped and noted; a list with no mark gets no note; a mark at the start of a later line stays the spec's own; a list of only the mark holds no spec. [proof: acceptance]
2. [S2] `Parse` drops the mark from the first line and writes `mrw: <label>: dropped the byte order mark at the start of the list` to `notes` when it is not nil; `specList` passes `os.Stderr` and `filesFrom` passes `nil`. Mutants, each killed: the mark is dropped from every line; the note is not written; the CLI passes `nil`. [proof: mutation]
3. [S3] The contract section, through the built binary: a list led by the mark serves both of its files, exit 0, and standard error carries the note; the pair: a mark at the start of the second line leaves that spec UNREADABLE, exit 1, with no note. [proof: acceptance]
4. [S4] `go test ./internal/speclist/ ./internal/mcp/` pass. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/speclist/ -count=1 -v -run '^TestAListThatStartsWithAByteOrderMarkDropsItAndSaysSo$' 2>&1) \
  && grep -qE '^--- PASS: TestAListThatStartsWithAByteOrderMarkDropsItAndSaysSo \(' <<<"$out" \
  && go test ./internal/speclist/ ./internal/mcp/ -count=1 >/dev/null 2>&1 \
  && t=$(mktemp -d) && mkdir "$t/r" && printf 'x\n' > "$t/r/a.txt" \
  && go build -o "$t/mrw" ./cmd/mrw \
  && printf '\xef\xbb\xbfa.txt\n' | env XDG_STATE_HOME="$t/state" "$t/mrw" --root "$t/r" read --files-from - >"$t/out" 2>"$t/err" \
  && grep -q '^==> a.txt' "$t/out" \
  && grep -qxF 'mrw: --files-from -: dropped the byte order mark at the start of the list' "$t/err" \
  && rm -rf "$t" \
  && grep -q '^# 341\. ' scripts/contract.sh \
  && f=$(gofmt -l internal cmd) && [ -z "$f" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/apply internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAListThatStartsWithAByteOrderMarkDropsItAndSaysSo` | `internal/speclist/speclist_test.go` | the mark leaves the first spec and the first comment and is noted; no note without it; a mark past the first line is kept; a list of only the mark is empty | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | every `mrw read --files-from` and every `mrw_read` with `files_from` calls `Parse`; the fence and the contract section drive the built binary |
| 3 — the caller can discover it | the note on standard error names the list |
| 4 — it is used | no telemetry (ADR-009); the evidence is the measurement in Goal |

## Mutation Log
- 2026-11-17 · 956090a5* · mutant killed · exit 1 · `internal/speclist/speclist.go` · the mark is dropped from every line, so the mark at the start of a later line must go red · acceptance-sha256:e9e64a809b43c91cfebe904c49f821d0c2057a9d3bec92e18f2fb1d9baa2aa5a · covers:the mark is dropped from the first line only
- 2026-11-17 · 956090a5* · mutant killed · exit 1 · `internal/speclist/speclist.go` · the note names no list, so the noted cases must go red · acceptance-sha256:e9e64a809b43c91cfebe904c49f821d0c2057a9d3bec92e18f2fb1d9baa2aa5a · covers:the note names the list
- 2026-11-17 · 956090a5* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI hands Parse no writer, so the built binary must say nothing and the fence go red · acceptance-sha256:e9e64a809b43c91cfebe904c49f821d0c2057a9d3bec92e18f2fb1d9baa2aa5a · covers:the CLI hands Parse its standard error

## Verification Log
- 2026-11-17 · 956090a5* · exit 1 · `set -o pipefail …` · acceptance-sha256:e9e64a809b43c91cfebe904c49f821d0c2057a9d3bec92e18f2fb1d9baa2aa5a · ms:189 · test-lock-sha256:8683eecf7d2757c40174c2c6684cd5e7bfca9af45d0dd0e425dd441384cbdb8c · test-lock-file:docs/adr/locks/8683eecf7d2757c40174c2c6684cd5e7bfca9af45d0dd0e425dd441384cbdb8c.lock
  ```
  ```
- 2026-11-17 · 956090a5* · exit 0 · `set -o pipefail …` · acceptance-sha256:e9e64a809b43c91cfebe904c49f821d0c2057a9d3bec92e18f2fb1d9baa2aa5a · ms:46051
- 2026-11-17 · 956090a5* · exit 0 · `set -o pipefail …` · acceptance-sha256:e9e64a809b43c91cfebe904c49f821d0c2057a9d3bec92e18f2fb1d9baa2aa5a · ms:51797
- 2026-11-17 · 956090a5* · exit 0 · `set -o pipefail …` · acceptance-sha256:e9e64a809b43c91cfebe904c49f821d0c2057a9d3bec92e18f2fb1d9baa2aa5a · ms:39105
- 2026-11-17 · 956090a5* · exit 0 · `set -o pipefail …` · acceptance-sha256:e9e64a809b43c91cfebe904c49f821d0c2057a9d3bec92e18f2fb1d9baa2aa5a · ms:41578

## Invariants

- A list with no leading mark parses exactly as before: the same specs, no note. Edge spaces are kept (ADR-069).
- The note goes to the writer the caller hands in and nowhere else; standard output is unchanged.

## Risks

- A list whose first spec is meant to begin with U+FEFF would be renamed. None is known; the Stop Condition names it.
- The note is one more line on standard error for a caller that parses it; standard output and the exit code are unchanged.

## Stop Condition

The fence exits 0. Stop and ask Zy if a spec that legitimately starts with U+FEFF on a list's first line turns up: it would be renamed.

## Out of Scope

- A note on `mrw_read`'s answer for a `files_from` list with the mark (deferred: docs/adr/BACKLOG.md)
- A spec that begins with U+FEFF on a list's first line (permanent: boundary: the mark is dropped; a path cannot be typed with one by accident, and none is known)
