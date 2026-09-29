# Task ADR-093-T2: both input schemas are closed; README and AGENTS.md name the refusal; contract §176

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `additionalProperties: false` on both input schemas; contract §176; the refusal sentence in `README.md` and `AGENTS.md`
**Consumes:** `undeclaredRefusal` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `every advertised input schema is closed`, `contract §176 names both claims`, `the golden changed by the two keys and nothing else, on the branch`, `the golden test compares and does not rewrite`, `the docs name the refusal`, `the rest of internal/mcp still passes`, `no engine file changes`, `go.mod keeps one requirement`, `the package is gofmt-clean and vets`

## Goal

`tools/list` tells a host what T1 enforces: both input schemas declare
`additionalProperties: false`, and the README and AGENTS.md say that an undeclared argument is
refused.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/mcp.go` | edit | `"additionalProperties": false` in `mrw_read`'s `InputSchema` (`:423-464`) and `mrw_write`'s (`:491-557`). `tools()` is what `tools/list` serves, so these maps are what SELECT the advertisement. |
| `internal/mcp/undeclared093_test.go` | edit | `TestEveryInputSchemaIsClosed` |
| `internal/mcp/testdata/legacy_golden.jsonl` | regenerate | the `tools/list` line gains the key once per tool. It was captured before ADR-067 T4 and re-captured by ADR-070, ADR-075, ADR-076, ADR-090 and ADR-091 (`era_test.go:214-223`). |
| `internal/mcp/era_test.go` | edit | the comment at `:214-223` gains an ADR-093 clause, as every re-capture has |
| `README.md` | edit | one sentence in "Use it from an MCP host" (`:246-250`, where `mrw_write` is said to run no check) |
| `AGENTS.md` | edit | one sentence after the MCP paragraph (`:114-122`), which the centralised `mrw` skill mirrors |
| `scripts/contract.sh` | edit | §176 |

## Ordered Steps

1. [S1] Write `TestEveryInputSchemaIsClosed` and confirm it is RED on T1's tree on its assertion.
   It reads `tools/list` through `serve`, the wire a host reads, not `tools()` directly. It asserts
   that every tool listed has `inputSchema.additionalProperties == false`, and that at least two
   tools were checked, so an empty list cannot pass.
2. [S2] Add `"additionalProperties": false` to both `InputSchema` maps, and confirm GREEN.
   [proof: mutation]
   Mutant: drop it from `mrw_read`'s map. This kills `TestEveryInputSchemaIsClosed` and the fence's count of the key in the golden; §176 goes red in the pre-commit contract run.
3. [S3] Regenerate the golden with
   `MRW_UPDATE_LEGACY_GOLDEN=1 go test ./internal/mcp/ -count=1 -run TestALegacyResultIsUnchangedByTheModernPath`,
   then run it again without the variable. Strip every `"additionalProperties":false,` from both the
   regenerated file and the merge-base copy, and the two must be equal byte for byte. The key must
   also occur exactly twice. The fence checks both against the merge-base, which makes it a
   branch-time check: once T2 merges, the merge-base is the commit itself and the comparison is
   trivially equal, so the Verification Log line `adr-verify` writes at execution is the lasting
   receipt. The fence unsets `MRW_UPDATE_LEGACY_GOLDEN`, because with it set the golden test rewrites
   its own baseline and passes (`era_test.go:227-234`). Append "and once by ADR-093, whose input
   schemas are closed: that key, once per tool" to the comment at `era_test.go:214-223`.
   [proof: acceptance]
4. [S4] Add this sentence to both `README.md` and `AGENTS.md`: "An argument a tool does not declare
   is refused, naming it and the arguments the tool takes; nothing is done."
   [proof: acceptance]
5. [S5] Add §176, which drives the binary through `m mcp`. `tools/list` shows
   `additionalProperties: false` for both tools, with the claim `every input schema is closed`. The
   pair is `mrw_read {"specs":["a.txt"]}`, which is still served, with the claim
   `and a call with declared arguments is still served`. The row is RED against v1.31.0 in a
   mini-harness. [proof: human: `./scripts/contract.sh` run unpiped before the commit, exit 0 with §176's rows printed — the whole contract is minutes long and a fence runs at least three times per task, so the fence greps the section and its claims and lifecycle §6 runs it]
6. [S6] Run `gofmt` and `go vet` on `internal/mcp`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
unset MRW_UPDATE_LEGACY_GOLDEN
G=internal/mcp/testdata/legacy_golden.jsonl
grep -q '^# 176\. ' scripts/contract.sh \
  && grep -qF 'every input schema is closed' scripts/contract.sh \
  && grep -qF 'and a call with declared arguments is still served' scripts/contract.sh \
  && go test ./internal/mcp/ -count=1 -v \
    -run 'TestEveryInputSchemaIsClosed|TestALegacyResultIsUnchangedByTheModernPath' 2>&1 | tee /tmp/adr093-t2.out \
  && grep -q '^--- PASS: TestEveryInputSchemaIsClosed ' /tmp/adr093-t2.out \
  && grep -q '^--- PASS: TestALegacyResultIsUnchangedByTheModernPath ' /tmp/adr093-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr093-t2.out \
  && [ "$(grep -o '"additionalProperties":false' "$G" | wc -l | tr -d ' ')" = "2" ] \
  && diff <(git show "$(git merge-base HEAD origin/main):$G" | sed 's/"additionalProperties":false,//g') <(sed 's/"additionalProperties":false,//g' "$G") > /dev/null \
  && grep -q 'An argument a tool does not declare is refused' README.md \
  && grep -q 'An argument a tool does not declare is refused' AGENTS.md \
  && go test ./internal/mcp/ -count=1 > /tmp/adr093-t2-all.out 2>&1 \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines cmd/mrw \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines cmd/mrw)" ] \
  && [ "$(go mod edit -json | python3 -c 'import json,sys; print(len(json.load(sys.stdin).get("Require") or []))')" = "1" ] \
  && [ -z "$(gofmt -l internal/mcp)" ] \
  && go vet ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryInputSchemaIsClosed` | `internal/mcp/undeclared093_test.go` | over the wire, every tool in `tools/list` declares `additionalProperties: false`, and at least two were checked | — | S1, S2 |
