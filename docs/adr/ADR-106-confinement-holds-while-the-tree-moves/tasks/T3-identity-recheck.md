# Task ADR-106-T3: a target replaced after validation is refused

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the identity recheck before each content commit rename
**Consumes:** `tree` (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a replaced target is refused before its rename`, `a change before the first rename writes nothing`, `a change at a later file stops the commit there`

## Goal

Validation keeps the `FileInfo` of each existing target, its ID loaded. Every target is checked through the tree
before the first commit rename — a change stops the plan with nothing written — and each again immediately before
its own rename, where a change stops the commit at that file with earlier files written (ADR-066). A different
file, size or modification time each count; the refusal names the file.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `pending` carries the validation `FileInfo`; the recheck before `commitRenameFn` |
| `internal/apply/identity106_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test `TestACommitRefusesATargetThatChangedAfterValidation`; confirm RED: the rename replaces the file swapped in inside the root. [proof: mutation]
2. [S2] The two rechecks. [proof: mutation] Mutants: the pre-commit pass removed; the per-target recheck removed; the size clause removed; the modification-time clause removed; the `os.SameFile` clause removed. The ID-loading mutant (removing the load at validation) is killed only on Windows, where `os.SameFile` is lazy: the equal-size, equal-time case runs there in CI, and the local log records the mutant as not killable on this platform.

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestACommitRefusesATargetThatChangedAfterValidation' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestACommitRefusesATargetThatChangedAfterValidation \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/check internal/links \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACommitRefusesATargetThatChangedAfterValidation` | `internal/apply/identity106_test.go` | a plan editing `y.txt` then `a.txt`: with `a.txt` replaced by a new file while staging, the plan fails naming `a.txt`, writes nothing, and the newcomer survives; with `a.txt` replaced just before its own rename (through `commitRenameFn`, after `y.txt` landed), `y.txt` stays written, `a.txt` fails and keeps the newcomer; with one existing target alone, a replacement of equal size and restored modification time (identity only), an in-place rewrite of a new size (size only), and an in-place rewrite of equal size with a new modification time (time only) are each refused | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the recheck in `apply.go` |
| 2 — something selects it | every commit of an existing file |
| 3 — the caller can discover it | the refusal names the file |
| 4 — it is used | every edit of an existing file |

## Mutation Log
- 2026-09-30 · 321e066* · mutant killed · exit 1 · `internal/apply/apply.go` · the pre-commit pass removed · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · covers:a change before the first rename writes nothing
- 2026-09-30 · 321e066* · mutant killed · exit 1 · `internal/apply/apply.go` · the per-target recheck removed · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · covers:a change at a later file stops the commit there
- 2026-09-30 · 321e066* · mutant killed · exit 1 · `internal/apply/apply.go` · the size clause removed · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · covers:a replaced target is refused before its rename
- 2026-09-30 · 321e066* · mutant killed · exit 1 · `internal/apply/apply.go` · the modification-time clause removed · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · covers:a replaced target is refused before its rename
- 2026-09-30 · 321e066* · mutant killed · exit 1 · `internal/apply/apply.go` · the identity clause removed · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · covers:a replaced target is refused before its rename
- 2026-09-30 · 321e066* · mutant survived · exit 0 · `internal/apply/apply.go` · the file ID not loaded at validation (blind only on Windows, where os.SameFile is lazy) · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · covers:a replaced target is refused before its rename
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```

## Invariants

- A plan whose targets did not change is unaffected.

## Risks

- A filesystem with coarse modification times: the recheck also compares identity and size, and a same-size same-time rewrite in place is left to the sha guard of the next read.

## Stop Condition

Stop and ask if the recheck refuses a plan whose target nobody touched.

## Out of Scope

- The other ADR-106 tasks, each in its own file.

## Verification Log
- 2026-09-30 · 321e066* · exit 1 · `set -o pipefail …` · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · ms:311 · test-lock-sha256:1de66c9cd29310274b9fac221c5def93684c86668ae8a0e83c795251a148618a · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L2lkZW50aXR5MTA2X3Rlc3QuZ28JVGVzdEFDb21taXRSZWZ1c2VzQVRhcmdldFRoYXRDaGFuZ2VkQWZ0ZXJWYWxpZGF0aW9uCWEwMmEyMTlhNWZkOGVkMzhkMTg5ZjFmM2QwZTUxY2ZiNGQzMjA0N2ExZjdhMGY2ZmRhOWRhZWZhYWI2Y2QyMzQ
  ```
  --- last 8 line(s) of stdout
  === RUN   TestACommitRefusesATargetThatChangedAfterValidation
      identity106_test.go:42: the file swapped in after validation was overwritten: a.txt holds "A\nb\nc\nd\ne\n" (err <nil>)
      identity106_test.go:45: a target replaced after validation was not refused by name: <nil> {Root:/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestACommitRefusesATargetThatChangedAfterValidation3344868686/001 DryRun:false Applied:true Files:[{Path:y.txt Created:false Written:true SHABefore:86dc03602dcf385217216784784a8ecf20e6400decc3208170b12fcb0afb6698 SHAAfter:2958accc685b73ccbbf51b90734268ed94cf82fadcea9631cd30545bd452f398 LinesFrom:5 LinesTo:5 Removed:false RenamedTo: Target:} {Path:a.txt Created:false Written:true SHABefore:86dc03602dcf385217216784784a8ecf20e6400decc3208170b12fcb0afb6698 SHAAfter:c208ece63dc6061a76a629fd98813ddd352f856348ad25b144f52aeb3355d883 LinesFrom:5 LinesTo:5 Removed:false RenamedTo: Target:}] Hunks:[{singleLineCode:false wrapTail:false Path:y.txt Addr:1 Op:replace Status:ok Reason: Removed:1 Added:1 SrcLine:0 RemovedFirst: RemovedLast: Echo:[] Balance: Kind:} {singleLineCode:false wrapTail:false Path:a.txt Addr:1 Op:replace Status:ok Reason: Removed:1 Added:1 SrcLine:0 RemovedFirst: RemovedLast: Echo:[] Balance: Kind:}] Failed:0 Advisories:0 DirsCreated:[] LeftBehind:[] StrictSingleLine:0 StrictWouldRefuse:0}
      identity106_test.go:48: y.txt was written by a plan that stopped: "Y\nb\nc\nd\ne\n"
  --- FAIL: TestACommitRefusesATargetThatChangedAfterValidation (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.074s
  FAIL
  ```
- 2026-09-30 · 321e066* · exit 0 · `set -o pipefail …` · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · ms:293
- 2026-09-30 · 321e066* · exit 0 · `set -o pipefail …` · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · ms:287
- 2026-09-30 · 321e066* · exit 0 · `set -o pipefail …` · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · ms:280
- 2026-09-30 · 321e066* · exit 0 · `set -o pipefail …` · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · ms:299
- 2026-09-30 · 321e066* · exit 0 · `set -o pipefail …` · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · ms:280
- 2026-09-30 · 321e066* · exit 0 · `set -o pipefail …` · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · ms:294
- 2026-09-30 · human-observed · note 2026-09-30 on the survived mutant 'the file ID not loaded at validation': on darwin and Linux os.SameFile compares the device and inode the Stat already holds, so removing the load changes nothing there; only on Windows does os.SameFile read the file ID lazily by name at comparison, and there the identity-only case (another file of equal size and the same time) is the kill. The Windows CI shards run that case; the local log cannot. The other five clauses each have a killed mutant; approved
- 2026-09-30 · 321e066* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · ms:0 · test-lock-sha256:45fbf7a87472b3016fd9f33a4f5061201eb84b69261ce24d06ae40bc87154b30 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L2lkZW50aXR5MTA2X3Rlc3QuZ28JVGVzdEFDb21taXRSZWZ1c2VzQVRhcmdldFRoYXRDaGFuZ2VkQWZ0ZXJWYWxpZGF0aW9uCTQ0MWFlNzgzODc5YWE3MDJiZDM3MmI1ZmJiZWZhZmVhZTIxOGU4NDBkM2U2NjU1MDNkMmVjYmE4NmZjMjA3NDMKYm9keQlpbnRlcm5hbC9hcHBseS9pZGVudGl0eTEwNl90ZXN0LmdvCXJlcGxhY2VkIGp1c3QgYmVmb3JlIGl0cyBvd24gcmVuYW1lOiB0aGUgY29tbWl0IHN0b3BzIHRoZXJlCTUxYWQzZTY3NmM4MTg3MDg5OTZiNTQyMTZlODQwOGMxNzBhMDBlMTBkOGI3ZTE3YTZiNjNjOGZiNWI5OTM0OWYKYm9keQlpbnRlcm5hbC9hcHBseS9pZGVudGl0eTEwNl90ZXN0LmdvCXJlcGxhY2VkIHdoaWxlIHN0YWdpbmc6IG5vdGhpbmcgaXMgd3JpdHRlbgkxY2Y0MzRmNTIyM2QyOGVhZmE0NjczNTc3ZDlhOTVkZDc1YWRhYTMyN2U1OTYxNGMzYzZkNmY4ZGUxOWI5YTQ1 · test-lock-kind:replace
- 2026-09-30 · human-observed · relock 2026-09-30 reviewed: after the red run TestACommitRefusesATargetThatChangedAfterValidation gained the cases the Codex review of the record asked for — a replacement just before its own rename (earlier files stay written), identity only, size only and time only — and its stageFileFn override takes the tree; the original staging case is kept; approved
- 2026-10-06 · d493df7* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · ms:0 · test-lock-sha256:3ec66cb5b91a149e9f1f197e1c28cb3204f57baadc8d4768a4263951fffdcab6 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvaWRlbnRpdHkxMDZfdGVzdC5nbwlUZXN0QUNvbW1pdFJlZnVzZXNBVGFyZ2V0VGhhdENoYW5nZWRBZnRlclZhbGlkYXRpb24JMWRkYzEwYTVlNGUwOGZjZjdkMzA1NzAxMTZkYzg5MmI5MDA1NGRmY2E1ODgxMTAyNGJiZTc1MzU1ODY2YzQ3NQpib2R5CWludGVybmFsL2FwcGx5L2lkZW50aXR5MTA2X3Rlc3QuZ28JcmVwbGFjZWQganVzdCBiZWZvcmUgaXRzIG93biByZW5hbWU6IHRoZSBjb21taXQgc3RvcHMgdGhlcmUJNTFhZDNlNjc2YzgxODcwODk5NmI1NDIxNmU4NDA4YzE3MGEwMGUxMGQ4YjdlMTdhNmI2M2M4ZmI1Yjk5MzQ5Zgpib2R5CWludGVybmFsL2FwcGx5L2lkZW50aXR5MTA2X3Rlc3QuZ28JcmVwbGFjZWQgd2hpbGUgc3RhZ2luZzogbm90aGluZyBpcyB3cml0dGVuCTFjZjQzNGY1MjIzZDI4ZWFmYTQ2NzM1NzdkOWE5NWRkNzVhZGFhMzI3ZTU5NjE0YzNjNmQ2ZjhkZTE5YjlhNDU · test-lock-kind:replace
- 2026-10-06 · 5da531b* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:832eb443f5972df3fed1e5cea98e2487916f97871b04b212634a26443d42e51b · ms:0 · test-lock-sha256:cc79b0ac5753b777ac710157dbc436fab0b68f5d0e912d7caa28be72d10cb33b · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvaWRlbnRpdHkxMDZfdGVzdC5nbwlUZXN0QUNvbW1pdFJlZnVzZXNBVGFyZ2V0VGhhdENoYW5nZWRBZnRlclZhbGlkYXRpb24JNDdkYzUxMzFjMWE1OTk1OTRmNGI2OWFkZDkzNmVmMTIwMWNjNjgyM2E1NGQwM2VhY2I2ZWY4Nzc2OWEwMTQ1Ngpib2R5CWludGVybmFsL2FwcGx5L2lkZW50aXR5MTA2X3Rlc3QuZ28JcmVwbGFjZWQganVzdCBiZWZvcmUgaXRzIG93biByZW5hbWU6IHRoZSBjb21taXQgc3RvcHMgdGhlcmUJN2E2MWUzOGJlMGU2NjYzMDI3YTE4NTExOGE3ZjdmYTZlZTQzODY5NDkzZDUzY2Q0MWM0NTU2MGFiNzQ3MzY1MQpib2R5CWludGVybmFsL2FwcGx5L2lkZW50aXR5MTA2X3Rlc3QuZ28JcmVwbGFjZWQgd2hpbGUgc3RhZ2luZzogbm90aGluZyBpcyB3cml0dGVuCTI3MGE0YWYxY2M5MzI0NzMwNGEyY2VmY2ZmNDgyOWI2NjAzZTBlZDFkODJmZGUxMWZlYmU1NWQ4NDAwNGQzNGY · test-lock-kind:replace
