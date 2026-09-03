# Task ADR-010-T1: A stdio JSON-RPC transport, hand-rolled, reachable

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `mcp.Serve(in io.Reader, out io.Writer, root string) error`, the JSON-RPC frame, `initialize` and `tools/list`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the newline-delimited framing`, `the absence of a new go.mod requirement`, `the subcommand registration`

## Goal

`mrw mcp` starts a stdio MCP server that completes a host's `initialize` handshake and lists its
tools, with no new module dependency.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/mcp.go` | add | the frame, the JSON-RPC envelope, `initialize`, `tools/list` |
| `internal/mcp/mcp_test.go` | add | its tests |
| `cmd/mrw/main.go` | edit | **THE CALL SITE.** `mcpCmd()` and its entry in the `Commands` slice — the line that makes the package reachable. `rooted.Descendable` was built, tested and deleted unused on 2026-09-03 for want of exactly this |
| `go.mod` | **must not change** | the go/no-go condition, checked by the fence rather than by intention |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): a framed `initialize` request gets a framed
   response carrying a protocol version; `tools/list` names both tools; a malformed frame is
   answered with a JSON-RPC error rather than a panic or a silent close.
2. [S2] Implement the framing the MCP stdio transport actually specifies: **one JSON-RPC message per
   line, delimited by newlines, and no embedded newline inside a message.** There is no
   `Content-Length` header in MCP stdio — that is Language Server Protocol, and a server that speaks
   it fails every host's `initialize` while passing any test written to the same mistake. Source:
   https://modelcontextprotocol.io/specification/2025-06-18/basic/transports#stdio — *"Messages are
   delimited by newlines, and MUST NOT contain embedded newlines."* The same page carries two rules
   that bind mrw specifically, because this binary prints to stdout everywhere else: the server
   *"MUST NOT write anything to its `stdout` that is not a valid MCP message"*, and it *"MAY write
   UTF-8 strings to its standard error"* for logging. A stray `fmt.Println` on the server path is a
   protocol violation, not a cosmetic one. [proof: acceptance]
3. [S3] Implement the JSON-RPC envelope to the letter, because a host validates it: every message
   carries `jsonrpc: "2.0"`; a request carries an `id` and gets exactly one response with the same
   `id`; an error object carries BOTH `code` and `message`, not a code alone. A NOTIFICATION — a
   message with no `id` — gets NO response at all, and `notifications/initialized` is the one this
   server must accept silently; answering it is a protocol violation that some hosts treat as fatal.
   A request with an unknown method is an error RESPONSE, never a dropped connection — a host that
   sees a closed pipe cannot tell a crash from a refusal.
4. [S4] Answer `initialize` with the WHOLE response the lifecycle requires — `protocolVersion`,
   `serverInfo` (name and version), and `capabilities` declaring `tools` — then answer `tools/list`.
   A response carrying only a protocol version parses as JSON and still fails negotiation, which is
   the failure this step exists to prevent. Source:
   https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle. The tool SCHEMAS are
   declared here and their handlers land in T2, so `tools/list` is honest from the first commit
   rather than growing later.
5. [S5] Add `mcpCmd()` and **register it** in the `Commands` slice, so `mrw mcp` resolves and
   `mrw --help` lists it. [proof: mutation]
   Answer `ping` too: the spec's ping utility says the receiver MUST respond promptly with an empty
   result, and hosts send them on a timer to check connection health — a server that errors every
   health check is one a host is entitled to drop. It is a base-protocol obligation beside
   `initialize`, not a capability this ADR chose not to implement.
6. [S6] Assert no dependency was added, and that the engine is untouched — the go/no-go conditions.
   The dependency clause counts requirement lines in `go.mod` and checks the one that remains names
   `urfave/cli/v3`; it needs no remote ref, so it answers the same question in a shallow CI checkout
   and on a detached HEAD. The engine clause is `git status --porcelain --untracked-files=all`,
   which sees an untracked new file where a `git diff` does not. A dependency added by accident is
   exactly what this project's one-dependency posture loses quietly. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr010-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr010-t1.out \
  && [ "$(grep -cE '^--- PASS: (TestInitializeCompletesTheHandshake|TestOneMessagePerLineRoundTrips|TestAMalformedFrameIsAnErrorNotAClose|TestToolsListNamesBothTools|TestAnUnknownMethodIsAnErrorResponse|TestTheInitializedNotificationGetsNoResponse|TestOnlyMCPMessagesReachStdout|TestPingIsAnsweredWithAnEmptyResult|TestAJSONArrayIsAnInvalidRequestNotAParseError)\b' /tmp/adr010-t1.out)" = "9" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && grep -q '^require github.com/urfave/cli/v3 ' go.mod \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ] \
  && go build -o /tmp/mrw-t1 ./cmd/mrw \
  && /tmp/mrw-t1 --help > /tmp/adr010-t1-help.out 2>&1 \
  && grep -q '^   mcp ' /tmp/adr010-t1-help.out \
  && go test ./...
