# Task ADR-067-T4: the server is dual-era — `server/discover`, per-request `_meta`, `-32022`, `resultType`, caching hints, a per-call reserve; contract §125

**Depends-on:** T1, T2, T3
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `server/discover`; the per-request era decision in `handle`; modern result decoration; the per-call ceiling reserve
**Consumes:** `supportedVersions` constant (legacy and modern lists) (T2); `serverInfo()` helper (name, title, version, description) (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `discover names every supported version`, `an unknown version is -32022 with the list`, `every advertised version is answered on retry`, `a modern request without clientCapabilities is -32602`, `a modern result is decorated`, `a modern tools/list carries caching hints`, `a legacy result is byte-identical`, `every in-call measurement reads the reserved ceiling`, `the era precedence is fixed`, `the binary serves a modern probe`, `no engine file changes`

## Goal

`handle` (`internal/mcp/mcp.go:148`) applies ADR-067 Decision 4, in its precedence order:
1. a notification is never answered, before any `_meta` is read;
2. `initialize` is always legacy;
3. `server/discover` is always modern-shaped;
4. every other method takes its era from `params._meta["io.modelcontextprotocol/protocolVersion"]`:
   - absent → today's behaviour, unchanged;
   - a version mrw does not speak → `-32022`, `data: {supported, requested}`;
   - `2026-07-28` without `io.modelcontextprotocol/clientCapabilities` → `-32602`;
   - `2026-07-28`, well formed → the method runs as today, and its result gains `resultType: "complete"` and `_meta["io.modelcontextprotocol/serverInfo"]`; `tools/list` also gains `ttlMs: 3600000` and `cacheScope: "public"`;
   - a legacy version in `_meta` → served undecorated.

A modern `tools/call` reserves its decoration's encoded size when it starts. Every in-call comparison against the ceiling reads the reserved limit, so no measurement sees a smaller answer than the one sent. That covers the write floor before applying, receipt elision, `firstPage`, `matchIndex`, the served-read check and `withinCeiling`. The sites are enumerated by `grep -n 'MaxResultChars' internal/mcp/tools.go`. A refusal built by the funnel is decorated like any modern result. The README's "Use it from an MCP host" section names the revisions mrw speaks and the era rule.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/mcp.go` | edit | precedence and era decision in `handle`; `server/discover`; `-32022` code and error data; decoration; the modern list joins T2's `supportedVersions` |
| `internal/mcp/tools.go` | edit | `callTool` takes the era; one per-call limit read by every in-call ceiling comparison; decoration applied inside the call |
| `internal/mcp/era_test.go` | new | the nine tests below |
| `internal/mcp/testdata/legacy_golden.jsonl` | new | legacy answers captured immediately before this task, on T1–T3's tree |
| `scripts/contract.sh` | edit | §125 |
| `README.md` | edit | the protocol revisions paragraph |

## Ordered Steps

1. [S1] Capture the legacy golden, then write the tests, and confirm each feature test RED on T1–T3's tree on an assertion. [proof: mutation]
   - **The golden first.** Before any T4 edit, record the legacy answers to `initialize`, `tools/list`, `ping`, a served `mrw_read`, a refused `mrw_read`, and a `mrw_write` dry run on a fixed fixture. Each `-- ck <id>` value and each `ack` id is normalised to a placeholder: ids are random (`ack.go:96`), and nothing else is normalised.
   - **Feature tests (red first):**
     - `TestServerDiscoverNamesEverySupportedVersion`: `supportedVersions` is `["2026-07-28","2025-11-25","2025-06-18"]`; `capabilities.tools` is present; `_meta` names `mrw`; `instructions` equals `initialize`'s; `ttlMs` ≥ 0; `cacheScope` is `"public"`; `resultType` is `"complete"`. It is the same shape when its `_meta` names `2025-11-25`.
     - `TestAModernRequestForAnUnknownVersionIsRefusedWithItsSupportedList`: `tools/list` naming `1900-01-01` → `-32022`, `data.requested`, and `data.supported` equal to discover's list. Then the same request retried with each listed version is answered, decorated only for `2026-07-28`.
     - `TestAModernRequestWithoutClientCapabilitiesIsInvalidParams`: `tools/list` naming `2026-07-28` with no `clientCapabilities` → `-32602`.
     - `TestAModernResultCarriesResultTypeAndServerInfo`: a modern `tools/call` of `mrw_read` → `resultType` `"complete"`, `_meta` names `mrw`, and `content[0]` is the served text.
     - `TestAModernToolsListCarriesCachingHints`: a modern `tools/list` → `ttlMs`, `cacheScope`, `resultType`, and the same tools as a legacy list.
     - `TestAModernReadStaysWithinTheCeiling`: at a ceiling where the undecorated served read fits and the decorated one would not, the modern answer is within the ceiling. It degrades to a page rather than being cut, and a page it served holds pending checkpoints that an `ack` then licenses.
     - `TestAModernWriteReceiptIsBudgetedWithItsDecoration`, two cases on a 300-hunk plan whose full receipt cannot fit:
       - a ceiling one byte below the decorated floor refuses before applying, the tree unchanged. The refusal is the JSON-RPC error ADR-032 decided on, so there is no result to decorate;
       - at exactly the decorated floor the write applies, and its elided answer is within the ceiling, decorated, and never says nothing was written.
       A partial commit is not driven here: the only seam that forces one, `commitRenameFn`, is internal to `internal/apply`. Elision takes the same budgeted path either way. Amended during execution, 2026-09-24.
     - `TestTheEraPrecedenceIsFixed`:
       - `initialize` carrying modern `_meta` answers the legacy shape;
       - a notification naming `1900-01-01`, or with no `clientCapabilities`, gets no reply;
       - legacy and modern requests interleaved on one `Serve` each answer in their own shape.
   - **Guard (holds from the start, proved by mutation):** `TestALegacyResultIsUnchangedByTheModernPath`. Every legacy answer equals the golden after normalisation, and none carries `resultType`.
2. [S2] Implement; confirm GREEN, and that all of `internal/mcp` stays green. [proof: mutation]
   Mutants:
   - legacy results decorated too: kills the legacy guard;
   - the reserve not read by the pre-apply floor check: kills the write test;
   - the reserve not read by `firstPage`: kills the read test;
   - `-32022` without `data.supported`: kills the unknown-version test;
   - the `clientCapabilities` check removed: kills the capabilities test;
   - `initialize` taking its era from `_meta`: kills the precedence test;
   - `tools/list` without `ttlMs`: kills the caching test.
3. [S3] §125 through the binary, one stdin session:
   - `server/discover` naming `2026-07-28` → `supportedVersions` names all three;
   - a modern `tools/call` of `mrw_read` serves the line with `resultType`;
   - the pair: `tools/list` naming `1900-01-01` → `-32022`.
   RED against v1.23.0 in a mini-harness, GREEN in the full `./scripts/contract.sh`. [proof: mutation]
4. [S4] README paragraph; `gofmt`, `go vet`, unpiped; re-run the Claude Code wire capture against the built binary and record that the legacy session is unchanged. [proof: human: the capture is a live host session outside the suite]

## Acceptance

```bash
set -o pipefail
grep -q '^# 125\. ' scripts/contract.sh \
  && grep -q '2026-07-28' README.md \
  && go test ./internal/mcp/ -count=1 -v \
    -run 'TestServerDiscoverNamesEverySupportedVersion|TestAModernRequestForAnUnknownVersionIsRefusedWithItsSupportedList|TestAModernRequestWithoutClientCapabilitiesIsInvalidParams|TestAModernResultCarriesResultTypeAndServerInfo|TestAModernToolsListCarriesCachingHints|TestALegacyResultIsUnchangedByTheModernPath|TestAModernReadStaysWithinTheCeiling|TestAModernWriteReceiptIsBudgetedWithItsDecoration|TestTheEraPrecedenceIsFixed' 2>&1 | tee /tmp/adr067-t4.out \
  && grep -q '^--- PASS: TestServerDiscoverNamesEverySupportedVersion ' /tmp/adr067-t4.out \
  && grep -q '^--- PASS: TestAModernRequestForAnUnknownVersionIsRefusedWithItsSupportedList ' /tmp/adr067-t4.out \
  && grep -q '^--- PASS: TestAModernRequestWithoutClientCapabilitiesIsInvalidParams ' /tmp/adr067-t4.out \
  && grep -q '^--- PASS: TestAModernResultCarriesResultTypeAndServerInfo ' /tmp/adr067-t4.out \
  && grep -q '^--- PASS: TestAModernToolsListCarriesCachingHints ' /tmp/adr067-t4.out \
  && grep -q '^--- PASS: TestALegacyResultIsUnchangedByTheModernPath ' /tmp/adr067-t4.out \
  && grep -q '^--- PASS: TestAModernReadStaysWithinTheCeiling ' /tmp/adr067-t4.out \
  && grep -q '^--- PASS: TestAModernWriteReceiptIsBudgetedWithItsDecoration ' /tmp/adr067-t4.out \
  && grep -q '^--- PASS: TestTheEraPrecedenceIsFixed ' /tmp/adr067-t4.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr067-t4.out \
  && ./scripts/contract.sh > /tmp/adr067-t4-contract.out 2>&1 \
  && grep -q '^  PASS  a modern request is served per request' /tmp/adr067-t4-contract.out \
  && grep -q '^  PASS  a modern request naming an unknown version is refused with the supported list' /tmp/adr067-t4-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines cmd/mrw \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines cmd/mrw)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l internal/mcp)" ] \
  && go vet ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestServerDiscoverNamesEverySupportedVersion` | `internal/mcp/era_test.go` | `server/discover`'s answer, whatever version it names | — | S1, S2 |
