# ADR-093: An MCP tool refuses an argument it does not declare

**Status:** Accepted
**Accepted:** 2026-09-29 by Zy — "ADR-093 MCP args", selected under "Which records do you accept as written, so I can execute them?", on the record as drafted after a Codex review
**Date:** 2026-09-29
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-009, ADR-010, ADR-011, ADR-016, ADR-017, ADR-025, ADR-032, ADR-044, ADR-054, ADR-067, ADR-075, ADR-076, ADR-078, ADR-088, ADR-089, ADR-092, docs/adr/BACKLOG.md
**Governs:** `internal/mcp/tools.go`, `internal/mcp/mcp.go`
**Enforced-by:** `internal/mcp/undeclared093_test.go::TestAnUndeclaredArgumentIsRefusedByName` (written by T1 [S1]; the record merges with it)
**Invalidates:** ADR-067 — Decision 3's "Acknowledgements sent with it are still promoted, as today" (line 74) does not hold for this refusal. It is answered before dispatch, so an `ack` sent beside an undeclared argument is not promoted. A decode failure already behaves that way: `internal/mcp/tools.go:255-257` and `:553-555` return before `promote` at `:260` and `:561`.
**Served-path change:** A `tools/call` to `mrw_read` or `mrw_write` whose `arguments` carry a key the tool does not declare (`check`, `then`, `max_lines`, or `Plan` for `plan`) is refused with an `isError` result. The result names the key, the arguments the tool takes and the CLI flag that has the capability, and nothing is served, written or acknowledged. Until now such a call returned an ordinary answer that ignored the key. `tools/list` declares `additionalProperties: false` on both input schemas.

## Context

**The survey.** Item C1 of the 2026-09-29 gap survey of v1.31.0 (agentsmemory drawer `349dbd2b`,
wing `wing_tool-multipathreadwrite`, room `findings`) found this: `mrw_write` and `mrw_read` decode
their arguments with a plain `json.Unmarshal`, so a key the tool does not declare is dropped without
a word. The survey's critic probed it from Claude Code: `mrw_read` with `max_lines: 1` served three
lines. That result does not say whether Claude Code forwarded `max_lines` and mrw ignored it, or
whether the host dropped the key before sending. The reproduction below goes to the binary directly.
The survey cited `internal/mcp/tools.go:541-555`. Re-read at `8cbb89e`, that range holds only
the write decoder (`writeTool`, decode at `:553`). The read decoder is `readTool`, `:231-257`, with
its decode at `:255`, and the survey's range missed it.

**Reproduced 2026-09-29 against the installed v1.31.0 (`2ea8bd5`).** Two `tools/call` requests were
piped to `mrw --root DIR mcp` on a scratch root:
- `mrw_read {"specs":["main.go:1-3"],"max_lines":1,"stat":true}` served lines 1-3 with their
  content and pending checkpoints, and no `isError`. `max_lines` and `stat` did nothing.
- `mrw_write {"plan":…,"check":true,"then":["vet"],"Dry_Run":true}` returned an ordinary receipt
  with `dry_run: true`, no `isError`, and nothing about a check or a step. `check` and `then` did
  nothing. `Dry_Run`, which the schema does not declare, was honoured, because `encoding/json`
  matches a key to a field case-insensitively.

A caller that asks for verification (`check`, `then`, `then_sh`), a line cap (`max_lines`) or a
stat-only answer (`stat`) therefore gets an ordinary answer that did not do what it asked. For
`mrw_write` that is a green result where verification was requested. That is the failure this
project exists to refuse: a write whose effect is not what the caller believes is invisible.

**Why the schema did not stop it.** The advertised input schemas (`internal/mcp/mcp.go:423-464`,
`:491-557`) name their properties. Neither says `additionalProperties: false`, so an extra key is
valid against both. Nobody has measured whether Claude Code forwards an undeclared key, or whether
any host enforces a closed schema.

**Two lists of the accepted keys, and nothing binds them.** The schema maps in `mcp.go` list the
arguments, and so do the anonymous decode structs in `tools.go` (`:231-251`, `:542-549`). Today
they agree: six names each. A check keyed on either list would still silently drop a key the other
lacks, which is the same defect in the other direction.

**The class is every tool the MCP server dispatches, and the `arguments` object it decodes.**
Enumerated 2026-09-29 at `8cbb89e`:
- tools dispatched: `grep -n 'case "mrw_' internal/mcp/tools.go` gives `:177` `mrw_read` and
  `:179` `mrw_write`, 2 in all;
