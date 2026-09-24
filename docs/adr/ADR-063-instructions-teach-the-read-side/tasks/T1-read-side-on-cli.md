# Task ADR-063-T1: read section on CLI(); every read flag named; help summaries; contract §115

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** read section on CLI(); every read flag named; root, `read` and `--ast-grep` Usage name finding; skill description; §115
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `CLI teaches the read and plan address forms`, `CLI names every read flag`, `a write's start pattern is taught as exactly once`, `help summaries name finding`, `the skill description names --grep`, `the handshake and the engine do not move`

## Goal

`guide.CLI()` carries the read section ADR-063 quotes, after the write cookbook. Every flag `readCmd()` defines appears in `guide.CLI()` as `--<name>`. The root `Usage` is `find, read and write many file ranges in one call`; the `read` `Usage` is `print line ranges, pattern matches or --grep hits from one or more files`; the `--ast-grep` placeholder is `PATTERN`. The repo skill description names the address forms and `--grep`. Contract §115 drives the shipped binary. `internal/mcp` and the engine packages do not move.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | read section appended to CLI(). |
| `internal/guide/guide_test.go` | edit | `TestCLITeachesTheReadSide`. |
| `cmd/mrw/main.go` | edit | root, `read` and `--ast-grep` Usage strings. |
| `cmd/mrw/instructions_test.go` | edit | `TestEveryReadFlagIsTaughtByInstructions` iterates `readCmd().Flags`; `TestHelpSummariesNameTheReadSide`. |
| `.claude/skills/mrw/SKILL.md` | edit | description names the address forms and `--grep`. |
| `scripts/contract.sh` | edit | **§115** — next free after §114. |
| `docs/adr/BACKLOG.md` | edit | ADR-063 row; centralised skill deferral receipt. |
| `AGENTS.md`, `README.md` | edit | their `rg -l … \| mrw read --files-from -` pipeline names `.` (chaos finding, 2026-09-24). |

## Ordered Steps

1. [S1] Write `TestCLITeachesTheReadSide`, `TestEveryReadFlagIsTaughtByInstructions` and `TestHelpSummariesNameTheReadSide`; confirm all three are RED on the current text. [proof: mutation]
2. [S2] Append the read section to `guide.CLI()` and change the three `Usage` strings. S1 GREEN. Deleting the `--files-from` line must fail S1; deleting "exactly once" must fail S1; unquoting the example spec must fail S1; reverting the `read` Usage must fail S1. [proof: mutation]
3. [S3] Write §115 against the v1.22.1 binary and confirm it is RED; rebuild and confirm GREEN. The good half: `$MRW instructions` carries the address list written literally into the row, and every flag in the OPTIONS block of `$MRW read --help` except `--help`; `$MRW read --help` shows `--ast-grep PATTERN`; the quoted example spec and `printf 'b.go:/X/\n' | $MRW read --files-from -` exit 0 against a fixture, and `$MRW read b.go:-2` and `$MRW read b.go:4,+9` on a 5-line file exit 0 (the second serving `@@ 4-5`). The pair, each of which must fail: a plan `@@ b.go /X/ replace` where `X` matches twice exits 1 and leaves the file byte-identical; `@@ b.go 4,+9 delete` on the 5-line file exits 1; `@@ b.go -2 delete` is refused. Deleting the `--grep` sentence from CLI() must fail §115. [proof: mutation]
4. [S4] Skill description clause, measured in bytes under 1,024. BACKLOG row and receipt. Scoped tests, `gofmt`, `go vet` unpiped; `./scripts/contract.sh` before commit. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 115\. ' scripts/contract.sh \
  && go test ./internal/guide/ ./cmd/mrw/ -count=1 -v \
    -run 'TestCLITeachesTheReadSide|TestEveryReadFlagIsTaughtByInstructions|TestHelpSummariesNameTheReadSide|TestCLIContainsSharedAndTheOperatorTraps|TestCLITeachesAlwaysAndAPlan|TestInstructionsCommandPrintsCLI' 2>&1 | tee /tmp/adr063-t1.out \
  && grep -q '^--- PASS: TestCLITeachesTheReadSide ' /tmp/adr063-t1.out \
  && grep -q '^--- PASS: TestEveryReadFlagIsTaughtByInstructions ' /tmp/adr063-t1.out \
  && grep -q '^--- PASS: TestHelpSummariesNameTheReadSide ' /tmp/adr063-t1.out \
  && grep -q '^--- PASS: TestInstructionsCommandPrintsCLI ' /tmp/adr063-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr063-t1.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/mcp internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(gofmt -l internal/guide cmd/mrw)" ] \
  && python3 -c "import sys,re; t=open('.claude/skills/mrw/SKILL.md').read(); d=re.search(r'^description: >-\n((?:  .*\n)+)', t, re.M).group(1); d=' '.join(l.strip() for l in d.splitlines()); sys.exit(0 if '--grep' in d and len(d.encode()) < 1024 else 1)" \
  && go vet ./internal/guide/ ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestCLITeachesTheReadSide` | `internal/guide/guide_test.go` | CLI() names the read and plan address forms (including `N-`, `-M`, `A,+N`, `/from/,/to/`), that a plan refuses `-M`, `--grep`, `--exclude`, `--ast-grep`, `--files-from`, that a write's start pattern must match exactly once, and quotes the example spec; Shared() carries none of it | — | S1, S2 |
