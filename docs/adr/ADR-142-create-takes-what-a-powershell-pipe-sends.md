# ADR-142: `mrw write --create` takes what a PowerShell pipe sends

**Status:** Accepted
**Accepted:** 2026-10-10 by Zy — "windows peers are back for testign", then the standing goal "/loop continue delivering, end to end, no dead code"; the findings are five Windows sessions' reports on v1.59.0
**Date:** 2026-10-10
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-139, ADR-005, ADR-065, docs/adr/BACKLOG.md
**Invalidates:** ADR-139 decision 3's refusal of content whose last line ends in CRLF after LF lines, in the one shape a pipe produces; every other refusal of ADR-139 stands
**Governs:** `internal/ingest/create.go`, `internal/apply/apply.go`, `cmd/mrw/main.go`, `scripts/contract.sh`, `AGENTS.md`
**Enforced-by:** `internal/ingest/create142_test.go::TestAPowerShellPipeTerminatorAfterLFContentIsDropped`
**Served-path change:** `mrw write --create PATH` on content that ends in `\r\n` after lines that end in `\n` and hold no other CR drops that final `\r\n`, as PowerShell appends one to everything it pipes, and says so on stderr. A leading UTF-8 byte order mark is kept and named on stderr. A `--create` of a path that exists and has not been read says that it exists, not that the file must be read first.

## Context

Windows PowerShell 5.1 and PowerShell 7 append CRLF to the text they pipe to a program, so `Get-Content -Raw f | mrw write --create p` of an LF file hands mrw `a\nb\n\r\n`. ADR-139 refused that as a line ending in a bare CR (the Codex review of #371, P2: the plan parser strips a CR quietly, so a CR must never be dropped in silence). Three of five Windows sessions met it on v1.59.0 (2026-10-10): every LF-only file piped from PowerShell is refused with "a line ends in a bare CR", on input with no bare CR in it. PowerShell 5.1 also prefixes a BOM, which `--create` carries into the file as content without a word.

A third finding is a message: `--create` of a path that exists and has not been read says `exists.txt has not been read … Run mrw read exists.txt first, or pass --force`. After the read it says `already exists — use replace or delete`, and `--force` still refuses, so the first message sends the caller down a path that cannot work.

**Audit of the class** — *a place where `--create` content is turned into lines*: `mrw read --grep 'CompileCreate' --exclude '*_test.go' cmd internal` names one production caller (`cmd/mrw/main.go`), so one place decides.

## Existing Primitives Audit

- **`lines.Split`** (ADR-005, ADR-065) — CRLF only when every newline is one; mixed content is LF with a stray CR kept in its line. Untouched.
- **`ingest.CompileCreate`** (ADR-139) — keeps every refusal; it receives the content already cleaned.
- **The `failK`/`fail` verdicts in `apply`** — the create-on-existing message is a different sentence on the same verdict.

## Decision

1. **A pipe terminator is dropped, narrowly and in the open.** `ingest.CreateContent` returns the content and a list of notes. When the content ends in `\r\n`, the text before that holds at least one `\n` and no `\r`, the final `\r\n` is removed and a note says so. Everything else is returned unchanged, so `a\nb\r`, `a\r\nb\n\r\n` and every other mixed shape stay refused by `CompileCreate` as ADR-139 decided. All-CRLF content is not touched: PowerShell's terminator there is a trailing blank line, which mrw cannot tell from the file's own, and AGENTS.md says so.
2. **A leading UTF-8 byte order mark is kept and named.** It is content (ADR-139's test pins it), and PowerShell 5.1 adds one; stderr says `mrw: --create PATH: the content begins with a UTF-8 byte order mark, kept as content`.
3. **A lone create hunk on a path that exists and was not read says `create: P already exists — a create never overwrites; read it and use replace or delete, or unlink it first`**, in place of the read-before-modify refusal. A path that was read, or a create under `--force`, keeps the line-count message it had; every other op keeps the read-before-modify refusal.
4. **The notes go to stderr, one line each, once the content has compiled** (a refused content is told nothing was done to it); the exit code and the receipt do not change.

## Alternatives Considered

- **Turn every CRLF into LF whenever the content has a lone LF** — rejected: it changes more than the pipe's terminator in a genuinely mixed file and says nothing; ADR-139 refused for exactly that reason.
- **Drop the final CRLF whenever the content ends in one** — rejected: an all-CRLF file ends in one by right, and a CRLF file would lose its last line break.
- **Tell PowerShell callers to write bytes** (`[IO.File]::ReadAllBytes … | mrw`) — rejected: PowerShell's pipe sends text to a native command whatever it was given; the caller cannot opt out of the terminator.
- **Leave it, document it** — rejected: three of five sessions hit it first-run and the refusal blames input that holds no bare CR.

## Component / Boundary Impact

`internal/ingest` (one function), `internal/apply` (one message), `cmd/mrw` (one call and the notes). No engine package changes beyond the message in `apply`; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `ingest.CreateContent` | new function | T1 | `cmd/mrw` |
| `mrw write --create` | a pipe terminator dropped with a stderr note; a BOM named | T1 | PowerShell callers |
| create on an existing path | its own message when the path was not read | T1 | CLI and MCP callers (`apply`) |
| `scripts/contract.sh` | §243 | T1 | CI Linux |
| `AGENTS.md` | the `--create` paragraph | T1 | every agent |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** an LF file piped from PowerShell is created as it is, with a line saying what was dropped; the existing-path message stops pointing at a dead end.
- **Negative:** one narrow rule that guesses a terminator; the note is what keeps it honest.
- **Neutral:** all-CRLF content from `-Raw` still gains a trailing blank line; documented, not guessed at.

## Out of Scope

- Normalising line endings in general (permanent: boundary: ADR-139 chose to refuse mixed content; this record narrows that refusal to the one shape a pipe makes)
- Windows PowerShell 5.1 turning non-ASCII text into `?` before mrw sees it (permanent: fact: the bytes mrw receives are already mangled; citation: file `docs/adr/ADR-139-write-create-makes-one-file-from-stdin.md:34`)
- Refusing NTFS stream names and the wording of name refusals (deferred: docs/adr/BACKLOG.md — "Windows names" entry, planned as the next record)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a mixed-ending file loses one CRLF at its end | Low | one byte pair differs, and it is named on stderr | the rule needs LF lines and no other CR; the note |
| the create-exists message hides a read the caller wanted | Low | none: a create cannot overwrite | the message names replace, delete and unlink |

## Rollback

Revert T1: the pipe terminator is refused again as before, and the old message returns. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up beyond the deferred item above.