- tools advertised: `grep -n 'Name: *"mrw_' internal/mcp/mcp.go` gives `:392` and `:467`, 2 in all;
- decoders of `arguments`: `grep -n 'json.Unmarshal(args' internal/mcp/tools.go` gives three:
  - `:255` (read) and `:553` (write);
  - `:1774` in `nonUTF8Arg` (ADR-078). It decodes into a map only to name an argument that is not
    UTF-8, and never acts on one.

Every other `json.Unmarshal` in the package decodes something other than a tool's arguments, and
each stays as it is:
- `tools.go:150` (`callParams`) and `mcp.go:73` (`requestEra`) decode the `params` object, which
  the protocol owns. Claude Code 2.1.281 puts `_meta.progressToken` and
  `_meta["claudecode/toolUseId"]` there (ADR-067, Context), so closing `params` would refuse the one
  measured host.
- `mcp.go:258` (the JSON-RPC message), `:81` (the `_meta` version value) and `:588` (the request
  id) decode the envelope.
- `mcp.go:358` decodes `initialize` params, and is lenient on purpose: "the handshake is how a host
  learns what this server speaks" (`:356-357`).
- `ack.go:430` decodes mrw's own pending-checkpoint store, not caller input.

**One wrapper sits in front of the class, and it is covered.** The opencode plugin (ADR-089) calls
`mrw mcp` for both tools. Its `set()` (`cmd/opencode/mrw-plugin/src/index.ts:100-104`) forwards only
a fixed map of keys (`:128`, `:156`), and the schemas declare all of them. The plugin cannot send an
undeclared key to mrw, so this record's refusal never reaches it.

**The same silent drop one layer up.** A model's `max_lines` sent to the plugin's tool never reaches
mrw at all. What opencode does with an argument its zod shape does not declare was not read.
`@opencode-ai/plugin` 1.18.32's `tool()` is the identity function (`dist/tool.js:2-4`), so that
validation is opencode's own. It is deferred below.

**The tests send declared keys.** A tally on 2026-09-29 counted the keys in the inline `arguments`
objects of `internal/mcp/*_test.go`, `cmd/mrw` and `scripts/contract.sh`: `specs` 43, `plan` 34,
`ack` 12, `grep` 10, `dry_run` 9, `format` 6, `exclude` 4, `echo_pad` 2. Every one is declared.
Arguments a test builds as a Go map were not in that tally, which is why the fences run the whole
`internal/mcp` package. `legacyTranscript` (`internal/mcp/era_test.go:176-187`) sends only declared
keys.

## Existing Primitives Audit

- **`errorResult`** (`tools.go:939`), the `isError` result ADR-067 routes a caller's argument
  mistakes through. **Reused** for the refusal.
- **The funnel `withinCeiling(decorateCall(…))`** (`tools.go:194`, `:211`; ADR-032, ADR-067).
  **Reused.** The refusal leaves through it, so it is bounded and era-decorated like every answer.
- **`tools()`** (`mcp.go:389`), whose `InputSchema` properties are the names a host is shown.
  **Reused as the one list** of declared arguments. The refusal reads the same map a host reads.
- **`nonUTF8Arg`** (`tools.go:1769`, ADR-078). **Kept first.** Arguments whose raw bytes are not
  UTF-8 still get its refusal, before the key check decodes them. A key that is valid UTF-8 is named
  as decoded, a genuine U+FFFD included: ADR-078 rejected refusing a decoded U+FFFD (`:80`).
- **ADR-016's two routing checks**: every flag the wire names is in that subcommand's `--help`
  (`internal/mcp/mcp_test.go:565-576`, through `cliHelp`), and no flag the wire calls CLI-only is a
  declared argument (`TestTheRoutingClaimsOnlyRealExclusives`, `:738`). **Reused** for the
  refusal's routing text.
- **The legacy golden and its regeneration switch** (`era_test.go:214-245`,
  `MRW_UPDATE_LEGACY_GOLDEN`). **Reused.** It has been re-captured five times, each time with a
  diff limited to the change a record named.
- **`json.Decoder.DisallowUnknownFields`**: audited and **not taken**. See Alternatives.
- **`SchemaOf`** (`schema.go:23`), which derives the output schema from its type: audited and
  **not taken** for the input schemas. See Alternatives.

## Decision

