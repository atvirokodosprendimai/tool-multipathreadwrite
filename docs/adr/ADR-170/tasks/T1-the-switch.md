# Task ADR-170-T1: `mrw mcp` serves no tools until `--enable` or `MRW_MCP=on`

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `mrw mcp --enable`, `MRW_MCP`, the off answer in both eras, contract §295, the harnesses that start `mrw mcp` enabled
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the switch tests run`, `the switch test passed`, `the legacy golden holds`, `the --json receipt still matches mrw_read's`, `the contract run`, `section 295 exists`, `the off row passes`, `the on row passes`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decisions 1 and 2: without the switch, `mrw mcp` validates its startup as today, answers `initialize`, `server/discover` and `ping` in both eras with off instructions, keeps `capabilities.tools`, returns an empty `tools/list`, and refuses `tools/call` of any tool, naming the switch, with no effect; with the switch it serves byte-for-byte as before. The switch follows Decision 2's table. Every harness that starts `mrw mcp` to exercise the tools starts it enabled.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | the `--enable` flag on `mrw mcp` — the line that selects it — and `MRW_MCP`, resolved once by Decision 2's table; the usage text |
| `internal/mcp/mcp.go` | edit | the off answer: empty `tools/list`, the `tools/call` refusal, off instructions in `initialize` and `server/discover` |
| `internal/mcp/instructions.go` | edit | the off instructions' text |
| `cmd/mrw/mcpoff170_test.go` | add | the test below |
| `scripts/contract.sh` | edit | §295; every existing row that drives `mrw mcp` passes `--enable` |
| `scripts/break-campaign.sh` | edit | its MCP probes (`:92-96`) start `mrw mcp --enable` |
| `scripts/chaos.py` | edit | its MCP processes (`:742`, `:847`) start `mrw mcp --enable` |
| Go tests that start the built binary's `mcp` | edit | pass `--enable` (e.g. `cmd/mrw/readjson165_test.go`) |

## Ordered Steps

1. [S1] Write `TestMrwMcpServesNoToolsUntilEnabled` (the built binary over stdio, `MRW_MCP` cleared in its environment) and confirm RED. Cases:
   - Off, legacy era: `initialize` instructions name `--enable` and `MRW_MCP=on` and keep `capabilities.tools`; `tools/list` is an empty JSON array; `tools/call` of `mrw_read` and of `mrw_write` (a plan that would create a file) are each a JSON-RPC error naming the switch, and afterwards the file does not exist and the ledger is unchanged; `ping` answers.
   - Off, modern era: `server/discover` carries the off instructions; `tools/list` is empty.
   - Decision 2's table, every row: `--enable`, `--enable=true`, `--enable=false` with `MRW_MCP=on`, `MRW_MCP=on`, `off`, empty, unset, and `yes` (off, and stderr names it); `--enable` beside `MRW_MCP=yes` names nothing.
   - On: both tools listed, a read answers, the answer equal to today's.
   - A bad root still fails at start, switch or no switch.
   [proof: acceptance]
