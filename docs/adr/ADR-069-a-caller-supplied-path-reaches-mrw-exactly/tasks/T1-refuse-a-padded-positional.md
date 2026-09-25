# Task ADR-069-T1: The CLI refuses a positional its parser trimmed; contract §128

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `refusePaddedArgs` and the exit-2 refusal naming `--`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a padded positional is refused`, `dash-dash still serves the padded path`, `the binary names --`, `no engine file changes`

## Goal

urfave/cli v3.11.0 trims every positional before `--` (`command_parse.go:81`, `:117`), so `mrw read 'x '`
serves `x`. A helper `refusePaddedArgs(cmd)` in `cmd/mrw/main.go`, run first in the `read`, `write`,
`iter` and `check` Actions, walks `cmd.Root().Args()` after the subcommand name up to the first `--`.
A token that differs from its trim, and whose trim is in `cmd.Args()`, is refused with `cli.Exit(…,
exitUsage)`. The message names the token and `--`. `iter note` is skipped: a note is free text.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `refusePaddedArgs`; four Action call sites |
| `cmd/mrw/paddedarg_test.go` | new | `TestAPaddedPositionalWithoutDashDashIsRefused` |
| `scripts/contract.sh` | edit | §128 |

## Ordered Steps

1. [S1] Write the test; confirm RED on an assertion (today `read 'x '` exits 0 and serves `x`). [proof: mutation]
   - Files `x` and `x ` with identical bytes. `read 'x '`, `write 'p '` (a plan file named `p `) and
     `check 'x '` each exit 2, and the error names `'x '` (or `'p '`) and `--`.
   - `read -- 'x '` exits 0 and its header names `x `.
   - `read --grep ' same' x` is NOT refused: a padded flag value whose trim is no positional passes. This kills a helper that refuses every padded token.
   - `iter note 'wip '` is NOT refused.
2. [S2] Add the helper and call it from the four Actions; GREEN; every `cmd/mrw` test stays green. [proof: mutation]
   Mutant: the helper returns nil.
3. [S3] §128 through the binary: `read 'x '` exits 2 naming `--`; `read -- 'x '` exits 0. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 128\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ -count=1 -v -run 'TestAPaddedPositionalWithoutDashDashIsRefused' 2>&1 | tee /tmp/adr069-T1.out \
  && grep -q '^--- PASS: TestAPaddedPositionalWithoutDashDashIsRefused ' /tmp/adr069-T1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr069-T1.out \
  && ./scripts/contract.sh > /tmp/adr069-T1-contract.out 2>&1 \
  && grep -q '^  PASS  a padded positional without -- is refused and names --' /tmp/adr069-T1-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/lines)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l cmd/mrw internal/iter internal/apply internal/ingest)" ] \
  && go vet ./cmd/mrw/ ./internal/iter/ ./internal/apply/ ./internal/ingest/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPaddedPositionalWithoutDashDashIsRefused` | `cmd/mrw/paddedarg_test.go` | read/write/check refuse a padded positional; `--` serves it; a padded flag value is not refused | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and §128 |
| 2 — something selects it | the CLI / compiler path every caller of that surface takes |
| 3 — the caller can discover it | the header, receipt or refusal names the exact path |
| 4 — it is used | §128 drives the built binary |

