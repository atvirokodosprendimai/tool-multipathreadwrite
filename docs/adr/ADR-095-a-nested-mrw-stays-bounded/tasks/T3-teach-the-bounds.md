# Task ADR-095-T3: Every surface teaches the bounds and what escapes them

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** none
**Consumes:** `check.DepthRefusal` (T1), `stopGroup` (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the instructions teach that a check counts a level`, `every surface names what clears the count`, `every surface says a group hears TERM first`, `no surface says ast-grep is killed at 2 s`, `ADR-092 T3's teaching still holds`, `ADR-092 T5's guide line still holds`, `each narrowed record names ADR-095`, `ADR-092's outcome table has the depth row`, `the tree is gofmt-clean`, `no other engine package changes`

## Goal

`mrw instructions`, AGENTS.md and README.md teach ADR-095 Decisions 1–6, and each accepted record
whose Decision it narrows carries one "Amended by ADR-095" line.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | line 44: a check and a step run one level deeper; at depth 8 `--then`, `--then-sh`, a write whose check is due and `mrw check` are refused, `--no-check` writes without it; `env -i` and `sudo` clear the count, as `setsid` leaves the group; on unix a stopped group hears TERM, and what ignores it is killed a second later — an mrw killed that way can leave its own check running. Line 64: a hanging ast-grep is sent TERM at 2 s, and one that ignores it is killed by 3 s |
| `AGENTS.md` | edit | the `--then` paragraph ("Using mrw", section 4) and the `--ast-grep` sentence ("Read many ranges") say the same |
| `README.md` | edit | the steps bullet and the `--ast-grep` table row say the same |
| `cmd/mrw/teach095_test.go` | add | the surface test |
| `docs/adr/ADR-092-each-step-after-a-write-gets-its-own-verdict.md` | edit | "Amended by ADR-095" on the T5 amendment; the outcome table gains a row that begins "at `MRW_STEP_DEPTH` ≥ 8, a check due or `mrw check`" — steps none; `write --json` the ADR-072 single refusal document, `check --json` one document holding only `error`, no `then`; exit 2 |
| `docs/adr/ADR-080-children-and-logs.md` | edit | "Amended by ADR-095" on Decision 1 |
| `docs/adr/ADR-072-the-exit-code-and-the-receipt-agree-with-the-tree.md` | edit | "Amended by ADR-095" on Decision 4 |
| `docs/adr/ADR-058-structural-find-shells-out-to-ast-grep.md` | edit | "Amended by ADR-095" on Decision 5 |
| `docs/adr/ADR-074-a-read-is-served-or-reported-never-hung.md` | edit | "Amended by ADR-095" on Decision 2 |

The class of stale wording, enumerated 2026-09-29 at 8cbb89e: `mrw read --grep 'killed at 2 s|kills? the
group on cancel|killed at the 2 s|a hang is killed' docs/adr AGENTS.md README.md internal/guide` → 11
lines in 10 files. The three live surfaces are rewritten; the Decisions in ADR-058, ADR-072, ADR-074
and ADR-080 get an amendment line; ADR-058's task T3, ADR-063's quotation of the instructions, and
BACKLOG's history stay as written — they record what was true when they were written.

## Ordered Steps

1. [S1] Write `TestEverySurfaceTeachesTheCheckDepthAndItsLimits`; red today. [proof: mutation]
2. [S2] `internal/guide/guide.go`. [proof: mutation]
3. [S3] AGENTS.md and README.md. [proof: mutation]
4. [S4] The five amendment lines, and ADR-092's outcome-table row. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestEverySurfaceTeachesTheCheckDepthAndItsLimits' -v 2>&1 | tee /tmp/adr095-T3.out \
  && grep -qE '^--- PASS: TestEverySurfaceTeachesTheCheckDepthAndItsLimits \(' /tmp/adr095-T3.out \
  && go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestEverySurfaceTeachesThen|TestTheStepDepthGuardRefusesADeepSequence' -v 2>&1 | tee /tmp/adr095-T3-reg.out \
  && [ -z "$(for t in TestEverySurfaceTeachesThen TestTheStepDepthGuardRefusesADeepSequence; do grep -qE "^--- PASS: $t \(" /tmp/adr095-T3-reg.out || echo "$t"; done)" ] \
  && [ -z "$(for f in docs/adr/ADR-058-structural-find-shells-out-to-ast-grep.md docs/adr/ADR-072-the-exit-code-and-the-receipt-agree-with-the-tree.md docs/adr/ADR-074-a-read-is-served-or-reported-never-hung.md docs/adr/ADR-080-children-and-logs.md docs/adr/ADR-092-each-step-after-a-write-gets-its-own-verdict.md; do grep -q 'Amended by ADR-095' "$f" || echo "$f"; done)" ] \
  && grep -qE '^ *[|] at `MRW_STEP_DEPTH` ≥ 8, a check due' docs/adr/ADR-092-each-step-after-a-write-gets-its-own-verdict.md \
  && [ -z "$(gofmt -l cmd internal)" ] \
  && [ -z "$(git diff --name-only "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEverySurfaceTeachesTheCheckDepthAndItsLimits` | `cmd/mrw/teach095_test.go` | `mrw instructions`, AGENTS.md and README.md each say a check runs one level deeper and is refused at depth 8 (the number read from `check.MaxStepDepth`), name `env -i` and `sudo` beside `setsid`, say that on unix a stopped group hears TERM first and that what ignores it is killed a second later, leaving the check of an mrw killed that way running, and say a hanging ast-grep is sent TERM at 2 s and killed by 3 s; none says ast-grep is "killed at 2 s" | — | S1, S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the text in `internal/guide/guide.go`, AGENTS.md, README.md |
| 2 — something selects it | `mrw instructions` prints `guide.CLI()`; the test reads what it prints |
| 3 — the caller can discover it | this task is that rung for T1 and T2 |
| 4 — it is used | nothing measures this yet; the centralised `mrw` skill mirrors AGENTS.md at the release (ADR-095 Follow-ups) |

## Mutation Log
(empty until execute)
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/guide/guide.go` · mrw instructions no longer says the check counts a level as a step does · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · covers:the instructions teach that a check counts a level
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `AGENTS.md` · AGENTS.md no longer names env -i and sudo · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · covers:every surface names what clears the count
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `README.md` · README no longer says a group hears SIGTERM first · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · covers:every surface says a group hears TERM first
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `README.md` · the README table row says ast-grep is killed at 2 s again · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · covers:no surface says ast-grep is killed at 2 s
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/guide/guide.go` · the --then-sh caveat no longer says what it grants · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · covers:ADR-092 T3's teaching still holds
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `cmd/mrw/main.go` · the depth guard T5 teaches no longer refuses a step at 8 · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · covers:ADR-092 T5's guide line still holds
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `docs/adr/ADR-080-children-and-logs.md` · ADR-080 no longer names ADR-095 on the Decision it narrows · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · covers:each narrowed record names ADR-095
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `docs/adr/ADR-092-each-step-after-a-write-gets-its-own-verdict.md` · the outcome table row is not the depth row the record names · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · covers:ADR-092's outcome table has the depth row
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/guide/guide.go` · guide.go is not gofmt-clean · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · covers:the tree is gofmt-clean
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · covers:no other engine package changes

## Invariants

- ADR-092 T3's surfaces still name both step flags and carry the `--then-sh` caveat (`TestEverySurfaceTeachesThen`).
- Frozen task files and BACKLOG history are not rewritten.

## Risks

- A sentence restating a limit goes stale when the limit moves. Mitigation: the surfaces say "depth
  8" and "a second" once each, and the test reads the constant's value from `check.MaxStepDepth`.

## Stop Condition

Stop if a surface would have to promise something T1 or T2 did not ship.

## Out of Scope

- The centralised `mrw` skill (deferred: ADR-095 Follow-ups — at the release that ships this record)

## Verification Log
(empty until execute)
- 2026-09-29 · 5794966* · exit 1 · `set -o pipefail …` · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · ms:912 · test-lock-sha256:fec000b226454ec5d36204cf4e64e7b55d0f6706309e3450bd18e836f040f13d · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdGVhY2gwOTVfdGVzdC5nbwlUZXN0RXZlcnlTdXJmYWNlVGVhY2hlc1RoZUNoZWNrRGVwdGhBbmRJdHNMaW1pdHMJNzY0ZTVmNjAyZDcwZTc0MTRjYmNiZDUzOTQwMTY0NThjNzIwOGQwYWU0ZmRmNmEyODFhZTY4MDRmNTA1MGRlNw
  ```
  --- last 10 line(s) of stdout (of 41 after folding 41 raw)
      teach095_test.go:51: README.md does not teach "hears SIGTERM first"
      teach095_test.go:51: README.md does not teach "killed a second later"
      teach095_test.go:51: README.md does not teach "can leave its own check running"
      teach095_test.go:51: README.md does not teach "sent SIGTERM at 2 s"
      teach095_test.go:51: README.md does not teach "killed by 3 s"
      teach095_test.go:55: README.md still says a hanging ast-grep is killed at 2 s
  --- FAIL: TestEverySurfaceTeachesTheCheckDepthAndItsLimits (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.340s
  FAIL
  ```
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · ms:968
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · ms:634
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · ms:675
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · ms:683
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · ms:683
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · ms:689
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · ms:650
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · ms:643
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · ms:687
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:2681ec643d9eb7788f34482f01252799a3df92bb6bf8235bc361e09348d481dd · ms:713