2. [S2] The switch and the off answer. Mutants: the default is on; `tools/list` lists the tools when off; `tools/call` of `mrw_write` serves when off; discovery keeps the on instructions when off; `MRW_MCP=on` ignored; `--enable=false` read as on; the variable read when the flag is given. [proof: mutation]
3. [S3] Contract §295 — `mrw mcp without --enable lists no tools and refuses a call` paired with `mrw mcp --enable serves both tools` — and every existing contract row, break-campaign probe, `chaos.py` process and binary-driven Go test that exercises the tools passes `--enable`. The break campaign's probe list and exit codes stay as they were. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ ./internal/mcp/ -count=1 -v -run '^(TestMrwMcpServesNoToolsUntilEnabled|TestALegacyResultIsUnchangedByTheModernPath|TestReadJSONCarriesTheServedTextAndTheReceiptMrwReadSends)$' 2>&1) \
  && for t in TestMrwMcpServesNoToolsUntilEnabled TestALegacyResultIsUnchangedByTheModernPath TestReadJSONCarriesTheServedTextAndTheReceiptMrwReadSends; do grep -qE "^--- PASS: $t \(" <<<"$out" || exit 1; done \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 295\.' scripts/contract.sh \
  && grep -qF 'PASS  mrw mcp without --enable lists no tools and refuses a call' <<<"$c" \
  && grep -qF 'PASS  mrw mcp --enable serves both tools' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestMrwMcpServesNoToolsUntilEnabled` | `cmd/mrw/mcpoff170_test.go` | the off answer in both eras, both tools refused with no effect, Decision 2's table, startup validation kept | none | S1, S2 |
| `TestALegacyResultIsUnchangedByTheModernPath` | `internal/mcp/era_test.go` | the enabled transcript is byte-identical | none | S3 |
| `TestReadJSONCarriesTheServedTextAndTheReceiptMrwReadSends` | `cmd/mrw/readjson165_test.go` | a binary-driven caller passes `--enable` and still matches | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `--enable` and `MRW_MCP` in `cmd/mrw/main.go` |
| 3 — the caller can discover it | the off instructions; `mrw mcp --help` (T2) |
| 4 — it is used | the opencode plugin (T2); MCP registrations |

## Mutation Log
(empty until execute)
- 2026-10-23 · b5f2b277* · mutant killed · exit 1 · `cmd/mrw/main.go` · the tools are on by default · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3
- 2026-10-23 · b5f2b277* · mutant killed · exit 1 · `internal/mcp/mcp.go` · tools/list lists the tools when off · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3
- 2026-10-23 · b5f2b277* · mutant killed · exit 1 · `internal/mcp/instructions.go` · initialize and discovery keep the on instructions when off · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3
- 2026-10-23 · b5f2b277* · mutant killed · exit 1 · `cmd/mrw/main.go` · MRW_MCP=on is ignored · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3
- 2026-10-23 · b5f2b277* · mutant killed · exit 1 · `cmd/mrw/main.go` · --enable=false is read as on · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3
- 2026-10-23 · b5f2b277* · mutant killed · exit 1 · `cmd/mrw/main.go` · the variable is read when the flag is given · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3
- 2026-10-23 · b5f2b277* · mutant killed · exit 1 · `internal/mcp/mcp.go` · tools/call serves when off: mrw_write writes · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3
- 2026-10-23 · b5f2b277* · mutant killed · exit 1 · `cmd/mrw/main.go` · a padded MRW_MCP value enables the tools · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3

## Invariants

- With the switch, every MCP answer is byte-identical to today's.
- The CLI is untouched.

## Risks

- In-process tests call `mcp.Serve` directly; the switch is resolved where `mrw mcp` starts the server, so they keep serving and only binary-driven callers need `--enable`.

## Stop Condition

The fence exits 0.

## Out of Scope

- The plugin and the words (deferred: ADR-170 T2)

## Verification Log
(empty until execute)
- 2026-10-23 · b5f2b277* · exit 1 · `set -o pipefail …` · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3 · ms:4086 · test-lock-sha256:2d1ea4ab3d35aff0101ef2a09a791084dfe037d503e2c8a2dead59cab1c8d7ae · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9tY3BvZmYxNzBfdGVzdC5nbwlUZXN0TXJ3TWNwU2VydmVzTm9Ub29sc1VudGlsRW5hYmxlZAk2NDM0ODZlNzUyMDJmNWU3MWFjZGQ5ZTQwZmEwMWZmOTA3ZmFkMTc0MmZiMTllMzI0YTdjNzJmZDBjMDNlYWNhCmJvZHkJY21kL21ydy9tY3BvZmYxNzBfdGVzdC5nbwlvZmYsIGxlZ2FjeSBlcmEJNjEwNDM0NmQxOGVmYmUxYzM5MTZjZjJjMzQ3ZTE2Y2RlNWQ0ZjIzNzg5MTVlYTkwY2E5OGUxYjIzNmM1OTQxOQpib2R5CWNtZC9tcncvbWNwb2ZmMTcwX3Rlc3QuZ28Jb2ZmLCBtb2Rlcm4gZXJhCWI1Yjc4YTJjZjA5Njk4YzBmYmUxYmVlNmQ2YTJhN2U1YTQ4OTUzOTc0NzBiZjNlZjY0MDkwZmNhZGU0YTM2MmMKYm9keQljbWQvbXJ3L21jcG9mZjE3MF90ZXN0LmdvCW9uLCBib3RoIHRvb2xzIGFuZCBhIHJlYWQJOTg3ZjQyM2ZlYWU2MmRhZTk1ZjJmZTUzYjAyY2I1OGM2YjAzMzUzNzQyZDZkNTUxYjg2MTIyNTI2MTM5ODJiMgpib2R5CWNtZC9tcncvbWNwb2ZmMTcwX3Rlc3QuZ28Jc3RhcnR1cCB2YWxpZGF0aW9uIGhvbGRzIHdpdGggdGhlIHN3aXRjaCBvciB3aXRob3V0CWM4OThhNTRhM2I2ODAwZmE3ZTYxY2ViZDJkYjNhNDRmOWE2ZDEzNjA4YTBmOTNlN2IzNTI1MmJmYjVhN2M0ZmYKYm9keQljbWQvbXJ3L3JlYWRqc29uMTY1X3Rlc3QuZ28JVGVzdFJlYWRKU09OQ2Fycmllc1RoZVNlcnZlZFRleHRBbmRUaGVSZWNlaXB0TXJ3UmVhZFNlbmRzCWQzYWYwYjE4ODM5NzI5OTcwZmMyZWNmY2I0Y2RkYmM4ZGMyNjY2ZmYzZmE4MGQ0YmRiYjQ3NDQyOTdhZmRjY2QKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFMZWdhY3lSZXN1bHRJc1VuY2hhbmdlZEJ5VGhlTW9kZXJuUGF0aAlmNzcxMTAxYWZjN2Y1YTJjZDlmYzZkMWIxOTJkZjMyMWY0ZTE3ZmJmODczNDVjOGE0OWNiYzc5NThhMGQ1YjBiCmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RBTW9kZXJuUmVhZFN0YXlzV2l0aGluVGhlQ2VpbGluZwk4NDhjMmIxMjVjNjA3YWZjMzBjOGQxOTc1YzI4N2ZmYTU0NTlhZWU0NDU4YjU4MDA1ZTJjNzRlZjdmNWViZjljCmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RBTW9kZXJuUmVxdWVzdEZvckFuVW5rbm93blZlcnNpb25Jc1JlZnVzZWRXaXRoSXRzU3VwcG9ydGVkTGlzdAkxODUzOWExYjRiMzQ1NGM2YzI0MWFkZTM1NDI1NmYyZTcyODE0N2U4YzM1MmQ5ZGEwOTZjNWZjZmJhM2RhNDI5CmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RBTW9kZXJuUmVxdWVzdFdpdGhvdXRDbGllbnRDYXBhYmlsaXRpZXNJc0ludmFsaWRQYXJhbXMJNDEzMDNmZGNkYzg0ZDZjZTk1NjM5ODU1NTc4YmM0NGE3ZTllZjQzYTliMWU4ZTIxYmQyNjFkM2NlNzE5NTg3Mgpib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0QU1vZGVyblJlc3VsdENhcnJpZXNSZXN1bHRUeXBlQW5kU2VydmVySW5mbwk4ZDRiZWVmYjdhZDAzMWZjMDEzNzdjYzQyYTJkMDUyY2YxODAwZWZmZjM4YzBhNTQzYjMwOGQxZTlmNWI3MzMzCmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RBTW9kZXJuVG9vbHNMaXN0Q2Fycmllc0NhY2hpbmdIaW50cwljNTQ1ZWM5Yzc3OTY2MzllMjE2NjVjMmJlMzViNWY0MDE4YjE5OTk1Mjg0ZThiZjQxMWQ4ZWQwMTA4OWE1ZWE1CmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RBTW9kZXJuV3JpdGVSZWNlaXB0SXNCdWRnZXRlZFdpdGhJdHNEZWNvcmF0aW9uCWY2MmJmYTNlMWMwMmE2ODZmNDAwYjNjYmRkNzYzODU3MDk4YzIwNGVhYzIxZDgyYzM3N2QwYzFiMDY3OTM4YTgKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdFNlcnZlckRpc2NvdmVyTmFtZXNFdmVyeVN1cHBvcnRlZFZlcnNpb24JYTlmZjY5MDMwZTU1ZTZkYzBhOGNjOGQwZmMxMWYxMzFhNjMxNzg5YTAzMGQwYTU3OWY0YTIxZjAxOTVjNzRhYQpib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0VGhlRXJhUHJlY2VkZW5jZUlzRml4ZWQJNTY2OWEyYjJjODIzZWJkMDFiZmI3ZGY5MGYxZmE4OWNiOTg0ZTI0OWY2N2RhMzM5Y2QwN2IyMDJkNGFiMmZmZA
  ```
  ```
- 2026-10-23 · b5f2b277* · exit 0 · `set -o pipefail …` · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3 · ms:95878
- 2026-10-23 · b5f2b277* · exit 0 · `set -o pipefail …` · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3 · ms:107881
- 2026-10-23 · b5f2b277* · exit 0 · `set -o pipefail …` · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3 · ms:81311
- 2026-10-23 · b5f2b277* · exit 0 · `set -o pipefail …` · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3 · ms:81729
- 2026-10-23 · b5f2b277* · exit 0 · `set -o pipefail …` · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3 · ms:81747
- 2026-10-23 · b5f2b277* · exit 0 · `set -o pipefail …` · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3 · ms:79967
- 2026-10-23 · b5f2b277* · exit 0 · `set -o pipefail …` · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3 · ms:80946
- 2026-10-23 · b5f2b277* · exit 0 · `set -o pipefail …` · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3 · ms:80050
- 2026-10-23 · b5f2b277* · exit 0 · `set -o pipefail …` · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3 · ms:79388
- 2026-10-23 · b5f2b277* · exit 0 · `set -o pipefail …` · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3 · ms:87105
- 2026-10-23 · b5f2b277* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:c308d85f3b69f94a7972dd08dcd7912323a4605df24c64c685283e3ce6f0f0d3 · ms:0 · test-lock-sha256:2b387b88b7e88de11936180a864c8a34639599a6ac4d7f6f89055f072e7f1b54 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9tY3BvZmYxNzBfdGVzdC5nbwlUZXN0TXJ3TWNwU2VydmVzTm9Ub29sc1VudGlsRW5hYmxlZAkwNDgzMjk5N2VkMTNlYjdhNGM3ZGI2NmNiZDczN2ZhNjI0MTE1YTU1YjNmMmFlZGNjMzZjNDFmMWY0NmNmMjU2CmJvZHkJY21kL21ydy9tY3BvZmYxNzBfdGVzdC5nbwlhIHJlZnVzZWQgY2FsbCBjaGFuZ2VzIG5vIHN0YXRlLCBpbiBlaXRoZXIgZXJhCWM3YmYwOWRjNzcwOGUxOWQ5Nzc4NmFjYThlMjAyOTQ3M2M2OGI1MWI2ZWNlZGQ5ZDQxNDczOGFmYmY2NWM4MWEKYm9keQljbWQvbXJ3L21jcG9mZjE3MF90ZXN0LmdvCW9mZiwgbGVnYWN5IGVyYQk2MTA0MzQ2ZDE4ZWZiZTFjMzkxNmNmMmMzNDdlMTZjZGU1ZDRmMjM3ODkxNWVhOTBjYTk4ZTFiMjM2YzU5NDE5CmJvZHkJY21kL21ydy9tY3BvZmYxNzBfdGVzdC5nbwlvZmYsIG1vZGVybiBlcmEJYjViNzhhMmNmMDk2OThjMGZiZTFiZWU2ZDZhMmE3ZTVhNDg5NTM5NzQ3MGJmM2VmNjQwOTBmY2FkZTRhMzYyYwpib2R5CWNtZC9tcncvbWNwb2ZmMTcwX3Rlc3QuZ28Jb24sIGJvdGggdG9vbHMgYW5kIGEgcmVhZAk5ODdmNDIzZmVhZTYyZGFlOTVmMmZlNTNiMDJjYjU4YzZiMDMzNTM3NDJkNmQ1NTFiODYxMjI1MjYxMzk4MmIyCmJvZHkJY21kL21ydy9tY3BvZmYxNzBfdGVzdC5nbwlzdGFydHVwIHZhbGlkYXRpb24gaG9sZHMgd2l0aCB0aGUgc3dpdGNoIG9yIHdpdGhvdXQJYzg5OGE1NGEzYjY4MDBmYTdlNjFjZWJkMmRiM2E0NGY5YTZkMTM2MDhhMGY5M2U3YjM1MjUyYmZiNWE3YzRmZgpib2R5CWNtZC9tcncvcmVhZGpzb24xNjVfdGVzdC5nbwlUZXN0UmVhZEpTT05DYXJyaWVzVGhlU2VydmVkVGV4dEFuZFRoZVJlY2VpcHRNcndSZWFkU2VuZHMJZDNhZjBiMTg4Mzk3Mjk5NzBmYzJlY2ZjYjRjZGRiYzhkYzI2NjZmZjNmYTgwZDRiZGJiNDc0NDI5N2FmZGNjZApib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0QUxlZ2FjeVJlc3VsdElzVW5jaGFuZ2VkQnlUaGVNb2Rlcm5QYXRoCWY3NzExMDFhZmM3ZjVhMmNkOWZjNmQxYjE5MmRmMzIxZjRlMTdmYmY4NzM0NWM4YTQ5Y2JjNzk1OGEwZDViMGIKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5SZWFkU3RheXNXaXRoaW5UaGVDZWlsaW5nCTg0OGMyYjEyNWM2MDdhZmMzMGM4ZDE5NzVjMjg3ZmZhNTQ1OWFlZTQ0NThiNTgwMDVlMmM3NGVmN2Y1ZWJmOWMKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5SZXF1ZXN0Rm9yQW5Vbmtub3duVmVyc2lvbklzUmVmdXNlZFdpdGhJdHNTdXBwb3J0ZWRMaXN0CTE4NTM5YTFiNGIzNDU0YzZjMjQxYWRlMzU0MjU2ZjJlNzI4MTQ3ZThjMzUyZDlkYTA5NmM1ZmNmYmEzZGE0MjkKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5SZXF1ZXN0V2l0aG91dENsaWVudENhcGFiaWxpdGllc0lzSW52YWxpZFBhcmFtcwk0MTMwM2ZkY2RjODRkNmNlOTU2Mzk4NTU1NzhiYzQ0YTdlOWVmNDNhOWIxZThlMjFiZDI2MWQzY2U3MTk1ODcyCmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RBTW9kZXJuUmVzdWx0Q2Fycmllc1Jlc3VsdFR5cGVBbmRTZXJ2ZXJJbmZvCThkNGJlZWZiN2FkMDMxZmMwMTM3N2NjNDJhMmQwNTJjZjE4MDBlZmZmMzhjMGE1NDNiMzA4ZDFlOWY1YjczMzMKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5Ub29sc0xpc3RDYXJyaWVzQ2FjaGluZ0hpbnRzCWM1NDVlYzljNzc5NjYzOWUyMTY2NWMyYmUzNWI1ZjQwMThiMTk5OTUyODRlOGJmNDExZDhlZDAxMDg5YTVlYTUKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5Xcml0ZVJlY2VpcHRJc0J1ZGdldGVkV2l0aEl0c0RlY29yYXRpb24JZjYyYmZhM2UxYzAyYTY4NmY0MDBiM2NiZGQ3NjM4NTcwOThjMjA0ZWFjMjFkODJjMzc3ZDBjMWIwNjc5MzhhOApib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0U2VydmVyRGlzY292ZXJOYW1lc0V2ZXJ5U3VwcG9ydGVkVmVyc2lvbglhOWZmNjkwMzBlNTVlNmRjMGE4Y2M4ZDBmYzExZjEzMWE2MzE3ODlhMDMwZDBhNTc5ZjRhMjFmMDE5NWM3NGFhCmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RUaGVFcmFQcmVjZWRlbmNlSXNGaXhlZAk1NjY5YTJiMmM4MjNlYmQwMWJmYjdkZjkwZjFmYTg5Y2I5ODRlMjQ5ZjY3ZGEzMzljZDA3YjIwMmQ0YWIyZmZk · test-lock-kind:replace