| `TestEveryReadFlagIsTaughtByInstructions` | `cmd/mrw/instructions_test.go` | every `readCmd().Flags` name appears in `guide.CLI()` as `--<name>` | — | S1, S2 |
| `TestHelpSummariesNameTheReadSide` | `cmd/mrw/instructions_test.go` | `rootCommand().Usage` names finding; `readCmd().Usage` names pattern matches and `--grep`; the `--ast-grep` flag's placeholder is `PATTERN` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §115 |
| 2 — something selects it | `mrw instructions` prints `guide.CLI()` (`TestInstructionsCommandPrintsCLI`); `mrw --help` prints `rootCommand().Usage` and `readCmd().Usage`; deleting the read section fails S1 and §115 |
| 3 — the caller can discover it | `mrw --help`'s NAME line says mrw finds; it lists `instructions`; the new `read` Usage names patterns and `--grep` on the first screen |
| 4 — it is used | ADR-009 refuses telemetry; the evidence for building it is M's 2026-09-23 question, answered only by a live run |

## Mutation Log
(empty until execute)
- 2026-09-24 · dc2a584* · mutant killed · exit 1 · `internal/guide/guide.go` · CLI drops the --files-from line: TestEveryReadFlagIsTaughtByInstructions and TestCLITeachesTheReadSide must go red · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · covers:CLI names every read flag
- 2026-09-24 · dc2a584* · mutant killed · exit 1 · `internal/guide/guide.go` · CLI stops teaching the exactly-once start pattern: TestCLITeachesTheReadSide must go red · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · covers:a write's start pattern is taught as exactly once
- 2026-09-24 · dc2a584* · mutant killed · exit 1 · `internal/guide/guide.go` · the example spec with a space is unquoted and would split in the shell: TestCLITeachesTheReadSide must go red · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · covers:CLI teaches the read and plan address forms
- 2026-09-24 · dc2a584* · mutant killed · exit 1 · `cmd/mrw/main.go` · read Usage reverts to line ranges only: TestHelpSummariesNameTheReadSide must go red · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · covers:help summaries name finding
- 2026-09-24 · dc2a584* · mutant killed · exit 1 · `.claude/skills/mrw/SKILL.md` · skill description stops naming --grep: the fence's description clause must go red · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · covers:the skill description names --grep
- 2026-09-24 · dc2a584* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the handshake text moves: the fence's merge-base diff over internal/mcp must go red · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · covers:the handshake and the engine do not move
- 2026-09-24 · dc2a584* · mutant killed · exit 1 · `internal/guide/guide.go` · the taught rg pipeline loses its path and hangs on a piped stdin (chaos 2026-09-24): TestCLITeachesTheReadSide must go red · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · covers:CLI teaches the read and plan address forms
- 2026-09-24 · dc2a584* · mutant killed · exit 1 · `internal/guide/guide.go` · CLI stops saying a partial read exits 1 (chaos 2026-09-24): TestCLITeachesTheReadSide must go red · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · covers:CLI teaches the read and plan address forms
- 2026-09-24 · c7ae1fa* · mutant killed · exit 1 · `internal/guide/guide.go` · CLI drops that an end past EOF clamps with exit 0 (Codex P2, PR #199): TestCLITeachesTheReadSide must go red · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · covers:CLI teaches the read and plan address forms
- 2026-09-24 · c7ae1fa* · mutant killed · exit 1 · `internal/guide/guide.go` · CLI restores the overbroad .git claim (Codex P2, PR #199): TestCLITeachesTheReadSide must go red · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · covers:CLI teaches the read and plan address forms

## Invariants

- Shared() stays five sentences, and carries no read section.
- `internal/mcp` is not edited; the handshake stays ≤ 4096 bytes.
- Extra args to `mrw instructions` stay exit 2.
- Engine packages are not edited.
- No §NN in this Tests table.

## Risks

- One flag's long name is a prefix of another's (`--x` beside `--x-y`): the flag test matches `--<name>` followed by a character that cannot continue a flag name, not a bare substring.

## Stop Condition

If the only way to go green is to edit `internal/mcp` or raise 4096, stop.
If the only way to go green is to put the read section in Shared(), stop.

## Out of Scope

- Centralised skill description (parent Follow-ups; BACKLOG receipt)
- `mrw read --help` Description and the other flag usages (already the source this record copies from; only the `--ast-grep` placeholder changes)

## Verification Log
(empty until execute)
- 2026-09-24 · dc2a584* · exit 1 · `set -o pipefail …` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:32 · test-lock-sha256:5eb62f9ce76da9b12fae5e03127e83245c72a19e34b781957366cb44d3a58acc · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEV2ZXJ5UmVhZEZsYWdJc1RhdWdodEJ5SW5zdHJ1Y3Rpb25zCTg3ZWRhNWQxZTIzZTBhOTdjZDY4NDZiYTM5YTU4NjBhNWE2ZWRjZmJlM2JhNGE4Zjk5NWMyMDg2NGJmYmExYWYKYm9keQljbWQvbXJ3L2luc3RydWN0aW9uc190ZXN0LmdvCVRlc3RIZWxwU3VtbWFyaWVzTmFtZVRoZVJlYWRTaWRlCTAxZmU4ZmYxZTY4OTBiM2M5NGVhZDM1YzkwZDc3YjkxNTQ1OTZlOGMyNDNkNzI4MzQzNWM3YTg1ZGU2OTdhYTYKYm9keQljbWQvbXJ3L2luc3RydWN0aW9uc190ZXN0LmdvCVRlc3RJbnN0cnVjdGlvbnNDb21tYW5kUHJpbnRzQ0xJCWUzMDYwMDM4ZDRmYTQxNDZhMzg1NTY4YzZkNTJmNDM1ZmNlZmJkMjc4YjAxZDNkNjU5ZmZhZWExZThkZWJjMDAKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RDTElDb250YWluc1NoYXJlZEFuZFRoZU9wZXJhdG9yVHJhcHMJNjU2YjFmNjM1NGQ0Y2M2NDZhMzYwMWQwZGFlM2NmODJjZDc5YTBjYTBmNjEzNDc0NGUyZDg1NWY0MGNhZTc1NQpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdENMSVRlYWNoZXNBbHdheXNBbmRBUGxhbgk4NzZkOGQ1ZWVlNjBiMDg5OWYyNDg4ZmM3YThlN2MzOGM0OTM5NTFmNGU0Zjg3MTQ3MWRiMzBjNWYwZjdlNzExCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Q0xJVGVhY2hlc1RoZVJlYWRTaWRlCTRjNWJkYjk0MTQ3OWU3ZTAyZjM5NmFkYTViMWVjNjgxY2FlYzAwNzY2OTYwMmZmMTQ3MWI1OTY0ODA3YTljNGUKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RFdmVyeVN1cmZhY2VDb250YWluc1RoZVNoYXJlZFNlbnRlbmNlcwljNGUzMTY5MzFlMzVmMTViZTg5Y2ZlNGVhM2E0ZjVkNTg2YjcyNzQxNDkxOTg2MGIzZGEzMTM2NzE1M2NmNzcw
  ```
  ```
- 2026-09-24 · dc2a584* · exit 0 · `set -o pipefail …` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:507
- 2026-09-24 · dc2a584* · exit 0 · `set -o pipefail …` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:308
- 2026-09-24 · dc2a584* · exit 0 · `set -o pipefail …` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:314
- 2026-09-24 · dc2a584* · exit 0 · `set -o pipefail …` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:302
- 2026-09-24 · dc2a584* · exit 0 · `set -o pipefail …` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:310
- 2026-09-24 · dc2a584* · exit 0 · `set -o pipefail …` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:306
- 2026-09-24 · dc2a584* · exit 0 · `set -o pipefail …` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:368
- 2026-09-24 · dc2a584* · exit 0 · `set -o pipefail …` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:302
- 2026-09-24 · dc2a584* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:0 · test-lock-sha256:4c18709aead7951cb96df31d4c7ea84f609686ab098d1697df810e3109963dc7 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEV2ZXJ5UmVhZEZsYWdJc1RhdWdodEJ5SW5zdHJ1Y3Rpb25zCTg3ZWRhNWQxZTIzZTBhOTdjZDY4NDZiYTM5YTU4NjBhNWE2ZWRjZmJlM2JhNGE4Zjk5NWMyMDg2NGJmYmExYWYKYm9keQljbWQvbXJ3L2luc3RydWN0aW9uc190ZXN0LmdvCVRlc3RIZWxwU3VtbWFyaWVzTmFtZVRoZVJlYWRTaWRlCTAxZmU4ZmYxZTY4OTBiM2M5NGVhZDM1YzkwZDc3YjkxNTQ1OTZlOGMyNDNkNzI4MzQzNWM3YTg1ZGU2OTdhYTYKYm9keQljbWQvbXJ3L2luc3RydWN0aW9uc190ZXN0LmdvCVRlc3RJbnN0cnVjdGlvbnNDb21tYW5kUHJpbnRzQ0xJCWUzMDYwMDM4ZDRmYTQxNDZhMzg1NTY4YzZkNTJmNDM1ZmNlZmJkMjc4YjAxZDNkNjU5ZmZhZWExZThkZWJjMDAKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RDTElDb250YWluc1NoYXJlZEFuZFRoZU9wZXJhdG9yVHJhcHMJNjU2YjFmNjM1NGQ0Y2M2NDZhMzYwMWQwZGFlM2NmODJjZDc5YTBjYTBmNjEzNDc0NGUyZDg1NWY0MGNhZTc1NQpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdENMSVRlYWNoZXNBbHdheXNBbmRBUGxhbgk4NzZkOGQ1ZWVlNjBiMDg5OWYyNDg4ZmM3YThlN2MzOGM0OTM5NTFmNGU0Zjg3MTQ3MWRiMzBjNWYwZjdlNzExCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Q0xJVGVhY2hlc1RoZVJlYWRTaWRlCTE5YTBlOWU3OGI2Yjc1YTRkNmViYjRhNzY2NzJkYWU0ODg4ZTQ4MzMzODNhZmI3ZDU0NDA3ZTE2YWQ3NTJlNGQKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RFdmVyeVN1cmZhY2VDb250YWluc1RoZVNoYXJlZFNlbnRlbmNlcwljNGUzMTY5MzFlMzVmMTViZTg5Y2ZlNGVhM2E0ZjVkNTg2YjcyNzQxNDkxOTg2MGIzZGEzMTM2NzE1M2NmNzcw · test-lock-kind:replace
- 2026-09-24 · dc2a584* · exit 0 · `set -o pipefail …` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:662
- 2026-09-24 · c7ae1fa* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:0 · test-lock-sha256:f445dc5f9aac088388659d81333d72fc8ad7ba93f484d64e53bffd047f78e363 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEV2ZXJ5UmVhZEZsYWdJc1RhdWdodEJ5SW5zdHJ1Y3Rpb25zCTg3ZWRhNWQxZTIzZTBhOTdjZDY4NDZiYTM5YTU4NjBhNWE2ZWRjZmJlM2JhNGE4Zjk5NWMyMDg2NGJmYmExYWYKYm9keQljbWQvbXJ3L2luc3RydWN0aW9uc190ZXN0LmdvCVRlc3RIZWxwU3VtbWFyaWVzTmFtZVRoZVJlYWRTaWRlCTFlYmJiOTc1NjAxMTdmMWMwYzc0ZGVlYjhjOTc1OThlYzExYWI3MWFmMDI3MGE1YmJlNDc4Y2FmZDI5N2Q1MmEKYm9keQljbWQvbXJ3L2luc3RydWN0aW9uc190ZXN0LmdvCVRlc3RJbnN0cnVjdGlvbnNDb21tYW5kUHJpbnRzQ0xJCWUzMDYwMDM4ZDRmYTQxNDZhMzg1NTY4YzZkNTJmNDM1ZmNlZmJkMjc4YjAxZDNkNjU5ZmZhZWExZThkZWJjMDAKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RDTElDb250YWluc1NoYXJlZEFuZFRoZU9wZXJhdG9yVHJhcHMJNjU2YjFmNjM1NGQ0Y2M2NDZhMzYwMWQwZGFlM2NmODJjZDc5YTBjYTBmNjEzNDc0NGUyZDg1NWY0MGNhZTc1NQpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdENMSVRlYWNoZXNBbHdheXNBbmRBUGxhbgk4NzZkOGQ1ZWVlNjBiMDg5OWYyNDg4ZmM3YThlN2MzOGM0OTM5NTFmNGU0Zjg3MTQ3MWRiMzBjNWYwZjdlNzExCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Q0xJVGVhY2hlc1RoZVJlYWRTaWRlCTAxZmMwM2ExNTBjMjE3YzExMWQzZWYxNjAwMjA5YjIzZWJmNmEwZWU2NTcwOGY0YjBjYjJiMTcwMzdkMWY5ZDMKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RFdmVyeVN1cmZhY2VDb250YWluc1RoZVNoYXJlZFNlbnRlbmNlcwljNGUzMTY5MzFlMzVmMTViZTg5Y2ZlNGVhM2E0ZjVkNTg2YjcyNzQxNDkxOTg2MGIzZGEzMTM2NzE1M2NmNzcw · test-lock-kind:replace
- 2026-09-24 · c7ae1fa* · exit 0 · `set -o pipefail …` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:586
- 2026-09-24 · c7ae1fa* · exit 0 · `set -o pipefail …` · acceptance-sha256:412a4010c7419043e0681178e97445e8befea1e503c10cae6278c0c23bc0987e · ms:343