```

Every clause names something this task creates, and the fence was RUN before any code existed and
observed to exit non-zero — `internal/mcp` does not compile, so it fails at the first clause. That
check is not optional: three fences in the 2026-09-03 session were green on an untouched tree
because nobody ran them, including one that named a `contract.sh` section which already existed.

**The `--help` clause was rewritten DURING execution, because it could never have gone green.** It
was `/tmp/mrw-t1 --help 2>&1 | grep -q 'mcp'`. Under the `set -o pipefail` on line 1, `grep -q` exits
the moment it matches, `--help` gets SIGPIPE writing the rest of its output, and the pipeline reports
**141**. Measured 2026-09-03 **on darwin**: ten runs, ten 141s. It is NOT deterministic everywhere —
a reviewer measured the same construct exiting 0 three times out of three on Linux, because whether
the writer takes SIGPIPE depends on whether `grep -q` has exited before `--help` finished writing.
That makes the original form worse than merely wrong: it is a fence whose verdict depends on the
machine. The redirect is correct on every platform. So the fence passed its
"run it before you write the code" check for the compile error while carrying a second clause that
was red before the work and would have stayed red after it. Verifying a fence is red BEFORE is
necessary and does not establish that it can go green AFTER; the two need separate checks, and the
first hides the second whenever an earlier clause fails for an honest reason.

The replacement redirects to a file and greps the file, so nothing short-circuits a writer. It also
anchors on `^   mcp ` — the command row `--help` prints — rather than the bare substring `mcp`,
which would match this task's own tool names or any future line mentioning the protocol.

**The named-test count is what makes "green" mean something.** `go test` on a package with no test
files prints `[no test files]` and exits **0**, and `no test files` is not the same string as `no
tests to run` — so the filter above was passable by an empty package. Counting `--- PASS:` lines
for the nine tests by name cannot be satisfied by a package that compiles, by an unrelated test,
or by a test that was skipped. It is deliberately not `-run`: a `-run` regex silently drops any
name it does not match, which is how an ADR-009 fence claimed five tests and ran three.

**The dependency clause went through two wrong forms before this one.** First
`[ "$(grep -c '^\t' go.mod)" = "1" ]`, which counts tab-indented lines rather than requirements and
returns **0** here — red on an untouched tree, which is the same defect as green on one. Then a
diff against `git merge-base HEAD origin/main`, which is right in intent and wrong in practice: it
exits 128 in a clone without that ref, and this repository's CI uses `actions/checkout@v4` at its
default `fetch-depth: 1`. Counting requirement lines needs no history at all.

The `--help` clause is rung 3 — a subcommand a caller cannot discover is not shipped.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestInitializeCompletesTheHandshake` | `internal/mcp/mcp_test.go` | S4 — the response carries `protocolVersion`, `serverInfo` and a `tools` capability, not just a version | — | S1, S4 |
| `TestOneMessagePerLineRoundTrips` | `internal/mcp/mcp_test.go` | S2 — newline-delimited framing, and that no message carries an embedded newline | — | S1, S2 |
| `TestAMalformedFrameIsAnErrorNotAClose` | `internal/mcp/mcp_test.go` | S3 — a host can tell a refusal from a crash | — | S1, S3 |
| `TestToolsListNamesBothTools` | `internal/mcp/mcp_test.go` | S4 — the surface is declared before it is implemented | — | S1, S4 |
| `TestAnUnknownMethodIsAnErrorResponse` | `internal/mcp/mcp_test.go` | S3 — the error object carries a `code` AND a `message` | — | S1, S3 |
| `TestTheInitializedNotificationGetsNoResponse` | `internal/mcp/mcp_test.go` | S3 — a notification has no `id` and must not be answered | — | S1, S3 |
| `TestOnlyMCPMessagesReachStdout` | `internal/mcp/mcp_test.go` | S2 — every line written to stdout parses as an MCP message; diagnostics go to stderr | — | S1, S2 |
| `TestPingIsAnsweredWithAnEmptyResult` | `internal/mcp/mcp_test.go` | S3 — the spec's ping utility: the receiver MUST respond with an empty result, and hosts send these on a timer | — | S1, S3 |
| `TestAJSONArrayIsAnInvalidRequestNotAParseError` | `internal/mcp/mcp_test.go` | S3 — an array is valid JSON and is not a message; refusing it is right, calling it a parse error is not | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the five tests above |
| 2 — something selects it | the `Commands` slice entry (S5); removing it makes the fence's `--help \| grep mcp` clause red, and that clause is inside the Acceptance command rather than beside it |
| 3 — the caller can discover it | `mrw --help` lists `mcp`, asserted by the fence; the README is T3's |
| 4 — it is used | nothing measures this yet — `mrw stats` counts plans, not transports, and counting the transport is not this ADR's question |

## Mutation Log

