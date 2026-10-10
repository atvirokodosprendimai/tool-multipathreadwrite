# Task ADR-141-T1: `--core` prints the eight rules, held to the binary and the full text

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `guide.Core`, `mrw instructions --core`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `instructions --core prints eight short rules that name only real flags and that the full contract also carries`

## Goal

Decisions 1–3 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | `Core()`; three sentences at the end of the full contract |
| `cmd/mrw/main.go` | edit | the `--core` flag on `instructions` |
| `internal/guide/core141_test.go`, `cmd/mrw/core141_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §242 |
| `AGENTS.md` | edit | the `mrw instructions` bullet |

## Ordered Steps

1. [S1] Write `TestTheCoreIsEightShortRulesTheFullContractStillCarries` (eight numbered rules, under 300 words, first line names the full form, and each rule's marker phrase is also in `CLI()`), and in `cmd/mrw` `TestTheCoreNamesOnlyFlagsTheBinaryHas` (every `--flag` in the core is a flag of `read`, `write` or the root command) and `TestInstructionsCoreFlagPrintsTheCore` (`instructions --core` prints `Core()`; plain `instructions` prints `CLI()`; `--core` with an argument is exit 2). Confirm RED.
2. [S2] `Core()`, the flag, the sentences at the end of the full contract. Mutants: the core names a flag the binary lacks; the flag prints the full text; the core drops the `--create` rule; the full contract loses the `(?i)` rule; and, per clause, a deleted clause of the core or of the full text. [proof: mutation]
3. [S3] Contract §242 and AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/guide/ -count=1 -timeout 300s -run 'TestTheCoreIsEightShortRulesTheFullContractStillCarries' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheCoreIsEightShortRulesTheFullContractStillCarries \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestTheCoreNamesOnlyFlagsTheBinaryHas|TestInstructionsCoreFlagPrintsTheCore|TestInstructionsCommandPrintsCLI' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheCoreNamesOnlyFlagsTheBinaryHas \(' "$out" \
  && grep -qE '^--- PASS: TestInstructionsCoreFlagPrintsTheCore \(' "$out" \
  && go test ./internal/guide/ -count=1 -timeout 900s \
  && grep -q '^# 242\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheCoreIsEightShortRulesTheFullContractStillCarries` | `internal/guide/core141_test.go` | eight numbered rules under 300 words, the full form named, each rule's marker also in `CLI()` | none | S1, S2 |
| `TestTheCoreNamesOnlyFlagsTheBinaryHas` | `cmd/mrw/core141_test.go` | every flag the core names exists on `read`, `write` or the root | none | S1, S2 |
| `TestInstructionsCoreFlagPrintsTheCore` | `cmd/mrw/core141_test.go` | `--core` prints the core, plain prints the full contract, an argument is exit 2 | none | S1, S2 |
| `TestInstructionsCommandPrintsCLI` | `cmd/mrw/instructions_test.go` | the plain command is still exactly `guide.CLI()` | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `guide.Core()` |
| 2 — something selects it | the `--core` flag of `instructions` |
| 3 — the caller can discover it | the last line of `mrw instructions`; `mrw instructions --help`; AGENTS.md |
| 4 — it is used | the 2026-10-09 survey; no telemetry (ADR-009) |

## Invariants

- `mrw instructions` without the flag is the full contract plus three sentences at its end.
- The MCP handshake document is unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the eight rules cannot stay under 300 words without dropping one the survey shows costing turns: the record would then name nine.

## Out of Scope

