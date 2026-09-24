# Task ADR-067-T3: a caller's argument mistake is a tool execution error; contract §124

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `isError` results for the seven argument-mistake sites
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `each argument mistake is an isError result with its sentence`, `a malformed call is still a protocol error`, `a non-object arguments is refused before dispatch`, `the binary answers format=git with an isError result`, `no engine file changes`

## Goal

The seven sites that return a JSON-RPC `-32602` for a caller's argument mistake return `errorResult` with the same sentence (SEP-1303, 2025-11-25). The sites are `internal/mcp/tools.go:204`, `:218`, `:485`, `:488`, `:496`, `:517` and `:519`. Params that do not decode (`:117`), an unknown tool (`:133`) and the too-small-ceiling write refusal (`:580`) stay JSON-RPC errors. `callTool` first refuses an `arguments` that is present but not an object (a number, string, array or `null`) with `-32602`: that is a malformed `CallToolRequest`, and it used to reach `:204`/`:485`, which this task moves. Found by the cold review of ADR-067.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | the seven returns; the `arguments`-is-an-object check in `callTool` |
| `internal/mcp/argerror_test.go` | new | the two tests below |
| `internal/mcp/writeformat_test.go` | edit | `TestWriteFormatGitIsRefused` (`:89-112`) expects an `isError` result naming the grammar, not an `error` |
| `internal/mcp/astgrep_test.go` | edit | only if its "needs at least one spec" read changes shape |
| `scripts/contract.sh` | edit | §83's `format=git` row (`:5082-5088`) follows; §124 |

## Ordered Steps

1. [S1] Write the tests and confirm each RED on `main` on an assertion. [proof: mutation]
   - `TestAnArgumentErrorIsAToolExecutionError`: table-driven over the seven mistakes:
     - `mrw_read` with an object field of the wrong type (`{"specs": 42}`);
     - `mrw_read` with no spec and no grep;
     - `mrw_write` with an object field of the wrong type (`{"plan": 7}`);
     - `echo_pad: -1`;
     - an empty plan;
     - `format: "git"`;
     - `format: "nope"`.
     Each answer has a `result` with `isError: true`, no `error`, and `content[0]` containing the old sentence.
   A guard that holds today and must keep holding, proved by mutation rather than a red run: `TestAMalformedCallIsStillAProtocolError`. `params` a string; an unknown tool name; and `arguments` as `42`, `"x"`, `[]` and `null` for both tools. Each answer is an `error` with `-32602`.
2. [S2] Implement; confirm GREEN, and update `writeformat_test.go:95`. [proof: mutation]
   Mutants:
   - one site reverted to `rpcError` (`:488`): kills the table;
   - the unknown-tool site made `isError`: kills the malformed test;
   - the `arguments`-is-an-object check removed: kills the malformed test (`arguments: 42` would reach `:204` as a tool error).
3. [S3] §124 through the binary: `mrw_write` with `echo_pad: -1` answers a `result` with `isError: true` naming `echo_pad`, and `params: "x"` answers an `error`. §83's `format=git` row asserts `isError` and the grammar sentence. RED against v1.23.0 in a mini-harness, GREEN in the full `./scripts/contract.sh`. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 124\. ' scripts/contract.sh \
  && go test ./internal/mcp/ -count=1 -v \
    -run 'TestAnArgumentErrorIsAToolExecutionError|TestAMalformedCallIsStillAProtocolError|TestWriteFormatGitIsRefused' 2>&1 | tee /tmp/adr067-t3.out \
  && grep -q '^--- PASS: TestAnArgumentErrorIsAToolExecutionError ' /tmp/adr067-t3.out \
  && grep -q '^--- PASS: TestAMalformedCallIsStillAProtocolError ' /tmp/adr067-t3.out \
  && grep -q '^--- PASS: TestWriteFormatGitIsRefused ' /tmp/adr067-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr067-t3.out \
  && ./scripts/contract.sh > /tmp/adr067-t3-contract.out 2>&1 \
  && grep -q '^  PASS  an argument mistake is a tool result the model reads' /tmp/adr067-t3-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines cmd/mrw \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines cmd/mrw)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l internal/mcp)" ] \
  && go vet ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnArgumentErrorIsAToolExecutionError` | `internal/mcp/argerror_test.go` | all seven mistakes are `isError` results with their sentence | — | S1, S2 |
