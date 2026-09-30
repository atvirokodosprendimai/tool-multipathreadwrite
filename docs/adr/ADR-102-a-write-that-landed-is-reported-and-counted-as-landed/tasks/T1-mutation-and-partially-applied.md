# Task ADR-102-T1: `writer.MutationOf` and the sixth outcome, `partially_applied`

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `writer.MutationOf(res)`; the `partially_applied` outcome; contract §196
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `every tally asks what landed`, `a partial commit is partially_applied`, `a contract row drives the binary`, `no engine package changes`

## Goal

`writer.MutationOf(res)` returns `writer.None`, `writer.Partial` or `writer.Complete`; a sixth outcome
`partially_applied` sits in `authoring`'s vocabulary between `refused_apply` and `check_not_run` and in `Landed()`;
the CLI and `mrw_write` count a commit that failed after a file landed as `partially_applied` (one that failed
before any file landed stays `refused_apply`); `stats` prints it at zero and its landed line names it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/writer/writer.go` | edit | `Mutation` type, `None`/`Partial`/`Complete`, `MutationOf(res)` from `DryRun`, `Applied`, `files[].written` |
| `internal/authoring/authoring.go` | edit | `PartiallyApplied`; `names` and `Plans()` derived from `Vocabulary()` (one list, not three) |
| `cmd/mrw/main.go` | edit | the tally at `:1324-1328` and `:1420-1429` and the recent ring gate (`:1395`) ask `writer.MutationOf`; the stats name column (`:516`, `%-14s`) fits the 17-character name; the landed line (`:520`) names `partially_applied` and says a partial commit ran no check |
| `internal/mcp/tools.go` | edit | the tally switch (`:722-735`), the recent ring gate (`:710`) and the unreportable branch (`:872-879`) ask `writer.MutationOf` |
| `internal/apply/mutation102_test.go`, `cmd/mrw/landed102_test.go`, `internal/mcp/landed102_test.go` | add | the tests below |
| `scripts/contract.sh` | edit | §196 |
| `AGENTS.md`, `README.md`, `cmd/opencode/mrw-plugin/src/index.ts` (+ `dist/`) | edit | the stats vocabulary and `Landed` |

## Ordered Steps

1. [S1] Write the failing tests `TestMutationSaysHowMuchOfAPlanLanded`, `TestAPartialCommitIsCountedAsPartiallyApplied` and `TestAnMCPPartialCommitIsCountedAsPartiallyApplied`; confirm RED on `f1d5996`. [proof: mutation]
2. [S2] Add `writer.MutationOf` and the outcome; route the four tallies through it. [proof: mutation] Mutants: `Mutation` reports a partial commit as none; the CLI tally records `refused_apply` for a partial commit; `Landed()` leaves `partially_applied` out.
3. [S3] Contract §196, driving `$MRW`: a plan replacing `a.txt` and unlinking `d/x.txt` in a read-only `d` exits 2 PARTIALLY APPLIED; `stats --json` counts `partially_applied` 1 and `landed` 1; the pair, the same plan with `d` writable, counts `applied`. [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §196's rows printed — the fence greps the section, since the whole contract takes minutes]
4. [S4] AGENTS, README and the opencode plugin describe the six names and `Landed`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/writer/ ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 300s -run 'TestMutationSaysHowMuchOfAPlanLanded|TestAPartialCommitIsCountedAsPartiallyApplied|TestAnMCPPartialCommitIsCountedAsPartiallyApplied' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestMutationSaysHowMuchOfAPlanLanded \(' "$out" \
  && grep -qE '^--- PASS: TestAPartialCommitIsCountedAsPartiallyApplied \(' "$out" \
  && grep -qE '^--- PASS: TestAnMCPPartialCommitIsCountedAsPartiallyApplied \(' "$out" \
  && grep -q '^# 196\. ' scripts/contract.sh \
  && grep -q 'partially_applied' AGENTS.md \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestMutationSaysHowMuchOfAPlanLanded` | `internal/writer/mutation102_test.go` | a dry run and an unwritten result are none; not applied with one file written is partial; applied is complete | — | S1, S2 |