**1. `callTool` refuses an argument its tool does not declare, before it dispatches.**
- A tool's declared arguments are the property names of the `inputSchema` it advertises in
  `tools()`. There is no second list.
- A key is declared only if it matches a property name exactly. `Plan`, `DRY_RUN` and a `_meta`
  inside `arguments` are all undeclared. `encoding/json`'s case-insensitive match is not the
  contract.
- The check sits in `callTool` (`tools.go:148`), after the era setup (`:166-172`) and before the
  `switch` (`:176`). Placed after the era setup, the refusal is decorated for its era. It does not
  fire for a tool name it does not know, so an unknown tool is still `-32602` (`:181-182`,
  ADR-067), whatever keys it carries. It does not fire when the arguments are not valid UTF-8, so
  ADR-078's refusal still answers those.
- The refusal is an `errorResult` assigned to `res`, and the `switch` is skipped. It leaves through
  the existing `return withinCeiling(decorateCall(res))` at `:194`, the one funnel ADR-032 put there
  because enumerating return sites let one escape (`:187-193`). No return site is added. At a
  ceiling too small for it, the funnel's own refusal replaces this text (`:216`), as it does for
  every answer. Its text names:
  - the tool and every undeclared key, each quoted, sorted;
  - that nothing was done, including that an `ack` sent with the call was not recorded;
  - every argument the tool takes, sorted, and the server's `Version`, because a key a later mrw
    declares is refused by this one;
  - where the CLI has the capability. For `mrw_write`, that this tool runs no check and no step
    (ADR-054, ADR-092), while `mrw write` runs the project's check after a write and takes
    `--then`. For `mrw_read`, the read flags only the CLI has, such as `--files-from`,
    `--max-lines` and `--stat`.
- Every `--flag` the text names must appear in that subcommand's own `--help`. None may be an
  argument the tool declares. These are ADR-016's two checks, applied to this text.
- Nothing is done: no `ack` is promoted, nothing is served or recorded in the ledger, and nothing
  is written. The refusal is not counted in the ADR-009 tally, just as a decode failure is not
  today (`tools.go:553-555` records nothing).

**2. The decoders decode exactly what the schemas declare.** The two anonymous structs become the
named types `readArgs` and `writeArgs`. A class test binds each tool's type's `json` tags to its
schema's properties. It does so through a table in the test file that must name every tool
`tools()` lists, so it fails on a third tool without an entry and on a declared property without a
field. The table lives in a `_test.go` file because ADR-088 refuses production code that only a
test reaches.

**3. Both input schemas declare `additionalProperties: false`.** The schema then describes what the
server enforces. A host that validates can refuse before sending, and a host that does not still
meets the server's refusal. Either way the server is the enforcement.

**4. A key from a later version is refused, by name.** Suppose a later mrw declares a key and a
caller sends it to this one. This one refuses it, naming the key, what this version takes and this
version's number. That is intended: an older server silently ignoring a newer caller's `check` is
exactly the defect this record removes. A caller that spans versions reads `tools/list`, which is
now the list the server enforces. There is no escape hatch, such as an `x-` prefix or a
pass-through `_meta` inside `arguments`. Anything a caller may send is declared.

**5. It ships as a minor release.** Input that used to be accepted is now refused, which changes the
public MCP contract. It ships in the next minor tag, as ADR-067's envelope change did (`7da61c1`,
first tagged `v1.24.0`). What a caller sees:
- An MCP host that sends an undeclared key gets a `tools/call` result with `isError: true`. Its
  first text block names the key. It used to get an ordinary answer that ignored the key.
- The opencode plugin sees nothing new, because it sends only declared keys.
- A caller that sent `Dry_Run` or `Plan` gets a refusal where the key used to be honoured. The
  schema never declared that spelling.

**What would make this decision fail:** a host that adds keys of its own to a tool's `arguments`,
because every call from it would be refused. No measured host does: Claude Code 2.1.281 put its
additions in `params._meta` (ADR-067, Context). Whether it forwards a key the schema does not declare
is unmeasured, and so is every host other than Claude Code. T1's Stop Condition and the Risks table
say what to do if one is found.

## Alternatives Considered

- **`json.Decoder.DisallowUnknownFields` at each decoder.** Rejected, for three reasons:
  - it also matches keys case-insensitively: measured 2026-09-29 on go1.27.1,
    `{"Plan":"x"}` decoded into a `json:"plan"` field with it set;
  - it goes at each decoder, so a third tool can forget it. ADR-032's funnel records that failure
    at `tools.go:187-193`: enumerating return sites is how one was missed;
  - its error, `json: unknown field "check"`, names no accepted argument and no CLI route.
