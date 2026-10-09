# Task ADR-138-T1: the walk counts a link to a directory and says so

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `WalkSkipped.LinkedDirs`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a link to a directory met by a walk inside the root is counted on the skipped line and not followed`

## Goal

Decisions 1–4 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/walk.go` | edit | the set, the count, the `SkipNote` clause and tail |
| `internal/read/linkeddirs138_test.go` | add | the tests |
| `internal/mcp/schema_test.go`, `docs/receipts.txt` | edit | `skipped.linked_dirs` in the read schema the receipt test holds, and the registry |
| `scripts/contract.sh` | edit | §239; the §216 row that holds `skipped` to an exact object lists the key |
| `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md` | edit | the `-- skipped:` paragraph; the taken bullet |

## Ordered Steps

1. [S1] Write `TestAWalkCountsTheLinksToDirectoriesItDoesNotFollow` (a tree with a directory holding a matching file and a link to it: the walk serves the real path only, `LinkedDirs` is 1, the note says `1 link(s) to a directory, not followed` and `name one to be told why`), `TestAnEscapingLinkToADirectoryIsNotCounted` and `TestALinkToAFileIsServedNotCounted`. Confirm RED.
2. [S2] The set, the count, the clause and tail. Mutants: the link is dropped uncounted again; a file link is counted. [proof: mutation]
3. [S3] `skipped.linked_dirs` in the schema and registry, contract §239, AGENTS.md, README, the BACKLOG bullet removed. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestAWalkCountsTheLinksToDirectoriesItDoesNotFollow|TestAnEscapingLinkToADirectoryIsNotCounted|TestALinkToAFileIsServedNotCounted' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWalkCountsTheLinksToDirectoriesItDoesNotFollow \(' "$out" \
  && grep -qE '^--- PASS: TestAnEscapingLinkToADirectoryIsNotCounted \(' "$out" \
  && grep -qE '^--- PASS: TestALinkToAFileIsServedNotCounted \(' "$out" \
  && go test ./internal/read/ ./internal/mcp/ -count=1 -timeout 900s \
  && grep -q '^# 239\. ' scripts/contract.sh \
  && grep -q '^mcp_read skipped.linked_dirs$' docs/receipts.txt \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWalkCountsTheLinksToDirectoriesItDoesNotFollow` | `internal/read/linkeddirs138_test.go` | the link is counted, the real path served once, the note says so | none | S1, S2 |
| `TestAnEscapingLinkToADirectoryIsNotCounted` | `internal/read/linkeddirs138_test.go` | a link out of the root stays silent | none | S1, S2 |
| `TestALinkToAFileIsServedNotCounted` | `internal/read/linkeddirs138_test.go` | a link to a file is a candidate, not a directory link | none | S1, S2 |
| `TestAFifoIsNotCountedAsALinkToADirectory` | `internal/read/linkeddirs138_unix_test.go` | a FIFO is not counted as a link to a directory | none | S1, S2 |
| `TestALinkTheCallerExcludedOrIgnoredIsNotCountedAsALink` | `internal/read/linkeddirs138_policy_test.go` | `--exclude` and a .gitignore rule naming the link apply before it is counted | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `LinkedDirs` and its set |
| 2 — something selects it | the walker's non-file skip, on every discovered path |
| 3 — the caller can discover it | the `-- skipped:` line and `skipped.linked_dirs`; AGENTS.md; README |
| 4 — it is used | the 2026-10-09 Windows chaos round; no telemetry (ADR-009) |

## Invariants

- What a walk serves and every exit code are unchanged.
- A path the caller names is refused by name as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if counting a link to a directory needs a read of the directory's contents: that is the oracle ADR-007 rule 2 shuts.

## Out of Scope

- A FIFO, socket or device (deferred: docs/adr/BACKLOG.md)

## Mutation Log
- 2026-10-10 · 37aee68 · mutant killed · exit 1 · `internal/read/walk.go` · S2: the link is dropped uncounted again · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · covers:a link to a directory met by a walk inside the root is counted on the skipped line and not followed
- 2026-10-10 · 37aee68* · mutant survived · exit 0 · `internal/read/walk.go` · S2: any link is counted as a link to a directory, a file link included · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · covers:a link to a directory met by a walk inside the root is counted on the skipped line and not followed
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-10 · 37aee68* · mutant killed · exit 1 · `internal/read/walk.go` · S2: the tail promises a flag walks the links when only links were skipped · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · covers:a link to a directory met by a walk inside the root is counted on the skipped line and not followed
- 2026-10-10 · e67592f · mutant killed · exit 1 · `internal/read/walk.go` · S2: any non-regular entry is counted as a link to a directory, a FIFO included · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · covers:a link to a directory met by a walk inside the root is counted on the skipped line and not followed
- 2026-10-10 · 69a570f · mutant killed · exit 1 · `internal/read/walk.go` · S2: an excluded link name is still counted · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · covers:a link to a directory met by a walk inside the root is counted on the skipped line and not followed
- 2026-10-10 · 69a570f* · mutant killed · exit 1 · `internal/read/walk.go` · S2: a link the ignore rules name is counted as a link · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · covers:a link to a directory met by a walk inside the root is counted on the skipped line and not followed

## Verification Log
- 2026-10-10 · 37aee68 · exit 0 · `set -o pipefail …` · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · ms:32088
- 2026-10-10 · 37aee68* · exit 0 · `set -o pipefail …` · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · ms:30895
- 2026-10-10 · 37aee68* · exit 0 · `set -o pipefail …` · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · ms:31792
- 2026-10-10 · e67592f · exit 0 · `set -o pipefail …` · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · ms:32136
- 2026-10-10 · e67592f* · exit 1 · `set -o pipefail …` · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · ms:135 · test-lock-sha256:70f55d35ecaa01689601d616722bfcce88aaedc525c91e47bfd3bf3946f2b237 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9saW5rZWRkaXJzMTM4X3Rlc3QuZ28JVGVzdEFMaW5rVG9BRmlsZUlzU2VydmVkTm90Q291bnRlZAkyMWYzODUyNTgwZjA2YmFhMmU0YTdmMDVhMDVkZDJkMjZlNGU3NjFlMzAzZmNjZmNlMGQ3YmVlOWYzZWI2NjQ4CmJvZHkJaW50ZXJuYWwvcmVhZC9saW5rZWRkaXJzMTM4X3Rlc3QuZ28JVGVzdEFXYWxrQ291bnRzVGhlTGlua3NUb0RpcmVjdG9yaWVzSXREb2VzTm90Rm9sbG93CTRjNTg3ZWMzZGFhOTJmNWYyZWFiNWZkMWI4OGEyMzZiZjIyMDMxYmZmNWQxYTc2NWQzYmMwMzdhYWNlOGQzMjkKYm9keQlpbnRlcm5hbC9yZWFkL2xpbmtlZGRpcnMxMzhfdGVzdC5nbwlUZXN0QW5Fc2NhcGluZ0xpbmtUb0FEaXJlY3RvcnlJc05vdENvdW50ZWQJNTRlM2JlZjNkYTEyODQ1MjJhMTE4ODZlMTZiYmJmZDBhZWRmZWU5YWMwNjhmZmQyZTcwYmE5MDdiYjhiMWYzZApib2R5CWludGVybmFsL3JlYWQvbGlua2VkZGlyczEzOF91bml4X3Rlc3QuZ28JVGVzdEFGaWZvSXNOb3RDb3VudGVkQXNBTGlua1RvQURpcmVjdG9yeQkzZjAyMTMzMTUxNjFmYmEwNWIxZGJiMmY2NDMyMDJmZjg3NTk1NThlMTY3YmYwMzU2YWI4M2U1MGM4YWY2MTZl
  ```
  --- last 9 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read.test]
  internal/read/linkeddirs138_test.go:40:8: sk.LinkedDirs undefined (type WalkSkipped has no field or method LinkedDirs)
  internal/read/linkeddirs138_test.go:41:50: sk.LinkedDirs undefined (type WalkSkipped has no field or method LinkedDirs)
  internal/read/linkeddirs138_test.go:48:44: unknown field LinkedDirs in struct literal of type WalkSkipped
  internal/read/linkeddirs138_test.go:70:8: sk.LinkedDirs undefined (type WalkSkipped has no field or method LinkedDirs)
  internal/read/linkeddirs138_test.go:86:8: sk.LinkedDirs undefined (type WalkSkipped has no field or method LinkedDirs)
  internal/read/linkeddirs138_unix_test.go:25:8: sk.LinkedDirs undefined (type WalkSkipped has no field or method LinkedDirs)
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read [build failed]
  FAIL
  ```
- 2026-10-10 · e67592f* · exit 0 · `set -o pipefail …` · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · ms:30825
- 2026-10-10 · 69a570f · exit 0 · `set -o pipefail …` · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · ms:32423
- 2026-10-10 · 69a570f* · exit 0 · `set -o pipefail …` · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · ms:31501
- 2026-10-10 · 69a570f* · exit 0 · `set -o pipefail …` · acceptance-sha256:10b4d8f1beab7fcf31cf26ca08177c0b25b60fdabb507c74915a03e880f95fc9 · ms:30695
