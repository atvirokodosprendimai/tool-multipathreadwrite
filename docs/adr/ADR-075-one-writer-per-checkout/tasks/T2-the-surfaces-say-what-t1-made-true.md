# Task ADR-075-T2: the surfaces and the records say what T1 made true

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the MCP routing by shell alone; `chaos.py` failing on a lost edit; the dispositions
**Consumes:** T1's lock
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the surfaces stop selling serialized writes`, `the handshake says writers take turns`, `the chaos pass fails on a lost edit`, `ADR-002's line is marked invalidated`, `the engine packages are unchanged`, `go.mod declares one requirement`

## Goal

The MCP handshake and both tool descriptions route callers who share a checkout to `mrw mcp` for
serialized writes; after T1 that is false. `chaos.py` counts a lost edit as an accepted risk. ADR-002,
the spec's UC-3 and BACKLOG say locking is out of scope. Make each say what is now true.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/instructions.go`, `internal/mcp/mcp.go` | edit | route by shell alone; writers take turns on either surface |
| `internal/mcp/mcp_test.go` | edit | `TestTheSurfaceSaysTheCLIIsRicher` asserts the new claim and forbids the old |
| `internal/mcp/era_test.go`, `internal/mcp/testdata/legacy_golden.jsonl` | edit | the legacy answers carry the new routing; the golden regenerated through its own switch, the diff that sentence alone |
| `scripts/contract.sh` | edit | §50 requires "one writer per checkout" where it required "serialized" |
| `AGENTS.md` | edit | the same, and a pointer that named a README passage that does not exist |
| `scripts/chaos.py` | edit | a lost edit fails by default |
| `docs/adr/ADR-002-mrw-will-not-edit-a-file-it-has-not-seen.md`, `docs/specs/2026-09-16-dangling-high-impact.md`, `docs/adr/BACKLOG.md` | edit | the invalidated lines marked; the round's entry closed; the unlocked-state leads filed |

## Ordered Steps

1. [S1] Change the test; confirm RED against the current wording. [proof: mutation]
2. [S2] Change the wording; GREEN. [proof: mutation]
   Mutants: the old clause restored in `instructions.go`; restored in one tool description.
