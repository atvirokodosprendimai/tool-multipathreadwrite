# Task ADR-114-T1: the engine edits and renames a file in one plan

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** the engine accepts line edits plus one rename on a file
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the engine edits and renames a file in one plan`

## Goal

`apply.Apply` accepts line edits plus exactly one `rename` on a file, commits the edit then the rename, keeps one record per path (the rename rewrites the source's record and appends the destination's with the edited sha and line count), and restores the source's edited record when a later failure undoes the rename.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | the mix check; the rename planned beside the edits; the two pendings; one dry-run record; one unwritten record on commit failure |
| `internal/apply/pathop.go` | edit | `renameOne` rewrites the edited source's record; the undo restores it |
| `internal/apply/editrename114_test.go` | add | the tests |
| `internal/apply/chaos114_test.go` | add | `TestEditRenameChaos`: seeded random plans with injected commit failures, against disk, receipt and verdict invariants |

## Ordered Steps

1. [S1] Write `TestAFileIsEditedAndRenamedInOnePlan`: an edit + rename lands the edited content at the destination, the source is gone, the receipt has one record per path with the edited sha at the destination; a dry run lists the source once; unlink beside an edit and two renames stay refused; a partial read refuses the rename; a later rename failure (injected with `commitRenameFn`) restores the source's record as written with the edited sha. Confirm RED. [proof: mutation]
2. [S2] The engine change. Mutants: the mix check refuses again; `renameOne` appends instead of rewriting; the undo drops the edited record. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestAFileIsEditedAndRenamedInOnePlan' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAFileIsEditedAndRenamedInOnePlan \(' "$out" \
  && go test ./internal/apply/ ./internal/writer/ -count=1 -timeout 600s \
  && [ -z "$(gofmt -l cmd internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFileIsEditedAndRenamedInOnePlan` | `internal/apply/editrename114_test.go` | edit + rename lands the edited content at the destination with one record per path; dry run lists the source once; unlink + edit and two renames refused; a partial read refuses; a later failure restores the source's edited record | none | S1, S2 |
| `TestEditRenameChaos` | `internal/apply/chaos114_test.go` | 300 seeded multi-file plans (edit, edit+rename, rename, unlink) with a commit rename failing at a random step: a clean apply leaves exactly the model; every file is in a state the plan explains; one record per path matching the disk; an ok verdict is on disk and a non-ok one is not; no unnamed .mrw-* file. 20,000 seeds clean on 2026-10-02; both planted defects (append instead of rewrite; the verdict fix reverted) caught | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every plan with edits and a rename on one file |
| 3 — the caller can discover it | the receipt's records; AGENTS.md (T2) |
| 4 — it is used | Codex apply_patch emits Move to with hunks (T2); telemetry is refused (ADR-009) |

## Mutation Log
- 2026-10-02 · e1c7646* · mutant killed · exit 1 · `internal/apply/apply.go` · the mix check refuses edit+rename again · acceptance-sha256:4ad5dc5d1ed364bffedd21c712f5a51702df3d6f22d6cf1463b0150eac6a8460
- 2026-10-02 · e1c7646* · mutant killed · exit 1 · `internal/apply/pathop.go` · renameOne appends a second source record instead of rewriting · acceptance-sha256:4ad5dc5d1ed364bffedd21c712f5a51702df3d6f22d6cf1463b0150eac6a8460
- 2026-10-02 · e1c7646* · mutant killed · exit 1 · `internal/apply/pathop.go` · the undo drops the edited record instead of restoring it · acceptance-sha256:4ad5dc5d1ed364bffedd21c712f5a51702df3d6f22d6cf1463b0150eac6a8460
- 2026-10-02 · e1c7646* · mutant killed · exit 1 · `internal/apply/apply.go` · the verdict fix reverted: an edit that landed reads failed · acceptance-sha256:4ad5dc5d1ed364bffedd21c712f5a51702df3d6f22d6cf1463b0150eac6a8460

## Invariants

- Every plan the engine accepted before applies as before; no receipt key changes.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The apply_patch compiler (permanent: boundary: T2)

