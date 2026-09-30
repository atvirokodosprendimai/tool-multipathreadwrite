# Task ADR-102-T4: the unreportable message says not to re-run

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the new `unreportableAt` text
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the message never advises a re-run`, `the floor still bounds the message`

## Goal

`unreportableAt` says the write happened and not to re-send the plan (it would apply again), to read the files
to see what changed, and — for the detail on later writes — to restart the server with a larger
`--max-result-chars`. `floorAt` measures the same function, so the floor rises with the sentence; the smallest
accepted ceiling rises with it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `unreportableAt` (`:909-918`) |
| `internal/mcp/landed102_test.go` | edit | the test below |

## Ordered Steps

1. [S1] Write the failing test `TestAnUnreportableWriteSaysNotToRerun`; confirm RED. [proof: mutation]
2. [S2] Change the text; `TestTheWriteFloorIsAFloor` stays green. [proof: mutation] Mutant: the old advice restored.

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestAnUnreportableWriteSaysNotToRerun|TestTheWriteFloorIsAFloor' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnUnreportableWriteSaysNotToRerun \(' "$out" \
  && grep -qE '^--- PASS: TestTheWriteFloorIsAFloor \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnUnreportableWriteSaysNotToRerun` | `internal/mcp/landed102_test.go` | for a complete and a partial write the message says the write happened and "do not re-send", and never "re-run with" | — | S1, S2 |
| `TestTheWriteFloorIsAFloor` | `internal/mcp/limit_test.go` | (existing) the floor still bounds the message at ten ceilings | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every `mrw write` and `mrw_write` goes through `writer.Apply` and the tallies; the tests drive `rootCommand` or `Serve` |
| 3 — the caller can discover it | the receipt, `stats`, and the ledger's licence on the next write |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 Codex design review |

