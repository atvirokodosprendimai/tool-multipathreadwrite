# Task ADR-067-T2: `initialize` negotiates; `serverInfo` says what mrw is; the wire says what failed; contract §123

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `supportedVersions` constant (legacy and modern lists); `serverInfo()` helper (name, title, version, description)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a spoken requested version is echoed`, `an unspoken one gets the latest`, `serverInfo carries title and description`, `an encode failure is internal`, `a null id is refused`, `the binary answers the host's version`

## Goal

- `initialize` reads `params.protocolVersion`. It answers that version when mrw speaks it (`2025-11-25`, `2025-06-18`), and `2025-11-25` otherwise.
- `serverInfo` carries `name`, `title` (`mrw`), `version` and a one-line `description`, from one helper T4 reuses.
- `resultResponse` reports an encode failure as `-32603` (`internal/mcp/mcp.go:418`).
- `handle` answers a request whose `id` is JSON `null` with `-32600` and a null id, and does not dispatch it (`mcp.go:174`).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/mcp.go` | edit | `protocolVersion` becomes `supportedVersions` (legacy list, latest first) with a `latestLegacy` accessor; `initializeResult(params)`; `serverInfo()`; `:418` code; null-id check |
| `internal/mcp/negotiate_test.go` | new | the five tests below |
| `internal/mcp/mcp_test.go` | edit | any assertion pinning `"2025-06-18"` as the only answer follows the negotiation |
| `scripts/contract.sh` | edit | §123; any existing row pinning `2025-06-18` in the initialize answer follows |

## Ordered Steps

1. [S1] Write the tests and confirm each RED on `main` on an assertion. [proof: mutation]
   - `TestInitializeEchoesASupportedRequestedVersion`: `2025-11-25` → `2025-11-25`, and `2025-06-18` → `2025-06-18`.
   - `TestInitializeAnswersItsLatestVersionForAnUnknownOne`: `2024-11-05` and a missing version → `2025-11-25`.
   - `TestServerInfoCarriesATitleAndADescription`: both non-empty, and `version` is `Version`.
   - `TestAResultThatCannotBeEncodedIsAnInternalError`: `resultResponse` on a value `json.Marshal` refuses (a `chan int`) → code `-32603`.
   - `TestARequestWithANullIdIsInvalid`: `{"jsonrpc":"2.0","id":null,"method":"tools/list"}` → `-32600` with a null id, and no tools list in the answer.
2. [S2] Implement; confirm GREEN, and that the rest of `internal/mcp` stays green. [proof: mutation]
   Mutants:
   - `initialize` ignores the requested version: kills the echo test;
   - the fallback answers `2025-06-18`: kills the latest test;
   - `:418` reverts to `-32600`: kills the encode test;
   - the null-id check removed: kills the null-id test.
3. [S3] §123 through the binary: an `initialize` asking `2025-11-25` is answered `2025-11-25`, and one asking `2025-06-18` is answered `2025-06-18` (the pair). RED against v1.23.0 in a mini-harness, GREEN in the full `./scripts/contract.sh`. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 123\. ' scripts/contract.sh \
  && go test ./internal/mcp/ -count=1 -v \
    -run 'TestInitializeEchoesASupportedRequestedVersion|TestInitializeAnswersItsLatestVersionForAnUnknownOne|TestServerInfoCarriesATitleAndADescription|TestAResultThatCannotBeEncodedIsAnInternalError|TestARequestWithANullIdIsInvalid' 2>&1 | tee /tmp/adr067-t2.out \
  && grep -q '^--- PASS: TestInitializeEchoesASupportedRequestedVersion ' /tmp/adr067-t2.out \
  && grep -q '^--- PASS: TestInitializeAnswersItsLatestVersionForAnUnknownOne ' /tmp/adr067-t2.out \
  && grep -q '^--- PASS: TestServerInfoCarriesATitleAndADescription ' /tmp/adr067-t2.out \
  && grep -q '^--- PASS: TestAResultThatCannotBeEncodedIsAnInternalError ' /tmp/adr067-t2.out \
  && grep -q '^--- PASS: TestARequestWithANullIdIsInvalid ' /tmp/adr067-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr067-t2.out \
  && ./scripts/contract.sh > /tmp/adr067-t2-contract.out 2>&1 \
  && grep -q '^  PASS  initialize answers the version the host asked for' /tmp/adr067-t2-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines cmd/mrw \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines cmd/mrw)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l internal/mcp)" ] \
  && go vet ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestInitializeEchoesASupportedRequestedVersion` | `internal/mcp/negotiate_test.go` | a spoken requested version is answered as asked | — | S1, S2 |
| `TestInitializeAnswersItsLatestVersionForAnUnknownOne` | `internal/mcp/negotiate_test.go` | an unspoken or missing version gets `2025-11-25` | — | S1, S2 |
| `TestServerInfoCarriesATitleAndADescription` | `internal/mcp/negotiate_test.go` | `serverInfo` says what mrw is | — | S1, S2 |
| `TestAResultThatCannotBeEncodedIsAnInternalError` | `internal/mcp/negotiate_test.go` | an encode failure is `-32603` | — | S1, S2 |
| `TestARequestWithANullIdIsInvalid` | `internal/mcp/negotiate_test.go` | a null id is refused, not dispatched | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the five tests and §123 |
| 2 — something selects it | `handle` passes `initialize`'s params to `initializeResult` |
| 3 — the caller can discover it | the answer's `protocolVersion` and `serverInfo` |
| 4 — it is used | Claude Code 2.1.281 asks `2025-11-25` on every start (wire capture, 2026-09-24) |

