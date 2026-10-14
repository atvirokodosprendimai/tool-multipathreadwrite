# Task ADR-151-T1: a plan flag on write and an all-digit short root are named

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `shortRootNote` in `cmd/mrw/shortroot.go`; the `write` table in `cmd/mrw/foreignflag.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `write given a plan-naming flag exits 2 with the argument's spelling appended`, `an all-digit short root before the subcommand is named on stderr with nothing else changed`

## Goal

Decisions 1–3 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/foreignflag.go` | edit | the `write` table and the per-command lookup |
| `cmd/mrw/shortroot.go` | add | `shortRootNote` |
| `cmd/mrw/main.go` | edit | `main` prints the note before the parser runs |
| `cmd/mrw/flag151_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §255 |
| `AGENTS.md`, `internal/guide/guide.go` | edit | one sentence each (README carries no ADR-140 sentence to sit beside) |
| `docs/adr/BACKLOG.md` | edit | the flag-friction entry: taken by ADR-151 |

## Ordered Steps

1. [S1] Write `TestAPlanFlagOnWriteIsAnsweredWithTheArgument` (each of `--plan`, `--file`, `--input`, `--stdin` on `write` is exit 2 with the old prefix and the sentence last; `write -f`, `write -p` and `read --plan` carry no such sentence) and `TestAnAllDigitShortRootBeforeTheVerbIsNamed` (the built binary, in a working directory holding a directory `3`: `-C 3`, `-C=3`, `--C 3`, `--C=3` before `read` print the note and serve exactly what `--root 3` serves with the same exit; `--root 3`, `-C sub`, and `read -C 3` after the verb print none; with no directory `3` the refusal and exit 1 are unchanged and the note is there). Confirm RED.
2. [S2] The `write` table and the per-command lookup in `foreignFlagHint`. Mutants: the write table empty; the write sentences given on `read` too. [proof: mutation]
3. [S3] `shortRootNote` and its call in `main`. Mutants: the call removed; any `-C` value noted, not only digits; `--root` noted as well. [proof: mutation]
4. [S4] Contract §255, AGENTS.md, the guide sentence and the BACKLOG entry. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAPlanFlagOnWriteIsAnsweredWithTheArgument|TestAnAllDigitShortRootBeforeTheVerbIsNamed' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAPlanFlagOnWriteIsAnsweredWithTheArgument \(' "$out" \
  && grep -qE '^--- PASS: TestAnAllDigitShortRootBeforeTheVerbIsNamed \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 900s -run 'Usage|Subcommand|Flag|Padded' \
  && grep -q '^# 255\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPlanFlagOnWriteIsAnsweredWithTheArgument` | `cmd/mrw/flag151_test.go` | each plan-naming flag on `write` is exit 2 with the old prefix and the argument's spelling; others untouched | none | S1, S2 |
| `TestAnAllDigitShortRootBeforeTheVerbIsNamed` | `cmd/mrw/flag151_test.go` | the built binary notes every accepted spelling of an all-digit short root, serves and exits as `--root` does, and notes nothing else | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `foreignFlagHint`'s `write` table; `shortRootNote` |
| 2 — something selects it | `usageError` on every parser rejection; `main` before every parse |
| 3 — the caller can discover it | the error and the stderr line themselves; AGENTS.md and `mrw instructions` |
| 4 — it is used | the survey (2 and 4 sessions) and the v1.65.0 probe; no telemetry (ADR-009) |

## Invariants

- Exit codes, stdout and the root mrw uses are unchanged.
- `read`'s table (ADR-140) is unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the argv walk cannot agree with the parser on a spelling the parser accepts: a note that fires where the parser did something else is a false statement.

## Out of Scope