- **Serve or apply as asked, and warn about the ignored key.** Rejected. The call still did
  something other than what was asked, and a caller that does not read warnings gets today's
  defect. For `mrw_write`, a requested check that never ran would come back `applied` with a note:
  a green answer where verification was asked. ADR-025 made the same choice for a read that served
  nothing.
- **Give the MCP tools the arguments (`check`, `then`, `max_lines`).** Not this record. ADR-054 (Out
  of Scope, line 132) keeps the check off this surface permanently. ADR-092 (Out of Scope, line 192)
  DEFERS steps here to `docs/adr/BACKLOG.md` ("From ADR-092", `:2284`), to be armed when an MCP-only
  host asks for verified writes, because a permanent MCP boundary contradicted M's direction
  (ADR-016). A line cap and a stat-only read are deferred below. Until one of them ships, the
  refusal says the tool does not do it and names the CLI flag that does.
- **Close the schemas and rely on hosts.** Rejected. Nobody has measured whether any host enforces a
  schema before sending, and only the server sees every call that reaches it.
- **Derive the input schemas from `readArgs` and `writeArgs` with `SchemaOf`, as the output schema
  is derived.** That would give one list by construction. Not taken now:
  - `SchemaOf` widens a nil slice to `["array","null"]` (`schema.go:141`), which would change the
    advertised type of `specs`, `ack` and `exclude`;
  - the examples, defaults and enum would need a description table like `writeDescriptions`.

  Decision 2's class test gives the same agreement with a smaller diff.
- **Refuse inside each tool, after decoding.** Rejected. It needs the check at both sites, and a
  funnel covers a third tool by construction.

## Component / Boundary Impact

- `internal/mcp` owns every code change:
  - `tools.go`: the check in `callTool`, `undeclaredRefusal`, `readArgs` and `writeArgs`;
  - `mcp.go`: the two input schemas.
- Tests: `internal/mcp/undeclared093_test.go` (new), and one row in the ceiling table of
  `internal/mcp/limit_test.go`. Also `internal/mcp/era_test.go`'s comment and
  `internal/mcp/testdata/legacy_golden.jsonl`: captured before ADR-067 T4 and re-captured by
  ADR-070, ADR-075, ADR-076, ADR-090 and ADR-091 (`era_test.go:214-223`). It is regenerated here
  with a diff limited to the two new keys.
- Docs: `README.md` "Use it from an MCP host", and the MCP paragraph of `AGENTS.md`.
- `scripts/contract.sh` §175 and §176.
- These stay byte-identical against the merge-base: `internal/read`, `internal/apply`,
  `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines` and
  `cmd/mrw`. `go.mod` keeps one requirement. The opencode plugin is unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `tools/call` arguments of `mrw_read` and `mrw_write` | an undeclared key, matched exactly, is an `isError` result naming it, the declared arguments, the version and the CLI route; nothing is done | T1 | MCP hosts; models; the opencode plugin (never sends one) |
| `tools/list` `inputSchema` | `additionalProperties: false` on both tools | T2 | MCP hosts that validate |
| `README.md`, `AGENTS.md` | one sentence each naming the refusal | T2 | readers; the centralised `mrw` skill mirrors `AGENTS.md` |
| contract §175, §176 | one row pair each | T1, T2 | CI; the full suite before each commit (lifecycle §6) |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `undeclaredRefusal`, the check in `callTool` that a closed schema advertises | T1 | T2 | No: T2 advertises what T1 enforces, and a closed schema without T1 would promise a refusal the server does not make |

## Implementation

See `docs/adr/ADR-093-an-mcp-tool-refuses-an-argument-it-does-not-declare/tasks/README.md`.

## Consequences

- **Positive:**
  - a caller that asks an MCP tool for something the tool does not do is told so in the same turn,
    with the CLI route where one exists;
  - the schema and the decoders cannot drift apart unnoticed;
  - the advertised schema says what is enforced.
- **Negative:**
  - a host that adds keys to `arguments` would be refused on every call (none measured);
  - a newer caller talking to an older server is refused rather than partly served;
  - a caller relying on a case-variant spelling is refused;
  - one line of the legacy golden changes, at two places.
