# Task ADR-086-T2: a name the filesystem refuses is refused at staging

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `probeName` and its calls at staging
**Consumes:** the byte-preserving `splitHeader` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a refused create writes nothing`, `a refused rename writes nothing`, `a probe leaves nothing behind`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

A rename to `moved/\xffdash.txt` on APFS failed at commit after the plan's content edit had landed: PARTIALLY APPLIED, exit 2.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/probe086_test.go` | new | through `probeNameFn`: a refused create and a refused rename each write nothing, on every platform |
| `internal/apply/probe086_darwin_test.go` | new | the real APFS refusal, skipped on a volume that accepts the byte |
| `internal/apply/apply.go` | edit | `probeName`, `probeNameFn`, the two staging calls |

## Ordered Steps

1. [S1] Write both tests; on `537b896` the seam does not exist and the darwin case lands PARTIALLY APPLIED. [proof: mutation]
2. [S2] Probe each create target and rename destination at staging. [proof: mutation]

## Acceptance

```bash
set -o pipefail
go test ./internal/apply/ -count=1 -timeout 180s -run 'TestANameTheFilesystemRefusesWritesNothing|TestAnAPFSInvalidNameIsRefusedBeforeAnyWrite' -v 2>&1 | tee /tmp/adr086-T2.out \
  && missing=$(for t in TestANameTheFilesystemRefusesWritesNothing; do grep -qE "^--- PASS: $t \(" /tmp/adr086-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && ! grep -qE '^--- FAIL' /tmp/adr086-T2.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestANameTheFilesystemRefusesWritesNothing` | `internal/apply/probe086_test.go` | a refused create and a refused rename each fail their hunk, write nothing and leave nothing | — | S1, S2 |
| `TestAnAPFSInvalidNameIsRefusedBeforeAnyWrite` | `internal/apply/probe086_darwin_test.go` | the same, with a real `\xff` name on a volume that refuses it | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `probeName` |
| 2 — something selects it | every create and rename in every plan |
| 3 — the caller can discover it | the failed hunk names the name and the filesystem's reason |
| 4 — it is used | chaos seed 25; ADR-009 refuses telemetry |

## Verification Log
(empty until execute)
- 2026-09-27 · 537b896* · exit 1 · `set -o pipefail …` · acceptance-sha256:7cc5460d59bf6c82ba7321a087bcfeaa8456e261a6f3315661aa1d13bfcfb219 · ms:227 · test-lock-sha256:82dc980d78c5e32bb799f5779b32e4bd18a279d33c2956dc35c4796f34708058 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L3Byb2JlMDg2X2Rhcndpbl90ZXN0LmdvCVRlc3RBbkFQRlNJbnZhbGlkTmFtZUlzUmVmdXNlZEJlZm9yZUFueVdyaXRlCWMyNDVmZjFhNzdmYTY1NDE2Mzc3OTg0OGFlZmVmNGU2ZDcwMzcwNDBmMzU4NTM2NTEwZmY3MTUyYjU5MDE0OTEKYm9keQlpbnRlcm5hbC9hcHBseS9wcm9iZTA4Nl90ZXN0LmdvCVRlc3RBTmFtZVRoZUZpbGVzeXN0ZW1SZWZ1c2VzV3JpdGVzTm90aGluZwkzNzIxZmQyYzJiZTYzYjhhMWM5ZWNkMzY5MmZjMWQxNTM2MGU0YWM4NzdlZTFlOTgwZTIzZDAzNzFjNjVhN2E5
  ```
  --- last 6 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply.test]
  internal/apply/probe086_test.go:18:10: undefined: probeNameFn
  internal/apply/probe086_test.go:19:21: undefined: probeNameFn
  internal/apply/probe086_test.go:20:2: undefined: probeNameFn
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [build failed]
  FAIL
  ```
- 2026-09-27 · 537b896* · exit 0 · `set -o pipefail …` · acceptance-sha256:7cc5460d59bf6c82ba7321a087bcfeaa8456e261a6f3315661aa1d13bfcfb219 · ms:303
- 2026-09-27 · 537b896* · exit 0 · `set -o pipefail …` · acceptance-sha256:7cc5460d59bf6c82ba7321a087bcfeaa8456e261a6f3315661aa1d13bfcfb219 · ms:438
- 2026-09-27 · 537b896* · exit 0 · `set -o pipefail …` · acceptance-sha256:7cc5460d59bf6c82ba7321a087bcfeaa8456e261a6f3315661aa1d13bfcfb219 · ms:262

## Mutation Log
(empty until execute)
- 2026-09-27 · 537b896* · mutant killed · exit 1 · `internal/apply/apply.go` · a create target is not probed at staging · acceptance-sha256:7cc5460d59bf6c82ba7321a087bcfeaa8456e261a6f3315661aa1d13bfcfb219 · covers:a refused create writes nothing
- 2026-09-27 · 537b896* · mutant killed · exit 1 · `internal/apply/apply.go` · a rename destination is not probed at staging · acceptance-sha256:7cc5460d59bf6c82ba7321a087bcfeaa8456e261a6f3315661aa1d13bfcfb219 · covers:a refused rename writes nothing

## Invariants

- A plan whose names the filesystem accepts writes exactly what it wrote before.
- Nothing is left in the tree by a probe that could be removed.

## Risks

- The real refusal is observed only where the volume refuses a name; CI's Linux and Windows jobs drive the seam test, and the darwin test runs on macOS.

## Out of Scope

- A contract row for the refusal (permanent: fact: the contract script runs on Linux CI only, where ext4 accepts the byte; citation: file `AGENTS.md:63`)

## Stop Condition

The fence exits 0 on macOS, and the seam test passes on the Linux and Windows CI jobs.
