# Task ADR-161-T1: read --sub prints a plan; write --grep --sub applies it in one command

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `mrw read --grep P --sub R`, `mrw write --grep P --sub R`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a sub read prints a plan and licenses only its lines`, `a sub write applies that plan as one plan`, `the refusals`

## Goal

A same-text change at many sites is one mrw command, with no line in the agent's context and no script.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/sub.go` | add | the plan from a `--grep` walk and a replacement: one hunk per changed line, the lines licensed |
| `internal/read/sub161_test.go` | add | the plan, the licence, the refusals, CRLF, the round trip through `plan.Parse` |
| `internal/seen/seen.go` | edit | a licence of partial spans that is not shown: `Written` with `Spans`, the ledger line `w<shown>\|<licensed>` (ledger format v6, so an mrw reading v5 leaves it alone), merged so it never widens what is known, and named in a refusal as what a `--sub` plan licensed |
| `internal/guide/guide.go` | edit | one sentence naming `--sub`, as every read flag must be (the recipes block is T3's) |
| `cmd/mrw/main.go` | edit | `--sub` on `read` and `write`, `--grep` on `write`, the usage refusals (what selects them) |
| `cmd/mrw/sub161_test.go` | add | `write --grep --sub` end to end: one receipt, all or nothing, the check |
| `docs/receipts.txt` | edit | any key the receipt gains |
| `scripts/contract.sh` | edit | §263: read --sub into write -, and write --sub, against the built binary |

## Ordered Steps

1. [S1] Write the failing tests and the contract row and see them red. [proof: acceptance]
2. [S2] The plan builder: for each line the `--grep` walk serves, `regexp.ReplaceAllString` on its text; an unchanged line is not planned; every hunk `@@ PATH N replace sha=<the file's sha256> body=1 raw=true`, PATH quoted and escaped as `splitHeader` reads it; the planned lines recorded as licensed and not shown (`Written`, empty `Shown`, ADR-118), so `occurrence=N` counts none; stderr `-- planned: N site(s) in M file(s)`; exit 1 `no match:` or `incomplete:` when nothing is planned; a minimal sentence naming `--sub` in `guide.CLI()`, which `TestEveryReadFlagIsTaughtByInstructions` and contract §115 require of every read flag. [proof: mutation]
3. [S3] The refusals of Decision 1: `--sub` without `--grep` and beside the flags it excludes, a ranged path, a REPLACEMENT holding CR or LF, a new text ending in a CR. A path starting with `@` is spelled `./@…`, never a working-set pointer. A round trip — plan text through `plan.Parse` gives back each path and body, for a path with a space and with both quote kinds, a body starting `@@`, `body=` or a byte order mark, and an empty body. [proof: mutation]
4. [S4] `write --grep P --sub R [paths]`: the same plan handed to the existing apply path, so verdicts, all or nothing, the check, `--then`, `--dry-run` and `--json` are the write's, a refusal under `--json` one document that keeps its exit; `--exclude` and `--no-ignore` shape the walk as `read --grep`'s do; a CRLF file keeps its endings; a refused or unreadable matching file applies nothing and exits 1 naming it; a skipped binary file is counted on stderr and left out. [proof: mutation]
5. [S5] Contract §263 drives the built binary: a scratch tree of files holding `Old`, `OldX` and `xOld`; `read --grep '\bOld\b' --sub New | mrw write -` and `write --grep '\bOld\b' --sub New` each change every `Old` and neither neighbour, exit 0; a sub that changes nothing exits 1 and writes nothing. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/read/ ./internal/guide/ ./cmd/mrw/ -count=1 -v 2>&1) \
  && for n in TestASubReadPrintsAPlanAndLicensesItsLines TestASubWriteAppliesThePlanInOneCommand; do grep -qE "^--- PASS: $n " <<<"$out" || exit 1; done \
  && test -z "$(gofmt -l .)" && grep -q '^# 263\. ' scripts/contract.sh \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  a sub read piped to write, and a sub write, change every site and no neighbour' <<<"$c"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestASubReadPrintsAPlanAndLicensesItsLines` | `internal/read/sub161_test.go` | the plan text, the licence of exactly the planned lines, unchanged lines left out, `raw=true` for `@@`, CRLF text, the stderr count, the exits and the refusals | none | S2, S3 |