## Verification Log
(empty until execute)
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:46fe9123080575a15e79a83e67cf199efcd4046e1956e5f7474b9c939d8f8f2a · ms:31283
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:46fe9123080575a15e79a83e67cf199efcd4046e1956e5f7474b9c939d8f8f2a · ms:30649
- 2026-09-25 · c43095d* · exit 1 · `set -o pipefail …` · acceptance-sha256:46fe9123080575a15e79a83e67cf199efcd4046e1956e5f7474b9c939d8f8f2a · ms:793 · test-lock-sha256:c7ab13ec334e4ff77984079a7ed920aee7ed9b08c54871b0515969c9c30acb7c · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkYXJnX3Rlc3QuZ28JVGVzdEFQYWRkZWRQb3NpdGlvbmFsV2l0aG91dERhc2hEYXNoSXNSZWZ1c2VkCTQ4MGUwZWRlZDMwMTM4OGQzNDg1ZmJiMDhhZDEwODk1NGQwZTA4MGQ3M2I1ZGVjNWVhZWU3Zjk0Yjc0NmMwMjc
  ```
  --- last 10 line(s) of stdout (of 18 after folding 18 raw)
              1| the same
      paddedarg_test.go:60: ["write" "--no-check" "/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAPaddedPositionalWithoutDashDashIsRefused172268536/001/p "]: the refusal does not name "/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAPaddedPositionalWithoutDashDashIsRefused172268536/001/p " and --:
          open /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAPaddedPositionalWithoutDashDashIsRefused172268536/001/p: no such file or directory
      paddedarg_test.go:60: ["check" "x "]: the refusal does not name "x " and --:
          check SKIPPED: no check declared and no go.mod found
          no check could run: no check declared and no go.mod found — declare one in .quality-harness.json
  --- FAIL: TestAPaddedPositionalWithoutDashDashIsRefused (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.008s
  FAIL
  ```
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:46fe9123080575a15e79a83e67cf199efcd4046e1956e5f7474b9c939d8f8f2a · ms:30408
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:46fe9123080575a15e79a83e67cf199efcd4046e1956e5f7474b9c939d8f8f2a · ms:30064
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:46fe9123080575a15e79a83e67cf199efcd4046e1956e5f7474b9c939d8f8f2a · ms:29393
- 2026-09-25 · c43095d* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:46fe9123080575a15e79a83e67cf199efcd4046e1956e5f7474b9c939d8f8f2a · ms:0 · test-lock-sha256:7710236ac2e4834c41b02eabac22769c6f2b21e8f95d80af5c751f5638bd5be3 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkYXJnX3Rlc3QuZ28JVGVzdEFQYWRkZWRQb3NpdGlvbmFsV2l0aG91dERhc2hEYXNoSXNSZWZ1c2VkCTFhMWJiNjQxMzFlZGJhZjY4OGJmMzE2YmJkMTNhMTA1OWEwYjRiM2Y3ZjFhMjYwYjhiNzE4NGYxZDNjYTdiMzE · test-lock-kind:replace
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:46fe9123080575a15e79a83e67cf199efcd4046e1956e5f7474b9c939d8f8f2a · ms:30650

## Mutation Log
(empty until execute)
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `cmd/mrw/main.go` · the helper sees no raw token: read/write/check of a padded positional must go back to exit 0 and §128 red · acceptance-sha256:46fe9123080575a15e79a83e67cf199efcd4046e1956e5f7474b9c939d8f8f2a · covers:a padded positional is refused
- 2026-09-25 · c43095d* · mutant survived · exit 0 · `cmd/mrw/main.go` · the helper ignores --: read -- x-space is refused, the test and §128 go red · acceptance-sha256:46fe9123080575a15e79a83e67cf199efcd4046e1956e5f7474b9c939d8f8f2a · covers:dash-dash still serves the padded path
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `cmd/mrw/main.go` · the refusal stops naming --: §128 goes red · acceptance-sha256:46fe9123080575a15e79a83e67cf199efcd4046e1956e5f7474b9c939d8f8f2a · covers:the binary names --
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-069 does not own changes: the go/no-go guard must go red · acceptance-sha256:46fe9123080575a15e79a83e67cf199efcd4046e1956e5f7474b9c939d8f8f2a · covers:no engine file changes
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `cmd/mrw/main.go` · the helper reads past --: read x -- x-space is refused, the test goes red · acceptance-sha256:46fe9123080575a15e79a83e67cf199efcd4046e1956e5f7474b9c939d8f8f2a · covers:dash-dash still serves the padded path

## Invariants

- `mrw read -- 'x '` and every unpadded positional behave exactly as before.
- A padded flag value is refused only when its trim equals a positional (Risks).

## Risks

- A padded flag value equal to a positional's trim (`--exclude ' x' x`) is refused. Loud, exit 2, and rare.
- Windows cannot hold `x` and `x ` apart; the space fixtures skip there, visibly.

## Out of Scope

- Recovering the raw positional instead of refusing (permanent: boundary: decided with M 2026-09-25; it needs urfave's flag arity)

## Stop Condition

Stop if keeping the path exactly needs a format change or an exit-code change.
