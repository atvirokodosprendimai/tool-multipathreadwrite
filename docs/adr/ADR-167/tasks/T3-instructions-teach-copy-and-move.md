# Task ADR-167-T3: the instructions teach copy and move

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** a `mrw instructions` recipe; one sentence each in the plan references; contract §287
**Consumes:** T2's receipt keys and refusal, which the text names
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the guide test run`, `the recipe is printed`, `the plan references name from=`, `the contract run`, `the recipe runs as printed`, `the engine packages it does not own`

## Goal

Decisions 5 and 7: `mrw instructions` carries a recipe that copies lines without re-sending them and moves a block as a `from=` copy plus a `delete`; the plan references on every surface name `from=PATH:ADDR` and that its lines must have been read.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | a recipe after ADR-161's nine, and a sentence beside the `body=@` one (`:84` on `48be90a4`; origin/main `f862d8a3` moved the file by one line) |
| `internal/guide/fromcopy167_test.go` | add | the test |
| `internal/mcp/mcp.go` | edit | a sentence beside the `body=@` one (`:741`) |
| `AGENTS.md`, `README.md` | edit | one sentence each beside the `body=@path` text (`AGENTS.md:349`) |
| `scripts/contract.sh` | edit | §287: the recipe taken from `$MRW instructions` and run in a scratch tree |

## Ordered Steps

1. [S1] Write the failing test and see it red: TestInstructionsTeachCopyAndMove (`mrw instructions` holds a recipe whose command line carries `from=` with an address and a `delete` of the same range; `--core` stays under ADR-141's word bound; the MCP instructions and the plan reference name `from=PATH:ADDR`). [proof: acceptance]
2. [S2] The recipe and the sentences. Mutant: the recipe's `delete` names another range than its `from=`. [proof: mutation]
3. [S3] Contract §287, one row with its pair: `the copy-and-move recipe from mrw instructions moves a block and refuses without its read`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/guide/ -count=1 -timeout 300s -v -run 'TestInstructionsTeachCopyAndMove$' 2>&1) \
  && grep -qE '^--- PASS: TestInstructionsTeachCopyAndMove \(' <<<"$out" \
  && grep -qF 'from=PATH:ADDR' AGENTS.md \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  the copy-and-move recipe from mrw instructions moves a block and refuses without its read' <<<"$c" \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/seen internal/check internal/state
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestInstructionsTeachCopyAndMove` | `internal/guide/fromcopy167_test.go` | the recipe, the `--core` bound, the MCP sentence | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the recipe text |
| 2 — something selects it | `mrw instructions`, the MCP server instructions |
| 3 — the caller can discover it | it is printed where ADR-161 puts recipes first |
| 4 — it is used | nothing measures it; the bench (ADR-160) could, and no arm is proposed here |

## Invariants

- Every sentence earlier records pin in `mrw instructions` stays (ADR-161 Decision 4).
- The recipe's command line is commands only, as ADR-161's Codex review required.

## Risks