| `TestAMalformedCallIsStillAProtocolError` | `internal/mcp/argerror_test.go` | guard: malformed params, a non-object `arguments` and an unknown tool stay `-32602` | — | S1, S2 |
| `TestWriteFormatGitIsRefused` | `internal/mcp/writeformat_test.go` | the git refusal still names the grammar, now as an `isError` result | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests and §124 |
| 2 — something selects it | `callTool` routes every answer through `withinCeiling`, `isError` results included |
| 3 — the caller can discover it | the sentence is in `content[0]`, which a host hands the model |
| 4 — it is used | SEP-1303 (2025-11-25) says clients SHOULD give tool execution errors to the model |

## Verification Log
(empty until execute)
- 2026-09-24 · ae040e4* · exit 1 · `set -o pipefail …` · acceptance-sha256:bbe6ed194dbfc4ce4fd87fc3dbd3ec75c68894de8702ac73ce4a618bc4d68d0f · ms:161 · test-lock-sha256:ea00d937db3344006e84947091d9d3adeb4c160d0316316deb9b3bfacfd49ad1 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9hcmdlcnJvcl90ZXN0LmdvCVRlc3RBTWFsZm9ybWVkQ2FsbElzU3RpbGxBUHJvdG9jb2xFcnJvcgk0NGRkMGM2ZTk1MmQ2ZDY3ZDJhOGFlMGE5ZGI4ZTczODllMzM0MzBlNjE3YmJmODlkOGM1ZWI0MDZlNjY3ZWIwCmJvZHkJaW50ZXJuYWwvbWNwL2FyZ2Vycm9yX3Rlc3QuZ28JVGVzdEFuQXJndW1lbnRFcnJvcklzQVRvb2xFeGVjdXRpb25FcnJvcgk0NTY5ZmZlNzc4YzMyMDEwZTVjMmJjYTVjNTgzZTcwMTMzMmNkYjJmZTlhNTQ4OTg0YTM5YTY3N2Y2ZTAxZWFjCmJvZHkJaW50ZXJuYWwvbWNwL3dyaXRlZm9ybWF0X3Rlc3QuZ28JVGVzdEFTZWFyY2hSZXBsYWNlQmxvYk9uV3JpdGVQbGFuSXNBQmFkTmF0aXZlUGxhbgkyZWU4NjQwOWUxMjIxNzU0OTAxNzNlYzAzZGNhMDAzMzRkMzYzM2Q0MzUwMDVlZWJjMmZjMjcxOWJhZTg0YzZmCmJvZHkJaW50ZXJuYWwvbWNwL3dyaXRlZm9ybWF0X3Rlc3QuZ28JVGVzdEFuQXBwbHlQYXRjaEJsb2JPbldyaXRlUGxhbklzQUJhZE5hdGl2ZVBsYW4JMTNmM2Q3MjFiZDg3ZGE5ZTRhZmU0MzU2MTllNmFlYmI4ZTU2ZTUyN2Y1NWRiMTE5NWQwODFiOTdlZTY1MmZjOApib2R5CWludGVybmFsL21jcC93cml0ZWZvcm1hdF90ZXN0LmdvCVRlc3RXcml0ZURlY2xhcmVzRm9ybWF0T25UaGVFeGlzdGluZ1Rvb2wJYzIzN2U3M2E3NDlmYTRkYmM3Njk3YjNjYmRmMmQ5ODFmZTNlNzMxYzM5NTNjNzc1YzhhOTVlNWYzYzA0MTc3NQpib2R5CWludGVybmFsL21jcC93cml0ZWZvcm1hdF90ZXN0LmdvCVRlc3RXcml0ZUZvcm1hdEFwcGx5UGF0Y2hVbnJlYWRXcml0ZXNOb3RoaW5nCWQ2NjBmODY3Njk3Mzg0MmFiYjRmNWExMjczN2Y2M2JhMDY2MWM5OTFlMWVhNjc4ZjcxYTBiM2RhZjkyNzQyNzMKYm9keQlpbnRlcm5hbC9tY3Avd3JpdGVmb3JtYXRfdGVzdC5nbwlUZXN0V3JpdGVGb3JtYXRHaXRJc1JlZnVzZWQJYWQ5OWVjOWE3NjUyZjc5NjhmZTk1ZDhiYWZmODkxYjY1YWQ1MThjNTlmYzc0MjU1OTY5NjgwODRiZmZlZGEwNQpib2R5CWludGVybmFsL21jcC93cml0ZWZvcm1hdF90ZXN0LmdvCVRlc3RXcml0ZUZvcm1hdFNlYXJjaFJlcGxhY2VVbnJlYWRXcml0ZXNOb3RoaW5nCWFlMTI2YmE4NDIxZTJiNmYyNjNmMGQ3NWNjZjFmOTg0Y2NmOGIzODRlOTQ5NzVmZmNjZTI3MTUzYTA4NDg3MTE
  ```
  ```
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:bbe6ed194dbfc4ce4fd87fc3dbd3ec75c68894de8702ac73ce4a618bc4d68d0f · ms:45211
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:bbe6ed194dbfc4ce4fd87fc3dbd3ec75c68894de8702ac73ce4a618bc4d68d0f · ms:45538
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:bbe6ed194dbfc4ce4fd87fc3dbd3ec75c68894de8702ac73ce4a618bc4d68d0f · ms:35873
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:bbe6ed194dbfc4ce4fd87fc3dbd3ec75c68894de8702ac73ce4a618bc4d68d0f · ms:34463
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:bbe6ed194dbfc4ce4fd87fc3dbd3ec75c68894de8702ac73ce4a618bc4d68d0f · ms:45864
- 2026-09-24 · ae040e4* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:bbe6ed194dbfc4ce4fd87fc3dbd3ec75c68894de8702ac73ce4a618bc4d68d0f · ms:0 · test-lock-sha256:9d42b64a459f84b8498d03ce70e266382bea6f9cd1938d6840a3ab46886113e2 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9hcmdlcnJvcl90ZXN0LmdvCVRlc3RBTWFsZm9ybWVkQ2FsbElzU3RpbGxBUHJvdG9jb2xFcnJvcgk0NGRkMGM2ZTk1MmQ2ZDY3ZDJhOGFlMGE5ZGI4ZTczODllMzM0MzBlNjE3YmJmODlkOGM1ZWI0MDZlNjY3ZWIwCmJvZHkJaW50ZXJuYWwvbWNwL2FyZ2Vycm9yX3Rlc3QuZ28JVGVzdEFuQXJndW1lbnRFcnJvcklzQVRvb2xFeGVjdXRpb25FcnJvcgk0NTY5ZmZlNzc4YzMyMDEwZTVjMmJjYTVjNTgzZTcwMTMzMmNkYjJmZTlhNTQ4OTg0YTM5YTY3N2Y2ZTAxZWFjCmJvZHkJaW50ZXJuYWwvbWNwL3dyaXRlZm9ybWF0X3Rlc3QuZ28JVGVzdEFTZWFyY2hSZXBsYWNlQmxvYk9uV3JpdGVQbGFuSXNBQmFkTmF0aXZlUGxhbgkyZWU4NjQwOWUxMjIxNzU0OTAxNzNlYzAzZGNhMDAzMzRkMzYzM2Q0MzUwMDVlZWJjMmZjMjcxOWJhZTg0YzZmCmJvZHkJaW50ZXJuYWwvbWNwL3dyaXRlZm9ybWF0X3Rlc3QuZ28JVGVzdEFuQXBwbHlQYXRjaEJsb2JPbldyaXRlUGxhbklzQUJhZE5hdGl2ZVBsYW4JMTNmM2Q3MjFiZDg3ZGE5ZTRhZmU0MzU2MTllNmFlYmI4ZTU2ZTUyN2Y1NWRiMTE5NWQwODFiOTdlZTY1MmZjOApib2R5CWludGVybmFsL21jcC93cml0ZWZvcm1hdF90ZXN0LmdvCVRlc3RXcml0ZURlY2xhcmVzRm9ybWF0T25UaGVFeGlzdGluZ1Rvb2wJYzIzN2U3M2E3NDlmYTRkYmM3Njk3YjNjYmRmMmQ5ODFmZTNlNzMxYzM5NTNjNzc1YzhhOTVlNWYzYzA0MTc3NQpib2R5CWludGVybmFsL21jcC93cml0ZWZvcm1hdF90ZXN0LmdvCVRlc3RXcml0ZUZvcm1hdEFwcGx5UGF0Y2hVbnJlYWRXcml0ZXNOb3RoaW5nCWQ2NjBmODY3Njk3Mzg0MmFiYjRmNWExMjczN2Y2M2JhMDY2MWM5OTFlMWVhNjc4ZjcxYTBiM2RhZjkyNzQyNzMKYm9keQlpbnRlcm5hbC9tY3Avd3JpdGVmb3JtYXRfdGVzdC5nbwlUZXN0V3JpdGVGb3JtYXRHaXRJc1JlZnVzZWQJNjc5MGM5Zjg4MjBmM2JiOTdhNGUwN2NjMmE0Y2U0YTRhNTlmZDUyYzNkYjg1NDVhYjJjZjNlMjlhYmRkMTc4Ngpib2R5CWludGVybmFsL21jcC93cml0ZWZvcm1hdF90ZXN0LmdvCVRlc3RXcml0ZUZvcm1hdFNlYXJjaFJlcGxhY2VVbnJlYWRXcml0ZXNOb3RoaW5nCWFlMTI2YmE4NDIxZTJiNmYyNjNmMGQ3NWNjZjFmOTg0Y2NmOGIzODRlOTQ5NzVmZmNjZTI3MTUzYTA4NDg3MTE · test-lock-kind:replace

