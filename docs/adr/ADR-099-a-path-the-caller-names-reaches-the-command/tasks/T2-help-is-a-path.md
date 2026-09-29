# Task ADR-099-T2: `help` is a path to read, write and check

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `HideHelpCommand` on `read`, `write`, `check`; contract §192; the BACKLOG entry closed
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a file named help is read`, `the help flag still prints`, `mrw help still exists`, `a contract row drives the binary`

## Goal

`mrw read help` and `mrw read h` serve files named `help` and `h`; `mrw write help` applies the plan file
`help`; `mrw check help` takes `help` as a PATH; `--help` and `-h` still print each command's help, and
`mrw help` still lists the commands.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `HideHelpCommand: true` in `readCmd` (`:641`), `writeCmd` (`:935`) and `checkCmd` (`:1978`) |
| `cmd/mrw/path099_test.go` | edit | the test below |
| `scripts/contract.sh` | edit | §192 |
| `docs/adr/BACKLOG.md` | edit | the ADR-097 `mrw read help` entry marked closed |

**What selects it:** urfave's `ensureHelp` reads `HideHelpCommand` on each command when the tree is set up;
the field on each of the three commands is the whole mechanism.

## Ordered Steps

1. [S1] Write `TestAFileNamedHelpIsAPath` and confirm it RED on `7ebdf51`. [proof: mutation]
2. [S2] Set the field on the three commands. [proof: mutation] Mutants: the field removed from each command.
3. [S3] Contract §192, driving `$MRW`: `read help` serves the file's content, and `read -- h` too; `write help`, run
   in the checkout, applies the plan in the file `help`; `check help` scopes a check to it; paired,
   `read --help` prints `USAGE` and `mrw help` prints its `COMMANDS:` list.
   [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §192's rows printed — the fence greps the section, since the whole contract takes minutes]
4. [S4] Mark the BACKLOG entry closed. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAFileNamedHelpIsAPath|TestAFlagNamedForASubcommandNamesTheSubcommand|TestEverySubcommandReachesTheAgentFacingGuide|TestAUsageErrorWritesNothingToStdout' -v 2>&1 | tee "$out" \
  && missing=$(for t in TestAFileNamedHelpIsAPath TestAFlagNamedForASubcommandNamesTheSubcommand TestEverySubcommandReachesTheAgentFacingGuide TestAUsageErrorWritesNothingToStdout; do grep -qE "^--- PASS: $t \(" "$out" || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 192\. ' scripts/contract.sh \
  && grep -q 'print `read`.s help\*\* .*Closed' docs/adr/BACKLOG.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFileNamedHelpIsAPath` | `cmd/mrw/path099_test.go` | in a tree with files `help` and `h`: `read help`, `read -- help` and `read h` serve their content and exit 0; run from the checkout (`t.Chdir`), `write help` applies the plan in the file `help` (the created file exists); `check help` exits 0 and its scoped check receives `help` as `{files}`; `read --help`, `read -h`, `write --help`, `check --help` still print USAGE; `mrw help` still lists `read` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the field on three commands |
| 2 — something selects it | urfave's `ensureHelp`; the test drives `rootCommand`, §192 the built binary |
| 3 — the caller can discover it | a path reaches the command it was given to |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the BACKLOG entry from ADR-097 |

## Mutation Log
- 2026-09-29 · 7ebdf51* · mutant killed · exit 1 · `cmd/mrw/main.go` · read keeps its help subcommand: read help prints help · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · covers:a file named help is read
- 2026-09-29 · 7ebdf51* · mutant killed · exit 1 · `cmd/mrw/main.go` · write keeps its help subcommand · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · covers:a file named help is read
- 2026-09-29 · 7ebdf51* · mutant killed · exit 1 · `cmd/mrw/main.go` · check keeps its help subcommand · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · covers:a file named help is read
- 2026-09-29 · 7ebdf51* · mutant killed · exit 1 · `cmd/mrw/main.go` · HideHelp instead: the --help flag disappears too · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · covers:the help flag still prints
- 2026-09-29 · 7ebdf51* · mutant killed · exit 1 · `cmd/mrw/main.go` · the root loses its help command: mrw help stops listing the commands · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · covers:mrw help still exists
- 2026-09-29 · 29a29ef* · mutant killed · exit 1 · `cmd/mrw/main.go` · write keeps its help subcommand: the plan is not applied · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · covers:a file named help is read
- 2026-09-29 · 29a29ef* · mutant killed · exit 1 · `cmd/mrw/main.go` · check keeps its help subcommand: the scoped check never sees help · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · covers:a file named help is read

## Invariants

- `--help`, `-h` and `mrw help` unchanged; ADR-097's named refusal unchanged.

## Risks

- `HideHelpCommand` could hide more than the subcommand; the test asserts the flag still prints.

## Stop Condition

Stop and ask if `--help` stops printing on any of the three.

## Out of Scope

- `iter` and the root — the record's Out of Scope.

