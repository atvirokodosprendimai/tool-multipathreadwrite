# Task ADR-018-T1: Refuse the root nobody named

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (one guard, one call site, one contract row)
**Owner:** unassigned
**Produces:** the fallback-root guard and its refusal text
**Consumes:** `mcp.ResolveRoot` and `mcp.Source` (ADR-011, unchanged)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the guard keys on the source, not on the path`, `an explicit root is still honoured`, `the guard is actually wired into the binary`

## Goal

A server that nobody told which tree to serve, and that landed on a directory which cannot be a
project, refuses to start and says how to name one — while every stated intent still starts.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/root.go` | edit | `CheckRoot` — the guard, keyed on `Source` |
| `internal/mcp/root_test.go` | edit | **the ADR's Enforced-by**, six cases |
| `cmd/mrw/main.go` | edit | the call site, before `Serve` — a guard nothing calls is not a guard |
| `README.md` | edit | its MCP section describes an unconditional fallback, which stops being true |
| `scripts/contract.sh` | edit | §53 — drive the built binary from `/` |

## Ordered Steps

1. [S1] Write the failing test first (TDD red): a fallback onto a filesystem root or the home
   directory is refused; an explicit `--root` naming either is NOT; a project-variable root is not;
   an ordinary directory reached by fallback is not. [proof: acceptance]
2. [S2] Key the guard on `Source`, not on a list of paths. The value already exists — this is why
   the record is small — and a path list would be a rule defended by taste rather than by a
   principle. [proof: mutation]
3. [S3] Detect a filesystem root as `filepath.Dir(p) == p` rather than comparing to `"/"`. This
   codebase has already shipped one guard that silently passed on Windows because `filepath.IsAbs`
   is FALSE for `/etc/hosts` there; a literal `"/"` would miss a volume root the same way.
   [proof: mutation]
4. [S4] Resolve symlinks on both sides of the home comparison. `/tmp` against `/private/tmp` on
   macOS is the case that makes an unresolved comparison quietly wrong. [proof: mutation]
5. [S5] Make the refusal name the directory, the reason, and the flag that fixes it. A refusal with
   no way forward is the dead end ADR-014 and ADR-015 both exist to remove. [proof: acceptance]
6. [S6] WIRE IT, before `Serve`, at exit 2. [proof: acceptance]
7. [S7] Fix README: its MCP section says the server "binds to its own working directory" with no
   qualification, which stops being true here. [proof: acceptance]
8. [S8] Add contract §53: drive the BUILT BINARY from `/` with the variable unset and assert exit 2;
   assert the refusal names `--root`; assert an explicit `--root /` still exits 0; assert an
   ordinary fallback still exits 0. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr018-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr018-t1.out \
  && [ "$(grep -cE '^--- PASS: (TestAFallbackRootThatIsNotAProjectIsRefused)\b' /tmp/adr018-t1.out)" = "1" ] \
  && grep -q '^# 53\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the test name, `# 53.`
and the Go identifier `CheckRoot`. **§53 rather than §51**: 50 is the highest on `main`, and ADR-017
holds 51 and 52 on #80, so this takes the next free one rather than colliding on merge — the same
choice that made #74 and #76 merge textually.

⚠ **THE UNIT TEST ALONE CANNOT PROVE THIS.** `CheckRoot` can be correct and called by nothing, and
the binary would go on serving `/` with every Go test green. §53 is the row that binds, because it
runs the built binary from `/` and reads its exit status — which is also why S6 is a numbered step
rather than an implementation detail.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFallbackRootThatIsNotAProjectIsRefused` | `internal/mcp/root_test.go` | **the ADR's Enforced-by** — the guard keys on source, refuses the two impossible roots, and honours every stated intent | — | S1, S2, S3, S4, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test above |
| 2 — something selects it | `cmd/mrw`'s `mcp` command calls it before `Serve`; §53 proves that through the real binary |
| 3 — the caller can discover it | the refusal itself names the flag, and README documents the rule |
| 4 — it is used | nothing counts how often a host is refused, and nothing will — telemetry is refused by ADR-009. The evidence for building it is issue #81's reproduction |

## Mutation Log
<!-- filled during execution -->
- 2026-09-04 · 419967d* · mutant killed · exit 1 · `cmd/mrw/main.go` · S6: unwire the guard — keep CheckRoot correct and stop calling it. This is the mutant a unit test CANNOT kill: every Go test on CheckRoot still passes while the built binary goes on serving / . Only contract SS53, which runs the real binary from / and reads its exit status, turns red. This corpus has already shipped a gate that passed because the thing it checked was never reached · acceptance-sha256:a537b7b00f060c6681774ad1c4c3d00f92d686cd826dfbfd53809e14d0ee6918

## Invariants

- A fallback root that is a filesystem root or the home directory is refused at exit 2.
- An explicit `--root` is honoured whatever it names, including both of those.
- A `CLAUDE_PROJECT_DIR` root is honoured.
- An ordinary directory reached by fallback still serves.
- The refusal names the directory and `--root`.
- No engine directory changes; `go.mod` declares exactly one requirement.

## Risks

- The guard is written and not wired, so the binary is unchanged. Mitigated by §53 driving the built
  binary rather than the function.
- "Explicit is honoured" regresses and the guard becomes a path blacklist. Asserted in both the unit
  test and §53.
- The home comparison misses through a symlink. Both sides are resolved first.

## Stop Condition

Stop if the rule needs to know what a project LOOKS like — a `.git`, a `go.mod`, a marker file. That
is the taste-based rule the parent ADR rejected, it would refuse the document folders this project is
trying to serve, and wanting it is the signal that the guard is being widened past what M accepted.

## Out of Scope

- Multi-root or repeatable `--root` (deferred: docs/adr/BACKLOG.md — the reach entry)
- Guarding the CLI's `--root` (permanent: boundary: parent ADR, go/no-go)
- Refusing on project markers (permanent: boundary: parent ADR, Alternatives)

## Verification Log
<!-- filled during execution -->
- 2026-09-04 · 419967d* · exit 0 · `set -o pipefail …` · acceptance-sha256:a537b7b00f060c6681774ad1c4c3d00f92d686cd826dfbfd53809e14d0ee6918 · ms:30174
