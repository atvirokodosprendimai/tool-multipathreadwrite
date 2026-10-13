# Task ADR-147-T1: the tree passes every new analyser

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** a tree every new analyser passes
**Consumes:** none
**Data dependency:** hermetic, except `go run` fetches actionlint on a cold cache
**Proof map:** v1
**Rests-on:** `ruff passes the tree`, `shellcheck passes the tree`, `actionlint passes the workflows`, `the static.sh comment reads once`, `the engine is untouched`, `go.mod keeps one requirement`

## Goal

Fix the 12 findings the record's Context lists at `f94694d`: ruff 2, shellcheck 9, actionlint 1, none silenced.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/chaos.py` | edit | F401 `hashlib`, F841 `wr` |
| `scripts/contract.sh` | edit | SC2164 at 75; SC2034 at 870, keeping the read the row depends on in a file, as other rows do; SC2010 at 7421, 8109, 8633 and 8975, each rewritten over a name list a glob produces, 8109's with `dotglob` because mrw's temp file is a dotfile (`internal/state/write.go:60`) |
| `scripts/measure.sh` | edit | SC2209 at 229: the assignment is a name, quoted |
| `.github/workflows/timing-123.yml` | edit | SC2034 at 60: the loop counter |
| `scripts/static.sh` | edit | lines 8-9 repeat a sentence and break the next one |

## Ordered Steps

1. [S1] The fence fails at `f94694d`: ruff reports F401 and F841. [proof: acceptance]
2. [S2] Fix each finding, with no `# shellcheck disable`. [proof: mutation]
3. [S3] Row 8109's rewrite still sees a leftover: a scratch directory holding only `.seen.tmp-x` makes the new test report it, and one holding only `seen` does not. [proof: human: the executor runs both probes and pastes their output into the PR]
4. [S4] `./scripts/contract.sh` exits 0 on the fixed tree, the case-folding rows run (this Mac folds case) and not skipped. [proof: human: the executor runs it once and pastes its last line and the 230/235 row lines into the PR]

## Acceptance

```bash
set -o pipefail
git ls-files -z '*.py' | xargs -0 ruff check --select F,E9 \
  && git ls-files -z '*.sh' | xargs -0 shellcheck -S warning \
  && SHELLCHECK_OPTS='-S warning' go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 \
  && ! grep -q 'python3 runs fence-prose\.$' scripts/static.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal cmd/mrw \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence runs the analysers itself; `scripts/contract.sh` keeps passing | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the fixed lines |
| 2 — something selects it | the fence's analysers; T2's gate |
| 3 — the caller can discover it | n/a: no declared interface |
| 4 — it is used | `scripts/contract.sh` and `scripts/chaos.py` run as before |

## Mutation Log
(empty until execute)
- 2026-10-13 · f94694d* · mutant killed · exit 1 · `scripts/chaos.py` · an unused import goes back into a tracked .py · acceptance-sha256:ac71e77db28bfd0e853b61a5f5ced2663834e19016fda3a41d17dd3ed4a85534 · covers:ruff passes the tree
- 2026-10-13 · f94694d* · mutant killed · exit 1 · `scripts/contract.sh` · a cd with no failure branch goes back into a tracked .sh · acceptance-sha256:ac71e77db28bfd0e853b61a5f5ced2663834e19016fda3a41d17dd3ed4a85534 · covers:shellcheck passes the tree
- 2026-10-13 · f94694d* · mutant killed · exit 1 · `.github/workflows/timing-123.yml` · an unused loop counter goes back into a workflow run: script · acceptance-sha256:ac71e77db28bfd0e853b61a5f5ced2663834e19016fda3a41d17dd3ed4a85534 · covers:actionlint passes the workflows

## Verification Log
(empty until execute)
- 2026-10-13 · f94694d* · exit 1 · `set -o pipefail …` · acceptance-sha256:ac71e77db28bfd0e853b61a5f5ced2663834e19016fda3a41d17dd3ed4a85534 · ms:48 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz
  ```
  --- last 10 line(s) of stdout (of 26 after folding 26 raw)
  801 | …     if "root:x:0:0" in o: fail("mcp", "served /etc/passwd", root, None, data, (p.returncode, o, e, 0))
  802 | …     wr = [r for r in resp if isinstance(r, dict) and r.get("id") == 4]
      |       ^^
  803 | …     if rd(os.path.join(root, names[0])).startswith(b"X\n"):
  804 | …         fail("mcp", "mrw_write applied with a bogus ack / no read", root, None, data, (p.returncode, o, e, 0))
      |
  help: Remove assignment to unused variable `wr`

  Found 2 errors.
  [*] 1 fixable with the `--fix` option (1 hidden fix can be enabled with the `--unsafe-fixes` option).
  ```
- 2026-10-13 · f94694d* · exit 0 · `set -o pipefail …` · acceptance-sha256:ac71e77db28bfd0e853b61a5f5ced2663834e19016fda3a41d17dd3ed4a85534 · ms:9148
- 2026-10-13 · f94694d* · exit 0 · `set -o pipefail …` · acceptance-sha256:ac71e77db28bfd0e853b61a5f5ced2663834e19016fda3a41d17dd3ed4a85534 · ms:9124
- 2026-10-13 · f94694d* · exit 0 · `set -o pipefail …` · acceptance-sha256:ac71e77db28bfd0e853b61a5f5ced2663834e19016fda3a41d17dd3ed4a85534 · ms:9851
- 2026-10-13 · human-observed · observed by the executor 2026-10-13: S3 row-8109 probe printed 'bad: a temp state file was left: .seen.tmp-x seen' for a dir holding .seen.tmp-x and 'ok: and no temp file stays beside it' for one holding only seen; S4 ./scripts/contract.sh exit 0 in 85 s ('contract holds'), rows 'step 1 ran and step 3 never did', 'and no temp file stays beside it', 'and the directory spells it as the plan asked', 'naming the source's spelling, the directory unchanged' and 'a pre-v2 whole-file licence is NOT honoured after upgrading' all PASS on case-folding APFS

## Invariants

- Every contract row edited asserts what it asserted: `./scripts/contract.sh` passes, and row 8109 still fails on a leftover temp file.
- Nothing under `internal/` or `cmd/mrw/` changes.

## Risks

- A rewritten `ls | grep` row passes for the wrong reason. Mitigation: S3 probes 8109 both ways, and the case-folding rows keep reading the directory's own spelling (the record's Decision 3 measured it).

## Out of Scope

- Wiring the analysers into the gate (T2's job), and knip, which finds nothing today (T2 runs it)

## Stop Condition

The fence exits 0 and `./scripts/contract.sh` exits 0.