| `TestAPartialCommitIsCountedAsPartiallyApplied` | `cmd/mrw/landed102_test.go` | the read-only-directory partial commit (a single-line `a.go` replace, then the unlink) through `mrw write` exits 2 and `stats --json` counts `partially_applied` 1, `refused_apply` 0, `landed` 1 and `strict_candidates` 1; `stats` prints `partially_applied` | — | S1, S2 |
| `TestAnMCPPartialCommitIsCountedAsPartiallyApplied` | `internal/mcp/landed102_test.go` | the same partial commit through `mrw_write` counts `partially_applied` 1 and `refused_apply` 0, one entry in the recent ring and one strict-balance candidate | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every `mrw write` and `mrw_write` goes through `writer.Apply` and the tallies; the tests drive `rootCommand` or `Serve` |
| 3 — the caller can discover it | the receipt, `stats`, and the ledger's licence on the next write |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 Codex design review |

## Mutation Log
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/writer/writer.go` · MutationOf calls a partial commit none · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · covers:every tally asks what landed
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI counts a partial commit refused_apply again · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · covers:a partial commit is partially_applied
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/authoring/authoring.go` · Landed leaves partially_applied out · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · covers:a partial commit is partially_applied
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_write counts a partial commit refused_apply again · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · covers:a partial commit is partially_applied
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package changed against the merge-base · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · covers:no engine package changes
- 2026-09-30 · 23beaf7* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI leaves a partial commit out of the pricing · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · covers:a partial commit is partially_applied
- 2026-09-30 · 23beaf7* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_write leaves a partial commit out of the ring and the pricing · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · covers:a partial commit is partially_applied

## Invariants

- Exit codes and hunk statuses are unchanged.
- A write that landed whole is counted and recorded exactly as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-102 tasks, each in its own file.