- The centralised `mrw` skill mirrors AGENTS.md: whoever merges T3 refreshes it (the record's Follow-up).

## Out of Scope

- A bench arm measuring the output-token saving (permanent: boundary: this record ships the form; ADR-160's bench owns what it measures)

## Stop Condition

Stop if the recipe cannot be written as commands only: the form is then too awkward to teach, and the record goes back to Zy.

## Mutation Log
(empty until execute)
- 2026-10-22 · f862d8a3* · mutant killed · exit 1 · `internal/guide/guide.go` · the recipe deletes another range than it copies · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · covers:the recipe is printed
- 2026-10-22 · f862d8a3* · mutant survived · exit 0 · `internal/mcp/mcp.go` · the MCP plan reference loses from=PATH:ADDR · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · covers:the plan references name from=
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-22 · f862d8a3* · mutant killed · exit 1 · `internal/guide/guide.go` · the recipe reads fewer lines than it copies · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · covers:the recipe runs as printed
- 2026-10-22 · f862d8a3* · mutant killed · exit 1 · `scripts/contract.sh` · the fixture breaks the row line counts · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · covers:the contract run
- 2026-10-22 · f862d8a3* · mutant killed · exit 1 · `internal/state/state.go` · internal/state changes · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · covers:the engine packages it does not own
- 2026-10-22 · f862d8a3* · mutant killed · exit 1 · `internal/mcp/mcp.go` · the MCP plan reference loses from=PATH:ADDR · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · covers:the plan references name from=
- 2026-10-22 · 874387ba* · mutant killed · exit 1 · `internal/guide/guide.go` · recipe 10 loses its label shape · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · covers:the guide test run

## Verification Log
(empty until execute)
- 2026-10-22 · f862d8a3* · exit 1 · `set -o pipefail …` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:616 · test-lock-sha256:86e86ca0582c2d81adfc777e8e497b828a03a6eba725f6cb77cd1ed74da8f656 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZnJvbWNvcHkxNjdfdGVzdC5nbwlUZXN0SW5zdHJ1Y3Rpb25zVGVhY2hDb3B5QW5kTW92ZQllOGFmYzE5ZTU4OWI5Y2E3OGI3ZDU0MWExNTlkMGRjODdlNGI0MGY3MzA4ZjkwNjBkZjRhOTRkOWI2YWU2NGNiCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZnJvbWNvcHkxNjdfdGVzdC5nbwlhIHJlY2lwZSBjb3BpZXMgYSByYW5nZSBhbmQgZGVsZXRlcyB0aGUgc2FtZSByYW5nZQk4ZTZiYjY2NjEzZmE3MzZjOWIzMzJlYjM0YTY5NThlNThhMjVmZTE2NjkyOTZlNzJiMzRmODIwZGRkZTBmOGZhCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZnJvbWNvcHkxNjdfdGVzdC5nbwl0aGUgTUNQIHBsYW4gZGVzY3JpcHRpb24gbmFtZXMgaXQJOGFkNjgyODZhZmMwOWUzMDQ4MGI1OGI3MDliMmE3ZWZhMmNmMzQ3ZTNkZWM1NTFiMjE4OGVjOGY5NWZhYWNiZgpib2R5CWludGVybmFsL2d1aWRlL2Zyb21jb3B5MTY3X3Rlc3QuZ28JdGhlIHBsYW4gcmVmZXJlbmNlIG5hbWVzIGZyb209UEFUSDpBRERSIGFuZCB0aGUgcmVhZCBpdCBuZWVkcwlmOWVmOTIwZTU2ZmUzMTkyODA0ZTdkMTRhZjk1NjgxOWQ1ZmNlNWUwNWQ2MmJmOGY0MDdjMTNlNjNlZDU2MDAw
  ```
  ```
- 2026-10-22 · f862d8a3* · exit 1 · `set -o pipefail …` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:150465
  ```
  ```
- 2026-10-22 · f862d8a3* · exit 1 · `set -o pipefail …` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:88938
  ```
  ```
- 2026-10-22 · f862d8a3* · exit 1 · `set -o pipefail …` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:81530
  ```
  ```
- 2026-10-22 · f862d8a3* · exit 1 · `set -o pipefail …` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:89211
  ```
  ```
- 2026-10-22 · f862d8a3* · exit 1 · `set -o pipefail …` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:90338
  ```
  ```
- 2026-10-22 · f862d8a3* · exit 0 · `set -o pipefail …` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:116263
- 2026-10-22 · f862d8a3* · exit 0 · `set -o pipefail …` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:97023
- 2026-10-22 · f862d8a3* · exit 0 · `set -o pipefail …` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:97834
- 2026-10-22 · f862d8a3* · exit 0 · `set -o pipefail …` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:94756
- 2026-10-22 · f862d8a3* · exit 0 · `set -o pipefail …` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:119800
- 2026-10-22 · f862d8a3* · exit 0 · `set -o pipefail …` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:108874
- 2026-10-22 · f862d8a3* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:0 · test-lock-sha256:a9ec71d9105a881748f70c2b24e12eef71a19967fd7853724eb9c7000806b130 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZnJvbWNvcHkxNjdfdGVzdC5nbwlUZXN0SW5zdHJ1Y3Rpb25zVGVhY2hDb3B5QW5kTW92ZQkyZTdmOTRhYzdhNWJhZDU4ZDI5YzdmZmM4N2VjYzRlNTgyMzNhNjQwOGZiNjU4MmZlZjc4OTZjOGRhM2U2NGUyCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZnJvbWNvcHkxNjdfdGVzdC5nbwlhIHJlY2lwZSBjb3BpZXMgYSByYW5nZSBhbmQgZGVsZXRlcyB0aGUgc2FtZSByYW5nZQk4ZTZiYjY2NjEzZmE3MzZjOWIzMzJlYjM0YTY5NThlNThhMjVmZTE2NjkyOTZlNzJiMzRmODIwZGRkZTBmOGZhCmJvZHkJaW50ZXJuYWwvZ3VpZGUvZnJvbWNvcHkxNjdfdGVzdC5nbwl0aGUgTUNQIHBsYW4gZGVzY3JpcHRpb24gbmFtZXMgaXQJYTU5NzU1OGJiY2Q5ZDUyOWQ2NjlhNzc5YzZlYjJlOTVjZTJjYTU1Yzc3NzE4MzgxMGU4MWM1NjYyNDI4Y2E4OApib2R5CWludGVybmFsL2d1aWRlL2Zyb21jb3B5MTY3X3Rlc3QuZ28JdGhlIHBsYW4gcmVmZXJlbmNlIG5hbWVzIGZyb209UEFUSDpBRERSIGFuZCB0aGUgcmVhZCBpdCBuZWVkcwlmOWVmOTIwZTU2ZmUzMTkyODA0ZTdkMTRhZjk1NjgxOWQ1ZmNlNWUwNWQ2MmJmOGY0MDdjMTNlNjNlZDU2MDAw · test-lock-kind:replace
- 2026-10-22 · 874387ba* · exit 0 · `set -o pipefail …` · acceptance-sha256:19970ee54819fee9a7345fbe8359dc7eea77e88b689689343565223b9aee1c6c · ms:85246
