# Task ADR-095-T1: A check counts a level, and at the limit none is started

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `check.DepthRefusal`, the check's `MRW_STEP_DEPTH`, contract §181
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the check runs one level deeper`, `a step still runs one level deeper`, `an unreadable depth counts as zero`, `a write whose check is due is refused at the limit`, `a write that starts no check lands at the limit`, `mrw check is refused at the limit`, `a recursion through the check stops at the limit`, `a red recursion stops itself`, `ADR-092 T5's guard still holds`, `the contract row exists`, `the tree is gofmt-clean`, `no other engine package changes`

## Goal

The check's shell gets `MRW_STEP_DEPTH` one higher than mrw's own, and at depth 8 or more a write
whose check is due and `mrw check` are refused, exit 2, before anything is written or run (ADR-095
Decisions 1–3).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | one unexported helper builds `MRW_STEP_DEPTH=StepDepth()+1`; SELECTS: `Run`'s call (`:242`) passes it, and `RunSteps` (`:792`) uses the same helper; `DepthRefusal(what string) error` builds the one refusal; `MaxStepDepth`'s comment says it bounds the check too |
| `internal/check/depth095_test.go` | add | the class test (check and step), the unreadable-value test |
| `cmd/mrw/main.go` | edit | SELECTS: `askedStepsError` (`:1603-1604`) returns `DepthRefusal`; the write refuses between `resolveSteps` (`:1256`) and `seen.Snapshot` (`:1259`); `checkDue` (`:1305-1311`) and that guard share one predicate; `checkCmd` refuses before `check.Run` (`:1935`) |
| `cmd/mrw/depth095_test.go` | add | the three CLI tests |
| `scripts/contract.sh` | edit | §181 drives the built binary |

## Ordered Steps

1. [S1] Write the five failing tests in the Tests table. Red today: the check sees its caller's
   depth (or none), and at depth 8 a `.go` write and `mrw check` run. Every test sets or clears
   `MRW_STEP_DEPTH` itself — under `mrw write` this repository's own check now runs at depth 1. The
   recursion test's check script appends a line per level and exits 97 past 12 lines, whatever
   `MRW_STEP_DEPTH` says, so the red run — and any guard mutant — unwinds by itself instead of
   recursing without end: nothing else bounds it, since `bounded` kills only the outer mrw and a
   nested check is in a group of its own (`scripts/contract.sh:135-139`). [proof: mutation]
2. [S2] `internal/check/check.go`: the helper; `Run` passes it to `run`; `RunSteps` builds its
   variable with the same helper. [proof: mutation]
3. [S3] `internal/check/check.go`: `DepthRefusal(what)` — `<what> refused <d> deep
   (MRW_STEP_DEPTH=<d>): a check or step that runs mrw again would recurse without end`;
   `askedStepsError` returns it for `--then and --then-sh are`. [proof: mutation]
4. [S4] `cmd/mrw/main.go` write: factor the due-rule into one predicate over "does this touch code";
   unless `--no-check` or `--dry-run` is set, compute it before the apply over the plan's paths — each `apply.Input.Path` after pointer
   resolution and each rename destination, so a `.md` → `.go` rename is due by its destination alone — and at `StepDepth() >= MaxStepDepth` refuse through
   `refuse()` with `DepthRefusal("a check is")` plus `; --no-check writes without it: nothing was
   written`. `checkDue` keeps computing it over `res.Files` with the same predicate. [proof: mutation]
5. [S5] `cmd/mrw/main.go` `checkCmd`: at the limit refuse through its `refuse` (one JSON document
   under `--json`) before the check runs. [proof: mutation]