## Verification Log
- 2026-09-29 · 7ebdf51* · exit 1 · `set -o pipefail …` · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · ms:358 · test-lock-sha256:f8421fcda4f4bc2041f1ac07b09524257341f74ca2856e270610ae51acab0475 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGF0aDA5OV90ZXN0LmdvCVRlc3RBQ2hlY2tXaXRoRnVsbEFuZEFQYXRoSXNSZWZ1c2VkCTc1OWQ1OTA0NWViMWU1M2UwNTNhZTdhMjJhNGM5ZjYzNjE0MDI5Yjc3ODAxMGYyMWZjZjc0MjYzZjdiYTRhMTQKYm9keQljbWQvbXJ3L3BhdGgwOTlfdGVzdC5nbwlUZXN0QUZpbGVOYW1lZEhlbHBJc0FQYXRoCWVjMzgzNDY1ZTE0MWU3NWIxOTQwNTE5YmNlZGZlNjk5ZmJlMzIwYmZkZjA1YzIxZDM1NDNmNTI2Zjg4YzUxNTI
  ```
  --- last 10 line(s) of stdout (of 60 after folding 60 raw)
             such file, a Go project falls back to "go test ./..." — and the output says the
             co
  --- FAIL: TestAFileNamedHelpIsAPath (0.00s)
  === RUN   TestAUsageErrorWritesNothingToStdout
  --- PASS: TestAUsageErrorWritesNothingToStdout (0.00s)
  === RUN   TestAFlagNamedForASubcommandNamesTheSubcommand
  --- PASS: TestAFlagNamedForASubcommandNamesTheSubcommand (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.100s
  FAIL
  ```
- 2026-09-29 · 7ebdf51* · exit 0 · `set -o pipefail …` · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · ms:332
- 2026-09-29 · 7ebdf51* · exit 0 · `set -o pipefail …` · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · ms:353
- 2026-09-29 · 7ebdf51* · exit 0 · `set -o pipefail …` · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · ms:347
- 2026-09-29 · 7ebdf51* · exit 0 · `set -o pipefail …` · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · ms:333
- 2026-09-29 · human-observed · S3 observed 2026-09-29: ./scripts/contract.sh run unpiped on this branch, exit 0 'contract holds', with §192 printed: read help and read -- h serve the files; read --help still prints read's help; mrw help still lists the commands
- 2026-09-29 · human-observed · relock 2026-09-29: after the red run, TestAFileNamedHelpIsAPath's write case changed from 'the plan in the file help applied' to 'help reached write as the plan path (open help, no USAGE)', because a plan file resolves from the working directory, not --root, so the in-process run cannot open it; still red without the fix (write help printed USAGE), every other assertion kept
- 2026-09-29 · 7ebdf51* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · ms:0 · test-lock-sha256:29e03e1a20aaf30934d42e04777b60dc8402e367e9c3fa1805c8b9be01ef802b · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGF0aDA5OV90ZXN0LmdvCVRlc3RBQ2hlY2tXaXRoRnVsbEFuZEFQYXRoSXNSZWZ1c2VkCTc1OWQ1OTA0NWViMWU1M2UwNTNhZTdhMjJhNGM5ZjYzNjE0MDI5Yjc3ODAxMGYyMWZjZjc0MjYzZjdiYTRhMTQKYm9keQljbWQvbXJ3L3BhdGgwOTlfdGVzdC5nbwlUZXN0QUZpbGVOYW1lZEhlbHBJc0FQYXRoCTE0ZjMyZDI3NzBhNzU3ZjgwMDdkZGQ0OTdlMDM2MjAyNzRkNjkxZWQ5NzNkZGVkODE4M2FlMDA2OTRkNTljNmY · test-lock-kind:replace
- 2026-09-29 · 7ebdf51* · exit 0 · `set -o pipefail …` · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · ms:531
- 2026-09-29 · human-observed · relock 2026-09-29 (Codex review of #285): TestACheckWithFullAndAPathIsRefused now asserts the exact refusal and empty stdout through runSplit; TestAFileNamedHelpIsAPath now runs write from the checkout (t.Chdir) and asserts the plan applied, and asserts check help exits 0 with its scoped check receiving help as {files}. Both stronger; nothing removed
- 2026-09-29 · 29a29ef* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · ms:0 · test-lock-sha256:c0172a75b813c499bb635d5cb0ab06833ec5f5cce4910c702932b809de584979 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGF0aDA5OV90ZXN0LmdvCVRlc3RBQ2hlY2tXaXRoRnVsbEFuZEFQYXRoSXNSZWZ1c2VkCWRhMDZhYzcyODQ4ZTNmMmQ4YmNiZjM4YTEzYWFhOWU0NjExMTk2ZDU1ZmNhMDYxNmNmZGE4ODZhNTJkOTUwYzcKYm9keQljbWQvbXJ3L3BhdGgwOTlfdGVzdC5nbwlUZXN0QUZpbGVOYW1lZEhlbHBJc0FQYXRoCTIxMjVmZmUwYzBkZGYxYjRhZWE2MWJjMDI4MjllZDk5Y2EwZGNlODJiODk3Y2VjM2Q4NDRlNGFkY2ZiOWY3MWQ · test-lock-kind:replace
- 2026-09-29 · 29a29ef* · exit 0 · `set -o pipefail …` · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · ms:330
- 2026-09-29 · 29a29ef* · exit 0 · `set -o pipefail …` · acceptance-sha256:11102d50275da6a804d4cfa139983c3ef68705fdc90a3959dc234f601fbcde49 · ms:324