- 2026-09-03 · ddc21f9 · mutant killed · exit 1 · `cmd/mrw/main.go` · rung 2: unregister the subcommand, so mrw --help no longer lists mcp and the call site is proved to be what makes the package reachable · acceptance-sha256:20f770bdf61525ab802d8bed88c3512009c7ed28cf28d200025d74cf2b861f59
- 2026-09-03 · da9caba · mutant killed · exit 1 · `internal/mcp/mcp.go` · answer notifications instead of staying silent: notifications/initialized has no id and answering it is a protocol violation some hosts treat as fatal · acceptance-sha256:20f770bdf61525ab802d8bed88c3512009c7ed28cf28d200025d74cf2b861f59
- 2026-09-03 · e5e30f0 · mutant killed · exit 1 · `internal/mcp/mcp.go` · drop the newline delimiter: two responses share one line, which is the framing the whole task is named for and which no host can parse · acceptance-sha256:4cac0f7fa998c80b55c98480cfa79411e79e29034080711b3c586d03301c85bc
- 2026-09-03 · ff9b67f · mutant killed · exit 1 · `cmd/mrw/main.go` · rung 2: unregister the subcommand so mrw --help no longer lists mcp · acceptance-sha256:4cac0f7fa998c80b55c98480cfa79411e79e29034080711b3c586d03301c85bc

## Invariants

- `go.mod` declares exactly one requirement and it is `urfave/cli/v3`.
- Everything the server writes to stdout is a valid MCP message; anything else goes to stderr.
- No message written to stdout contains an embedded newline.
- No file under `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`,
  `internal/check` or `internal/state` is added, changed or removed — and the fence checks this
  rather than only asserting it, which it did not do in the first draft.
- A malformed or unknown request produces a JSON-RPC error response; the connection stays open.

## Risks

- Framing is where hosts differ in practice, and the first draft of this task specified LSP framing
  outright. T2's §38 drives a real pipe, but note that a real-pipe test written against the wrong
  frame passes too — so the spec citation in S2 is the actual mitigation, and §38 is what catches
  everything else.
- The envelope is easy to get half-right: valid JSON, wrong lifecycle. Mitigated by naming
  `protocolVersion`, `serverInfo`, `capabilities` and the initialized notification in the steps and
  in two tests, rather than leaving "completes the handshake" to interpretation.
- A long-running mode is new to this binary. Nothing else about it changes, and T2 owns the
  behaviour that touches files.

## Stop Condition

Stop if the transport needs anything from the engine to answer `initialize` or `tools/list`. It must
be startable in a directory with no ledger, no plan and no git — a server that cannot say hello
without touching the tree has coupled the wire to the filesystem, and T2's tools become the only
place that coupling belongs.

## Out of Scope

- The tool handlers — T2's job.
- README documentation — T3's job.

## Verification Log
- 2026-09-03 · 71aa42f* · exit 1 · `set -o pipefail …` · acceptance-sha256:eecd865509376c9fc8b9d3c136714cff0266fd43eee03225454169f021fbd545 · ms:196
  ```
  --- last 4 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp.test]
  internal/mcp/mcp_test.go:17:12: undefined: Serve
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  FAIL
  ```
- 2026-09-03 · 71aa42f* · exit 141 · `set -o pipefail …` · acceptance-sha256:eecd865509376c9fc8b9d3c136714cff0266fd43eee03225454169f021fbd545 · ms:816
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  === RUN   TestToolsListNamesBothTools
  --- PASS: TestToolsListNamesBothTools (0.00s)
  === RUN   TestAnUnknownMethodIsAnErrorResponse
  --- PASS: TestAnUnknownMethodIsAnErrorResponse (0.00s)
  === RUN   TestTheInitializedNotificationGetsNoResponse
  --- PASS: TestTheInitializedNotificationGetsNoResponse (0.00s)
  === RUN   TestOnlyMCPMessagesReachStdout
  --- PASS: TestOnlyMCPMessagesReachStdout (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	(cached)
  ```
- 2026-09-03 · ddc21f9 · exit 0 · `set -o pipefail …` · acceptance-sha256:20f770bdf61525ab802d8bed88c3512009c7ed28cf28d200025d74cf2b861f59 · ms:1630
- 2026-09-03 · da9caba · exit 0 · `set -o pipefail …` · acceptance-sha256:20f770bdf61525ab802d8bed88c3512009c7ed28cf28d200025d74cf2b861f59 · ms:2409
- 2026-09-03 · e5e30f0 · exit 0 · `set -o pipefail …` · acceptance-sha256:4cac0f7fa998c80b55c98480cfa79411e79e29034080711b3c586d03301c85bc · ms:1904
- 2026-09-03 · ff9b67f · exit 0 · `set -o pipefail …` · acceptance-sha256:4cac0f7fa998c80b55c98480cfa79411e79e29034080711b3c586d03301c85bc · ms:1538