| `TestASubWriteAppliesThePlanInOneCommand` | `cmd/mrw/sub161_test.go` | one receipt with a verdict per site, nothing written when a site fails, the check run, `--dry-run` writes nothing, CRLF kept | none | S4 |
| `TestASubReadRefusalsAndExits` | `cmd/mrw/sub161_test.go` | through the binary: the plan is all of stdout, the count and the skipped binary on stderr, a refusal of `occurrence=N` over a planned-but-unshown line naming the `--sub` licence, an unplanned line refused, the plan applying with no further read, every usage refusal, `no match:` and `incomplete:` | none | S2, S3 |
| `TestASubPlanParsesBackToTheSamePathsAndBodies` | `internal/read/sub161_test.go` | the round trip through `plan.Parse`: paths with a space, both quote kinds and a backslash; bodies starting `@@`, `body=`, a byte order mark, and an empty one | none | S3 |
| `TestAPlanLicenceThatShowsNothingSurvivesTheLedger` | `internal/seen/sub161_test.go` | the `w<shown>\|<licensed>` line written and parsed back; merges with a plan, a whole read and a partial read never widen the licence | none | S2 |
| `TestASubRefusesANewTextEndingInACR` | `internal/read/sub161_test.go` | a line of a mixed-ending file whose new text would end in a CR is refused by name, nothing planned or licensed | none | S3 |
| `TestASubWriteNeverReadsAFileNameAsAPointer` | `cmd/mrw/sub161_test.go` | a file named `@1`, with working-set entry 1 on another file holding the same text, licensed and excluded: the write changes `@1` only | none | S3, S4 |
| `TestALedgerNamesItsWriter` | `internal/seen/writer150_test.go` | the ledger is written as format v6; v4, v5 and v6 are read (ADR-150, raised by this task) | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `internal/read/sub.go` |
| 2 — something selects it | the `--sub` flag on `read` and `write` |
| 3 — the caller can discover it | `--help`, `mrw instructions` (T3) |
| 4 — it is used | T5's pilot |

## Mutation Log
(empty until execute)
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `internal/read/sub.go` · S2: plan and license every served line, changed or not · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · covers:a sub read prints a plan and licenses only its lines
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: drop the -C refusal beside --sub · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · covers:the refusals
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `cmd/mrw/main.go` · S4: apply the sub plan without recording what it read · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · covers:a sub write applies that plan as one plan
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `internal/read/sub.go` · S2: plan and license every served line, changed or not · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · covers:a sub read prints a plan and licenses only its lines
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `internal/read/read.go` · S2: the planned lines count as shown, so occurrence=N trusts a count nobody saw · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · covers:a sub read prints a plan and licenses only its lines
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: drop the -C refusal beside --sub · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · covers:the refusals
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `cmd/mrw/sub.go` · S3: let a line break through the replacement · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · covers:the refusals
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `cmd/mrw/main.go` · S4: apply the sub plan without recording what it read · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · covers:a sub write applies that plan as one plan
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `cmd/mrw/main.go` · S4: apply a partial plan when the walk refused or could not read a matching file · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · covers:a sub write applies that plan as one plan
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `cmd/mrw/main.go` · S4: a --sub write refusal under --json is plain text, not one document · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `cmd/mrw/main.go` · S4: write --grep --sub ignores --exclude · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `internal/read/sub.go` · S3: a path starting with @ is printed bare and read back as a working-set pointer · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `internal/read/sub.go` · S3: a new text ending in CR is planned and written a byte short · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0
- 2026-10-16 · 8c4b3f31* · mutant killed · exit 1 · `internal/seen/seen.go` · S2: the w<shown>|<licensed> line goes out in a v5 ledger an older mrw misreads · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0

## Invariants

- A read without `--sub` and a write with a PLAN are unchanged.
- The engine packages change only where this record owns it (`internal/read`).

## Risks

- `ReplaceAllString`'s `$` expansion surprises a caller who meant a literal `$`; `--help` says `$$`.

## Stop Condition

The plan grammar cannot carry a line some file holds and no refusal can name it.

## Out of Scope

- A substitution across lines (permanent: boundary: Decision 1 rewrites line by line)