## Mutation Log
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/mcp/tools.go` · the old re-run advice restored · acceptance-sha256:7e2dc87bccf676be6b021770891ce1af7473edf869c0eb3bbfe707ff765e01a7 · covers:the message never advises a re-run

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
- 2026-09-30 · f1d5996* · exit 1 · `set -o pipefail …` · acceptance-sha256:7e2dc87bccf676be6b021770891ce1af7473edf869c0eb3bbfe707ff765e01a7 · ms:330 · test-lock-sha256:f51d36a3e660552ae0a5309ec965fa842c08b88b6d35c19632f25772979f0e60 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9sYW5kZWQxMDJfdGVzdC5nbwlUZXN0QUxlZGdlckZhaWx1cmVTdGlsbFNlbmRzVGhlTUNQUmVjZWlwdAk3Njc5NTA5Mzg5ZmNhMDE5NDAyYzZhM2RkM2NkMzFlOTJhMWZmZmFjMzlkOTYyMmUzY2FkZmZkYmJjODY5YjczCmJvZHkJaW50ZXJuYWwvbWNwL2xhbmRlZDEwMl90ZXN0LmdvCVRlc3RBbk1DUFBhcnRpYWxDb21taXRJc0NvdW50ZWRBc1BhcnRpYWxseUFwcGxpZWQJZWI4OTY4ZmMyZGNlZDk3ZTBjZDFhZDg3N2JjNDUyOTBkMjg3MmYxNWMwNTFhM2NjZGE1OTYxN2RmYmZmNzU0ZQpib2R5CWludGVybmFsL21jcC9sYW5kZWQxMDJfdGVzdC5nbwlUZXN0QW5VbnJlcG9ydGFibGVXcml0ZVNheXNOb3RUb1JlcnVuCTllOWQ2NDM0ZjgyMzgzM2M1NWY3ZTdhYjY0YjRjODJmZTY1ZjU3OWQxZmUwNTIxM2FkYWE5YWIxYzI3ZmUwNzUKdW5wcm92ZW4JaW50ZXJuYWwvbWNwLwlUZXN0VGhlV3JpdGVGbG9vcklzQUZsb29y
  ```
  --- last 10 line(s) of stdout (of 29 after folding 29 raw)
      --- PASS: TestTheWriteFloorIsAFloor/budget-300 (0.00s)
      --- PASS: TestTheWriteFloorIsAFloor/budget-440 (0.00s)
      --- PASS: TestTheWriteFloorIsAFloor/budget-445 (0.00s)
      --- PASS: TestTheWriteFloorIsAFloor/budget-450 (0.00s)
      --- PASS: TestTheWriteFloorIsAFloor/budget-500 (0.00s)
      --- PASS: TestTheWriteFloorIsAFloor/budget-5000 (0.00s)
      --- PASS: TestTheWriteFloorIsAFloor/budget-200000 (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.150s
  FAIL
  ```
- 2026-09-30 · f1d5996* · exit 1 · `set -o pipefail …` · acceptance-sha256:7e2dc87bccf676be6b021770891ce1af7473edf869c0eb3bbfe707ff765e01a7 · ms:292
  ```
  --- last 10 line(s) of stdout (of 29 after folding 29 raw)
      --- PASS: TestTheWriteFloorIsAFloor/budget-300 (0.00s)
      --- PASS: TestTheWriteFloorIsAFloor/budget-440 (0.00s)
      --- PASS: TestTheWriteFloorIsAFloor/budget-445 (0.00s)
      --- PASS: TestTheWriteFloorIsAFloor/budget-450 (0.00s)
      --- PASS: TestTheWriteFloorIsAFloor/budget-500 (0.00s)
      --- PASS: TestTheWriteFloorIsAFloor/budget-5000 (0.00s)
      --- PASS: TestTheWriteFloorIsAFloor/budget-200000 (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.113s
  FAIL
  ```
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:7e2dc87bccf676be6b021770891ce1af7473edf869c0eb3bbfe707ff765e01a7 · ms:387
- 2026-09-30 · human-observed · relock 2026-09-30: the Tests table named internal/mcp/ as the file of TestTheWriteFloorIsAFloor, which cannot be hashed; it is internal/mcp/limit_test.go. No test body changed
- 2026-09-30 · f1d5996* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:7e2dc87bccf676be6b021770891ce1af7473edf869c0eb3bbfe707ff765e01a7 · ms:0 · test-lock-sha256:65b96e4c90ecf4932603881c438077ef5b2fdddbe4297d4bac17a67d61270185 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9sYW5kZWQxMDJfdGVzdC5nbwlUZXN0QUxlZGdlckZhaWx1cmVTdGlsbFNlbmRzVGhlTUNQUmVjZWlwdAk3Njc5NTA5Mzg5ZmNhMDE5NDAyYzZhM2RkM2NkMzFlOTJhMWZmZmFjMzlkOTYyMmUzY2FkZmZkYmJjODY5YjczCmJvZHkJaW50ZXJuYWwvbWNwL2xhbmRlZDEwMl90ZXN0LmdvCVRlc3RBbk1DUFBhcnRpYWxDb21taXRJc0NvdW50ZWRBc1BhcnRpYWxseUFwcGxpZWQJZWI4OTY4ZmMyZGNlZDk3ZTBjZDFhZDg3N2JjNDUyOTBkMjg3MmYxNWMwNTFhM2NjZGE1OTYxN2RmYmZmNzU0ZQpib2R5CWludGVybmFsL21jcC9sYW5kZWQxMDJfdGVzdC5nbwlUZXN0QW5VbnJlcG9ydGFibGVXcml0ZVNheXNOb3RUb1JlcnVuCTllOWQ2NDM0ZjgyMzgzM2M1NWY3ZTdhYjY0YjRjODJmZTY1ZjU3OWQxZmUwNTIxM2FkYWE5YWIxYzI3ZmUwNzUKYm9keQlpbnRlcm5hbC9tY3AvbGltaXRfdGVzdC5nbwlUZXN0QVBhcnRpYWxBcHBsaWNhdGlvbklzTm90UmVwb3J0ZWRBc05vdGhpbmdXcml0dGVuCWU4ZGNhNzFlYWQ4MGVmNWU5NmQyYjE2NjkwMDc0YzllMzdkNDFkMmZkMDM2OGJjNTU5YzgxZTFjZjQ1OTVmNjMKYm9keQlpbnRlcm5hbC9tY3AvbGltaXRfdGVzdC5nbwlUZXN0QVJlYWRXaG9zZVJlY2VpcHRPdmVyZmxvd3NJc05vdFNlcnZlZAlkNDhiMWFkMTI3NDkwMzM2NmY0NmFmZWM4YTY5ZDViZWViNDQzY2IzM2FhNDBlYWI1YzYxNTkxMGRkNTEzMzcwCmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdEFTbWFsbENlaWxpbmdSZWZ1c2VzVGhlV3JpdGVCZWZvcmVBcHBseWluZwkyNGY2OTc4OWZjZmZjYjliNDIwMzYwMjc2NDYyZDQ1YzQyOGQ5YmRjMjE2ZGMzYjUzZjI1MzcwZWE0NDM3YzE4CmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdEFXcml0ZVJlY2VpcHRFbGlkZXNTdWNjZXNzZXNOb3RGYWlsdXJlcwk2Njk0MzMwNWRlNzQ5OGU1ZjdmMGNmYTVhN2UzMDE3MzRjMWQwNjdjNWVhN2Q5OGQzYTRmMzU3OGNhNDA3NjY1CmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdEV2ZXJ5QW5zd2VyRml0c0luY2x1ZGluZ1RoZVJlZnVzYWxzCWM3NmE3ODI4NDA2MzljOGUxMmU4YTA5NDBiZGIzNmE2YzYwYmYzYmEzMzkxNjhlNGMxYjdiNzZkN2FiMDZlZjcKYm9keQlpbnRlcm5hbC9tY3AvbGltaXRfdGVzdC5nbwlUZXN0VGhlQWR2ZXJ0aXNlZENlaWxpbmdCb3VuZHNFdmVyeUFuc3dlcgliNzJhOTQzNmRmZmI1MzkyNzU1OWJjOGI1MDhmZmJmZDUxNGM5MjRhYmQyNzcwMjc5ZTA5NDY0ZjNhY2QzYjBmCmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdFRoZUNlaWxpbmdOZXZlclNocmlua3NBU2VydmVkUmVhZAk1Y2ExYjFhNjNjYjU3Zjg4MWViYmJkZDRkODhhMmYyOWE0MzNkZTFkMmI5YTlhNzYwYzZmOWVjMjczMDE0ZGUzCmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdFRoZVNlY29uZFN0YWdlRWxpc2lvbkRyb3BzRmlsZVJlY29yZHMJMWJlODVkODI0OTY1NmQ2MDMwNmVkYjFjMzYxMmRiYzUwMmFlODYzMGQxODI3MDE0MzY4NTdlNGZmYmVkZTAxOQpib2R5CWludGVybmFsL21jcC9saW1pdF90ZXN0LmdvCVRlc3RUaGVTZWNvbmRTdGFnZU5ldmVyRWxpZGVzQVdyaXR0ZW5GaWxlCTE4ZGNkOGViNjNkOTgzNDQ5YWZlZWRmNzc5MmUzMWU3NDhmY2I0MWQ4ZWQ2YmRiNmVkODAxYjYzNjM4MmQ0ZGMKYm9keQlpbnRlcm5hbC9tY3AvbGltaXRfdGVzdC5nbwlUZXN0VGhlV3JpdGVGbG9vcklzQUZsb29yCWYxYWMzNmI2ODI5ZjE4OGE5MzllOWFhNTM2OTIyZDRhYzdhOWUwMWI0ZjNmYTQ1NzIzZjg1M2ZlZTI1YWYzOTA · test-lock-kind:replace
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:7e2dc87bccf676be6b021770891ce1af7473edf869c0eb3bbfe707ff765e01a7 · ms:535
- 2026-09-30 · human-observed · signed off 2026-09-30: T4 observed complete — the fence passed after the relock that pointed the Tests table at internal/mcp/limit_test.go, and its mutant (the old re-run advice restored) is killed
- 2026-09-30 · human-observed · relock 2026-09-30 (Codex review of #293): T4's lock spans internal/mcp/landed102_test.go, where the partial-commit and ledger tests gained ring and pricing assertions; TestAnUnreportableWriteSaysNotToRerun is unchanged; approved
- 2026-09-30 · 23beaf7* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:7e2dc87bccf676be6b021770891ce1af7473edf869c0eb3bbfe707ff765e01a7 · ms:0 · test-lock-sha256:80765798f04e2f82ffd70bd11148601b06fe1ad9755737ddeba7a10dab8b5218 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9sYW5kZWQxMDJfdGVzdC5nbwlUZXN0QUxlZGdlckZhaWx1cmVTdGlsbFNlbmRzVGhlTUNQUmVjZWlwdAkwZDU4NmVmZTJhOTQ3NWRkOTdkNWJkMTVhM2EzZDI5ZDM0ZDI0MGRiNTgxYWRjYTBlYzllMjE1Y2RmMTFlMGZhCmJvZHkJaW50ZXJuYWwvbWNwL2xhbmRlZDEwMl90ZXN0LmdvCVRlc3RBbk1DUFBhcnRpYWxDb21taXRJc0NvdW50ZWRBc1BhcnRpYWxseUFwcGxpZWQJNTUzNDA4ZDRmYTdkZjc4N2ViYTE1M2NiY2M5Y2RkN2YzMGRhY2JlNzk1ODIwZWRkZjVkOTdhOWU2OWI0ZjJhMQpib2R5CWludGVybmFsL21jcC9sYW5kZWQxMDJfdGVzdC5nbwlUZXN0QW5VbnJlcG9ydGFibGVXcml0ZVNheXNOb3RUb1JlcnVuCTllOWQ2NDM0ZjgyMzgzM2M1NWY3ZTdhYjY0YjRjODJmZTY1ZjU3OWQxZmUwNTIxM2FkYWE5YWIxYzI3ZmUwNzUKYm9keQlpbnRlcm5hbC9tY3AvbGltaXRfdGVzdC5nbwlUZXN0QVBhcnRpYWxBcHBsaWNhdGlvbklzTm90UmVwb3J0ZWRBc05vdGhpbmdXcml0dGVuCWU4ZGNhNzFlYWQ4MGVmNWU5NmQyYjE2NjkwMDc0YzllMzdkNDFkMmZkMDM2OGJjNTU5YzgxZTFjZjQ1OTVmNjMKYm9keQlpbnRlcm5hbC9tY3AvbGltaXRfdGVzdC5nbwlUZXN0QVJlYWRXaG9zZVJlY2VpcHRPdmVyZmxvd3NJc05vdFNlcnZlZAlkNDhiMWFkMTI3NDkwMzM2NmY0NmFmZWM4YTY5ZDViZWViNDQzY2IzM2FhNDBlYWI1YzYxNTkxMGRkNTEzMzcwCmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdEFTbWFsbENlaWxpbmdSZWZ1c2VzVGhlV3JpdGVCZWZvcmVBcHBseWluZwkyNGY2OTc4OWZjZmZjYjliNDIwMzYwMjc2NDYyZDQ1YzQyOGQ5YmRjMjE2ZGMzYjUzZjI1MzcwZWE0NDM3YzE4CmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdEFXcml0ZVJlY2VpcHRFbGlkZXNTdWNjZXNzZXNOb3RGYWlsdXJlcwk2Njk0MzMwNWRlNzQ5OGU1ZjdmMGNmYTVhN2UzMDE3MzRjMWQwNjdjNWVhN2Q5OGQzYTRmMzU3OGNhNDA3NjY1CmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdEV2ZXJ5QW5zd2VyRml0c0luY2x1ZGluZ1RoZVJlZnVzYWxzCWM3NmE3ODI4NDA2MzljOGUxMmU4YTA5NDBiZGIzNmE2YzYwYmYzYmEzMzkxNjhlNGMxYjdiNzZkN2FiMDZlZjcKYm9keQlpbnRlcm5hbC9tY3AvbGltaXRfdGVzdC5nbwlUZXN0VGhlQWR2ZXJ0aXNlZENlaWxpbmdCb3VuZHNFdmVyeUFuc3dlcgliNzJhOTQzNmRmZmI1MzkyNzU1OWJjOGI1MDhmZmJmZDUxNGM5MjRhYmQyNzcwMjc5ZTA5NDY0ZjNhY2QzYjBmCmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdFRoZUNlaWxpbmdOZXZlclNocmlua3NBU2VydmVkUmVhZAk1Y2ExYjFhNjNjYjU3Zjg4MWViYmJkZDRkODhhMmYyOWE0MzNkZTFkMmI5YTlhNzYwYzZmOWVjMjczMDE0ZGUzCmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdFRoZVNlY29uZFN0YWdlRWxpc2lvbkRyb3BzRmlsZVJlY29yZHMJMWJlODVkODI0OTY1NmQ2MDMwNmVkYjFjMzYxMmRiYzUwMmFlODYzMGQxODI3MDE0MzY4NTdlNGZmYmVkZTAxOQpib2R5CWludGVybmFsL21jcC9saW1pdF90ZXN0LmdvCVRlc3RUaGVTZWNvbmRTdGFnZU5ldmVyRWxpZGVzQVdyaXR0ZW5GaWxlCTE4ZGNkOGViNjNkOTgzNDQ5YWZlZWRmNzc5MmUzMWU3NDhmY2I0MWQ4ZWQ2YmRiNmVkODAxYjYzNjM4MmQ0ZGMKYm9keQlpbnRlcm5hbC9tY3AvbGltaXRfdGVzdC5nbwlUZXN0VGhlV3JpdGVGbG9vcklzQUZsb29yCWYxYWMzNmI2ODI5ZjE4OGE5MzllOWFhNTM2OTIyZDRhYzdhOWUwMWI0ZjNmYTQ1NzIzZjg1M2ZlZTI1YWYzOTA · test-lock-kind:replace
- 2026-09-30 · 23beaf7* · exit 0 · `set -o pipefail …` · acceptance-sha256:7e2dc87bccf676be6b021770891ce1af7473edf869c0eb3bbfe707ff765e01a7 · ms:406