## Verification Log
(empty until execute)
- 2026-09-24 · ae040e4* · exit 1 · `set -o pipefail …` · acceptance-sha256:21f0587e50e331ceec556e664182d4f9a3ded14bad95a93e08a19c1239974c47 · ms:110 · test-lock-sha256:4c7a37b541f4e17935055b112ad48d1a7dfbf0bf05f265c4917051f08072159a · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9uZWdvdGlhdGVfdGVzdC5nbwlUZXN0QVJlcXVlc3RXaXRoQU51bGxJZElzSW52YWxpZAlkZDdmYzllZDMzNzg3ZDc0Y2NjNjg4NjJkNmM2YTVkZjI0MjI5OTk0OGM2ZTg3ZWZmNDkzNjU0MmVmZmViN2U0CmJvZHkJaW50ZXJuYWwvbWNwL25lZ290aWF0ZV90ZXN0LmdvCVRlc3RBUmVzdWx0VGhhdENhbm5vdEJlRW5jb2RlZElzQW5JbnRlcm5hbEVycm9yCTYzNjA4N2ZhZDI1ZTFkOWE1MGI1ZGNmZDg5NjU1M2M0ZWYwN2U4OGY4MTYzMWZkYTBlZjI5ZDM3MjJjN2FkYjMKYm9keQlpbnRlcm5hbC9tY3AvbmVnb3RpYXRlX3Rlc3QuZ28JVGVzdEluaXRpYWxpemVBbnN3ZXJzSXRzTGF0ZXN0VmVyc2lvbkZvckFuVW5rbm93bk9uZQliY2I2ZGNhMmNhODU5M2VmMDJhNjM0N2U4ZTZkMWEyOTcxYjEwNWZmYTBlMDk1ODhiYzEwYzEzMTExOTFhMTdkCmJvZHkJaW50ZXJuYWwvbWNwL25lZ290aWF0ZV90ZXN0LmdvCVRlc3RJbml0aWFsaXplRWNob2VzQVN1cHBvcnRlZFJlcXVlc3RlZFZlcnNpb24JOTlhOTIyYmUxNGVjODc1OTU1Y2U1ODQ4YzI2NTYyODg0NThjNzliZjA2Y2NkZDhlMDU2YWI2Y2JlZDQxNTk4ZQpib2R5CWludGVybmFsL21jcC9uZWdvdGlhdGVfdGVzdC5nbwlUZXN0U2VydmVySW5mb0NhcnJpZXNBVGl0bGVBbmRBRGVzY3JpcHRpb24JZDg3MWJlMTViYmEwMTNkMmI2MjM2YWU1YjJmYzU5ZmRhODY5MzBmYzlmOTQxZTg1ZWM3MjAxZjMwZGE3YzViMA
  ```
  ```
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:21f0587e50e331ceec556e664182d4f9a3ded14bad95a93e08a19c1239974c47 · ms:56654
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:21f0587e50e331ceec556e664182d4f9a3ded14bad95a93e08a19c1239974c47 · ms:49916
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:21f0587e50e331ceec556e664182d4f9a3ded14bad95a93e08a19c1239974c47 · ms:37803
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:21f0587e50e331ceec556e664182d4f9a3ded14bad95a93e08a19c1239974c47 · ms:30491
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:21f0587e50e331ceec556e664182d4f9a3ded14bad95a93e08a19c1239974c47 · ms:35202
- 2026-09-24 · ae040e4* · exit 0 · `set -o pipefail …` · acceptance-sha256:21f0587e50e331ceec556e664182d4f9a3ded14bad95a93e08a19c1239974c47 · ms:35027

## Mutation Log
(empty until execute)
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/mcp.go` · negotiate never echoes a spoken version · acceptance-sha256:21f0587e50e331ceec556e664182d4f9a3ded14bad95a93e08a19c1239974c47 · covers:a spoken requested version is echoed
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/mcp.go` · an unspoken version gets 2025-06-18, not the latest · acceptance-sha256:21f0587e50e331ceec556e664182d4f9a3ded14bad95a93e08a19c1239974c47 · covers:an unspoken one gets the latest
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/mcp.go` · serverInfo loses its title · acceptance-sha256:21f0587e50e331ceec556e664182d4f9a3ded14bad95a93e08a19c1239974c47 · covers:serverInfo carries title and description
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/mcp.go` · an encode failure reverts to -32600 · acceptance-sha256:21f0587e50e331ceec556e664182d4f9a3ded14bad95a93e08a19c1239974c47 · covers:an encode failure is internal
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/mcp.go` · the null-id check never fires · acceptance-sha256:21f0587e50e331ceec556e664182d4f9a3ded14bad95a93e08a19c1239974c47 · covers:a null id is refused
- 2026-09-24 · ae040e4* · mutant killed · exit 1 · `internal/mcp/mcp.go` · initialize answers 2025-06-18 whatever was asked (v1.23.0): §123 must go red · acceptance-sha256:21f0587e50e331ceec556e664182d4f9a3ded14bad95a93e08a19c1239974c47 · covers:the binary answers the host's version

## Invariants

- `capabilities` and `instructions` in the `initialize` answer are unchanged.
- A request with no id stays a notification and is never answered.

## Risks

- An existing test or contract row pins `2025-06-18` as the only answer; S2 finds it with `grep -rn '2025-06-18' internal/mcp scripts/contract.sh` and moves it to the negotiation.

## Out of Scope

- Answering `2025-03-26` or `2024-11-05` as asked (permanent: boundary: mrw has never implemented those revisions' batching and has not been measured against them; the legacy lifecycle lets a server answer its own latest)

## Stop Condition

Stop if a host measured on this machine rejects `2025-11-25` in the answer.