| `TestAModernRequestForAnUnknownVersionIsRefusedWithItsSupportedList` | `internal/mcp/era_test.go` | `-32022` and its data; every listed version is answered on retry | — | S1, S2 |
| `TestAModernRequestWithoutClientCapabilitiesIsInvalidParams` | `internal/mcp/era_test.go` | a malformed modern request | — | S1, S2 |
| `TestAModernResultCarriesResultTypeAndServerInfo` | `internal/mcp/era_test.go` | modern decoration on `tools/call` | — | S1, S2 |
| `TestAModernToolsListCarriesCachingHints` | `internal/mcp/era_test.go` | `ttlMs`/`cacheScope` on `tools/list` | — | S1, S2 |
| `TestALegacyResultIsUnchangedByTheModernPath` | `internal/mcp/era_test.go` | guard: legacy answers equal the pre-T4 golden | — | S1, S2 |
| `TestAModernReadStaysWithinTheCeiling` | `internal/mcp/era_test.go` | the reserve reaches the read's measurements; checkpoints held | — | S1, S2 |
| `TestAModernWriteReceiptIsBudgetedWithItsDecoration` | `internal/mcp/era_test.go` | the reserve reaches the floor and receipt elision; a refusal is decorated | — | S1, S2 |
| `TestTheEraPrecedenceIsFixed` | `internal/mcp/era_test.go` | `initialize` legacy, notifications silent, interleaved eras | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the nine tests and §125 |
| 2 — something selects it | `handle` decides the era from the request's `_meta`; `callTool` reads the reserved limit |
| 3 — the caller can discover it | `server/discover`, and `-32022` naming the supported list |
| 4 — it is used | no measured host sends the modern era yet (Claude Code 2.1.281, 2026-09-24); the parent ADR's Follow-up re-captures it |

