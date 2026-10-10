# Task ADR-143-T1: A write whose path lands in `.git` is refused

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `rooted.GitDir`, the refusal in `apply.resolve`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a write whose path has a .git component, spelled or real, is refused before anything is written, and a read of it is not`

## Goal

Decisions 1–5 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/gitdir.go` | add | `GitDir` |
| `internal/apply/apply.go` | edit | `resolve` calls it |
| `internal/rooted/gitdir143_test.go`, `internal/apply/gitdir143_test.go`, `cmd/mrw/gitdir143_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §246 |
| `AGENTS.md` | edit | one sentence in section 4 |

## Ordered Steps

1. [S1] Write `TestAPathInsideADotGitIsRefused` (rooted), `TestAPlanThatTouchesDotGitWritesNothing` (apply) and `TestWriteIntoDotGitIsRefusedAndReadIsNot` (cmd/mrw). Confirm RED.
2. [S2] `rooted.GitDir` and its call in `apply.resolve`. Mutants: the call removed; the real-location check removed; the case fold removed; the 8.3 short name never matched. [proof: mutation]
3. [S3] Contract §246 and AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/rooted/ -count=1 -timeout 300s -run 'TestAPathInsideADotGitIsRefused' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAPathInsideADotGitIsRefused \(' "$out" \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestAPlanThatTouchesDotGitWritesNothing' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAPlanThatTouchesDotGitWritesNothing \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestWriteIntoDotGitIsRefusedAndReadIsNot' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestWriteIntoDotGitIsRefusedAndReadIsNot \(' "$out" \
  && go test ./internal/rooted/ ./internal/apply/ -count=1 -timeout 900s \
  && grep -q '^# 246\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPathInsideADotGitIsRefused` | `internal/rooted/gitdir143_test.go` | `.git` components refused (spelled, folded, through a link, in the root, `git~1` on a Windows build); `.github`, `.gitignore`, `x.git` and `git` are not | none | S1, S2 |
| `TestAPlanThatTouchesDotGitWritesNothing` | `internal/apply/gitdir143_test.go` | create, replace, delete, unlink and a rename into or out of `.git` each fail their hunk and write nothing, siblings skip | none | S1, S2 |
| `TestWriteIntoDotGitIsRefusedAndReadIsNot` | `cmd/mrw/gitdir143_test.go` | the CLI exits 1 with the named reason for `--create` and a plan, and `mrw read` of the same path exits 0 | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `rooted.GitDir` |
| 2 — something selects it | the call in `apply.resolve`, which every hunk path and rename destination passes |
| 3 — the caller can discover it | the refusal names the cause and git as the tool; AGENTS.md |
| 4 — it is used | the Windows retest of v1.60.0 (2026-10-10) found the gap; no telemetry (ADR-009) |

## Invariants

- Reads of `.git` are unchanged.
- A refused plan writes nothing and its siblings skip (ADR-001); exit codes are the contract.
- A name that is not `.git` (`.github`, `.gitignore`, `x.git`) is never refused by this rule.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if refusing by the real location turns away a root or a path the existing suite or the contract legitimately writes.

## Out of Scope

- A hard link to a file under `.git`, and HFS+ ignorable code points (deferred: docs/adr/BACKLOG.md)