## Verification Log
(empty until execute)
- 2026-10-16 · 8c4b3f31* · exit 1 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:78930 · test-lock-sha256:9eb39bbc355f2e17dacdd2e519c72021ba22b843a810279f87680da18bb1b979 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9zdWIxNjFfdGVzdC5nbwlUZXN0QVN1YlJlYWRSZWZ1c2Fsc0FuZEV4aXRzCTVkNzA2YjE2MzNmMmE2YmZiNTQ2N2M3MTk5ODQwMzlmYjI5ZTZkY2IwMDVjMTAxMDY5MGNiNjFkNmUxZWIxNDgKYm9keQljbWQvbXJ3L3N1YjE2MV90ZXN0LmdvCVRlc3RBU3ViV3JpdGVBcHBsaWVzVGhlUGxhbkluT25lQ29tbWFuZAk3NjU1ODBiMDAyNzI4NGEyYzRlODM4YzczZjExOWNlMWJjNTFhMDI0MTM5NDI5NDg5MWRiZWJlODcxMzE5OWQ5CmJvZHkJaW50ZXJuYWwvcmVhZC9zdWIxNjFfdGVzdC5nbwlUZXN0QVN1YlJlYWRQcmludHNBUGxhbkFuZExpY2Vuc2VzSXRzTGluZXMJYjNjM2U3ZTFiMzcwZDAwNTI0OGM5NDE0MDhhMzkwNDA0MGNiNzJkMTU0YzExNjk3ZTYxM2IxYjQ2ZWZhNDRmYw
  ```
  ```
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:123275
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:126371
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:130360
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:145458
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:119243
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:123062
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:134860
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:122747
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:120748
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:119485
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:118417
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:121659
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:118081
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:116988
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:124730
- 2026-10-16 · 8c4b3f31* · exit 0 · `set -o pipefail …` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:119877
- 2026-10-16 · 8c4b3f31* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:0 · test-lock-sha256:e0485fa0535f9596b082406a6c00bd241910a6a38f989402191b466a148f5d73 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9zdWIxNjFfdGVzdC5nbwlUZXN0QVN1YlJlYWRSZWZ1c2Fsc0FuZEV4aXRzCWVhYTg2YmJiNWYwMzFlOTI5ZmFhNTNmOTM2YjhjOWQ4ZTA2NzdhZWZkZDhlYjg3NDI5NzFkYTUwMTk2ZTZjMjEKYm9keQljbWQvbXJ3L3N1YjE2MV90ZXN0LmdvCVRlc3RBU3ViV3JpdGVBcHBsaWVzVGhlUGxhbkluT25lQ29tbWFuZAk3ZDYwYTQ1MzEyMTE3YmY2ZDU4NmU5YmVjNTY1ZWUyNjE2NzhhNmFiYThlZDM0NDI2NzE5MjEzM2JjODQzYzBkCmJvZHkJY21kL21ydy9zdWIxNjFfdGVzdC5nbwlUZXN0QVN1YldyaXRlTmV2ZXJSZWFkc0FGaWxlTmFtZUFzQVBvaW50ZXIJZTMyMDQ1ZGEzZjNjMmJiYjRmZThlYWQ0ODJhZWFlNTEwOTk3YzdmMDM4NmQ5YjM4Mjc4MGJmNzE2ZjNlNjkyMwpib2R5CWludGVybmFsL3JlYWQvc3ViMTYxX3Rlc3QuZ28JVGVzdEFTdWJQbGFuUGFyc2VzQmFja1RvVGhlU2FtZVBhdGhzQW5kQm9kaWVzCTk0ODg4NDZkMWM1NTczMGQ3OTlmNmI0NzFjY2VmZjFjOGY0NzE3MmM4MDhkOWFhNWE5ZGU2OWNiMmIwOGFlYjIKYm9keQlpbnRlcm5hbC9yZWFkL3N1YjE2MV90ZXN0LmdvCVRlc3RBU3ViUmVhZFByaW50c0FQbGFuQW5kTGljZW5zZXNJdHNMaW5lcwkzZmJhMjVkMDdlMmJkYWIwNGQzYWZjMjAxODkwZGQ4MWRmY2Q4OGM2ODE0NTQ4M2E5NWEwZTdhZTY5OTNmMTk5CmJvZHkJaW50ZXJuYWwvcmVhZC9zdWIxNjFfdGVzdC5nbwlUZXN0QVN1YlJlZnVzZXNBTmV3VGV4dEVuZGluZ0luQUNSCTA2Y2ZmY2Q0Mzc0OGFiNzgxYTZmYjE2NTdkNGI3MTA3NTlmZGViYWEzNDI1M2Y5Yzk2MDYxMWY1OWFmZGRhNDIKYm9keQlpbnRlcm5hbC9zZWVuL3N1YjE2MV90ZXN0LmdvCVRlc3RBUGxhbkxpY2VuY2VUaGF0U2hvd3NOb3RoaW5nU3Vydml2ZXNUaGVMZWRnZXIJMTkxNjEwOWQzYzkwNWZhMjQyNDIxMGMwYmE3ZjE5ODU2MzZmZWJiYzRkODk5ZjNiZDNhY2Y5OWNjNWUxMjRkOApib2R5CWludGVybmFsL3NlZW4vd3JpdGVyMTUwX3Rlc3QuZ28JVGVzdEFMZWRnZXJOYW1lc0l0c1dyaXRlcglmMGNjMDYyMjY2MDcyYWFhZDRlMmU3OWFkYjViYTE4MWQ5MDEyODEwNTFmYjc2YjZiNzFlOGE4NzE4ZGVmNDdlCmJvZHkJaW50ZXJuYWwvc2Vlbi93cml0ZXIxNTBfdGVzdC5nbwlUZXN0QU5ld2VyTGVkZ2VySXNOZWl0aGVyVXNlZE5vck92ZXJ3cml0dGVuCTNiZWFmZDBlOGNmNjYyNWZhMDdkN2JlOWQyZGQyNjY5ZTIzNTBlZDk3ZWRmMjhkNDBlM2I4MjI5MDQxMjQ1MmM · test-lock-kind:replace
- 2026-10-29 · 41a53637* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ad4798bceed21e7530ffc1e487f2462634ca9cac384fb0720482ee36559164c0 · ms:0 · test-lock-sha256:02cac1c68e5a1a2046183f672a25813404e58e8a0577a0e1fa17edfc27192588 · test-lock-file:docs/adr/locks/02cac1c68e5a1a2046183f672a25813404e58e8a0577a0e1fa17edfc27192588.lock · test-lock-kind:replace