| `TestALegacyResultIsUnchangedByTheModernPath` | `internal/mcp/era_test.go` | the re-captured golden holds | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestEveryInputSchemaIsClosed` and §176 |
| 2 — something selects it | the two `InputSchema` maps in `tools()`, which `tools/list` serves; dropping the key kills the test and the golden count, and §176 goes red in the pre-commit contract run |
| 3 — the caller can discover it | `tools/list` is the schema a host reads; README and AGENTS.md say it in prose |
| 4 — it is used | nobody has measured whether any host enforces a closed schema before sending (deferred in ADR-093's Out of Scope); the server's refusal from T1 holds either way |

## Mutation Log
(empty until execute)
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/mcp/mcp.go` · S2: drop additionalProperties from mrw_read's inputSchema; kills TestEveryInputSchemaIsClosed and the fence's golden count · acceptance-sha256:5830fb06337df7eafdfcddf4b69d383ddeab754c24dd758dd9e1e987515d938c

## Invariants

- The golden changes by `"additionalProperties":false` once per tool, and by nothing else.
- Every other `tools/list` field, and the instructions, stay byte-identical.

## Risks

- Go orders a map's keys when it encodes one, so the key lands first inside each `inputSchema`. The
  golden comparison strips the key together with its comma, `"additionalProperties":false,`, which
  depends on that position. If the key is ever not first, the fence goes red rather than green.
- `tools()`'s `InputSchema` is read elsewhere (`conformance_test.go:309-314`, `mcp_test.go:176`,
  `:418`, `:744-745`, contract `:2049-2052`). Checked 2026-09-29: each reads `properties` or the
  whole map as text, and none enumerates the schema's top-level keys, so the new key breaks none.
  The schema walkers that do read `additionalProperties` (`conformance_test.go:237`, `:535`,
  contract `:2188`) take it only when it is an object, so a boolean `false` is skipped. The
  whole-package run in the fence and the pre-commit contract run would catch one that did.

## Stop Condition

Stop and ask M if the re-captured golden differs by anything other than the two keys.

## Out of Scope

- The refusal itself: T1's job.