## Mutation Log
- 2026-10-10 · 194e48d* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: apply.resolve does not act on the refusal · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · covers:a write whose path has a .git component, spelled or real, is refused before anything is written, and a read of it is not
- 2026-10-10 · 194e48d* · mutant killed · exit 1 · `internal/rooted/gitdir.go` · S2: the real location is not judged, so a link into .git passes · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · covers:a write whose path has a .git component, spelled or real, is refused before anything is written, and a read of it is not
- 2026-10-10 · 194e48d* · mutant killed · exit 1 · `internal/rooted/gitdir.go` · S2: the case fold is removed · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · covers:a write whose path has a .git component, spelled or real, is refused before anything is written, and a read of it is not
- 2026-10-10 · 194e48d* · mutant killed · exit 1 · `internal/rooted/gitdir.go` · S2: the 8.3 short name is never matched · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · covers:a write whose path has a .git component, spelled or real, is refused before anything is written, and a read of it is not
- 2026-10-10 · 7c7c192* · mutant inconclusive · exit 1 · `internal/rooted/gitdir.go` · S2: a link entry inside .git is judged by its target only · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · covers:a write whose path has a .git component, spelled or real, is refused before anything is written, and a read of it is not
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-10 · 7c7c192* · mutant inconclusive · exit 1 · `internal/rooted/gitdir.go` · S2: the root as spelled is not judged, only where it leads · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · covers:a write whose path has a .git component, spelled or real, is refused before anything is written, and a read of it is not
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-10 · 7c7c192* · mutant killed · exit 1 · `internal/rooted/gitdir.go` · S2: the 8.3 name git~1 is never matched · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · covers:a write whose path has a .git component, spelled or real, is refused before anything is written, and a read of it is not
- 2026-10-10 · e42cca4 · mutant killed · exit 1 · `internal/rooted/gitdir.go` · S2: a link entry inside .git is judged by its target only · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · covers:a write whose path has a .git component, spelled or real, is refused before anything is written, and a read of it is not
- 2026-10-10 · e42cca4* · mutant killed · exit 1 · `internal/rooted/gitdir.go` · S2: the root as spelled is not judged, only where it leads · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · covers:a write whose path has a .git component, spelled or real, is refused before anything is written, and a read of it is not

## Verification Log
- 2026-10-10 · 194e48d* · exit 1 · `set -o pipefail …` · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · ms:1639 · test-lock-sha256:32dac01d13c7d7958eddc7104f69545ef70d38bc59a49f8dc5944160a65e6652 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9naXRkaXIxNDNfdGVzdC5nbwlUZXN0V3JpdGVJbnRvRG90R2l0SXNSZWZ1c2VkQW5kUmVhZElzTm90CTdjZDk5MDU3ODE3ZWI3OTE0Y2M5MjI4NjkwNDgzNzE3NDk3MjdiMWNmMjY4NDI4MThhMjNiNDUwOTJkZjFmNjYKYm9keQlpbnRlcm5hbC9hcHBseS9naXRkaXIxNDNfdGVzdC5nbwlUZXN0QVBsYW5UaGF0VG91Y2hlc0RvdEdpdFdyaXRlc05vdGhpbmcJNjVkMWM1YjQ5YjBiZDIzMjI3YTA0OTk2MmU1NjAxMWQ5NjdkMzEyM2I0OTZhNjkwYzVlYWQ4YjllMjU4Yjc3ZQpib2R5CWludGVybmFsL3Jvb3RlZC9naXRkaXIxNDNfdGVzdC5nbwlUZXN0QVBhdGhJbnNpZGVBRG90R2l0SXNSZWZ1c2VkCTZmOTgzNjMwMDE4MzNlMjJhNmQzOWU1NjFiMWFjYTBhMjg2OGQ4Y2ZhMjk3NzYwMDBiMmY4OTQ1NGQ0NmNkNGUKYm9keQlpbnRlcm5hbC9yb290ZWQvZ2l0ZGlyMTQzX3Rlc3QuZ28JVGVzdFRoZVNob3J0TmFtZU9mRG90R2l0SXNNYXRjaGVkT25seVdoZXJlSXRFeGlzdHMJNWQyN2FlNWZmMWQyM2JjZWRmNzY5MjE5MzI1OWEwYWNlMTQ0MmE4YmYwMjAwMmQ1ZTc1OTAyMTBlYzY4ZGJjOQ
  ```
  --- last 10 line(s) of stdout
  === RUN   TestAPathInsideADotGitIsRefused
  --- PASS: TestAPathInsideADotGitIsRefused (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted	0.105s
  === RUN   TestAPlanThatTouchesDotGitWritesNothing
      gitdir143_test.go:33: create: applied=true hunks=[{singleLineCode:false wrapTail:false Path:a.txt Addr:1 Op:replace Status:ok Reason: Removed:1 Added:1 SrcLine:0 RemovedFirst: RemovedLast: Echo:[] Balance: Closer: Kind:} {singleLineCode:false wrapTail:false Path:.git/hooks/pre-commit Addr:0 Op:create Status:ok Reason: Removed:0 Added:1 SrcLine:0 RemovedFirst: RemovedLast: Echo:[] Balance: Closer: Kind:}], want a refused plan with two verdicts
  --- FAIL: TestAPlanThatTouchesDotGitWritesNothing (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.186s
  FAIL
  ```
