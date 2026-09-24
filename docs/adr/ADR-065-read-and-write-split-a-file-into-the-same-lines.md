# ADR-065: Read, write and the plan compilers split a file into the same lines

**Status:** Accepted
**Accepted:** 2026-09-24 by M — *"accepted"*
**Date:** 2026-09-24
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-036, ADR-051, ADR-058, ADR-066, docs/adr/BACKLOG.md
**Invalidates:** ADR-051's apply_patch spec fact F-10 ("the on-disk file is not CR LF-normalised … compile refuses") — accepted by M 2026-09-24, *"Supersede F-10 in ADR-065"*. The compilers now read a target as the write engine does; the write engine keeps the file's CR LF.
**Governs:** `internal/lines` (new), `internal/apply/apply.go` (`readLines`), `internal/read/read.go` (`split`), `internal/mcp/tools.go` (`countFileLines`), `internal/ingest/applypatch.go` (`fileLines`), `internal/ingest/searchreplace.go` (`searchFileLines`), `internal/read/astgrep.go`, `scripts/contract.sh`
**Enforced-by:** `internal/read/lines_agree_test.go::TestACROnlyFileIsServedAsTheLinesAWriteAddresses`
**Served-path change:** Every surface now numbers a file's lines as the write engine always has (ADR-005 §3):
- **A CR-only file** (`one\rtwo\rthree\r`) is served as 3 lines. It was served as 1, and a write to its line 2 applied though read had never served it.
- **CRLF lines** are served without their `\r`, on every surface including `--no-numbers`. So a read's `/one$/` matches where a write's does, and `apply_patch`/`search_replace` edit a CRLF or CR-only file instead of refusing it with "matched no lines".
- **A file that mixes line endings** keeps its stray `\r` on both sides, as before.
- **`--ast-grep` and MCP `ast_grep`** report a hit in a CR-only file as a problem naming the file, instead of serving the line ast-grep numbered by `\n`, which after this change would be a different line.
- **The break campaign's `[cr-only-file-read]` header** changes from `1L` to `3L`.

## Context

**What was observed.** A randomized chaos pass on 2026-09-24 (`main` `c3f5661`, v1.22.3; about 12,600 mrw runs checked against an independent model) found that read and write count the lines of a CR-only file differently. It was reproduced by hand:

```
printf 'one\rtwo\rthree\r' > cr.txt
mrw read cr.txt                          # ==> cr.txt  1L …   1| one\rtwo\rthree
printf '@@ cr.txt 1 replace\nREPLACED\n' | mrw write -   # exit 0, "3L -> 3L"; only "one" replaced
printf '@@ cr.txt 2 replace\nX\n' | mrw write -          # exit 0; line 2 was never served
```

- `internal/read/read.go:699-707` `split` breaks lines at `\n` only, keeping any `\r` inside the content.
- The write engine's `readLines`/`eolOf` (`internal/apply/apply.go:1521-1556`, decided by ADR-005 §3) treats a file whose every newline is `\r\n` as CRLF, and a file with no `\n` but a `\r` as CR-only.
- A bare read records "whole file" in the ledger (`internal/seen/seen.go:53-66`, nil spans) with no line count, and apply's `covered` short-circuits on it (`apply.go:824-840`). So the licence covered lines the caller never saw as lines.
- ADR-005 §4 says the caller is the one who counts the numbers. ADR-002 forbids editing what was not served.
- On CRLF files the counts agree, but read serves `one\r`: read's `/one$/` misses where the write's matches, which ADR-036 says must resolve the same way.
- The break campaign's `[cr-only-file-read]` probe pins read's `1L` and never writes, which is why nothing caught it.

**A second member, found while enumerating.** `--format=apply_patch` and `--format=search_replace` compile against the target's lines read by `fileLines` (`internal/ingest/applypatch.go:269-286`) and `searchFileLines` (`internal/ingest/searchreplace.go:126-143`), which also break at `\n` only. The document itself is normalised to LF first (ADR-051 F-9). So, probed 2026-09-24 on v1.22.3, a one-line `apply_patch` or `search_replace` edit:
- **on an LF file:** applies;
- **on the same file as CRLF or CR-only:** exits 2 with "old side matched no lines" / "SEARCH matched no lines".

That failure is loud, not silent, but a CRLF file cannot be edited through either format.

**The class this record governs:** code that splits a TARGET file's content into the lines a caller addresses. Enumerated 2026-09-24 with

```
git grep -nE 'strings\.Split\([^)]*"\\n"\)|bufio\.NewScanner|bytes\.Split\([^)]*\\n' -- 'internal/*.go' 'cmd/*.go' ':!*_test.go'
```