3. [S3] `chaos.py`, ADR-002, the spec and BACKLOG. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -timeout 180s -run 'TestTheSurfaceSaysTheCLIIsRicher|TestALegacyResultIsUnchangedByTheModernPath' -v 2>&1 | tee /tmp/adr075-T2.out \
  && missing=$(for t in TestTheSurfaceSaysTheCLIIsRicher TestALegacyResultIsUnchangedByTheModernPath; do grep -qE "^--- PASS: $t \(" /tmp/adr075-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && ! grep -q 'writes serialized' internal/mcp/instructions.go \
  && ! grep -q 'writes serialized' internal/mcp/mcp.go \
  && ! grep -q 'serializes in-process' AGENTS.md \
  && ! grep -q 'race-strict' scripts/chaos.py \
  && grep -q 'invalidated by ADR-075' docs/adr/ADR-002-mrw-will-not-edit-a-file-it-has-not-seen.md \
  && grep -q 'superseded by ADR-075' docs/specs/2026-09-16-dangling-high-impact.md \
  && grep -q 'Fixed by ADR-075' docs/adr/BACKLOG.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/read internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc internal/seen ':(exclude)internal/seen/seen.go' ':(exclude)internal/seen/writelock_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/read internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc internal/seen ':(exclude)internal/seen/seen.go' ':(exclude)internal/seen/writelock_test.go')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheSurfaceSaysTheCLIIsRicher` | `internal/mcp/mcp_test.go` | the handshake says writers take turns and no longer sells serialized writes; no description does | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the new wording |
| 2 — something selects it | every MCP `initialize` and `tools/list` |
| 3 — the caller can discover it | it is in the handshake |
| 4 — it is used | a user-scope registration puts the routing in front of every project's agent |

## Verification Log
(empty until execute)
- 2026-09-26 · da2fd0a* · exit 1 · `set -o pipefail …` · acceptance-sha256:1438e2ffdd99dc04735f0f965995c1c1b96eca5abefdcb9891871cf1f4bf79ce · ms:1071 · test-lock-sha256:9a3e55d843275e79a7cd6137141a688cb825aeac7c98c63bd9c96170324a9e82 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9tY3BfdGVzdC5nbwlUZXN0QUpTT05BcnJheUlzQW5JbnZhbGlkUmVxdWVzdE5vdEFQYXJzZUVycm9yCThhYTIzNWI0Njk1YjI3MWQxNTdmZWNlYjUwMDUzNmIxZjA4YmRhMjFlNzZjZTkzOGY3YTUwN2QwOGNjZjIyZDEKYm9keQlpbnRlcm5hbC9tY3AvbWNwX3Rlc3QuZ28JVGVzdEFNYWxmb3JtZWRGcmFtZUlzQW5FcnJvck5vdEFDbG9zZQk5M2E3OWEyMWEyNWZhZGI0OGQ3NDhkZTU5NTczNTMzMjNjMDI3ZWYwNDE3YjQwYTJkMTk3NzljODFhOWRkNjk5CmJvZHkJaW50ZXJuYWwvbWNwL21jcF90ZXN0LmdvCVRlc3RBblVua25vd25NZXRob2RJc0FuRXJyb3JSZXNwb25zZQlhNTI4NGVlZGIxMDViOTcyN2MzYjQ5Njc4MzQ3ZThhMWQ4ZWJmOGFmZjUyMDg0ZDZlNGUzMDJkNDRiNWM3YzU0CmJvZHkJaW50ZXJuYWwvbWNwL21jcF90ZXN0LmdvCVRlc3RJbml0aWFsaXplQ29tcGxldGVzVGhlSGFuZHNoYWtlCTgzMTM1MDVkYTZhYzhlNWI3YmM5MDg4MzIxZmQ3ZjgyNzFiNDY5NTRhZTk1NWJiMTM0NzAzYzcyNjkxYTgzNDkKYm9keQlpbnRlcm5hbC9tY3AvbWNwX3Rlc3QuZ28JVGVzdE1DUEluc3RydWN0aW9uc0NvbnRhaW5TaGFyZWQJNDFiMGUxZTNhNGE5MTkwYzk3OGM5Y2ExYTVjYTNkOTE5NzI0Mjk1NGI4ZmIxYTM4MmUyYmE0MzUxODkyYzVkYgpib2R5CWludGVybmFsL21jcC9tY3BfdGVzdC5nbwlUZXN0TUNQSW5zdHJ1Y3Rpb25zVGVhY2hUaGVXaHkJNzhiNzliYWVkNmUwZWMzMjg0NDM5NzRmOWNiMzY2OTc0Yjg2ODVkOTEyYzFkMGRiY2ZkNDY2MGY5YWVhZmI0Ygpib2R5CWludGVybmFsL21jcC9tY3BfdGVzdC5nbwlUZXN0TWFpbgk2Y2VlZmQ4ZGIyMmFlYjMwZjQxNDMzMmIwZjA1NGIyMzVlZTQyNmIzNmI2MjhlODU3OTMxNjM5YTA2NzlmMGZhCmJvZHkJaW50ZXJuYWwvbWNwL21jcF90ZXN0LmdvCVRlc3RNcndSZWFkRGVjbGFyZXNOb091dHB1dFNjaGVtYQk4OWZlMWFhYmFhN2Q3NjJiOWJmYWRiNzA3NTJiMGU1NWZmMmRkZmNiOGMxZjVlNDAwZTVjNTdhOWE5ZWYxNWUxCmJvZHkJaW50ZXJuYWwvbWNwL21jcF90ZXN0LmdvCVRlc3RPbmVNZXNzYWdlUGVyTGluZVJvdW5kVHJpcHMJY2EzNjQxZjM3OTE4M2ZmYjY0MTlkNTY5MDBlYTgzYWZkNzBiYmFkMWEwOTRlYWVmMDcyYTYyMWU2NTVjNmNhNgpib2R5CWludGVybmFsL21jcC9tY3BfdGVzdC5nbwlUZXN0T25seU1DUE1lc3NhZ2VzUmVhY2hTdGRvdXQJYjAyOWM3YTIwNTNjMTRmODhjZTkxN2I5ZjdiZTdlMTgzOTdiMmZiNTY1MGIzNjJmZGZhMzk4ZGFmNjMxN2EyMwpib2R5CWludGVybmFsL21jcC9tY3BfdGVzdC5nbwlUZXN0UGluZ0lzQW5zd2VyZWRXaXRoQW5FbXB0eVJlc3VsdAk2OTk4MGYwZTUzNmMzMDNiYzhlNzA1ZGM4MzEyODg1ZTY0MjBkYzU3YTcwNWRjOWJlNTUyMmZhMThjMDVmZGVjCmJvZHkJaW50ZXJuYWwvbWNwL21jcF90ZXN0LmdvCVRlc3RUaGVCdWlsdENMSVN1cnZpdmVzQVNlY29uZENhbGxlcgk2OGM0ZDFiOTAzN2RkMjkxZDA0ZWIxMzUwYjk1YjFjYWU1MjE2ZTI2YWVlY2I0NmQyYTFjMTQ2MGQ4OTU3YzIyCmJvZHkJaW50ZXJuYWwvbWNwL21jcF90ZXN0LmdvCVRlc3RUaGVEZXNjcmlwdGlvbnNTYXlXaGVuVG9SZWFjaEZvclRoZVRvb2wJZTk0ZjcwMjM3YmRmMjZjYzI5NDk1ODQzOTJkNGY0OGNiOGI3YWE3MzQ2Y2I0MjFiMmEwMmNjNzM2MTYzYzNjYwpib2R5CWludGVybmFsL21jcC9tY3BfdGVzdC5nbwlUZXN0VGhlSGFuZHNoYWtlRG9lc05vdFRlYWNoQUxlZGdlclJhY2UJNmQyNThiMTEwZTJkOGFiM2U1ZWZjMjk3MGNkMWViZTlhNWRiYmRmMmYwNmU1MDQ1NzliN2QwY2FkOWIxMzkzMApib2R5CWludGVybmFsL21jcC9tY3BfdGVzdC5nbwlUZXN0VGhlSW5pdGlhbGl6ZWROb3RpZmljYXRpb25HZXRzTm9SZXNwb25zZQkxZjczYTYxNjBhYWIwODkzMDZlMjU2ZDQ1Zjk2ODg1MjMyZWU0NzFiY2M1MDU1ZDVmNzM1OGNhZjk0MThhMTgxCmJvZHkJaW50ZXJuYWwvbWNwL21jcF90ZXN0LmdvCVRlc3RUaGVJbnN0cnVjdGlvbnNUZWFjaFRoZUNvbnRpbnVhdGlvbgkwZDliMTBiNjkxZTQ2YjY4Zjg2YWM3NGQ5MzRkODczNzAzNzNkOTBiY2UwZThhNDA3MDcwNDkyOWEwNjU1ZDg3CmJvZHkJaW50ZXJuYWwvbWNwL21jcF90ZXN0LmdvCVRlc3RUaGVJbnN0cnVjdGlvbnNUZWFjaFRoZVBhdHRlcm5BZGRyZXNzCTQ0NjVkNjE1NDQ2ZWZlMTc0N2IwYzJiMDEwYTM2MjY5MjRhZGQ5Yzg5NTljYjRhZGIzYzc5MDIzMGI4NmE5Y2IKYm9keQlpbnRlcm5hbC9tY3AvbWNwX3Rlc3QuZ28JVGVzdFRoZUluc3RydWN0aW9uc1RlbGxBSG9zdEhvd1RvQXV0aG9yQVBsYW4JZDY4MmE2MjNhNTk5MGExZGM2NTI4OTlhNjRjOTQwMTIwMDI1MzUxYzkyMjM2NDJlYWNjN2Y3NmU5OWZmNzk5ZQpib2R5CWludGVybmFsL21jcC9tY3BfdGVzdC5nbwlUZXN0VGhlUm91dGluZ0NsYWltc09ubHlSZWFsRXhjbHVzaXZlcwljZWM4ZGRiMmZhOTkwNzJjMzYxZGY3NTYzZjA2MDdlYzU5N2VkMzdmMjQ2ZTNlMDkwN2RiZTkwM2I5ZGM5OGRjCmJvZHkJaW50ZXJuYWwvbWNwL21jcF90ZXN0LmdvCVRlc3RUaGVTdXJmYWNlTmFtZXNUaGVSb290VGhlUGlja0Nob3NlCTYxNWU2MDhiNmUzNmVlOTc5ODhmMGQ1YWI3YjQ5MGY4Zjk5NWEzNmZjNDJlNmRlNjA3YjYzY2QxNmE1Yzc2MmMKYm9keQlpbnRlcm5hbC9tY3AvbWNwX3Rlc3QuZ28JVGVzdFRoZVN1cmZhY2VTYXlzVGhlQ0xJSXNSaWNoZXIJNWZhYjRlMTA0N2IxNzllNTUwZTBjNjNiNjM1MGJhM2MyMDlmNWE0ZjkyOWEzODRjY2FkNTc0ZTZmM2RjMDg1MQpib2R5CWludGVybmFsL21jcC9tY3BfdGVzdC5nbwlUZXN0VGhlU3VyZmFjZVRlYWNoZXNGaW5kaW5nCTA3NGFmMDYxNzA1ZmRkMDIxYjk5ZjQzYTJkYTIyYzY3NzJkZjI5MGJhYzM2NTYwZTA3MTIwOWE5OTE5MGY1ZTMKYm9keQlpbnRlcm5hbC9tY3AvbWNwX3Rlc3QuZ28JVGVzdFRvb2xzTGlzdE5hbWVzQm90aFRvb2xzCWYxZDIzNjE3MDhlNGUwY2U0YTM0Y2Y5Yjg1ZTJhY2VhNDM3MDBlMGE5OWQyMTg4YTE0NjMxNTBiZmI2YjVkMDY
  ```
  --- last 10 line(s) of stdout
  === RUN   TestTheSurfaceSaysTheCLIIsRicher
      mcp_test.go:530: the instructions do not say that writers take turns on either surface
      mcp_test.go:552: mrw_read's description still sells serialized writes as this surface's advantage:
          Use mrw always: plan the activity as one read of every site, then one plan, then one write. One mrw_read serves every site, and each served line is recorded so mrw_write may later edit it — served lines record nothing until you acknowledge them (see ack). Specs use mrw's own syntax: path, path:10-20, path:A,+N for the line A plus the N lines after it, path:/regexp/ so the read finds its own site, or path:$ for the last line. A read too large for one answer comes back as a PAGE, not a failure: the lines that fit, a -- PARTIAL: line, and a next_read spec for the rest. Repeat until next_read is absent — its absence is how you know you have the whole file, and you may only edit lines a page served. The served text is the first text block; the receipt (observed spans, problems, next_read) is the second, as JSON. Set `grep` to a regexp to FIND files you cannot name: it walks the paths (or the whole root) and serves every match, and returns an index of matching files when the matches are too large to serve. With a shell and mrw on PATH, prefer the CLI `mrw read` — it also has --files-from, and `mrw --root DIR read` for any checkout (--root BEFORE the subcommand; after `read`, -C is the context flag). Prefer THIS tool with no shell, or when callers sharing one checkout want their ledger writes serialized.
      mcp_test.go:552: mrw_write's description still sells serialized writes as this surface's advantage:
          Use mrw always: plan the activity as one read of every site, then one plan, then one write. Every edit travels in ONE plan and every hunk gets a verdict, so a replacement that matched nothing is reported rather than silently skipped. All or nothing: if any hunk fails validation, nothing is written. Every address resolves against the ORIGINAL file, so several hunks in one file need no offset arithmetic. mrw will not edit a line it has not served you — read it with mrw_read first. If you can run shell commands, prefer the CLI `mrw write` — it also has --check, which runs the project's tests scoped to what it just wrote. Prefer THIS tool with no shell, or when callers sharing one checkout want their ledger writes serialized; it needs no --json because its answer is already structured.
  --- FAIL: TestTheSurfaceSaysTheCLIIsRicher (0.33s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.492s
  FAIL
  ```
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:1438e2ffdd99dc04735f0f965995c1c1b96eca5abefdcb9891871cf1f4bf79ce · ms:1371
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:1438e2ffdd99dc04735f0f965995c1c1b96eca5abefdcb9891871cf1f4bf79ce · ms:954
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:1438e2ffdd99dc04735f0f965995c1c1b96eca5abefdcb9891871cf1f4bf79ce · ms:591
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6587edaff54c72f27fb94d5e964ba19e0055d68c3466b84337f25ff993e0e85e · ms:975
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6587edaff54c72f27fb94d5e964ba19e0055d68c3466b84337f25ff993e0e85e · ms:624
- 2026-09-26 · da2fd0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6587edaff54c72f27fb94d5e964ba19e0055d68c3466b84337f25ff993e0e85e · ms:735
- 2026-09-26 · 2c23bd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:6587edaff54c72f27fb94d5e964ba19e0055d68c3466b84337f25ff993e0e85e · ms:956

## Mutation Log
(empty until execute)
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the handshake sells serialized writes as this surface's advantage again · acceptance-sha256:1438e2ffdd99dc04735f0f965995c1c1b96eca5abefdcb9891871cf1f4bf79ce · covers:the surfaces stop selling serialized writes
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/mcp/mcp.go` · mrw_read's description sells serialized writes again · acceptance-sha256:1438e2ffdd99dc04735f0f965995c1c1b96eca5abefdcb9891871cf1f4bf79ce · covers:the surfaces stop selling serialized writes
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the handshake sells serialized writes as this surface's advantage again · acceptance-sha256:6587edaff54c72f27fb94d5e964ba19e0055d68c3466b84337f25ff993e0e85e · covers:the surfaces stop selling serialized writes
- 2026-09-26 · da2fd0a* · mutant killed · exit 1 · `internal/mcp/mcp.go` · mrw_read's description sells serialized writes again · acceptance-sha256:6587edaff54c72f27fb94d5e964ba19e0055d68c3466b84337f25ff993e0e85e · covers:the surfaces stop selling serialized writes

## Invariants

- A single writer on a checkout behaves exactly as before.

## Risks

- See the record.

## Out of Scope

- Everything the record lists (permanent: boundary: ADR-075 Out of Scope)

## Stop Condition

Stop if the lock needs the target file or a package the record does not govern.