## Verification Log
(empty until execute)
- 2026-09-29 · 0b2f304* · exit 1 · `set -o pipefail …` · acceptance-sha256:5830fb06337df7eafdfcddf4b69d383ddeab754c24dd758dd9e1e987515d938c · ms:34 · test-lock-sha256:52f2fc255126017da0decb291c96ea33819124120369d018fe37477479818a7b · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0QUxlZ2FjeVJlc3VsdElzVW5jaGFuZ2VkQnlUaGVNb2Rlcm5QYXRoCWY3NzExMDFhZmM3ZjVhMmNkOWZjNmQxYjE5MmRmMzIxZjRlMTdmYmY4NzM0NWM4YTQ5Y2JjNzk1OGEwZDViMGIKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5SZWFkU3RheXNXaXRoaW5UaGVDZWlsaW5nCTg0OGMyYjEyNWM2MDdhZmMzMGM4ZDE5NzVjMjg3ZmZhNTQ1OWFlZTQ0NThiNTgwMDVlMmM3NGVmN2Y1ZWJmOWMKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5SZXF1ZXN0Rm9yQW5Vbmtub3duVmVyc2lvbklzUmVmdXNlZFdpdGhJdHNTdXBwb3J0ZWRMaXN0CTE4NTM5YTFiNGIzNDU0YzZjMjQxYWRlMzU0MjU2ZjJlNzI4MTQ3ZThjMzUyZDlkYTA5NmM1ZmNmYmEzZGE0MjkKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5SZXF1ZXN0V2l0aG91dENsaWVudENhcGFiaWxpdGllc0lzSW52YWxpZFBhcmFtcwk0MTMwM2ZkY2RjODRkNmNlOTU2Mzk4NTU1NzhiYzQ0YTdlOWVmNDNhOWIxZThlMjFiZDI2MWQzY2U3MTk1ODcyCmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RBTW9kZXJuUmVzdWx0Q2Fycmllc1Jlc3VsdFR5cGVBbmRTZXJ2ZXJJbmZvCThkNGJlZWZiN2FkMDMxZmMwMTM3N2NjNDJhMmQwNTJjZjE4MDBlZmZmMzhjMGE1NDNiMzA4ZDFlOWY1YjczMzMKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5Ub29sc0xpc3RDYXJyaWVzQ2FjaGluZ0hpbnRzCWM1NDVlYzljNzc5NjYzOWUyMTY2NWMyYmUzNWI1ZjQwMThiMTk5OTUyODRlOGJmNDExZDhlZDAxMDg5YTVlYTUKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5Xcml0ZVJlY2VpcHRJc0J1ZGdldGVkV2l0aEl0c0RlY29yYXRpb24JZjYyYmZhM2UxYzAyYTY4NmY0MDBiM2NiZGQ3NjM4NTcwOThjMjA0ZWFjMjFkODJjMzc3ZDBjMWIwNjc5MzhhOApib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0U2VydmVyRGlzY292ZXJOYW1lc0V2ZXJ5U3VwcG9ydGVkVmVyc2lvbglhOWZmNjkwMzBlNTVlNmRjMGE4Y2M4ZDBmYzExZjEzMWE2MzE3ODlhMDMwZDBhNTc5ZjRhMjFmMDE5NWM3NGFhCmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RUaGVFcmFQcmVjZWRlbmNlSXNGaXhlZAk1NjY5YTJiMmM4MjNlYmQwMWJmYjdkZjkwZjFmYTg5Y2I5ODRlMjQ5ZjY3ZGEzMzljZDA3YjIwMmQ0YWIyZmZkCmJvZHkJaW50ZXJuYWwvbWNwL3VuZGVjbGFyZWQwOTNfdGVzdC5nbwlUZXN0QVJlZnVzZWRVbmRlY2xhcmVkQXJndW1lbnRQcm9tb3Rlc05vQWNrCWZkZGRlNTk1NjcxYTQyODQ1MDIwMzYyZDkzMTViMGRjZmM0M2FmZmMzNDhhNjBkZThmYzYxMTIwZjZjMzZlMWYKYm9keQlpbnRlcm5hbC9tY3AvdW5kZWNsYXJlZDA5M190ZXN0LmdvCVRlc3RBblVuZGVjbGFyZWRBcmd1bWVudElzUmVmdXNlZEJ5TmFtZQljZTUxNjNkN2ZkMzUwZTg3ZmY0Y2FiYjlhZTczZWQ1OWQyNzM2NjQ1MzAyODg0ZmQ4MTkyZDM0YWFiZTRkOTJjCmJvZHkJaW50ZXJuYWwvbWNwL3VuZGVjbGFyZWQwOTNfdGVzdC5nbwlUZXN0RXZlcnlJbnB1dFNjaGVtYUlzQ2xvc2VkCTRjY2ZkMzUzMjdmYzlhNDFkY2I0MTg3ZThmMzcxZTgyYzBkM2E0NDk0Zjk3MjRkODc5M2Q2N2Q2MDUyYzRiNDUKYm9keQlpbnRlcm5hbC9tY3AvdW5kZWNsYXJlZDA5M190ZXN0LmdvCVRlc3RFdmVyeVRvb2xEZWNvZGVzRXhhY3RseVRoZUFyZ3VtZW50c0l0RGVjbGFyZXMJNGUzNTNkMzZjYTc4NGExMWFmMDM5ZDRkYjgwYjYxZWM0NzJlMTA1OTU2YjM2ZmYwOTU1YmFkNmZmNjk3ZDEyMgpib2R5CWludGVybmFsL21jcC91bmRlY2xhcmVkMDkzX3Rlc3QuZ28JVGVzdFRoZVJlZnVzYWxSb3V0ZXNPbmx5VG9GbGFnc1RoZUNMSUhhcwk2ODAzOGE4Y2I5OGIwZjcxMGQ4YTY2MzQzOGVlMmQyNjAyZWE5Y2UxMmJkZmRlMzdkMGI0MWZmZGNhMDMyNTlk
  ```
  ```
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:5830fb06337df7eafdfcddf4b69d383ddeab754c24dd758dd9e1e987515d938c · ms:7387
- 2026-09-29 · human-observed · S5 observed by the executing agent (lane A-093): ./scripts/contract.sh run unpiped before the commit, exit 0, contract holds, with §176's two rows printed PASS