## Mutation Log
(empty until execute)
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/tools.go` · echo_pad reverts to a JSON-RPC -32602 · acceptance-sha256:bbe6ed194dbfc4ce4fd87fc3dbd3ec75c68894de8702ac73ce4a618bc4d68d0f · covers:each argument mistake is an isError result with its sentence
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/tools.go` · an unknown tool becomes a tool error · acceptance-sha256:bbe6ed194dbfc4ce4fd87fc3dbd3ec75c68894de8702ac73ce4a618bc4d68d0f · covers:a malformed call is still a protocol error
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/tools.go` · the arguments-is-an-object check never fires, so arguments:42 reaches the decoder · acceptance-sha256:bbe6ed194dbfc4ce4fd87fc3dbd3ec75c68894de8702ac73ce4a618bc4d68d0f · covers:a non-object arguments is refused before dispatch
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/tools.go` · format=git reverts to a JSON-RPC error: §83 and TestWriteFormatGitIsRefused must go red · acceptance-sha256:bbe6ed194dbfc4ce4fd87fc3dbd3ec75c68894de8702ac73ce4a618bc4d68d0f · covers:the binary answers format=git with an isError result
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/lines/lines.go` · a behaviour-neutral engine edit: only the fence engine guard can see it · acceptance-sha256:bbe6ed194dbfc4ce4fd87fc3dbd3ec75c68894de8702ac73ce4a618bc4d68d0f · covers:no engine file changes

## Invariants

- Every sentence is unchanged; only its envelope moves.
- A refusal applies nothing. Acknowledgements sent with it are still promoted, as today: `promote` runs before validation (`tools.go:208`, `:492`), and this task does not reorder it.

## Risks

- A test elsewhere reads `resp["error"]` for one of these mistakes; S2 greps each sentence in `internal/mcp/*_test.go` and `scripts/contract.sh`.

## Out of Scope

- The `-32602` on the too-small-ceiling write refusal at `tools.go:580` (permanent: boundary: ADR-032 keeps it a JSON-RPC error, and its fix is a server flag, not an argument the model can change)

## Stop Condition

Stop if a host measured on this machine hides `isError` results from the model.