- Shortening the full text or the prose copies (permanent: boundary: the record's Out of Scope)

## Mutation Log
- 2026-10-10 · 8c0436a · mutant killed · exit 1 · `internal/guide/guide.go` · S2: the core names a flag the binary lacks · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · covers:instructions --core prints eight short rules that name only real flags and that the full contract also carries
- 2026-10-10 · 8c0436a* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: --core prints the full contract · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · covers:instructions --core prints eight short rules that name only real flags and that the full contract also carries
- 2026-10-10 · 8c0436a* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: the core drops the --create rule · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · covers:instructions --core prints eight short rules that name only real flags and that the full contract also carries
- 2026-10-10 · 8c0436a* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: the full contract loses the (?i) rule the core teaches · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · covers:instructions --core prints eight short rules that name only real flags and that the full contract also carries
- 2026-10-10 · b38d44d · mutant killed · exit 1 · `internal/guide/guide.go` · S2: a deleted clause of the core (rule 6 --max-cols) · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · covers:instructions --core prints eight short rules that name only real flags and that the full contract also carries
- 2026-10-10 · b38d44d* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: a deleted clause of the full text (mrw write -) · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · covers:instructions --core prints eight short rules that name only real flags and that the full contract also carries

## Verification Log
- 2026-10-10 · 8c0436a · exit 0 · `set -o pipefail …` · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · ms:2104
- 2026-10-10 · 8c0436a* · exit 0 · `set -o pipefail …` · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · ms:1858
- 2026-10-10 · 8c0436a* · exit 0 · `set -o pipefail …` · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · ms:1798
- 2026-10-10 · 8c0436a* · exit 0 · `set -o pipefail …` · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · ms:1827
- 2026-10-10 · 8c0436a* · exit 1 · `set -o pipefail …` · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · ms:205 · test-lock-sha256:72f288e9bbbdc6c4d23a4f2cf3fe0b28e682469d7d7b64cb6b6c1f51df880d48 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9jb3JlMTQxX3Rlc3QuZ28JVGVzdEluc3RydWN0aW9uc0NvcmVGbGFnUHJpbnRzVGhlQ29yZQllMGU4NmFiNGQ0NDcwMTYyMjRlYTFmYWVmMmE0YjM5MTY1MGYyZDhkNDc2OWJkMjM0YmZkNDFjZmY1NzQ3YTE1CmJvZHkJY21kL21ydy9jb3JlMTQxX3Rlc3QuZ28JVGVzdFRoZUNvcmVOYW1lc09ubHlGbGFnc1RoZUJpbmFyeUhhcwkyZjA4Y2MzZmI3MjFmYmVmZThlZGQzNzdlZTg2MThlMGQxNTI2NjQzMmI4YjIwMjBiZmIzNjNmNmRkYTJjNWM2CmJvZHkJY21kL21ydy9pbnN0cnVjdGlvbnNfdGVzdC5nbwlUZXN0RXZlcnlSZWFkRmxhZ0lzVGF1Z2h0QnlJbnN0cnVjdGlvbnMJODdlZGE1ZDFlMjNlMGE5N2NkNjg0NmJhMzlhNTg2MGE1YTZlZGNmYmUzYmE0YThmOTk1YzIwODY0YmZiYTFhZgpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEhlbHBTdW1tYXJpZXNOYW1lVGhlUmVhZFNpZGUJMWViYmI5NzU2MDExN2YxYzBjNzRkZWViOGM5NzU5OGVjMTFhYjcxYWYwMjcwYTViYmU0NzhjYWZkMjk3ZDUyYQpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEluc3RydWN0aW9uc0NvbW1hbmRQcmludHNDTEkJZTMwNjAwMzhkNGZhNDE0NmEzODU1NjhjNmQ1MmY0MzVmY2VmYmQyNzhiMDFkM2Q2NTlmZmFlYTFlOGRlYmMwMApib2R5CWludGVybmFsL2d1aWRlL2NvcmUxNDFfdGVzdC5nbwlUZXN0VGhlQ29yZUlzRWlnaHRTaG9ydFJ1bGVzVGhlRnVsbENvbnRyYWN0U3RpbGxDYXJyaWVzCWUyOTY3ZGY5NWY4YWZmNjBiYzU0YzVhMjJmMGI1NmU4N2I5MmJhOWZjZDRhNGZkYThhOWEwZmVmZjI4NDU0NWI
  ```
  --- last 4 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide.test]
  internal/guide/core141_test.go:14:10: undefined: Core
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide [build failed]
  FAIL
  ```
- 2026-10-10 · 8c0436a* · exit 0 · `set -o pipefail …` · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · ms:1153
- 2026-10-10 · 76a65cb* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · ms:0 · test-lock-sha256:c7a4c9a105c62959f792713200bd199cd1e51d9bbf7c2bc5ec2d64fa74fe73a3 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9jb3JlMTQxX3Rlc3QuZ28JVGVzdEluc3RydWN0aW9uc0NvcmVGbGFnUHJpbnRzVGhlQ29yZQllMGU4NmFiNGQ0NDcwMTYyMjRlYTFmYWVmMmE0YjM5MTY1MGYyZDhkNDc2OWJkMjM0YmZkNDFjZmY1NzQ3YTE1CmJvZHkJY21kL21ydy9jb3JlMTQxX3Rlc3QuZ28JVGVzdFRoZUNvcmVOYW1lc09ubHlGbGFnc1RoZUJpbmFyeUhhcwkyZjA4Y2MzZmI3MjFmYmVmZThlZGQzNzdlZTg2MThlMGQxNTI2NjQzMmI4YjIwMjBiZmIzNjNmNmRkYTJjNWM2CmJvZHkJY21kL21ydy9pbnN0cnVjdGlvbnNfdGVzdC5nbwlUZXN0RXZlcnlSZWFkRmxhZ0lzVGF1Z2h0QnlJbnN0cnVjdGlvbnMJODdlZGE1ZDFlMjNlMGE5N2NkNjg0NmJhMzlhNTg2MGE1YTZlZGNmYmUzYmE0YThmOTk1YzIwODY0YmZiYTFhZgpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEhlbHBTdW1tYXJpZXNOYW1lVGhlUmVhZFNpZGUJMWViYmI5NzU2MDExN2YxYzBjNzRkZWViOGM5NzU5OGVjMTFhYjcxYWYwMjcwYTViYmU0NzhjYWZkMjk3ZDUyYQpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEluc3RydWN0aW9uc0NvbW1hbmRQcmludHNDTEkJZTMwNjAwMzhkNGZhNDE0NmEzODU1NjhjNmQ1MmY0MzVmY2VmYmQyNzhiMDFkM2Q2NTlmZmFlYTFlOGRlYmMwMApib2R5CWludGVybmFsL2d1aWRlL2NvcmUxNDFfdGVzdC5nbwlUZXN0VGhlQ29yZUlzRWlnaHRTaG9ydFJ1bGVzVGhlRnVsbENvbnRyYWN0U3RpbGxDYXJyaWVzCWUzY2VjNGNmZDRmMDUyMGViMTI2YTg1NDQzMWEzOGZjNTU0N2VmOGQ4N2ZkNGM1NTNjMmNjYTM1NWUzNDgzNGM · test-lock-kind:replace
- 2026-10-10 · b38d44d · exit 0 · `set -o pipefail …` · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · ms:2476
- 2026-10-10 · b38d44d* · exit 0 · `set -o pipefail …` · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · ms:2737
- 2026-10-10 · b911dba* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · ms:0 · test-lock-sha256:c7a4c9a105c62959f792713200bd199cd1e51d9bbf7c2bc5ec2d64fa74fe73a3 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9jb3JlMTQxX3Rlc3QuZ28JVGVzdEluc3RydWN0aW9uc0NvcmVGbGFnUHJpbnRzVGhlQ29yZQllMGU4NmFiNGQ0NDcwMTYyMjRlYTFmYWVmMmE0YjM5MTY1MGYyZDhkNDc2OWJkMjM0YmZkNDFjZmY1NzQ3YTE1CmJvZHkJY21kL21ydy9jb3JlMTQxX3Rlc3QuZ28JVGVzdFRoZUNvcmVOYW1lc09ubHlGbGFnc1RoZUJpbmFyeUhhcwkyZjA4Y2MzZmI3MjFmYmVmZThlZGQzNzdlZTg2MThlMGQxNTI2NjQzMmI4YjIwMjBiZmIzNjNmNmRkYTJjNWM2CmJvZHkJY21kL21ydy9pbnN0cnVjdGlvbnNfdGVzdC5nbwlUZXN0RXZlcnlSZWFkRmxhZ0lzVGF1Z2h0QnlJbnN0cnVjdGlvbnMJODdlZGE1ZDFlMjNlMGE5N2NkNjg0NmJhMzlhNTg2MGE1YTZlZGNmYmUzYmE0YThmOTk1YzIwODY0YmZiYTFhZgpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEhlbHBTdW1tYXJpZXNOYW1lVGhlUmVhZFNpZGUJMWViYmI5NzU2MDExN2YxYzBjNzRkZWViOGM5NzU5OGVjMTFhYjcxYWYwMjcwYTViYmU0NzhjYWZkMjk3ZDUyYQpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEluc3RydWN0aW9uc0NvbW1hbmRQcmludHNDTEkJZTMwNjAwMzhkNGZhNDE0NmEzODU1NjhjNmQ1MmY0MzVmY2VmYmQyNzhiMDFkM2Q2NTlmZmFlYTFlOGRlYmMwMApib2R5CWludGVybmFsL2d1aWRlL2NvcmUxNDFfdGVzdC5nbwlUZXN0VGhlQ29yZUlzRWlnaHRTaG9ydFJ1bGVzVGhlRnVsbENvbnRyYWN0U3RpbGxDYXJyaWVzCWUzY2VjNGNmZDRmMDUyMGViMTI2YTg1NDQzMWEzOGZjNTU0N2VmOGQ4N2ZkNGM1NTNjMmNjYTM1NWUzNDgzNGM · test-lock-kind:replace
- 2026-10-10 · b911dba* · exit 0 · `set -o pipefail …` · acceptance-sha256:3472305e900a551e9258cd3f5e6b129dca832b57e4c4a882484ec0594db32ace · ms:1075
