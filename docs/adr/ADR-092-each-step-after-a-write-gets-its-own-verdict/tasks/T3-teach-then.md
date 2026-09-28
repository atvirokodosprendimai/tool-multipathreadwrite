# Task ADR-092-T3: Every surface teaches the steps

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `TestEverySurfaceTeachesThen`
**Consumes:** `--then`, `--then-sh`, the `then` receipt (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `mrw instructions teaches both flags`, `AGENTS.md teaches both flags`, `README teaches both flags`, `write --help teaches both flags`, `the allow-rule caveat is named`, `the tree is gofmt-clean`, `no other engine package changes`

## Goal

A caller who reads `mrw instructions`, AGENTS.md or `write --help` learns both flags, that a step is
POSIX shell on every platform, and what `--then-sh` grants.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | the CLI text names `--then`, `--then-sh`, `steps` and stop-at-first-failure |
| `cmd/mrw/teach092_test.go` | add | `TestEverySurfaceTeachesThen` — `cmd/mrw` can run `write --help` and read AGENTS.md |
| `AGENTS.md` | edit | "Using mrw" §4 bullet and §5 `mrw check` bullet |
| `README.md` | edit | the flags, `steps`, the receipt and the allow-rule caveat |
| `cmd/mrw/main.go` | edit | the two flags' usage names the caveat |
| `scripts/contract.sh` | edit | §115 lets the instructions name `--then` and `--then-sh`, the two non-read flags they now teach |

## Ordered Steps

1. [S1] Write the failing test `TestEverySurfaceTeachesThen`: `guide.CLI()`, AGENTS.md, README.md and
   `write --help` each contain `--then`, `--then-sh` and the caveat sentence. [proof: mutation]
2. [S2] The text in each surface, one shared caveat sentence. [proof: mutation]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestEverySurfaceTeachesThen' -v 2>&1 | tee /tmp/adr092-T3.out \
  && grep -qE '^--- PASS: TestEverySurfaceTeachesThen \(' /tmp/adr092-T3.out \
  && [ -z "$(gofmt -l .)" ] \
  && [ -z "$(git diff --name-only 8ecb059 -- internal/read internal/apply internal/plan internal/seen internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEverySurfaceTeachesThen` | `cmd/mrw/teach092_test.go` | `mrw instructions`, AGENTS.md, README.md and `write --help` name both flags and the allow-rule caveat | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the text |
| 2 — something selects it | `mrw instructions` prints `guide.CLI()`; the test reads the printed help |
| 3 — the caller can discover it | the three surfaces themselves |
| 4 — it is used | nothing measures this yet |

## Mutation Log
(empty until execute)
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/guide/guide.go` · the instructions stop naming --then · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · covers:mrw instructions teaches both flags
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `AGENTS.md` · AGENTS.md drops the caveat · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · covers:AGENTS.md teaches both flags
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `README.md` · README drops the caveat · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · covers:README teaches both flags
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `cmd/mrw/main.go` · write --help drops the caveat · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · covers:write --help teaches both flags
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/guide/guide.go` · the caveat stops saying what it grants · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · covers:the allow-rule caveat is named
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/guide/guide.go` · guide.go is not gofmt-clean · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · covers:the tree is gofmt-clean
- 2026-09-28 · 8ecb059* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · covers:no other engine package changes

## Invariants

- ADR-037's shared sentences stay on every surface (`TestEverySurfaceContainsTheSharedSentences`).

## Risks

- None beyond the record's.

## Out of Scope

- The centralised `mrw` skill: updated at the release that ships this record, the parent's Follow-up

## Stop Condition

The fence exits 0.

## Verification Log
(empty until execute)
- 2026-09-28 · 8ecb059* · exit 1 · `set -o pipefail …` · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · ms:852 · test-lock-sha256:efa2de2a516c965a8dd48f3284b4967533a3c7e0e500e73c5382cb7c1fb10ba8 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdGVhY2gwOTJfdGVzdC5nbwlUZXN0RXZlcnlTdXJmYWNlVGVhY2hlc1RoZW4JOGQ5OTRiYzRlODAzYzRmYmZhYmVkYWU0MWMyODE0N2QyYTJiMjQzNGJjYzA3ZWZjZDE2ZGVhZmI0ZTk3ODMwYw
  ```
  --- last 6 line(s) of stdout
  === RUN   TestEverySurfaceTeachesThen
      teach092_test.go:27: the caveat does not say what --then-sh grants: ""
  --- FAIL: TestEverySurfaceTeachesThen (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.191s
  FAIL
  ```
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · ms:323
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · ms:322
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · ms:336
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · ms:333
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · ms:340
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · ms:337
- 2026-09-28 · 8ecb059* · exit 0 · `set -o pipefail …` · acceptance-sha256:f0135235e4208d0ecfccf2924207fdf6f2768a294811a40da32526c509197231 · ms:320