## Verification Log
(empty until execute)
- 2026-09-24 · ae040e4* · exit 1 · `set -o pipefail …` · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · ms:70 · test-lock-sha256:0ae405e88c1b3bdfef2d31405d69e71b79c6479c37333a35d66ef8817ec82932 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0QUxlZ2FjeVJlc3VsdElzVW5jaGFuZ2VkQnlUaGVNb2Rlcm5QYXRoCWY3NzExMDFhZmM3ZjVhMmNkOWZjNmQxYjE5MmRmMzIxZjRlMTdmYmY4NzM0NWM4YTQ5Y2JjNzk1OGEwZDViMGIKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5SZWFkU3RheXNXaXRoaW5UaGVDZWlsaW5nCWIyN2E1NTk5NWM5Yzk1Y2QxNjFjZWE4NGM2N2U1NGM3N2NjYjc3ZmM4OWZmYjQ3Njg2YzE0ZmEyM2Q5ZDFlNTIKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5SZXF1ZXN0Rm9yQW5Vbmtub3duVmVyc2lvbklzUmVmdXNlZFdpdGhJdHNTdXBwb3J0ZWRMaXN0CTE4NTM5YTFiNGIzNDU0YzZjMjQxYWRlMzU0MjU2ZjJlNzI4MTQ3ZThjMzUyZDlkYTA5NmM1ZmNmYmEzZGE0MjkKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5SZXF1ZXN0V2l0aG91dENsaWVudENhcGFiaWxpdGllc0lzSW52YWxpZFBhcmFtcwk0MTMwM2ZkY2RjODRkNmNlOTU2Mzk4NTU1NzhiYzQ0YTdlOWVmNDNhOWIxZThlMjFiZDI2MWQzY2U3MTk1ODcyCmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RBTW9kZXJuUmVzdWx0Q2Fycmllc1Jlc3VsdFR5cGVBbmRTZXJ2ZXJJbmZvCThkNGJlZWZiN2FkMDMxZmMwMTM3N2NjNDJhMmQwNTJjZjE4MDBlZmZmMzhjMGE1NDNiMzA4ZDFlOWY1YjczMzMKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5Ub29sc0xpc3RDYXJyaWVzQ2FjaGluZ0hpbnRzCWM1NDVlYzljNzc5NjYzOWUyMTY2NWMyYmUzNWI1ZjQwMThiMTk5OTUyODRlOGJmNDExZDhlZDAxMDg5YTVlYTUKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5Xcml0ZVJlY2VpcHRJc0J1ZGdldGVkV2l0aEl0c0RlY29yYXRpb24JNTNhNDEwZTQ3ODYzYzJlZTI5ZjA1NWMzOWFlZTE3ZTQ0NDVkMWQ1ZTMyYTI4MjljMWY5NDVlNzJjZjkzNGMxMgpib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0U2VydmVyRGlzY292ZXJOYW1lc0V2ZXJ5U3VwcG9ydGVkVmVyc2lvbglhOWZmNjkwMzBlNTVlNmRjMGE4Y2M4ZDBmYzExZjEzMWE2MzE3ODlhMDMwZDBhNTc5ZjRhMjFmMDE5NWM3NGFhCmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RUaGVFcmFQcmVjZWRlbmNlSXNGaXhlZAk1NjY5YTJiMmM4MjNlYmQwMWJmYjdkZjkwZjFmYTg5Y2I5ODRlMjQ5ZjY3ZGEzMzljZDA3YjIwMmQ0YWIyZmZk
  ```
  ```
