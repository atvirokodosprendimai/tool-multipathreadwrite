# Task ADR-093-T1: `callTool` refuses an argument its tool does not declare; contract §175

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `undeclaredRefusal`, `readArgs`, `writeArgs`; contract §175
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the call in callTool that refuses an undeclared key`, `exact key matching`, `a refused call promotes no ack and is not tallied`, `the refusal leaves through the one funnel`, `an unknown tool stays a protocol error whatever keys it carries`, `invalid raw UTF-8 keeps ADR-078's refusal`, `the decode types and the schemas declare one set`, `the routing names only flags the CLI has`, `contract §175 names both claims`, `the rest of internal/mcp still passes`, `no engine file changes`, `go.mod keeps one requirement`, `the package is gofmt-clean and vets`

## Goal

A `tools/call` whose `arguments` carry a key the tool's advertised `inputSchema` does not declare
is answered with an `isError` result naming the key, and nothing is done. Two cases fall outside
this refusal. An unknown tool is still `-32602`. A non-UTF-8 argument still gets ADR-078's refusal.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `callTool` (`:148-195`) calls `undeclaredRefusal` after the era setup (`:166-172`) and before the `switch` (`:176`). A refusal is assigned to `res` and skips the `switch`, so it reaches the existing funnel `return withinCeiling(decorateCall(res))` at `:194`; no return site is added. **This call is the line that SELECTS the refusal.** `undeclaredRefusal` is defined here and reads the declared names from `tools()`. The anonymous structs in `readTool` (`:231-251`) and `writeTool` (`:542-549`) become `readArgs` and `writeArgs`, with fields and tags unchanged. |
| `internal/mcp/undeclared093_test.go` | new | the four tests below, and the tool-to-type table the class test needs (ADR-088: a table only tests read lives in a test file) |
| `internal/mcp/limit_test.go` | edit | one oversize row in `TestEveryAnswerFitsIncludingTheRefusals`'s table (`:347-358`), so the refusal is shown to pass the ceiling funnel |
| `scripts/contract.sh` | edit | §175 |

## Ordered Steps

1. [S1] Write the three behaviour tests in `internal/mcp/undeclared093_test.go`, and one row in
   `TestEveryAnswerFitsIncludingTheRefusals`, and confirm each is RED on `main` on an assertion, not
   a build failure.
   - `TestAnUndeclaredArgumentIsRefusedByName` is table-driven:
     - `mrw_write` with `check: true`, `then: ["vet"]`, `then_sh: "true"`, `force: true`, and a
       case variant (`Plan` in place of `plan`), each alone;
     - `mrw_read` with `context: 3`, `max_lines: 1`, `stat: true`, and a `_meta` object inside
       `arguments`, each alone;
     - one call carrying two undeclared keys;
     - one `mrw_write` row sent in the modern era, `callReq(…, modernMeta)` as
       `era_test.go:144` sends it;
     - `{"name":"no_such_tool","arguments":{"x":1}}`, which must still answer `-32602`. This row
       is a guard that holds on `main`; S3's fifth mutant proves it.

     Each refused answer is a `result` with `isError: true` and no `error`. Its `content[0]` names
     every undeclared key and every property the tool's schema declares, each list in sorted order,
     the properties read from `tools()` so the test and the server read one list. It names
     `Version`, which the test sets to a sentinel such as `v0.0.0-adr093` and restores with
     `t.Cleanup`, because `"dev"` is a substring of ordinary words. It says that an `ack` sent with
     the call was not recorded. The modern-era row carries `resultType: "complete"`. On a refused
     `mrw_write` whose plan creates a file, the file does not exist afterwards, and
     `authoring.Load(root)` is the same before and after (as `tally083_test.go:28` reads it). On a
     refused `mrw_read`, `content[0]` carries no `-- ck` marker. The paired valid sibling deletes an
     extra key, and renames a case variant to its declared spelling: the file is created, and the
     read serves its lines.
   - `TestARefusedUndeclaredArgumentPromotesNoAck` reads a file and keeps the pending `ck` id. It
     sends `mrw_read` with that id in `ack` beside `max_lines: 1`, which is refused. Then an
     `mrw_write` replacing a served line, with no `ack`, is refused as unread: the test attempts the
     write rather than inspecting a response (`testing.md`, "Assert the thing"). Then the same id in
     a declared call is promoted, and the same write applies.
   - `TestTheRefusalRoutesOnlyToFlagsTheCLIHas` triggers a refusal on each tool, the `mrw_write`
     one with `then` alone, so that the word `check` in its text comes from the routing and not from
     the key. It runs three checks:
     - every `--flag` in `content[0]` appears in `cliHelp(t, "read")` or `cliHelp(t, "write")`,
       whichever matches the tool (`internal/mcp/mcp_test.go:569-576`);
     - no named flag, with its dashes made underscores, is a property the tool declares (as
       `TestTheRoutingClaimsOnlyRealExclusives`, `:738`);
     - each refusal names at least one flag, so an empty set cannot pass, and the `mrw_write`
       refusal names `mrw write` and the word `check`, and says the tool runs no check and no step.
   - The new row in `TestEveryAnswerFitsIncludingTheRefusals` (`limit_test.go:347-358`) is
     `mrw_write` with `plan: "x"` and an undeclared key of 3,000 characters, marked oversize. On
     `main` the key is ignored and the parse refusal is short, so the funnel's own sentence is absent
     and the row is RED.