- **Neutral:** no receipt field, tally outcome or engine file changes. Every argument a caller can
  usefully send today still works.

## Out of Scope

- Closing `params` to keys such as `_meta` (permanent: fact: Claude Code 2.1.281 sends `_meta.progressToken` and `_meta["claudecode/toolUseId"]` in `params`, so a closed `params` would refuse the one measured host; citation: file `docs/adr/ADR-067-the-mcp-surface-speaks-the-current-protocol.md:23`)
- Checking the keys of `initialize` params (permanent: fact: `initialize` answers missing or malformed params with the latest version, on purpose; citation: file `internal/mcp/mcp.go:356`)
- Giving `mrw_write` a check (permanent: fact: MCP `mrw_write` runs no check by ADR-054's decision, which cites ADR-044's two tools; citation: file `docs/adr/ADR-054-a-write-that-applied-can-still-leave-a-broken-tree.md:132`)
- Giving `mrw_write` steps (deferred: docs/adr/BACKLOG.md — "From ADR-092": steps on the MCP surface, armed when an MCP-only host asks for verified writes)
- Giving `mrw_read` a line cap or a stat-only answer (deferred: docs/adr/BACKLOG.md — "From ADR-093": arm when an MCP caller shows it needs `max_lines` or `stat` rather than the ceiling and the paging it has)
- Duplicate keys in one `arguments` object (permanent: fact: RFC 8259 §4 says object names SHOULD be unique and leaves duplicates' behaviour to the implementation; this record leaves `encoding/json`'s handling of a repeated declared key as it is, including that a later `null` leaves a string field unchanged; citation: url https://www.rfc-editor.org/rfc/rfc8259)
- The values of declared arguments (permanent: boundary: ADR-067 Decision 3 owns a caller's value mistakes; this record is about names)
- Counting the refusal in ADR-009's tally (permanent: boundary: a decode refusal is not counted today, `internal/mcp/tools.go:553-555`, and an undeclared argument is the same kind of mistake, made before any plan is parsed)
- What opencode does with a tool argument the plugin's zod shape does not declare (deferred: docs/adr/BACKLOG.md — "From ADR-093": measure whether opencode strips or refuses it before the plugin's `execute` runs)
- Whether a host, Claude Code included, forwards an undeclared key, or enforces `additionalProperties: false` before sending (deferred: docs/adr/BACKLOG.md — "From ADR-093": capture a host's wire for a call carrying an undeclared key)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A host adds keys of its own to `arguments`, and every call from it is refused | Low: Claude Code 2.1.281 puts its keys in `params._meta` | High | The refusal names the key, so the first call reports it; T1's Stop Condition; a host's wire capture is deferred to BACKLOG |
| A test or caller in the tree builds an `arguments` map with an undeclared key | Low: the inline tally of 2026-09-29 found none | Med | Both fences run the whole `internal/mcp` package, and the full contract suite runs before each commit (lifecycle §6); T1 stops when the key is a deliberate caller shape rather than a typo |
| The refusal's routing names a flag the CLI later renames | Low | Med | `TestTheRefusalRoutesOnlyToFlagsTheCLIHas` checks each flag against that subcommand's `--help`, as ADR-016 does for the handshake |
| The golden regeneration hides a second change | Low | Med | T2's fence strips the new key from the regenerated golden and the merge-base copy, compares them, and counts the key: exactly two. That is a branch-time check: once merged, the merge-base is the commit itself, so the Verification Log line written at execution is the lasting receipt |
| A host that validates shows its own error instead of mrw's | Low | Low | The same call is refused either way; the server's text is the fuller one |
| The refusal exceeds a small ceiling | Low | Low | It leaves through `withinCeiling`, which refuses legibly |
| A host strips an undeclared key before sending, so its model never meets the refusal | Unmeasured | Med | The opencode shape, one layer up; deferred to BACKLOG with the wire capture; T2's closed schema at least tells such a host what is accepted |

## Rollback

Revert the `tools.go` and `mcp.go` changes, the tests, §175 and §176, the golden and the two doc
sentences. Nothing persistent moves: no ledger, receipt field, tally outcome or state file changes
shape. A caller that adapted by dropping undeclared keys is unaffected by the revert.

## Follow-ups

- [ ] Update the centralised `mrw` skill from `AGENTS.md` at the release that ships this record.
- [x] File the three `docs/adr/BACKLOG.md` entries under "From ADR-093" in the commit that lands this record.