- 2026-09-24 · ae040e4* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · ms:0 · test-lock-sha256:1430c68502124efdac3c7c9508da6b832d47d46cec504ff38f67d26679b08b55 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0QUxlZ2FjeVJlc3VsdElzVW5jaGFuZ2VkQnlUaGVNb2Rlcm5QYXRoCWY3NzExMDFhZmM3ZjVhMmNkOWZjNmQxYjE5MmRmMzIxZjRlMTdmYmY4NzM0NWM4YTQ5Y2JjNzk1OGEwZDViMGIKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5SZWFkU3RheXNXaXRoaW5UaGVDZWlsaW5nCTg0OGMyYjEyNWM2MDdhZmMzMGM4ZDE5NzVjMjg3ZmZhNTQ1OWFlZTQ0NThiNTgwMDVlMmM3NGVmN2Y1ZWJmOWMKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5SZXF1ZXN0Rm9yQW5Vbmtub3duVmVyc2lvbklzUmVmdXNlZFdpdGhJdHNTdXBwb3J0ZWRMaXN0CTE4NTM5YTFiNGIzNDU0YzZjMjQxYWRlMzU0MjU2ZjJlNzI4MTQ3ZThjMzUyZDlkYTA5NmM1ZmNmYmEzZGE0MjkKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5SZXF1ZXN0V2l0aG91dENsaWVudENhcGFiaWxpdGllc0lzSW52YWxpZFBhcmFtcwk0MTMwM2ZkY2RjODRkNmNlOTU2Mzk4NTU1NzhiYzQ0YTdlOWVmNDNhOWIxZThlMjFiZDI2MWQzY2U3MTk1ODcyCmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RBTW9kZXJuUmVzdWx0Q2Fycmllc1Jlc3VsdFR5cGVBbmRTZXJ2ZXJJbmZvCThkNGJlZWZiN2FkMDMxZmMwMTM3N2NjNDJhMmQwNTJjZjE4MDBlZmZmMzhjMGE1NDNiMzA4ZDFlOWY1YjczMzMKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5Ub29sc0xpc3RDYXJyaWVzQ2FjaGluZ0hpbnRzCWM1NDVlYzljNzc5NjYzOWUyMTY2NWMyYmUzNWI1ZjQwMThiMTk5OTUyODRlOGJmNDExZDhlZDAxMDg5YTVlYTUKYm9keQlpbnRlcm5hbC9tY3AvZXJhX3Rlc3QuZ28JVGVzdEFNb2Rlcm5Xcml0ZVJlY2VpcHRJc0J1ZGdldGVkV2l0aEl0c0RlY29yYXRpb24JZWFkNDg4ZmY3NGQzNmU0YTY0OGE3MmI2OWFkNjY5YTYyMzI3NTlhOWFlZDFkZTIzZTM4ZTNkNDM5Y2EzYmRlZApib2R5CWludGVybmFsL21jcC9lcmFfdGVzdC5nbwlUZXN0U2VydmVyRGlzY292ZXJOYW1lc0V2ZXJ5U3VwcG9ydGVkVmVyc2lvbglhOWZmNjkwMzBlNTVlNmRjMGE4Y2M4ZDBmYzExZjEzMWE2MzE3ODlhMDMwZDBhNTc5ZjRhMjFmMDE5NWM3NGFhCmJvZHkJaW50ZXJuYWwvbWNwL2VyYV90ZXN0LmdvCVRlc3RUaGVFcmFQcmVjZWRlbmNlSXNGaXhlZAk1NjY5YTJiMmM4MjNlYmQwMWJmYjdkZjkwZjFmYTg5Y2I5ODRlMjQ5ZjY3ZGEzMzljZDA3YjIwMmQ0YWIyZmZk · test-lock-kind:replace
- 2026-09-24 · human-observed · Claude (session d856d76a) observed 2026-09-24: a headless Claude Code 2.1.281 session against the branch build (ae040e4 + ADR-067, dirty) through a logging wrapper sent initialize protocolVersion 2025-11-25 and was answered 2025-11-25 with serverInfo name/title/version/description; tools/list and a tools/call of mrw_read were served legacy-shaped with no resultType; no server/discover probe was sent
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · ms:44201
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · ms:39299
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · ms:59503
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · ms:34176
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · ms:37919
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · ms:37738
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · ms:39308
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · ms:52982