2. [S2] Name the decode types `readArgs` and `writeArgs`, with fields and tags unchanged, and add
   `TestEveryToolDecodesExactlyTheArgumentsItDeclares`. Its table maps `"mrw_read"` to `readArgs{}`
   and `"mrw_write"` to `writeArgs{}`. The test asserts that the table's keys equal the names
   `tools()` lists, and that each type's `json` tag names equal its schema's property names. It is a
   guard that holds once the types are named, so it is proved by mutation, not by a red run: adding
   ``Check bool `json:"check"` `` to `writeArgs` turns it red. [proof: mutation]
3. [S3] Implement `undeclaredRefusal(name string, args json.RawMessage) string` and call it from
   `callTool` as Affected Files says: when it returns a message, `res = errorResult(msg)` and the
   `switch` is skipped, so the refusal reaches the existing `return withinCeiling(decorateCall(res))`
   at `:194` and no return site is added. It returns `""` for three cases: a tool `tools()` does not
   list, so the `switch` default still answers `-32602`; arguments that are not valid UTF-8, so the
   tool's `nonUTF8Arg` refusal still answers; and arguments with no undeclared key. Keys are compared
   exactly, never with `strings.EqualFold`, and quoted in the text. The text follows ADR-093
   Decision 1. Confirm S1 is GREEN, and `TestAMalformedCallIsStillAProtocolError` and
   `TestArgumentsThatAreNotUTF8AreRefusedByName` are still GREEN. [proof: mutation]
   Mutants:
   - delete the `undeclaredRefusal` call in `callTool`: kills `TestAnUndeclaredArgumentIsRefusedByName` and `TestARefusedUndeclaredArgumentPromotesNoAck`;
   - compare keys with `strings.EqualFold`: kills the `Plan` row;
   - make the `mrw_write` routing name `--checks`: kills `TestTheRefusalRoutesOnlyToFlagsTheCLIHas`;
   - return `errorResult(msg), nil` from the check instead of assigning `res`: kills the modern-era row and the oversize row of `TestEveryAnswerFitsIncludingTheRefusals`;
   - drop the unknown-tool case, so an unlisted tool declares nothing: kills the `no_such_tool` row;
   - drop the UTF-8 case: kills `TestArgumentsThatAreNotUTF8AreRefusedByName`, whose empty key (`protocol078_test.go:90-97`) is undeclared, so the refusal names it instead of saying `not valid UTF-8`.
