# ADR-149: `mrw version --check-latest` asks for the latest release, and nothing else does

**Status:** Accepted
**Accepted:** 2026-10-14 by Zy — chose "Opt-in network check" (`mrw version --check-latest` asks GitHub for the latest tag; never automatic) over an offline-only staleness line, approved the plan naming this record, and set the goal "implement this plan end to end".
**Date:** 2026-10-14
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-009 (amended: its telemetry rejection), ADR-040 (`mrw version`), ADR-141 (the guidance core)
**Governs:** `cmd/mrw/latest149.go`, `cmd/mrw/main.go`, `README.md`, `AGENTS.md`
**Enforced-by:** `cmd/mrw/latest149_test.go::TestCheckLatestSaysWhetherANewerReleaseExists`
**Invalidates:** none — checked: `mrw version`, `-v` and `--version` print what they printed; ADR-009's tally is untouched and still sends nothing; the one sentence of ADR-009 this narrows carries a pointer here
**Served-path change:** `mrw version --check-latest` is new: it asks GitHub's releases API for the latest release, prints it beside the running version and exits 0 (current, newer, or a build that cannot be compared), 1 (a newer release exists) or 2 (it could not ask); without the flag nothing changes and no command makes a network call

## Context

The 14-session survey of 2026-10-09 (`docs/adr/BACKLOG.md`, "Guidance is about 900 words") asked for a staleness line on `mrw version`: all five Windows repositories had v1.48.0 on PATH while v1.52.0 was current, and eight sessions never acted on the version line they were shown. It happened again on 2026-10-13: the Windows PATH binary was one release behind v1.65.0, and Claude Desktop's `mrw-docs` entry pinned v1.22.3. A binary cannot know it is old from anything it carries: the release is decided after it is built.

The binary makes no network call today (`go.mod` names one requirement; `net` is imported only by `_test.go` files), and ADR-009 rejected telemetry: "a tool that phones home is a different product". Zy chose, on 2026-10-14, a network check the caller asks for by flag, never an automatic one.

**Audit of the class** — *a path in the shipped binary that opens a network connection*: `mrw read --grep '"net(/[a-z]+)?"' --exclude '*_test.go' cmd/ internal/` matches no file at `6b565cb`. After this record it matches one, `cmd/mrw/latest149.go`, and a test holds it to that one.

## Existing Primitives Audit

- `version` (`cmd/mrw/main.go`) is the release stamp, set by `-ldflags "-X main.version=${GITHUB_REF_NAME}"` in `.github/workflows/ci.yml`'s `build` job; a local build says `dev`. `versionString` adds the commit and is what `mrw version` prints first, unchanged.
- The `build` job only runs for a strict `vX.Y.Z` tag, so a release binary's `version` is always strict semver; anything else is a local or hand-stamped build.
- `cli.Exit` and `exitCode` (`cmd/mrw/main.go`) carry an exit status to `main`, which prints `mrw: <message>` on stderr.
- Go's `net/http` never sends a loopback request through a proxy, so a test server on 127.0.0.1 is reached directly whatever `HTTPS_PROXY` says.

## Decision

1. **`mrw version --check-latest`** prints the version line `mrw version` prints, then asks `GET https://api.github.com/repos/atvirokodosprendimai/tool-multipathreadwrite/releases/latest` with `User-Agent: mrw/<version>`, a 5 s timeout, stdlib `net/http`, and nothing in the request but its URL and headers. It reads `tag_name` from the answer and prints one line:
   - `latest release: vB — this mrw is current` when the running version equals it, exit 0;
   - `latest release: vB — this mrw (vA) is newer` when the running version is above it, exit 0;
   - `latest release: vB — this mrw (vA) is older; download https://github.com/atvirokodosprendimai/tool-multipathreadwrite/releases/latest`, then on stderr `mrw: a newer release exists: vB`, exit 1;
   - `latest release: vB — this build (dev) cannot be compared` when `version` is not a strict `vX.Y.Z` (a local build says `dev`), exit 0: there is no verdict to give, and a script that checks a dev build has not been told a release is newer.
2. **It could not ask** — a connection error, a timeout, a status other than 200, a body that is not JSON, or a `tag_name` that is not strict `vX.Y.Z` — prints `mrw: could not ask for the latest release: <reason>` on stderr, exit 2. The version line was already printed.
3. **It does not count releases.** "N releases behind" needs a second request for the release list; the line names both versions instead.
4. **Nothing else makes a network call.** Without the flag, `mrw version` is unchanged; no other command, the MCP server included, opens a connection. A test holds `net/http` to the one file. ADR-009's telemetry rejection stands for everything automatic and gains a pointer to this record.
5. **The API base is a package-level string variable** (`releasesAPI`, default `https://api.github.com`), so a test and the contract row point the built binary at a local server with `-ldflags -X`, as `version` is stamped. No environment variable or flag changes where it asks.

## Alternatives Considered

- **Offline staleness only** (the commit date from build info, or a ledger written by a newer mrw). Not chosen by Zy: it can say a binary is old, not that a newer release exists.
- **Check on every `mrw version`, or on a schedule.** Rejected: that is the automatic call ADR-009 rejected, and it would make a plain version query depend on the network.
- **An environment variable for the API base.** Rejected: a knob a caller can set to send mrw's requests elsewhere, for the benefit of tests only; the `-X` stamp reaches the built binary without one.

## Component / Boundary Impact

None — one file in `cmd/mrw`; no engine package changes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw version --check-latest` | new flag, stdout line, exit 0/1/2 | T1 | callers, scripts |
| README.md, AGENTS.md | one sentence each | T1 | agents, people |
| ADR-009 | an "Amended by ADR-149" pointer on its telemetry rejection | T1 | readers of the corpus |

## Inter-task Contracts

None — one task.

## Implementation

See `tasks/`.

## Consequences

- **Positive:** a caller can learn, in one call, whether the mrw it runs is behind, and a script can act on the exit code.
- **Negative:** the binary now links `net/http` (larger), and `govulncheck` gains a stdlib surface to report on.
- **Neutral:** GitHub's unauthenticated limit (60 requests an hour per address) bounds a script that loops; a 403 is reported as "could not ask", exit 2.

## Out of Scope

- Downloading or replacing the binary (permanent: boundary: mrw reports, the caller installs; README's Install section says how)
- The MCP server asking (permanent: boundary: Decision 4 — an MCP host calls tools without a person choosing the flag)
- Counting how many releases behind (permanent: boundary: Decision 3, one request)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A later change adds a network call elsewhere | Low | High | `TestOnlyTheCheckLatestFileImportsTheNetwork` fails on any other non-test file importing `net` or `net/http` |
| The repository moves and the URL answers 404 | Low | Low | exit 2 names the status; the URL is one constant |

## Rollback

Revert T1: one file, one flag, the doc sentences and the ADR-009 pointer.

## Follow-ups

None.