## Mutation Log
(empty until execute)
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/mcp.go` · legacy results are decorated too · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · covers:a legacy result is byte-identical
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/tools.go` · the pre-apply floor ignores the reserve · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · covers:every in-call measurement reads the reserved ceiling
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/mcp.go` · -32022 carries no supported list · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · covers:an unknown version is -32022 with the list
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/mcp.go` · the clientCapabilities check never fires · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · covers:a modern request without clientCapabilities is -32602
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/mcp.go` · initialize takes its era from _meta · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · covers:the era precedence is fixed
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/mcp.go` · a modern tools/list carries no ttlMs · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · covers:a modern tools/list carries caching hints
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/tools.go` · a modern tools/call result is not decorated · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · covers:a modern result is decorated
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/mcp.go` · server/discover is unrouted: §125 must go red · acceptance-sha256:142597d95ea6f8a72682fb1301358b03cff3ed6c53da2d38d2b9a9a9a26ecc04 · covers:the binary serves a modern probe

## Invariants

- A request without the `_meta` version is answered as before T4, byte for byte after id normalisation.
- `ping` and `initialize` behave as after T2 in every era.
- The ledger, the advertised ceiling (`anthropic/maxResultSizeChars`) and the two-tool cargo (ADR-044) are unchanged.

## Risks

- The per-call limit touches every ceiling comparison; the enumeration command above is re-run after the edit and every hit must read the per-call value.
- The golden must be captured on T1–T3's tree before any T4 edit; S1 does it first and commits it beside the test.

## Out of Scope

- `subscriptions/listen`, `notifications/tools/list_changed`, MRTR, `logLevel` (permanent: boundary: the parent ADR's Out of Scope; the tool list never changes during a process's life)

## Stop Condition

Stop if the modern path needs connection state, a new tool, or a change to what a legacy host receives.