## Verification Log
- 2026-09-30 · f1d5996* · exit 1 · `set -o pipefail …` · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · ms:800 · test-lock-sha256:64a2a55d8b8cf549b42c1a2bd7abe22b1dda982bfda7a8cc5a43f2996f1b7d16 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvbGFuZGVkMTAyX3Rlc3QuZ28JVGVzdEFQYXJ0aWFsQ29tbWl0SXNDb3VudGVkQXNQYXJ0aWFsbHlBcHBsaWVkCWM4M2ViMWJmODY3YjQxYzZlZDQ3YjJlNmFlZWIzOWMxOGU0MTFmZTNmNTdlMWI1NWI4MWFmMTg5YmIxZmQzZGMKYm9keQlpbnRlcm5hbC9tY3AvbGFuZGVkMTAyX3Rlc3QuZ28JVGVzdEFMZWRnZXJGYWlsdXJlU3RpbGxTZW5kc1RoZU1DUFJlY2VpcHQJNzY3OTUwOTM4OWZjYTAxOTQwMmM2YTNkZDNjZDMxZTkyYTFmZmZhYzM5ZDk2MjJlM2NhZGZmZGJiYzg2OWI3Mwpib2R5CWludGVybmFsL21jcC9sYW5kZWQxMDJfdGVzdC5nbwlUZXN0QW5NQ1BQYXJ0aWFsQ29tbWl0SXNDb3VudGVkQXNQYXJ0aWFsbHlBcHBsaWVkCWViODk2OGZjMmRjZWQ5N2UwY2QxYWQ4NzdiYzQ1MjkwZDI4NzJmMTVjMDUxYTNjY2RhNTk2MTdkZmJmZjc1NGUKYm9keQlpbnRlcm5hbC9tY3AvbGFuZGVkMTAyX3Rlc3QuZ28JVGVzdEFuVW5yZXBvcnRhYmxlV3JpdGVTYXlzTm90VG9SZXJ1bgk5ZTlkNjQzNGY4MjM4MzNjNTVmN2U3YWI2NGI0YzgyZmU2NWY1NzlkMWZlMDUyMTNhZGFhOWFiMWMyN2ZlMDc1CmJvZHkJaW50ZXJuYWwvd3JpdGVyL211dGF0aW9uMTAyX3Rlc3QuZ28JVGVzdE11dGF0aW9uU2F5c0hvd011Y2hPZkFQbGFuTGFuZGVkCWNjZGIyNTZkMWNiNTc4MWIwMDM0OTQxMjdjYmE4M2M3N2YwOTQzNTRlMDhmNzNhYWEyNmZjYjNlYmFiOTljYjY
  ```
  --- last 10 line(s) of stdout (of 24 after folding 24 raw)
      landed102_test.go:39: open /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAPartialCommitIsCountedAsPartiallyApplied491733738/002/d/x.txt: no such file or directory
  --- FAIL: TestAPartialCommitIsCountedAsPartiallyApplied (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.259s
  === RUN   TestAnMCPPartialCommitIsCountedAsPartiallyApplied
      landed102_test.go:59: tally map[refused_apply:1] landed 0, want partially_applied 1, refused_apply 0, landed 1
  --- FAIL: TestAnMCPPartialCommitIsCountedAsPartiallyApplied (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.264s
  FAIL
  ```
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · ms:570
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · ms:420
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · ms:421
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · ms:390
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · ms:381
- 2026-09-30 · human-observed · S3 observed 2026-09-30: ./scripts/contract.sh run unpiped on this branch (base f1d5996), exit 0 'contract holds', with §196 printed: the read-only-directory commit exits 2 PARTIALLY APPLIED with a.txt written and d/x.txt kept; stats counts partially_applied 1 and landed 1; the written file takes the next write without a re-read; the pair unlink applies with d writable. §91 now also asserts --json carries partially_applied
- 2026-09-30 · human-observed · relock 2026-09-30 (Codex review of #293, findings 1-2): the partial-commit tests now replace a single line of a.go rather than a.txt and also assert the landed write is priced (strict_candidates 1) and, over MCP, joins the recent ring; TestALedgerFailureStillSendsTheMCPReceipt also asserts the ring and the receipt's pattern.window; every earlier assertion kept
- 2026-09-30 · 23beaf7* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · ms:0 · test-lock-sha256:2854dec3e1108c90757740a123a4176e48e0c1443eb15de4cbc4fe7dcb1b1b3b · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvbGFuZGVkMTAyX3Rlc3QuZ28JVGVzdEFQYXJ0aWFsQ29tbWl0SXNDb3VudGVkQXNQYXJ0aWFsbHlBcHBsaWVkCTA0NDU3MDE3MmVlYWFmYmRkZWYyM2Y3ZWEyN2Y5ZjUzMTQyZjNlNGYyNDA4NjRhM2Q0YTJhNThiMmRiOTJhM2MKYm9keQlpbnRlcm5hbC9tY3AvbGFuZGVkMTAyX3Rlc3QuZ28JVGVzdEFMZWRnZXJGYWlsdXJlU3RpbGxTZW5kc1RoZU1DUFJlY2VpcHQJMGQ1ODZlZmUyYTk0NzVkZDk3ZDViZDE1YTNhM2QyOWQzNGQyNDBkYjU4MWFkY2EwZWM5ZTIxNWNkZjExZTBmYQpib2R5CWludGVybmFsL21jcC9sYW5kZWQxMDJfdGVzdC5nbwlUZXN0QW5NQ1BQYXJ0aWFsQ29tbWl0SXNDb3VudGVkQXNQYXJ0aWFsbHlBcHBsaWVkCTU1MzQwOGQ0ZmE3ZGY3ODdlYmExNTNjYmNjOWNkZDdmMzBkYWNiZTc5NTgyMGVkZGY1ZDk3YTllNjliNGYyYTEKYm9keQlpbnRlcm5hbC9tY3AvbGFuZGVkMTAyX3Rlc3QuZ28JVGVzdEFuVW5yZXBvcnRhYmxlV3JpdGVTYXlzTm90VG9SZXJ1bgk5ZTlkNjQzNGY4MjM4MzNjNTVmN2U3YWI2NGI0YzgyZmU2NWY1NzlkMWZlMDUyMTNhZGFhOWFiMWMyN2ZlMDc1CmJvZHkJaW50ZXJuYWwvd3JpdGVyL211dGF0aW9uMTAyX3Rlc3QuZ28JVGVzdE11dGF0aW9uU2F5c0hvd011Y2hPZkFQbGFuTGFuZGVkCWNjZGIyNTZkMWNiNTc4MWIwMDM0OTQxMjdjYmE4M2M3N2YwOTQzNTRlMDhmNzNhYWEyNmZjYjNlYmFiOTljYjY · test-lock-kind:replace
- 2026-09-30 · 23beaf7* · exit 0 · `set -o pipefail …` · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · ms:580
- 2026-09-30 · 23beaf7* · exit 0 · `set -o pipefail …` · acceptance-sha256:2c02d10df7d600b228c81ee0835103df5115f35b1a3629bb24c40a60e814c8fe · ms:402
