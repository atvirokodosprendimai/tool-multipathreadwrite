# Task ADR-098-T3: README and AGENTS teach the ast_grep index's paging

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** none
**Consumes:** `after` with `ast_grep` (`afterCursor`) (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the README clause`, `the AGENTS clause`, `the behaviour it documents holds`

## Goal

README.md's MCP paragraph and AGENTS.md's MCP paragraph each say, in one clause, that an `ast_grep` index
pages with `after` like a `grep` index.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `README.md` | edit | the MCP `grep` / `exclude` paragraph ("A grep too large to serve returns an index") |
| `AGENTS.md` | edit | the MCP paragraph ("When the matches are too large to serve, the tool returns an INDEX") |

**What selects it:** nothing in code; these are the two places a reader of the MCP surface lands.

## Ordered Steps

1. [S1] Confirm the fence RED before the edit: neither paragraph says an ast_grep index pages with `after`. [proof: mutation]
2. [S2] Add the clause inside each paragraph: the literal `an ast_grep index pages with` followed by
   `after`, like a grep index. [proof: mutation] Mutants: either clause removed; either clause moved out of
   its paragraph.

## Acceptance

```bash
set -o pipefail
readme=$(awk '/^`grep` \/ `exclude` map onto/{f=1} f&&/^$/{exit} f' README.md | tr '\n' ' ' | tr -s ' ') \
  && agents=$(awk '/^`mrw read --grep P` maps onto/{f=1} f&&/^$/{exit} f' AGENTS.md | tr '\n' ' ' | tr -s ' ') \
  && printf '%s' "$readme" | grep -qF 'an ast_grep index pages with' \
  && printf '%s' "$readme" | grep -qF 'after' \
  && printf '%s' "$agents" | grep -qF 'an ast_grep index pages with' \
  && printf '%s' "$agents" | grep -qF 'after' \
  && out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestAnAstGrepIndexPagesToTheEnd' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnAstGrepIndexPagesToTheEnd \(' "$out"
```

Each check reads only its own paragraph (README's MCP `grep`/`exclude` paragraph, AGENTS.md's
`mrw read --grep P` paragraph), so a mention elsewhere cannot satisfy it; both extractions are non-empty
on `db39d42` and hold neither literal.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnAstGrepIndexPagesToTheEnd` | `internal/mcp/astgrepindex098_test.go` | the behaviour the clauses document (T1) | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two clauses |
| 2 — something selects it | the fence's paragraph greps; once `scripts/fence-prose.py` (PR #283) is on `main`, `static.sh` reruns them on every push |
| 3 — the caller can discover it | README's and AGENTS' MCP paragraphs |
| 4 — it is used | nothing measures this |

## Mutation Log
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `README.md` · the README clause removed · acceptance-sha256:b883033b3e29f52ffeed86df538853769803372757daa24a8f8fdcbeb2e00af7 · covers:the README clause
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `AGENTS.md` · the AGENTS clause removed · acceptance-sha256:b883033b3e29f52ffeed86df538853769803372757daa24a8f8fdcbeb2e00af7 · covers:the AGENTS clause
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `README.md` · the README clause moved out of its paragraph · acceptance-sha256:b883033b3e29f52ffeed86df538853769803372757daa24a8f8fdcbeb2e00af7 · covers:the README clause
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `AGENTS.md` · the AGENTS clause moved out of its paragraph · acceptance-sha256:b883033b3e29f52ffeed86df538853769803372757daa24a8f8fdcbeb2e00af7 · covers:the AGENTS clause
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/mcp/tools.go` · the documented paging broken: the ast_grep call site drops the cursor · acceptance-sha256:b883033b3e29f52ffeed86df538853769803372757daa24a8f8fdcbeb2e00af7 · covers:the behaviour it documents holds

## Invariants

- No other paragraph changes; the centralised `mrw` skill re-syncs at the next tag.

## Risks

- The AGENTS clause sits in "Using mrw", which the centralised skill mirrors; it drifts until the release re-sync.

## Stop Condition

Stop and ask if T1 is not `done`.

## Out of Scope

- The code — T1; the served text — T2.

## Verification Log
- 2026-09-29 · db39d42* · exit 1 · `set -o pipefail …` · acceptance-sha256:b883033b3e29f52ffeed86df538853769803372757daa24a8f8fdcbeb2e00af7 · ms:57 · test-lock-sha256:79b69449c2f352188ca5a0c21cd1a8384c3c2c20b2fe70d203df193c85778517 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9hc3RncmVwaW5kZXgwOThfdGVzdC5nbwlUZXN0QWZ0ZXJXaXRob3V0QUZpbmRlcklzUmVmdXNlZE5hbWluZ0JvdGgJOTUwNTUzZWNjNmRjNDE5ZjI3ZmY5NDhiN2MxM2M4ZmMxMTg0MzNlNjQ0NDY5YWE2ZTM3MWRjNmEwOWU4MjJmMwpib2R5CWludGVybmFsL21jcC9hc3RncmVwaW5kZXgwOThfdGVzdC5nbwlUZXN0QW5Bc3RHcmVwSW5kZXhQYWdlc1RvVGhlRW5kCTE4ZGY4MzQxNmZiMDkzYjI4YzNhNWIzMjU4MDczNTcxMzhjZjkwMTFkMjdkZTFkMjIyNDgxODYxOGQwZjk2YTUKYm9keQlpbnRlcm5hbC9tY3AvYXN0Z3JlcGluZGV4MDk4X3Rlc3QuZ28JVGVzdFRoZUluZGV4TmFtZXNUaGVGaW5kZXJUaGF0TWFkZUl0CThlMzY4ZWRkNzUyZjA1OTcxNDExZGRmZTg0NmY3MGU2OTc0ZWQxMWNkYWM4ZmNlNjMyOTBmMjNkOWYzNzU2N2Q
  ```
  ```
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b883033b3e29f52ffeed86df538853769803372757daa24a8f8fdcbeb2e00af7 · ms:655
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b883033b3e29f52ffeed86df538853769803372757daa24a8f8fdcbeb2e00af7 · ms:677
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b883033b3e29f52ffeed86df538853769803372757daa24a8f8fdcbeb2e00af7 · ms:697
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b883033b3e29f52ffeed86df538853769803372757daa24a8f8fdcbeb2e00af7 · ms:679
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b883033b3e29f52ffeed86df538853769803372757daa24a8f8fdcbeb2e00af7 · ms:1232
