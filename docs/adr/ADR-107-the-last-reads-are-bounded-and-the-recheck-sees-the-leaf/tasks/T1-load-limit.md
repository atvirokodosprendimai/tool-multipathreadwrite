# Task ADR-107-T1: a file over the edit limit is refused before it is read

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** a file over the edit limit is refused before it is read
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a file over the limit is refused before it is read`

## Goal

Validation refuses a file a plan edits whose size is over `maxLoadBytes` (1 GiB) on its hunk, naming size and limit; `readLines` reads through a bound of the limit plus one byte.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `maxLoadBytes`; the size refusal at validation; `readLines` bounded |
| `internal/apply/load107_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test `TestAFileOverTheEditLimitIsRefusedBeforeItIsRead`; confirm RED. [proof: mutation]
2. [S2] The size refusal and the bounded read. Mutants: the size refusal removed; the bound removed from `readLines`. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestAFileOverTheEditLimitIsRefusedBeforeItIsRead' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAFileOverTheEditLimitIsRefusedBeforeItIsRead \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/check internal/links \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFileOverTheEditLimitIsRefusedBeforeItIsRead` | `internal/apply/load107_test.go` | with `maxLoadBytes` set small, a plan editing a larger file fails its hunk naming the size and the limit and writes nothing; a file grown past the limit between its stat and its read (through `loadFn`) fails its hunk with its sibling skipped; `readLines` of a 4 MB file past the limit allocates under 256 KB; a file under the limit edits as before | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every write, or every MCP acknowledgement |
| 3 — the caller can discover it | the refusal names the file and the reason |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 final design review |

## Mutation Log
- 2026-09-30 · 433ed20* · mutant killed · exit 1 · `internal/apply/apply.go` · the size refusal at validation removed · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · covers:a file over the limit is refused before it is read
- 2026-09-30 · 433ed20* · mutant survived · exit 0 · `internal/apply/apply.go` · readLines' bound removed · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · covers:a file over the limit is refused before it is read
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-30 · 433ed20* · mutant killed · exit 1 · `internal/apply/apply.go` · readLines' length check removed: a file grown past the limit reads as if whole · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · covers:a file over the limit is refused before it is read
- 2026-09-30 · 7f75ca1* · mutant killed · exit 1 · `internal/apply/apply.go` · readLines' bound removed (now asserted by allocation) · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · covers:a file over the limit is refused before it is read
- 2026-09-30 · 7f75ca1* · mutant killed · exit 1 · `internal/apply/apply.go` · the growth refusal returned as a bare error again · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · covers:a file over the limit is refused before it is read

## Invariants

- Exit codes keep their meanings; plans under the limits are unaffected.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-107 tasks, each in its own file.

## Verification Log
- 2026-09-30 · 433ed20* · exit 1 · `set -o pipefail …` · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · ms:302 · test-lock-sha256:33c33dca4d959f07e718b931856ba55287721282c52f407aa8b3161800895a86 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L2xvYWQxMDdfdGVzdC5nbwlUZXN0QUZpbGVPdmVyVGhlRWRpdExpbWl0SXNSZWZ1c2VkQmVmb3JlSXRJc1JlYWQJZmUzYjJlNmE2MWU0YTg1OTQwMTU3ZGVhMGY2NDNjNDdiMjUzNDMxMDU1YzRjYTJjZTE0ZDU0ZWMzNzQwYWFkOQ
  ```
  --- last 6 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply.test]
  internal/apply/load107_test.go:16:9: undefined: maxLoadBytes
  internal/apply/load107_test.go:17:21: undefined: maxLoadBytes
  internal/apply/load107_test.go:18:2: undefined: maxLoadBytes
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [build failed]
  FAIL
  ```
- 2026-09-30 · 433ed20* · exit 0 · `set -o pipefail …` · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · ms:297
- 2026-09-30 · 433ed20* · exit 0 · `set -o pipefail …` · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · ms:282
- 2026-09-30 · 433ed20* · exit 0 · `set -o pipefail …` · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · ms:284
- 2026-09-30 · human-observed · note 2026-09-30 on the survived mutant 'readLines' bound removed': without the LimitReader the length check after the read still refuses, so the outcome the test observes is the same and only the memory held differs; the length check itself has a killed mutant. The bound is kept because it is what makes the refusal cost at most limit plus one byte; approved
- 2026-09-30 · 7f75ca1* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · ms:0 · test-lock-sha256:eb82780c68f2606e334615e927aac95f63f2eb6d1cd120e17bacb003fe2768b8 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L2xvYWQxMDdfdGVzdC5nbwlUZXN0QUZpbGVPdmVyVGhlRWRpdExpbWl0SXNSZWZ1c2VkQmVmb3JlSXRJc1JlYWQJNjA0YjFmOTg3MTE5ZTE2Y2UyY2RjMDY0MjI0N2I3NzI5OGFlOGQ0ZjljNTQ1ZWQ1N2I2N2ZjMDllNzAzMjA3NA · test-lock-kind:replace
- 2026-09-30 · human-observed · relock 2026-09-30 reviewed: the Codex review of #302 asked for the growth refusal to keep every hunk verdict and for the read's bound to be asserted, so TestAFileOverTheEditLimitIsRefusedBeforeItIsRead gained a file grown past the limit through loadFn (its hunk failed, its sibling skipped) and an allocation bound on readLines of a 4 MB file; every earlier assertion kept; approved
- 2026-09-30 · 7f75ca1* · exit 1 · `set -o pipefail …` · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · ms:550
  ```
  --- last 4 line(s) of stdout
  === RUN   TestAFileOverTheEditLimitIsRefusedBeforeItIsRead
  --- PASS: TestAFileOverTheEditLimitIsRefusedBeforeItIsRead (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.071s
  ```
- 2026-09-30 · 7f75ca1* · exit 1 · `set -o pipefail …` · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · ms:413
  ```
  --- last 4 line(s) of stdout
  === RUN   TestAFileOverTheEditLimitIsRefusedBeforeItIsRead
  --- PASS: TestAFileOverTheEditLimitIsRefusedBeforeItIsRead (0.01s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.074s
  ```
- 2026-09-30 · 7f75ca1* · exit 0 · `set -o pipefail …` · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · ms:867
- 2026-09-30 · 7f75ca1* · exit 0 · `set -o pipefail …` · acceptance-sha256:96ccf4ce7b9da0d2ce5a695457fc3011d779ecb85095bd49e8253f4e8718a606 · ms:294