- `-f` and `-p` on `write` (permanent: boundary: each reads as another tool's flag)

## Mutation Log
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/foreignflag.go` · S2: the write table is empty · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/foreignflag.go` · S2: the write sentences are given on read too · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: main never passes argv to the note · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/shortroot.go` · S3: any -C value is noted, not only digits · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/shortroot.go` · S3: the long --root is noted as well · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/foreignflag.go` · S2: the write table is empty · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · covers:write given a plan-naming flag exits 2 with the argument's spelling appended
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: main never passes argv to the note · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · covers:an all-digit short root before the subcommand is named on stderr with nothing else changed

## Verification Log
- 2026-10-14 · 6b565cb* · exit 1 · `set -o pipefail …` · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · ms:2108 · test-lock-sha256:2dda7880127aa8939768bf11fce2ce06e81c6bfefea1d7961de3cab1ce19c950 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9mbGFnMTUxX3Rlc3QuZ28JVGVzdEFQbGFuRmxhZ09uV3JpdGVJc0Fuc3dlcmVkV2l0aFRoZUFyZ3VtZW50CWUwZDcxNTk4OTY1MjIyODkyNjYxMWFjMjRmOTI0M2NmYzhlM2Q4NmJkOGU0MmE4ZTI0MzAzODg2YTE3OGUzNzQKYm9keQljbWQvbXJ3L2ZsYWcxNTFfdGVzdC5nbwlUZXN0QW5BbGxEaWdpdFNob3J0Um9vdEJlZm9yZVRoZVZlcmJJc05hbWVkCWY4YzBkY2M1NTk4NTA5N2IyZjhmY2Y1NTU1ZjJhMDhkODkwNzU5ODc2ZWFkZjI2M2ZmOTRmMDNlMTNjYWMyMDI
  ```
  --- last 10 line(s) of stdout (of 22 after folding 22 raw)
      flag151_test.go:76: mrw -C=3 read f.txt: exit 0, stdout same as --root 3: true, want the note on stderr:
      flag151_test.go:76: mrw --C 3 read f.txt: exit 0, stdout same as --root 3: true, want the note on stderr:
      flag151_test.go:76: mrw --C=3 read f.txt: exit 0, stdout same as --root 3: true, want the note on stderr:
      flag151_test.go:96: -C 3 with no directory 3: exit 1, want 1, the refusal and the note:
          ==> f.txt  REFUSED  the root 3 does not exist: read it with --root pointed where you mean
          mrw: incomplete: 1 range(s) could not be served
  --- FAIL: TestAnAllDigitShortRootBeforeTheVerbIsNamed (0.86s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	1.067s
  FAIL
  ```
- 2026-10-14 · 6b565cb* · exit 1 · `set -o pipefail …` · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · ms:12421
  ```
  --- last 7 line(s) of stdout
  === RUN   TestAPlanFlagOnWriteIsAnsweredWithTheArgument
  --- PASS: TestAPlanFlagOnWriteIsAnsweredWithTheArgument (0.00s)
  === RUN   TestAnAllDigitShortRootBeforeTheVerbIsNamed
  --- PASS: TestAnAllDigitShortRootBeforeTheVerbIsNamed (2.11s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	2.337s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	3.340s
  ```
- 2026-10-14 · 6b565cb* · exit 1 · `set -o pipefail …` · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · ms:7651
  ```
  --- last 7 line(s) of stdout
  === RUN   TestAPlanFlagOnWriteIsAnsweredWithTheArgument
  --- PASS: TestAPlanFlagOnWriteIsAnsweredWithTheArgument (0.01s)
  === RUN   TestAnAllDigitShortRootBeforeTheVerbIsNamed
  --- PASS: TestAnAllDigitShortRootBeforeTheVerbIsNamed (1.95s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	2.294s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	1.419s
  ```
- 2026-10-14 · 6b565cb* · exit 1 · `set -o pipefail …` · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · ms:11327
  ```
  --- last 7 line(s) of stdout
  === RUN   TestAPlanFlagOnWriteIsAnsweredWithTheArgument
  --- PASS: TestAPlanFlagOnWriteIsAnsweredWithTheArgument (0.00s)
  === RUN   TestAnAllDigitShortRootBeforeTheVerbIsNamed
  --- PASS: TestAnAllDigitShortRootBeforeTheVerbIsNamed (2.27s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	2.509s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	1.634s
  ```
- 2026-10-14 · 6b565cb* · exit 1 · `set -o pipefail …` · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · ms:8798
  ```
  --- last 7 line(s) of stdout
  === RUN   TestAPlanFlagOnWriteIsAnsweredWithTheArgument
  --- PASS: TestAPlanFlagOnWriteIsAnsweredWithTheArgument (0.03s)
  === RUN   TestAnAllDigitShortRootBeforeTheVerbIsNamed
  --- PASS: TestAnAllDigitShortRootBeforeTheVerbIsNamed (1.64s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	1.896s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.909s
  ```
- 2026-10-14 · 6b565cb* · exit 1 · `set -o pipefail …` · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · ms:4144
  ```
  --- last 7 line(s) of stdout
  === RUN   TestAPlanFlagOnWriteIsAnsweredWithTheArgument
  --- PASS: TestAPlanFlagOnWriteIsAnsweredWithTheArgument (0.00s)
  === RUN   TestAnAllDigitShortRootBeforeTheVerbIsNamed
  --- PASS: TestAnAllDigitShortRootBeforeTheVerbIsNamed (1.06s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	1.262s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.278s
  ```
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · ms:3051
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · ms:3158
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · ms:3399
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · ms:2897
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · ms:2742
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · ms:2468
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5ed6c91c1942b2f3ac0c49d335844e87bf3c2b8592b484de4b7e43fb2951e4e · ms:2373