6. [S6] Contract §181: (a) a fixture whose check is `sh "$R/rec181.sh"`, the script written with
   `$MRW`'s absolute path in it, as §174 writes `rec174.sh` (`$MRW` is not exported,
   `scripts/contract.sh:115`) — `echo "$MRW_STEP_DEPTH" >> depth181; [ "$(wc -l < depth181)" -gt 12 ]
   && exit 97; exec <mrw> check --full` — run as `bounded 90 … env -u MRW_STEP_DEPTH "$MRW" -C "$R"
   check --full`, must exit 3 with `depth181` holding 1 to 8, last line 8, in under 60 s, and nothing
   matching `$R/rec181.sh` left running (`pkill -9 -f` it in the failing branch, as §174 does); (b) at
   `MRW_STEP_DEPTH=8` a `.go` write in a tree whose check appends to a marker exits 2 with `a.go`
   unchanged and no marker, while `--no-check` lands it, exit 0; (c) `check --full` exits 2 at 8 and 0
   at 7 and under `MRW_STEP_DEPTH=x`. [proof: human: `./scripts/contract.sh` run unpiped before the commit, exit 0 with §181's rows printed]

## Acceptance

```bash
set -o pipefail
go test ./internal/check/ ./cmd/mrw/ -count=1 -timeout 600s -run 'TestTheCheckAndEveryStepRunOneLevelDeeper|TestAnUnreadableDepthCountsAsZero|TestAWriteWhoseCheckIsDueIsRefusedAtTheDepthLimit|TestACheckIsRefusedAtTheDepthLimit|TestACheckThatRunsMrwAgainStopsAtTheDepthLimit' -v 2>&1 | tee /tmp/adr095-T1.out \
  && missing=$(for t in TestTheCheckAndEveryStepRunOneLevelDeeper TestAnUnreadableDepthCountsAsZero TestAWriteWhoseCheckIsDueIsRefusedAtTheDepthLimit TestACheckIsRefusedAtTheDepthLimit TestACheckThatRunsMrwAgainStopsAtTheDepthLimit; do grep -qE "^--- PASS: $t \(" /tmp/adr095-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && go test ./internal/check/ ./cmd/mrw/ -count=1 -timeout 600s -run 'TestAStepRunsOneLevelDeeper|TestTheStepDepthGuardRefusesADeepSequence|TestAPlainWriteIgnoresAMalformedStepsBlock' -v 2>&1 | tee /tmp/adr095-T1-reg.out \
  && regmissing=$(for t in TestAStepRunsOneLevelDeeper TestTheStepDepthGuardRefusesADeepSequence TestAPlainWriteIgnoresAMalformedStepsBlock; do grep -qE "^--- PASS: $t \(" /tmp/adr095-T1-reg.out || echo "$t"; done) \
  && [ -z "$regmissing" ] \
  && grep -q '^# 181\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd internal)" ] \
  && [ -z "$(git diff --name-only "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheCheckAndEveryStepRunOneLevelDeeper` | `internal/check/depth095_test.go` | under `MRW_STEP_DEPTH=3`, `Run`'s check and a `RunSteps` step each write 4 — the whole class of project commands | — | S1, S2 |
| `TestAnUnreadableDepthCountsAsZero` | `internal/check/depth095_test.go` | `""`, `x`, `-3`, ` 8` and `99999999999999999999` each read as 0 and a check under each sees 1; `8` reads 8 | — | S1, S2 |
| `TestAWriteWhoseCheckIsDueIsRefusedAtTheDepthLimit` | `cmd/mrw/depth095_test.go` | at 8: a `.go` replace, a `.go` unlink, a `.go` → `.md` rename, a `.md` → `.go` rename (due by its destination alone), `--check` on a `.md` plan, and a `.go` replace in a tree with a `go.mod` and no harness (the inferred check) each exit 2 with the tree unchanged and no check run, naming `MRW_STEP_DEPTH=8` and `--no-check`; under `--json` the shape `jsonRefusal` checks (`applied` false, `files` and `hunks` empty, the `error`) and no `then`; `mrw stats --json` then counts one `refused_apply` and no `applied`; `--no-check`, `--dry-run`, a `.md`-only plan and a `.go` replace in a tree with neither a harness nor a `go.mod` exit 0 with no check run; at 7 the `.go` write lands and its check writes 8 | — | S1, S4 |
| `TestACheckIsRefusedAtTheDepthLimit` | `cmd/mrw/depth095_test.go` | at 8 `check --full`, `check a.go` and the working-set form exit 2 naming `MRW_STEP_DEPTH`, no check marker; `check --full --json` prints one document whose only key is `error`; `check --then a` at 8 still exits 2 naming it; at 7 and under `MRW_STEP_DEPTH=x` `check --full` exits 0 | — | S1, S3, S5 |
| `TestACheckThatRunsMrwAgainStopsAtTheDepthLimit` | `cmd/mrw/depth095_test.go` | a built mrw (`go build -o <tmp> .`, as `receipt_before_check_test.go` does) whose check script runs it again with `check --full` and stops itself past 12 levels (S1), started with `MRW_STEP_DEPTH` cleared, exits 3 and records levels 1 to 8 and no ninth; skipped on Windows, where §181 cannot run either | — | S1, S2, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the helper and `DepthRefusal`; `TestTheCheckAndEveryStepRunOneLevelDeeper` |
| 2 — something selects it | `internal/check/check.go:242`'s call passes the helper's variable (delete it and the class test goes red); the write's guard and `checkCmd`'s guard in `cmd/mrw/main.go` (delete either and its CLI test goes red); §181 drives the built binary into a real recursion |
| 3 — the caller can discover it | the refusal names `MRW_STEP_DEPTH` and `--no-check`; T3 teaches it |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-29 survey probe and this record's probe |

## Mutation Log
(empty until execute)
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/check/check.go` · the check inherits its caller depth unchanged · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · covers:the check runs one level deeper
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/check/check.go` · a step inherits its caller depth unchanged · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · covers:a step still runs one level deeper
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/check/check.go` · a negative depth is read as written · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · covers:an unreadable depth counts as zero
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `cmd/mrw/main.go` · a write whose check is due lands at the limit · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · covers:a write whose check is due is refused at the limit
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `cmd/mrw/main.go` · every write is refused at the limit, --no-check and prose included · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · covers:a write that starts no check lands at the limit
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `cmd/mrw/main.go` · mrw check runs its check at the limit · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · covers:mrw check is refused at the limit
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/check/check.go` · the limit is one level too deep · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · covers:a recursion through the check stops at the limit
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/check/check.go` · no depth is ever refused, so only the script stops the recursion · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · covers:a red recursion stops itself
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `cmd/mrw/main.go` · steps are started at the limit · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · covers:ADR-092 T5's guard still holds
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/check/check.go` · check.go is not gofmt-clean · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · covers:the tree is gofmt-clean
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · covers:no other engine package changes
- 2026-09-29 · 5794966* · mutant killed · exit 1 · `scripts/contract.sh` · the section header 181 is gone (the line becomes a no-op command) · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · covers:the contract row exists

## Invariants

- Below depth 8 nothing changes but the value of `MRW_STEP_DEPTH` a check sees.
- At every depth, a write with `--no-check` or `--dry-run`, or one that starts no check, behaves as on v1.31.0.
- A refusal at the limit writes nothing and starts nothing.
- One counter, one limit (`MaxStepDepth`), one refusal function.

## Risks

- The pre-apply predicate is wider or narrower than `checkDue`. Mitigation: one predicate; the write
  test exercises replace, unlink, rename, `--check` and prose at the limit.
- A test that counts levels inherits depth 1 when this repository's check runs under `mrw write`.
  Mitigation: every test here sets or clears the variable itself.
- A red run of the recursion test or of §181, or a guard mutant, has no guard to stop it. Mitigation:
  the check script stops itself past 12 levels (S1), and §181 kills anything left matching its
  script's path, as §174 does.

## Stop Condition

Stop and bring it back if a plan that applies whole can touch a file set the pre-apply predicate cannot
see (so the refusal would promise a check that would not run, or miss one that would), or if an engine
package other than `internal/check` has to change.

## Out of Scope

- TERM before KILL (that's T2's job)
- Teaching and the record amendments (that's T3's job)

## Verification Log
(empty until execute)
- 2026-09-29 · 5794966* · exit 1 · `set -o pipefail …` · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · ms:1177 · test-lock-sha256:12c4a49603af70d1072a4737a47a0bb13b3dd3a7546fceec657b38fdd55ca21f · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvZGVwdGgwOTVfdGVzdC5nbwktLWpzb24gYW5kIHRoZSB0YWxseQkzOTFkZTUzOGQ3OWE2YmI5Y2RlYThiM2ViOWUxZWFiOTlkOGZiNjlhMDIwMzljZGFjY2UyOWQ3NDg5NmI3YjczCmJvZHkJY21kL21ydy9kZXB0aDA5NV90ZXN0LmdvCVRlc3RBQ2hlY2tJc1JlZnVzZWRBdFRoZURlcHRoTGltaXQJNGU1YWRjMjdkNmNiNDRlOTM0NjdkZDlhNTM3MDNiZGIzZmEwOTUwMGFjMDFmZGFmNzcxZWIxNDg1M2Y3ZTBjMgpib2R5CWNtZC9tcncvZGVwdGgwOTVfdGVzdC5nbwlUZXN0QUNoZWNrVGhhdFJ1bnNNcndBZ2FpblN0b3BzQXRUaGVEZXB0aExpbWl0CTIyMGNjYmE4MDRiN2RjZjFiMzllYWY3ZDFkYzIzYjg1OWE0NzFjN2U0YjIyMDI4NzFlZWNlZjhlOTAzMmVhMzgKYm9keQljbWQvbXJ3L2RlcHRoMDk1X3Rlc3QuZ28JVGVzdEFXcml0ZVdob3NlQ2hlY2tJc0R1ZUlzUmVmdXNlZEF0VGhlRGVwdGhMaW1pdAk1MjQwYTllNjZmODc0OTg2ZmE1YjBhNTc0NjJhN2FhNGRhMDRmNTk5NjliNzg5NDA5ZTBjZDUxN2IzYjZhOTMxCmJvZHkJY21kL21ydy9kZXB0aDA5NV90ZXN0LmdvCWEgd3JpdGUgdGhhdCBzdGFydHMgbm8gY2hlY2sgbGFuZHMJNDA4NDFhMDg0MjZiZTUzY2U1MjIwMGU4OWE2ZjFhNGIzYzVmZWYyZTQ3MzUxNTNiMWVjNmJmZmJlYzg2NDg0NQpib2R5CWNtZC9tcncvZGVwdGgwOTVfdGVzdC5nbwlhdCA3IHRoZSB3cml0ZSBsYW5kcyBhbmQgaXRzIGNoZWNrIHJ1bnMgYXQgOAllMTAyZjE1MDI1YTcwNzNhMTRjNmYwNzgzNzlmOTdlZWVlNjRlOWZiYmQ4YTQ2YjQ0MzRmYzcyMTBiYzFiN2VjCmJvZHkJY21kL21ydy9kZXB0aDA5NV90ZXN0LmdvCXRoZSBpbmZlcnJlZCBjaGVjawkzNWYwMDlmN2NlODZlNGY5MGNhNTkwMzlkOTJhZTg1N2IyNDlmMWNlZmY5MDQ4NDcyYTU3M2JiNWU4YmUxZjk5CmJvZHkJaW50ZXJuYWwvY2hlY2svZGVwdGgwOTVfdGVzdC5nbwlUZXN0QW5VbnJlYWRhYmxlRGVwdGhDb3VudHNBc1plcm8JNGE3ZDYwOWZhMDJmNzk2ZWIwODA3NDM5MjA5NGY4M2YwZDM0OGZlMzNkZWRiNmJlNTA5YWU5NDlhYWMyYWUyNQpib2R5CWludGVybmFsL2NoZWNrL2RlcHRoMDk1X3Rlc3QuZ28JVGVzdFRoZUNoZWNrQW5kRXZlcnlTdGVwUnVuT25lTGV2ZWxEZWVwZXIJY2QyMjhlNmM4N2ZlMmE5NzVkYzY2NzlhNGYzMzEwMWQyMDljMDlhYzg2MWU2ZWNjYzA5NGY1OTZhNTg2YjU5Nw
  ```
  --- last 10 line(s) of stdout (of 194 after folding 194 raw)
            | check last: mrw: check did not pass
            | check FAIL (exit 3, 197ms) — full output: /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/mrw-check-3905400324.log
            | mrw: check did not pass
          check last: mrw: check did not pass
          check FAIL (exit 3, 214ms) — full output: /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/mrw-check-480499550.log
          mrw: check did not pass
  --- FAIL: TestACheckThatRunsMrwAgainStopsAtTheDepthLimit (0.56s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.869s
  FAIL
  ```
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · ms:1301
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · ms:1136
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · ms:1152
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · ms:1167
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · ms:1236
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · ms:1236
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · ms:1331
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · ms:1477
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · ms:1214
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · ms:1212
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · ms:1187
- 2026-09-29 · 5794966* · exit 0 · `set -o pipefail …` · acceptance-sha256:661244bade99c8785f9a9afad14ce3a976ace10b217ded6f154522b4b49ac1eb · ms:1208
- 2026-09-29 · human-observed · S6 observed: PR #279 records 'contract.sh holds' (rebased on 53e0e11); contract.sh exit 0 with §181 on main db39d42 on 2026-09-29 (this session)