## Verification Log
- 2026-10-02 · e1c7646* · exit 1 · `set -o pipefail …` · acceptance-sha256:4ad5dc5d1ed364bffedd21c712f5a51702df3d6f22d6cf1463b0150eac6a8460 · ms:335 · test-lock-sha256:1506f927edd816f87097c4f930e92264df2b9c886510cc02c71224fcff29f468 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvZWRpdHJlbmFtZTExNF90ZXN0LmdvCVRlc3RBRmlsZUlzRWRpdGVkQW5kUmVuYW1lZEluT25lUGxhbgljZTk0ZDRlNTBlODBiYzJmMDJlNDMyYjI1OTkzZTkwYjQ1NzZkYWNjMDNlNGQxMGE2N2U1ODc2MDA5NjFkMWEwCmJvZHkJaW50ZXJuYWwvYXBwbHkvZWRpdHJlbmFtZTExNF90ZXN0LmdvCWEgZHJ5IHJ1biBsaXN0cyB0aGUgc291cmNlIG9uY2UgYW5kIHdyaXRlcyBub3RoaW5nCTMyYjU3YmU1NDRmYzg0OTdlYjRjZGMwOTg5ODkzZmZkY2Q2MTMyMDNhOWUyZmJjNTY2ZDdkYzI5NGE0MzA2MjIKYm9keQlpbnRlcm5hbC9hcHBseS9lZGl0cmVuYW1lMTE0X3Rlc3QuZ28JYSBsYXRlciBmYWlsdXJlIHVuZG9lcyB0aGUgcmVuYW1lIGFuZCByZXN0b3JlcyB0aGUgZWRpdGVkIHJlY29yZAk5NmRmZDhjYTc5MzE0ZTllYTllNTU2Y2NkZTY4MWRmNWY0YmQ4MjgzYzE1ZGE4NjU5NDJhMTQ5NDdiZjA1NGI0CmJvZHkJaW50ZXJuYWwvYXBwbHkvZWRpdHJlbmFtZTExNF90ZXN0LmdvCWEgcGFydGlhbCByZWFkIHJlZnVzZXMgdGhlIHJlbmFtZQkxOGVhMDZmOTdjNTMxMDdlYmNhODYxYTJmODQ4Y2VkYjQ2ZjRlODVhYWIxYTVkMDRiMmQ2MmQ3OWViMDE4NDEyCmJvZHkJaW50ZXJuYWwvYXBwbHkvZWRpdHJlbmFtZTExNF90ZXN0LmdvCWEgcmVuYW1lIHRoYXQgZmFpbHMgbGVhdmVzIHRoZSBlZGl0IGxhbmRlZCBhbmQgbmFtZWQJZWQ3N2JjMzY2NjhjYTE0ZDA4YTNlMGU2YzA5NjA5OTEzMzQyNzk0ODE0ZTc3MWI2YTQ3YjRjMzUwOWU3ZDlhYwpib2R5CWludGVybmFsL2FwcGx5L2VkaXRyZW5hbWUxMTRfdGVzdC5nbwlhbiB1bmxpbmsgYmVzaWRlIGFuIGVkaXQsIGFuZCB0d28gcmVuYW1lcywgc3RheSByZWZ1c2VkCTQxZGI5MGM1Y2U2YWFhZjM3Yjg5ZDQwODVhODVjMDM1N2EwNDNjOTVlZjg0NmU3MDAzMTc3N2Q5OTE1YzhlMGUKYm9keQlpbnRlcm5hbC9hcHBseS9lZGl0cmVuYW1lMTE0X3Rlc3QuZ28JdGhlIGVkaXQgbGFuZHMgYXQgdGhlIGRlc3RpbmF0aW9uLCBlYWNoIHBhdGggbmFtZWQgb25jZQkzMzAxYWE4MTlkZDM4MjI0MTVhZGJmY2Y0NjQ2YjI4NGZkZmU2YWQwMGU0MTNkMGE4NjNhZGFiZjY3MTBjNWE5
  ```
  --- last 10 line(s) of stdout (of 21 after folding 21 raw)
  --- FAIL: TestAFileIsEditedAndRenamedInOnePlan (0.01s)
      --- FAIL: TestAFileIsEditedAndRenamedInOnePlan/the_edit_lands_at_the_destination,_each_path_named_once (0.00s)
      --- FAIL: TestAFileIsEditedAndRenamedInOnePlan/a_dry_run_lists_the_source_once_and_writes_nothing (0.00s)
      --- PASS: TestAFileIsEditedAndRenamedInOnePlan/an_unlink_beside_an_edit,_and_two_renames,_stay_refused (0.00s)
      --- PASS: TestAFileIsEditedAndRenamedInOnePlan/a_partial_read_refuses_the_rename (0.00s)
      --- FAIL: TestAFileIsEditedAndRenamedInOnePlan/a_rename_that_fails_leaves_the_edit_landed_and_named (0.00s)
      --- FAIL: TestAFileIsEditedAndRenamedInOnePlan/a_later_failure_undoes_the_rename_and_restores_the_edited_record (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.095s
  FAIL
  ```
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ad5dc5d1ed364bffedd21c712f5a51702df3d6f22d6cf1463b0150eac6a8460 · ms:4062
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ad5dc5d1ed364bffedd21c712f5a51702df3d6f22d6cf1463b0150eac6a8460 · ms:3848
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ad5dc5d1ed364bffedd21c712f5a51702df3d6f22d6cf1463b0150eac6a8460 · ms:3315
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ad5dc5d1ed364bffedd21c712f5a51702df3d6f22d6cf1463b0150eac6a8460 · ms:3519
- 2026-10-02 · e1c7646* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:4ad5dc5d1ed364bffedd21c712f5a51702df3d6f22d6cf1463b0150eac6a8460 · ms:0 · test-lock-sha256:f45f93d61566806d95cec1fcbe67c44bea806727887662db480bc616ce09ce3c · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvY2hhb3MxMTRfdGVzdC5nbwlUZXN0RWRpdFJlbmFtZUNoYW9zCTUzNjRiZmE4MGY2Y2RhM2QwOWQ5MTA3ZmY2YjUwZjg1YzljODI3NDY3OThlYjBjNTdlOGU0YmQwNzliMjdmMDMKYm9keQlpbnRlcm5hbC9hcHBseS9lZGl0cmVuYW1lMTE0X3Rlc3QuZ28JVGVzdEFGaWxlSXNFZGl0ZWRBbmRSZW5hbWVkSW5PbmVQbGFuCWNlOTRkNGU1MGU4MGJjMmYwMmU0MzJiMjU5OTNlOTBiNDU3NmRhY2MwM2U0ZDEwYTY3ZTU4NzYwMDk2MWQxYTAKYm9keQlpbnRlcm5hbC9hcHBseS9lZGl0cmVuYW1lMTE0X3Rlc3QuZ28JYSBkcnkgcnVuIGxpc3RzIHRoZSBzb3VyY2Ugb25jZSBhbmQgd3JpdGVzIG5vdGhpbmcJMzJiNTdiZTU0NGZjODQ5N2ViNGNkYzA5ODk4OTNmZmRjZDYxMzIwM2E5ZTJmYmM1NjZkN2RjMjk0YTQzMDYyMgpib2R5CWludGVybmFsL2FwcGx5L2VkaXRyZW5hbWUxMTRfdGVzdC5nbwlhIGxhdGVyIGZhaWx1cmUgdW5kb2VzIHRoZSByZW5hbWUgYW5kIHJlc3RvcmVzIHRoZSBlZGl0ZWQgcmVjb3JkCTk2ZGZkOGNhNzkzMTRlOWVhOWU1NTZjY2RlNjgxZGY1ZjRiZDgyODNjMTVkYTg2NTk0MmExNDk0N2JmMDU0YjQKYm9keQlpbnRlcm5hbC9hcHBseS9lZGl0cmVuYW1lMTE0X3Rlc3QuZ28JYSBwYXJ0aWFsIHJlYWQgcmVmdXNlcyB0aGUgcmVuYW1lCTE4ZWEwNmY5N2M1MzEwN2ViY2E4NjFhMmY4NDhjZWRiNDZmNGU4NWFhYjFhNWQwNGIyZDYyZDc5ZWIwMTg0MTIKYm9keQlpbnRlcm5hbC9hcHBseS9lZGl0cmVuYW1lMTE0X3Rlc3QuZ28JYSByZW5hbWUgdGhhdCBmYWlscyBsZWF2ZXMgdGhlIGVkaXQgbGFuZGVkIGFuZCBuYW1lZAllZDc3YmMzNjY2OGNhMTRkMDhhM2UwZTZjMDk2MDk5MTMzNDI3OTQ4MTRlNzcxYjZhNDdiNGMzNTA5ZTdkOWFjCmJvZHkJaW50ZXJuYWwvYXBwbHkvZWRpdHJlbmFtZTExNF90ZXN0LmdvCWFuIHVubGluayBiZXNpZGUgYW4gZWRpdCwgYW5kIHR3byByZW5hbWVzLCBzdGF5IHJlZnVzZWQJNDFkYjkwYzVjZTZhYWFmMzdiODlkNDA4NWE4NWMwMzU3YTA0M2M5NWVmODQ2ZTcwMDMxNzc3ZDk5MTVjOGUwZQpib2R5CWludGVybmFsL2FwcGx5L2VkaXRyZW5hbWUxMTRfdGVzdC5nbwl0aGUgZWRpdCBsYW5kcyBhdCB0aGUgZGVzdGluYXRpb24sIGVhY2ggcGF0aCBuYW1lZCBvbmNlCTMzMDFhYTgxOWRkMzgyMjQxNWFkYmZjZjQ2NDZiMjg0ZmRmZTZhZDAwZTQxM2QwYTg2M2FkYWJmNjcxMGM1YTk · test-lock-kind:replace
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:4ad5dc5d1ed364bffedd21c712f5a51702df3d6f22d6cf1463b0150eac6a8460 · ms:2194