4. [S4] Add §175, which drives the binary through `m mcp`. The first call is `mrw_write` with
   `{"plan":"@@ b.txt 0 create\nX\n","check":true}`: it answers a `result` with `isError: true`
   whose text names `check` and `plan`, and `b.txt` does not exist. The second is the same call
   without `check`: it answers with `isError` absent and `b.txt` holding `X`. Then `mrw_read`
   `{"specs":["a.txt"],"max_lines":1}` is refused naming `max_lines`, and `{"specs":["a.txt"]}` is
   served. The claims are `an undeclared argument is refused by name` and
   `and the same call without it is served and applied`. The row is RED against v1.31.0 in a
   mini-harness. [proof: human: `./scripts/contract.sh` run unpiped before the commit, exit 0 with §175's rows printed — the whole contract is minutes long and a fence runs at least three times per task, so the fence greps the section and its claims and lifecycle §6 runs it]
5. [S5] Run the whole `internal/mcp` package, `gofmt` and `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 175\. ' scripts/contract.sh \
  && grep -qF 'an undeclared argument is refused by name' scripts/contract.sh \
  && grep -qF 'and the same call without it is served and applied' scripts/contract.sh \
  && go test ./internal/mcp/ -count=1 -v \
    -run 'TestAnUndeclaredArgumentIsRefusedByName|TestARefusedUndeclaredArgumentPromotesNoAck|TestTheRefusalRoutesOnlyToFlagsTheCLIHas|TestEveryToolDecodesExactlyTheArgumentsItDeclares|TestEveryAnswerFitsIncludingTheRefusals' 2>&1 | tee /tmp/adr093-t1.out \
  && grep -q '^--- PASS: TestAnUndeclaredArgumentIsRefusedByName ' /tmp/adr093-t1.out \
  && grep -q '^--- PASS: TestARefusedUndeclaredArgumentPromotesNoAck ' /tmp/adr093-t1.out \
  && grep -q '^--- PASS: TestTheRefusalRoutesOnlyToFlagsTheCLIHas ' /tmp/adr093-t1.out \
  && grep -q '^--- PASS: TestEveryToolDecodesExactlyTheArgumentsItDeclares ' /tmp/adr093-t1.out \
  && grep -q '^--- PASS: TestEveryAnswerFitsIncludingTheRefusals ' /tmp/adr093-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr093-t1.out \
  && go test ./internal/mcp/ -count=1 > /tmp/adr093-t1-all.out 2>&1 \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines cmd/mrw \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines cmd/mrw)" ] \
  && [ "$(go mod edit -json | python3 -c 'import json,sys; print(len(json.load(sys.stdin).get("Require") or []))')" = "1" ] \
  && [ -z "$(gofmt -l internal/mcp)" ] \
  && go vet ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnUndeclaredArgumentIsRefusedByName` | `internal/mcp/undeclared093_test.go` | each survey key, a case variant and a `_meta` inside `arguments` are `isError` results naming the key, every declared argument (both sorted), the version and the unrecorded `ack`; the modern-era row is decorated; the refused write creates nothing and leaves the tally as it was, and the refused read serves nothing; the same call with the key deleted, or renamed to its declared spelling, is applied or served; an unknown tool with a key is still `-32602` | — | S1, S3 |
| `TestARefusedUndeclaredArgumentPromotesNoAck` | `internal/mcp/undeclared093_test.go` | an `ack` beside an undeclared key licenses nothing, because the next write is refused as unread; the same `ack` in a declared call then licenses that write | — | S1, S3 |
| `TestTheRefusalRoutesOnlyToFlagsTheCLIHas` | `internal/mcp/undeclared093_test.go` | every `--flag` a refusal names is in that subcommand's `--help` and is not a declared argument; the write refusal names `mrw write` and the check | — | S1, S3 |
| `TestEveryToolDecodesExactlyTheArgumentsItDeclares` | `internal/mcp/undeclared093_test.go` | for every tool `tools()` lists, its decode type's `json` tags equal its schema's properties, and the table names every tool | — | S2 |
| `TestAMalformedCallIsStillAProtocolError` | `internal/mcp/argerror_test.go` | guard, unchanged: an unknown tool and a non-object `arguments` are still `-32602` | — | S3 |
| `TestEveryAnswerFitsIncludingTheRefusals` | `internal/mcp/limit_test.go` | one new row: an undeclared key of 3,000 characters is brought under the 900-byte ceiling by the funnel's own refusal | — | S1, S3 |
| `TestArgumentsThatAreNotUTF8AreRefusedByName` | `internal/mcp/protocol078_test.go` | guard, unchanged: invalid raw UTF-8, under a declared key or an empty one, still gets ADR-078's refusal | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests and §175 |
| 2 — something selects it | the `undeclaredRefusal` call in `callTool`; deleting it kills the refusal test and the `ack` test (S3's first mutant), and §175 goes red in the pre-commit contract run |
| 3 — the caller can discover it | the refusal lists the accepted arguments in `content[0]`, which a host hands the model; T2 closes the advertised schema |
| 4 — it is used | nothing measures this, and ADR-009 refuses telemetry. The evidence for building it is the survey's probe and the reproduction in ADR-093's Context |

## Mutation Log
(empty until execute)
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/mcp/tools.go` · S3: delete the undeclaredRefusal call in callTool; kills TestAnUndeclaredArgumentIsRefusedByName and TestARefusedUndeclaredArgumentPromotesNoAck · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/mcp/tools.go` · S3: compare keys with strings.EqualFold; kills the Plan row of TestAnUndeclaredArgumentIsRefusedByName · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/mcp/tools.go` · S3: the mrw_write routing names --checks; kills TestTheRefusalRoutesOnlyToFlagsTheCLIHas · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/mcp/tools.go` · S3: return the refusal instead of assigning res, bypassing the funnel; kills the modern-era row and the oversize row of TestEveryAnswerFitsIncludingTheRefusals · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/mcp/tools.go` · S3: drop the unknown-tool case, so an unlisted tool declares nothing; kills the no_such_tool row of TestAnUndeclaredArgumentIsRefusedByName · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/mcp/tools.go` · S3: drop the UTF-8 case; kills TestArgumentsThatAreNotUTF8AreRefusedByName, whose empty key is undeclared, so the refusal names it instead of saying not valid UTF-8 · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: add Check bool `json:"check"` to writeArgs; kills TestEveryToolDecodesExactlyTheArgumentsItDeclares · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee

## Invariants

- A call carrying only declared arguments is answered as before. T1 does not change the legacy
  golden, because `legacyTranscript` sends only declared keys.
- An unknown tool and a non-object `arguments` are still `-32602` (ADR-067 T3).
- Arguments whose raw bytes are not UTF-8 still get ADR-078's refusal, because the key check does
  not decode them. A key that is valid UTF-8 is named as decoded, a genuine U+FFFD or an escaped
  surrogate included, since ADR-078 rejected refusing a decoded U+FFFD.
- Every refusal passes `withinCeiling`.
- A refused call promotes no `ack`, serves nothing, records nothing in the ledger, writes nothing
  and is not counted in ADR-009's tally.

## Risks

- A test elsewhere in `internal/mcp` builds an `arguments` map with an undeclared key. The
  whole-package run in the fence finds it. If the key is a typo, fix the test.
- The refusal text is long enough to meet a small ceiling. `withinCeiling` answers that, as it does
  for every refusal.

## Stop Condition

Stop and ask M in two cases:
- A host measured on this machine adds keys of its own to a tool's `arguments`. The refusal would
  reject every call from it, which is M's decision and not this task's.
- A test or caller in the tree sends an undeclared key on purpose, as a caller shape rather than a
  typo.

## Out of Scope

- `additionalProperties: false` in the advertised schemas, the golden and the docs: T2's job.

## Verification Log
(empty until execute)
- 2026-09-29 · 0b2f304* · exit 1 · `set -o pipefail …` · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee · ms:29 · test-lock-sha256:f198d1abb083c55bb8cf70f4fd9057deeffa3c0c01b959bd229969d3dc57cd8b · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9hcmdlcnJvcl90ZXN0LmdvCVRlc3RBTWFsZm9ybWVkQ2FsbElzU3RpbGxBUHJvdG9jb2xFcnJvcgk0NGRkMGM2ZTk1MmQ2ZDY3ZDJhOGFlMGE5ZGI4ZTczODllMzM0MzBlNjE3YmJmODlkOGM1ZWI0MDZlNjY3ZWIwCmJvZHkJaW50ZXJuYWwvbWNwL2FyZ2Vycm9yX3Rlc3QuZ28JVGVzdEFuQXJndW1lbnRFcnJvcklzQVRvb2xFeGVjdXRpb25FcnJvcgk0NTY5ZmZlNzc4YzMyMDEwZTVjMmJjYTVjNTgzZTcwMTMzMmNkYjJmZTlhNTQ4OTg0YTM5YTY3N2Y2ZTAxZWFjCmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdEFQYXJ0aWFsQXBwbGljYXRpb25Jc05vdFJlcG9ydGVkQXNOb3RoaW5nV3JpdHRlbgllOGRjYTcxZWFkODBlZjVlOTZkMmIxNjY5MDA3NGM5ZTM3ZDQxZDJmZDAzNjhiYzU1OWM4MWUxY2Y0NTk1ZjYzCmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdEFSZWFkV2hvc2VSZWNlaXB0T3ZlcmZsb3dzSXNOb3RTZXJ2ZWQJZDQ4YjFhZDEyNzQ5MDMzNjZmNDZhZmVjOGE2OWQ1YmVlYjQ0M2NiMzNhYTQwZWFiNWM2MTU5MTBkZDUxMzM3MApib2R5CWludGVybmFsL21jcC9saW1pdF90ZXN0LmdvCVRlc3RBU21hbGxDZWlsaW5nUmVmdXNlc1RoZVdyaXRlQmVmb3JlQXBwbHlpbmcJMjRmNjk3ODlmY2ZmY2I5YjQyMDM2MDI3NjQ2MmQ0NWM0MjhkOWJkYzIxNmRjM2I1M2YyNTM3MGVhNDQzN2MxOApib2R5CWludGVybmFsL21jcC9saW1pdF90ZXN0LmdvCVRlc3RBV3JpdGVSZWNlaXB0RWxpZGVzU3VjY2Vzc2VzTm90RmFpbHVyZXMJNjY5NDMzMDVkZTc0OThlNWY3ZjBjZmE1YTdlMzAxNzM0YzFkMDY3YzVlYTdkOThkM2E0ZjM1NzhjYTQwNzY2NQpib2R5CWludGVybmFsL21jcC9saW1pdF90ZXN0LmdvCVRlc3RFdmVyeUFuc3dlckZpdHNJbmNsdWRpbmdUaGVSZWZ1c2FscwljNzZhNzgyODQwNjM5YzhlMTJlOGEwOTQwYmRiMzZhNmM2MGJmM2JhMzM5MTY4ZTRjMWI3Yjc2ZDdhYjA2ZWY3CmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdFRoZUFkdmVydGlzZWRDZWlsaW5nQm91bmRzRXZlcnlBbnN3ZXIJYjcyYTk0MzZkZmZiNTM5Mjc1NTliYzhiNTA4ZmZiZmQ1MTRjOTI0YWJkMjc3MDI3OWUwOTQ2NGYzYWNkM2IwZgpib2R5CWludGVybmFsL21jcC9saW1pdF90ZXN0LmdvCVRlc3RUaGVDZWlsaW5nTmV2ZXJTaHJpbmtzQVNlcnZlZFJlYWQJNWNhMWIxYTYzY2I1N2Y4ODFlYmJiZGQ0ZDg4YTJmMjlhNDMzZGUxZDJiOWE5YTc2MGM2ZjllYzI3MzAxNGRlMwpib2R5CWludGVybmFsL21jcC9saW1pdF90ZXN0LmdvCVRlc3RUaGVTZWNvbmRTdGFnZUVsaXNpb25Ecm9wc0ZpbGVSZWNvcmRzCTFiZTg1ZDgyNDk2NTZkNjAzMDZlZGIxYzM2MTJkYmM1MDJhZTg2MzBkMTgyNzAxNDM2ODU3ZTRmZmJlZGUwMTkKYm9keQlpbnRlcm5hbC9tY3AvbGltaXRfdGVzdC5nbwlUZXN0VGhlU2Vjb25kU3RhZ2VOZXZlckVsaWRlc0FXcml0dGVuRmlsZQkxOGRjZDhlYjYzZDk4MzQ0OWFmZWVkZjc3OTJlMzFlNzQ4ZmNiNDFkOGVkNmJkYjZlZDgwMWI2MzYzODJkNGRjCmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdFRoZVdyaXRlRmxvb3JJc0FGbG9vcglmMWFjMzZiNjgyOWYxODhhOTM5ZTlhYTUzNjkyMmQ0YWM3YTllMDFiNGYzZmE0NTcyM2Y4NTNmZWUyNWFmMzkwCmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFCYWRFeGNsdWRlR2xvYklzUmVmdXNlZE92ZXJNQ1AJN2E5NTM5YmY5NWQzNzMyMzBlNTk5YmJlMGZiMWNkMzQ3NGM4ODQyNTNhNzMxYWYzN2M2OGNmNGIyYjM3NzA5Ygpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBTG9uZ0xpbmVJblRoZU1pZGRsZUlzTmFtZWRXaXRoVGhlUmFuZ2VzQXJvdW5kSXQJZDNiODAwNGUxOGE3NGU3NTJiNzE1OTJmZDI1YmZjOTZjMjBhMGNiYWY4M2UwZGJmZjQ5ZTQ4YWNjYmFhYTMyYwpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBTWFueVNwZWNSZWFkU3RvcHNPbmNlSXRJc092ZXJUaGVDZWlsaW5nCTdlYTYxYmFiYTQ1YjhiOTVlMDgxZDY5YjJiYTI2OTk2NzczMDUxYmFjZGU0Y2ZiNTM4ZDM2ZTk3NWNkZjhmMjgKYm9keQlpbnRlcm5hbC9tY3AvcHJvdG9jb2wwNzhfdGVzdC5nbwlUZXN0QVJlbmRlcmVkRml0UmVmdXNhbE5hbWVzVGhlTGluZQliM2RiMzVkYmM5OGU3NjFkNWZlYWM2NWQxZDVmYzBjYzhjNmMzNjZhMjc4MzUwNzNhZWRhNzQ4MDE0MzQzZTFjCmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFSZXF1ZXN0SWRUaGF0SXNOZWl0aGVyQVN0cmluZ05vckFuSW50ZWdlcklzSW52YWxpZAk3ZWE1MTM0MzY1MzkyZDMxMzgzNzlmNTg3OGJkNTEwNzA4ZjVjMTViZDk2YjI5NzJmYTliY2MyN2FlMjg5NjUxCmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFyZ3VtZW50c1RoYXRBcmVOb3RVVEY4QXJlUmVmdXNlZEJ5TmFtZQk3NTY2NmI2MjljODNiNzE5MDU0NTYwYjMwZGZmMWJmMzJkMDVkZGRjZGVhNzE4OTdkZmZhNmUwMGE0YTA5ZDU2CmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdFRoZVJlZnVzYWxOYW1lc1RoZUxpbmVUaGF0RW5jb2Rlc1Bhc3RUaGVDZWlsaW5nCTIwNDM1MGY4NWE4MTg2ZGQxMGQ4YzhkNDcxODBmMjlhODA1YzE1MGMzZWQ3ZmZlZDFjZWNhOGJkMTUzOTc4NGMKYm9keQlpbnRlcm5hbC9tY3AvdW5kZWNsYXJlZDA5M190ZXN0LmdvCVRlc3RBUmVmdXNlZFVuZGVjbGFyZWRBcmd1bWVudFByb21vdGVzTm9BY2sJZmRkZGU1OTU2NzFhNDI4NDUwMjAzNjJkOTMxNWIwZGNmYzQzYWZmYzM0OGE2MGRlOGZjNjExMjBmNmMzNmUxZgpib2R5CWludGVybmFsL21jcC91bmRlY2xhcmVkMDkzX3Rlc3QuZ28JVGVzdEFuVW5kZWNsYXJlZEFyZ3VtZW50SXNSZWZ1c2VkQnlOYW1lCWNlNTE2M2Q3ZmQzNTBlODdmZjRjYWJiOWFlNzNlZDU5ZDI3MzY2NDUzMDI4ODRmZDgxOTJkMzRhYWJlNGQ5MmMKYm9keQlpbnRlcm5hbC9tY3AvdW5kZWNsYXJlZDA5M190ZXN0LmdvCVRlc3RUaGVSZWZ1c2FsUm91dGVzT25seVRvRmxhZ3NUaGVDTElIYXMJNjgwMzhhOGNiOThiMGY3MTBkOGE2NjM0MzhlZTJkMjYwMmVhOWNlMTJiZGZkZTM3ZDBiNDFmZmRjYTAzMjU5ZAp1bnByb3ZlbglpbnRlcm5hbC9tY3AvdW5kZWNsYXJlZDA5M190ZXN0LmdvCVRlc3RFdmVyeVRvb2xEZWNvZGVzRXhhY3RseVRoZUFyZ3VtZW50c0l0RGVjbGFyZXM
  ```
  ```
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee · ms:7021
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee · ms:7476
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee · ms:7187
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee · ms:7518
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee · ms:7242
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee · ms:7129
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee · ms:7330
- 2026-09-29 · 0b2f304* · exit 0 · `adr-verify --relock` · acceptance-sha256:3ffb201350cbee9c38603e0314c3fee42a50b8c9bc6f574719d5c4639eb60aee · ms:0 · test-lock-sha256:4d598f1a19736d102d67c8210973db97bd822bdefb6d40b41ae4989994088e36 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9hcmdlcnJvcl90ZXN0LmdvCVRlc3RBTWFsZm9ybWVkQ2FsbElzU3RpbGxBUHJvdG9jb2xFcnJvcgk0NGRkMGM2ZTk1MmQ2ZDY3ZDJhOGFlMGE5ZGI4ZTczODllMzM0MzBlNjE3YmJmODlkOGM1ZWI0MDZlNjY3ZWIwCmJvZHkJaW50ZXJuYWwvbWNwL2FyZ2Vycm9yX3Rlc3QuZ28JVGVzdEFuQXJndW1lbnRFcnJvcklzQVRvb2xFeGVjdXRpb25FcnJvcgk0NTY5ZmZlNzc4YzMyMDEwZTVjMmJjYTVjNTgzZTcwMTMzMmNkYjJmZTlhNTQ4OTg0YTM5YTY3N2Y2ZTAxZWFjCmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdEFQYXJ0aWFsQXBwbGljYXRpb25Jc05vdFJlcG9ydGVkQXNOb3RoaW5nV3JpdHRlbgllOGRjYTcxZWFkODBlZjVlOTZkMmIxNjY5MDA3NGM5ZTM3ZDQxZDJmZDAzNjhiYzU1OWM4MWUxY2Y0NTk1ZjYzCmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdEFSZWFkV2hvc2VSZWNlaXB0T3ZlcmZsb3dzSXNOb3RTZXJ2ZWQJZDQ4YjFhZDEyNzQ5MDMzNjZmNDZhZmVjOGE2OWQ1YmVlYjQ0M2NiMzNhYTQwZWFiNWM2MTU5MTBkZDUxMzM3MApib2R5CWludGVybmFsL21jcC9saW1pdF90ZXN0LmdvCVRlc3RBU21hbGxDZWlsaW5nUmVmdXNlc1RoZVdyaXRlQmVmb3JlQXBwbHlpbmcJMjRmNjk3ODlmY2ZmY2I5YjQyMDM2MDI3NjQ2MmQ0NWM0MjhkOWJkYzIxNmRjM2I1M2YyNTM3MGVhNDQzN2MxOApib2R5CWludGVybmFsL21jcC9saW1pdF90ZXN0LmdvCVRlc3RBV3JpdGVSZWNlaXB0RWxpZGVzU3VjY2Vzc2VzTm90RmFpbHVyZXMJNjY5NDMzMDVkZTc0OThlNWY3ZjBjZmE1YTdlMzAxNzM0YzFkMDY3YzVlYTdkOThkM2E0ZjM1NzhjYTQwNzY2NQpib2R5CWludGVybmFsL21jcC9saW1pdF90ZXN0LmdvCVRlc3RFdmVyeUFuc3dlckZpdHNJbmNsdWRpbmdUaGVSZWZ1c2FscwljNzZhNzgyODQwNjM5YzhlMTJlOGEwOTQwYmRiMzZhNmM2MGJmM2JhMzM5MTY4ZTRjMWI3Yjc2ZDdhYjA2ZWY3CmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdFRoZUFkdmVydGlzZWRDZWlsaW5nQm91bmRzRXZlcnlBbnN3ZXIJYjcyYTk0MzZkZmZiNTM5Mjc1NTliYzhiNTA4ZmZiZmQ1MTRjOTI0YWJkMjc3MDI3OWUwOTQ2NGYzYWNkM2IwZgpib2R5CWludGVybmFsL21jcC9saW1pdF90ZXN0LmdvCVRlc3RUaGVDZWlsaW5nTmV2ZXJTaHJpbmtzQVNlcnZlZFJlYWQJNWNhMWIxYTYzY2I1N2Y4ODFlYmJiZGQ0ZDg4YTJmMjlhNDMzZGUxZDJiOWE5YTc2MGM2ZjllYzI3MzAxNGRlMwpib2R5CWludGVybmFsL21jcC9saW1pdF90ZXN0LmdvCVRlc3RUaGVTZWNvbmRTdGFnZUVsaXNpb25Ecm9wc0ZpbGVSZWNvcmRzCTFiZTg1ZDgyNDk2NTZkNjAzMDZlZGIxYzM2MTJkYmM1MDJhZTg2MzBkMTgyNzAxNDM2ODU3ZTRmZmJlZGUwMTkKYm9keQlpbnRlcm5hbC9tY3AvbGltaXRfdGVzdC5nbwlUZXN0VGhlU2Vjb25kU3RhZ2VOZXZlckVsaWRlc0FXcml0dGVuRmlsZQkxOGRjZDhlYjYzZDk4MzQ0OWFmZWVkZjc3OTJlMzFlNzQ4ZmNiNDFkOGVkNmJkYjZlZDgwMWI2MzYzODJkNGRjCmJvZHkJaW50ZXJuYWwvbWNwL2xpbWl0X3Rlc3QuZ28JVGVzdFRoZVdyaXRlRmxvb3JJc0FGbG9vcglmMWFjMzZiNjgyOWYxODhhOTM5ZTlhYTUzNjkyMmQ0YWM3YTllMDFiNGYzZmE0NTcyM2Y4NTNmZWUyNWFmMzkwCmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFCYWRFeGNsdWRlR2xvYklzUmVmdXNlZE92ZXJNQ1AJN2E5NTM5YmY5NWQzNzMyMzBlNTk5YmJlMGZiMWNkMzQ3NGM4ODQyNTNhNzMxYWYzN2M2OGNmNGIyYjM3NzA5Ygpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBTG9uZ0xpbmVJblRoZU1pZGRsZUlzTmFtZWRXaXRoVGhlUmFuZ2VzQXJvdW5kSXQJZDNiODAwNGUxOGE3NGU3NTJiNzE1OTJmZDI1YmZjOTZjMjBhMGNiYWY4M2UwZGJmZjQ5ZTQ4YWNjYmFhYTMyYwpib2R5CWludGVybmFsL21jcC9wcm90b2NvbDA3OF90ZXN0LmdvCVRlc3RBTWFueVNwZWNSZWFkU3RvcHNPbmNlSXRJc092ZXJUaGVDZWlsaW5nCTdlYTYxYmFiYTQ1YjhiOTVlMDgxZDY5YjJiYTI2OTk2NzczMDUxYmFjZGU0Y2ZiNTM4ZDM2ZTk3NWNkZjhmMjgKYm9keQlpbnRlcm5hbC9tY3AvcHJvdG9jb2wwNzhfdGVzdC5nbwlUZXN0QVJlbmRlcmVkRml0UmVmdXNhbE5hbWVzVGhlTGluZQliM2RiMzVkYmM5OGU3NjFkNWZlYWM2NWQxZDVmYzBjYzhjNmMzNjZhMjc4MzUwNzNhZWRhNzQ4MDE0MzQzZTFjCmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFSZXF1ZXN0SWRUaGF0SXNOZWl0aGVyQVN0cmluZ05vckFuSW50ZWdlcklzSW52YWxpZAk3ZWE1MTM0MzY1MzkyZDMxMzgzNzlmNTg3OGJkNTEwNzA4ZjVjMTViZDk2YjI5NzJmYTliY2MyN2FlMjg5NjUxCmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdEFyZ3VtZW50c1RoYXRBcmVOb3RVVEY4QXJlUmVmdXNlZEJ5TmFtZQk3NTY2NmI2MjljODNiNzE5MDU0NTYwYjMwZGZmMWJmMzJkMDVkZGRjZGVhNzE4OTdkZmZhNmUwMGE0YTA5ZDU2CmJvZHkJaW50ZXJuYWwvbWNwL3Byb3RvY29sMDc4X3Rlc3QuZ28JVGVzdFRoZVJlZnVzYWxOYW1lc1RoZUxpbmVUaGF0RW5jb2Rlc1Bhc3RUaGVDZWlsaW5nCTIwNDM1MGY4NWE4MTg2ZGQxMGQ4YzhkNDcxODBmMjlhODA1YzE1MGMzZWQ3ZmZlZDFjZWNhOGJkMTUzOTc4NGMKYm9keQlpbnRlcm5hbC9tY3AvdW5kZWNsYXJlZDA5M190ZXN0LmdvCVRlc3RBUmVmdXNlZFVuZGVjbGFyZWRBcmd1bWVudFByb21vdGVzTm9BY2sJZmRkZGU1OTU2NzFhNDI4NDUwMjAzNjJkOTMxNWIwZGNmYzQzYWZmYzM0OGE2MGRlOGZjNjExMjBmNmMzNmUxZgpib2R5CWludGVybmFsL21jcC91bmRlY2xhcmVkMDkzX3Rlc3QuZ28JVGVzdEFuVW5kZWNsYXJlZEFyZ3VtZW50SXNSZWZ1c2VkQnlOYW1lCWNlNTE2M2Q3ZmQzNTBlODdmZjRjYWJiOWFlNzNlZDU5ZDI3MzY2NDUzMDI4ODRmZDgxOTJkMzRhYWJlNGQ5MmMKYm9keQlpbnRlcm5hbC9tY3AvdW5kZWNsYXJlZDA5M190ZXN0LmdvCVRlc3RFdmVyeVRvb2xEZWNvZGVzRXhhY3RseVRoZUFyZ3VtZW50c0l0RGVjbGFyZXMJNGUzNTNkMzZjYTc4NGExMWFmMDM5ZDRkYjgwYjYxZWM0NzJlMTA1OTU2YjM2ZmYwOTU1YmFkNmZmNjk3ZDEyMgpib2R5CWludGVybmFsL21jcC91bmRlY2xhcmVkMDkzX3Rlc3QuZ28JVGVzdFRoZVJlZnVzYWxSb3V0ZXNPbmx5VG9GbGFnc1RoZUNMSUhhcwk2ODAzOGE4Y2I5OGIwZjcxMGQ4YTY2MzQzOGVlMmQyNjAyZWE5Y2UxMmJkZmRlMzdkMGI0MWZmZGNhMDMyNTlk · test-lock-kind:relock
- 2026-09-29 · human-observed · S4 observed by the executing agent (lane A-093): ./scripts/contract.sh run unpiped before the commit, exit 0, contract holds, with §175's four rows printed PASS