That finds 18 sites.
- Governed here: `read.go:706` (which `--grep`'s `walk.go:207` also goes through), `mcp/tools.go:1050` (`countFileLines`, `next_read` paging), `ingest/applypatch.go:285`, and `ingest/searchreplace.go:142`.
- Left out, because they split something other than a target file:
  - mrw's own state (`seen.go:133,186`, `authoring.go:182,267,401`, `iter.go:58`);
  - plan and patch documents (`plan.go:194`, `main.go:1691`, `applypatch.go:64`, `searchreplace.go:24`);
  - rendered reports (`mcp/ack.go:115,189`);
  - the curve scorer's `changed()` (`curve/score.go:199-200`), which compares texts it produced, not a caller's file.

## Existing Primitives Audit

- **`readLines` / `eolOf` / `text`** (`internal/apply/apply.go:1493-1556`). They are the rule and are moved as-is into `internal/lines`; `apply` keeps `text` and calls the package. Neither `read` nor `ingest` may import `apply` for it, because that would pull the write engine into the read path.
- **The raw-bytes hash** in `read.split` (`sha256` of `b`, agreeing with `seen.SHA`, `seen.go:405-410`). Kept on the raw bytes. Hashing rejoined lines would make every CRLF file read as "changed behind mrw's back".
- **`countFileLines`** (`internal/mcp/tools.go:1043`). Its `bufio.Scanner` counts `\n`; replaced by a count from `lines.Split`. `firstPage` reads the same file straight after (`:942`), so holding it costs no new peak.

## Decision

**1. One function numbers a target's lines: `lines.Split(s string) (ls []string, eol string, final bool)`** in a new leaf package `internal/lines`, with ADR-005 §3's rule unchanged:
- **CRLF** when every `\n` is `\r\n`;
- **CR-only** when there is no `\n` but there is a `\r`;
- **LF** otherwise, including mixed files, where a stray `\r` stays in its line;
- **an empty input** has no lines.

**2. Every governed site calls it:** apply's `readLines`, read's `split` (and so `--grep`), MCP `countFileLines`, and both ingest compilers. Read still hashes the raw bytes.

**3. A line number mrw does not produce is not served on a file whose lines it numbers differently.** ast-grep numbers rows by `\n` (`internal/read/astgrep.go:103`). On a CR-only file that is not mrw's numbering, so the hit is reported as a problem naming the file and its line terminator. On LF, CRLF and mixed files the two numberings agree.

## Alternatives Considered

- **Fix CR-only only; keep serving CRLF lines with their `\r`.** Rejected. It closes the licence hole but leaves read's `/one$/` disagreeing with the write's on every CRLF file (ADR-036) and the ingest compilers refusing CRLF targets. With one shared function the rest costs nothing more.
- **Make the write engine split at `\n` only, as read does.** Rejected. It reverses ADR-005 §3, makes a CR-only file one unaddressable line, and leaves `\r` in every CRLF line a plan must reproduce byte for byte.
- **Store a line count in the ledger's whole-file observation.** Rejected as the fix: it would refuse the write, but read and write would still number lines differently, and ADR-036 and the ingest refusals remain. The licence hole closes because the counts agree.

## Component / Boundary Impact

- New package `internal/lines`, imported by `internal/apply`, `internal/read`, `internal/mcp` and `internal/ingest`. It imports only the standard library.
- Changed functions: `internal/apply` `readLines`, `internal/read` `split`, `internal/mcp` `countFileLines`, and `internal/ingest` `fileLines`/`searchFileLines`.
- Byte-identical: `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, `internal/guide`.
- `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read` / MCP `mrw_read` served lines | CR-only split; CRLF without `\r` | T1 | callers, the ledger |
| `--grep` / MCP `grep` matching | same lines as served | T1 | callers |
| MCP `next_read` paging | counts the same lines | T1 | MCP hosts |
| `--format=apply_patch` / `search_replace` | compile against the same lines | T2 | `mrw write` |
| `--ast-grep` / MCP `ast_grep` | a CR-only hit is a problem, not a served range | T1 | callers |
| contract §120 | CR-only licence, CRLF pattern and write | T1 | CI, `adr-verify` |
| contract §121 | apply_patch on CRLF and search_replace on CR-only apply | T2 | CI, `adr-verify` |
| break campaign `[cr-only-file-read]` | header `1L` → `3L` | T1 | release campaign diff |

Exit codes are unchanged. A CR-only read that addressed `2` used to exit 1 ("file has 1 lines") and now serves line 2.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `lines.Split` | T1 | T2 | No |
| served CRLF text without `\r` | T1 | — | Partly: a caller that matched a trailing `\r` in served text no longer finds it, and the write side never had it |

## Implementation

See `docs/adr/ADR-065-read-and-write-split-a-file-into-the-same-lines/tasks/README.md`.

## Consequences

- **Positive:** a line number means one line on every surface. The CR-only licence hole is closed, read and write patterns agree on CRLF, and both foreign plan formats can edit CRLF and CR-only files.
- **Negative:** served CRLF text changes (no `\r`), and the campaign's text for one probe changes.
- **Neutral:** ADR-005 §3's rule is unchanged; it moves to one place.

## Out of Scope

- Translating ast-grep's byte offsets into mrw's lines on a CR-only file (deferred: docs/adr/BACKLOG.md)
- A line count in the ledger's whole-file observation (permanent: boundary: agreement of the counts closes the hole; the ledger stays spans plus sha)
- The curve scorer's `changed()` (permanent: boundary: it compares texts mrw produced, not a caller's file)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| read hashes rejoined lines by mistake | Low | High | a mutant hashing the rejoined lines must go red; a test writes a CRLF file after a whole read |
| MCP ack or paging miscount after the change | Low | Med | a CR-only file large enough to page is followed to exhaustion, and only acknowledged lines become writable |
| a caller relied on seeing `\r` | Low | Low | stated in the Served-path change and the release tag |

## Rollback

Revert the package and its four call sites, the ast-grep check, the tests, §120, §121 and the campaign expectation. The ledger format does not change.

## Follow-ups

- [ ] Release with ADR-066 as v1.23.0, with the campaign diffed against v1.22.3 and `[cr-only-file-read]` named as an expected difference.