- 2026-10-10 · 194e48d* · exit 0 · `set -o pipefail …` · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · ms:4847
- 2026-10-10 · 194e48d* · exit 0 · `set -o pipefail …` · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · ms:5030
- 2026-10-10 · 194e48d* · exit 0 · `set -o pipefail …` · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · ms:5095
- 2026-10-10 · 194e48d* · exit 0 · `set -o pipefail …` · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · ms:4786
- 2026-10-10 · 7c7c192 · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · ms:0 · test-lock-sha256:0ccd1429040a625a61da712381637f2f64cb6bf6f857bcf99c4acc4d584d0f74 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9naXRkaXIxNDNfdGVzdC5nbwlUZXN0V3JpdGVJbnRvRG90R2l0SXNSZWZ1c2VkQW5kUmVhZElzTm90CTdjZDk5MDU3ODE3ZWI3OTE0Y2M5MjI4NjkwNDgzNzE3NDk3MjdiMWNmMjY4NDI4MThhMjNiNDUwOTJkZjFmNjYKYm9keQlpbnRlcm5hbC9hcHBseS9naXRkaXIxNDNfdGVzdC5nbwlUZXN0QVBsYW5UaGF0VG91Y2hlc0RvdEdpdFdyaXRlc05vdGhpbmcJNWI1OTc4NTk1YjZhMThmMDk4OWViYmZjMzMwM2NkOTUxM2E5MmU3ODViMWE3YjMzZGY5NDdjNzlmODEwNTNhZgpib2R5CWludGVybmFsL3Jvb3RlZC9naXRkaXIxNDNfdGVzdC5nbwlUZXN0QVBhdGhJbnNpZGVBRG90R2l0SXNSZWZ1c2VkCTFjNzRiM2IxY2QxZmUzMzg1NzVkMjdiMTU4NjVhYWMyMDI4NDM0YmQ0OTk1ZGNmOTczODFlNWUxNWQxMjIwMGYKYm9keQlpbnRlcm5hbC9yb290ZWQvZ2l0ZGlyMTQzX3Rlc3QuZ28JVGVzdFRoZVNob3J0TmFtZU9mRG90R2l0SXNNYXRjaGVkT25seVdoZXJlSXRFeGlzdHMJMWU2YjE5OWE5NGNmNTlkZmNhZTIxZDk3MDhjYjIwNzVhNDgzYWQ0NDAwYjQ0YTQ5ZmVlNGQ4YmY3ZjI5NTFiOQ · test-lock-kind:replace
- 2026-10-10 · 7c7c192* · exit 0 · `set -o pipefail …` · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · ms:7437
- 2026-10-10 · 7c7c192* · exit 0 · `set -o pipefail …` · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · ms:7332
- 2026-10-10 · 7c7c192* · exit 0 · `set -o pipefail …` · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · ms:6762
- 2026-10-10 · e42cca4 · exit 0 · `set -o pipefail …` · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · ms:7127
- 2026-10-10 · e42cca4* · exit 0 · `set -o pipefail …` · acceptance-sha256:664a82e74a24fc67ccc942c8ae9b407d34f92b65bcf220d6906237544783139f · ms:7780
