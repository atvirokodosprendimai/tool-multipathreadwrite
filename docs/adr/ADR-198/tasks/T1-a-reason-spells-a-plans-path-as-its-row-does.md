# Task ADR-198-T1: A refusal reason spells a plan's path as its row does

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the reason separator and its one formatting function in `internal/apply`; the audit's converted sites formatting through it; `TestAReasonSpellsAPlansPathAsItsRowDoes`; contract §338
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a path inside a reason is spelled as the row spells it`, `a path the caller typed, a system path and the attrib advice are kept`, `the path fields keep the platform's spelling`, `the exit-2 line still finds the reason in its error`, `the built binary does not rewrite a name holding a backslash`, `ADR-197's endings are unchanged`, `the receipts of the CLI and mrw_write hold their own tests`, `the packages the task does not own are unchanged`

## Goal

Decisions 1-5 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/reason198_test.go` | new | the red test |
| `internal/apply/wirepath.go` | edit | the reason separator (the platform's, settable by a test) and the function that spells a path with it |
| `internal/apply/apply.go` | edit | the audit's converted sites format their path through it |
| `internal/apply/pathop.go` | edit | the rename and unlink sites |
| `internal/apply/fromcopy.go` | edit | `loadSource`, `sourceGate` and the `from=P: WHY` site |
| `internal/apply/replace.go` | edit | `identityRefusal`'s second path, through `openRefusal` |
| `scripts/contract.sh` | edit | §338, through the built binary |
| `docs/adr/BACKLOG.md` | edit | the deferral of `rooted`'s identity error |

## Ordered Steps

1. [S1] Write the failing test `TestAReasonSpellsAPlansPathAsItsRowDoes` and see it RED. Where the platform's separator is `/` the test sets the reason separator to `\` and its files are literally named with a backslash (`d\f.txt`, one file name); on Windows it uses the real separator and a real directory. Each case runs `Apply` in a temporary root and reads the reason `Apply` returned. One case for every converted site of the record's audit:
   - unread lines (`SPAN of d/f.txt has not been read`), with a ledger; and with `NoForce`, the `Read d/f.txt with mrw_read first` advice;
   - an unlink beside an edit; a `create` of a file that exists, alone and beside another hunk; `sha=` on a path that does not exist, for a line op and for an unlink;
   - the six pattern refusals: fewer matches than `occurrence=`, unread matches before it, no match, an ambiguous start, an end with no match, an end matching only above the start;
   - the anchor in no line;
   - two spellings of one existing file; two names that differ only in case, neither existing, one of them a rename's destination;
   - rename destinations: the source itself, named twice, also written, nesting with another destination and with a written file (names holding a literal `\` under a real `/` directory where the separator is `/`), existing, unusable, and the `such as D/NAME` example;
   - `from=`: a source that does not exist, a directory, an unread source (the `mrw read P:A-B` advice), unshown lines, and a span that runs past the source;
   - a rename destination that appears before commit (the reason and the error `Apply` returns carry the same text), one under a link to nothing, and one whose directory left the root after validation (added after the first green run: the implementer had found no way to reach `rename dest D: ERR`, and ADR-106's swap seam reaches it);
   - the `ALREADY WRITTEN` list of a commit that stopped after one file landed;
   - where the filesystem folds case, the respelling refusals (`respells the directory`, `rename source … is not spelled that way`); these subcases skip where it does not, and the test says which ran.
   Kept, each a case:
   - a trailing-separator path and an absolute rename destination are quoted as typed; `from=X` echoes the header;
   - a read-only file: the reason begins with the path and its `chmod u+w` or `attrib -r` command keeps the platform's spelling;
   - an absolute path in a reason (`mrw looked for FULL`, the root's own name holding a `\` where the separator is `/`) is byte for byte the system's;
   - every path field of the result `Apply` returned keeps the platform's spelling;
   - with `/` as the separator, a file named `a\b.txt` is named `a\b.txt` in its reason.
2. [S2] The sites format through the one function. Mutants, each killed:
   - the function returns its argument unchanged, so the anchor case goes red;
   - the unread-lines site formats the bare path, so that case goes red;
   - a rename-destination site formats the bare path, so that case goes red;
   - `sourceGate` formats the bare path, so the `from=` advice case goes red;
   - the `ALREADY WRITTEN` list joins the bare paths, so that case goes red; the staging refusal `rename dest D: ERR` formats the bare path, so the swapped-directory case goes red;
   - the read-only command formats through the function, so the kept `attrib`/`chmod` case goes red;
   - a path field is converted, so the path-fields case goes red;
   - the reason separator is fixed at `\`, so contract §338's first row goes red.
   [proof: mutation]
3. [S3] Contract §338, through the built binary on Linux, a guard of Decision 5 and not of the conversion: in a root holding a file named `a\b.txt`, a replace whose anchor is in no line exits 1, writes nothing, and its reason ends `no line of a\b.txt holds it`; the pair: the same plan on `sub/f.txt` ends `no line of sub/f.txt holds it`. [proof: acceptance]
4. [S4] `go test ./internal/apply/ ./cmd/mrw/ ./internal/mcp/ ./internal/writer/` pass: ADR-132's and ADR-167's `Slashed` cases, the exit-2 line, `mrw skill --dir`'s reason and the appended ack advice hold unchanged. [proof: acceptance]

Before the merge, outside the fence (no gate here runs Windows shells): the built binary is probed on Windows with issue #642's plan, and the row and the reason both read `sub/f.txt`.

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/apply/ -count=1 -v -run '^(TestAReasonSpellsAPlansPathAsItsRowDoes|TestAnAnchorMissNamesTheLinesThatHoldIt)$' 2>&1) \
  && grep -qE '^--- PASS: TestAReasonSpellsAPlansPathAsItsRowDoes \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAnAnchorMissNamesTheLinesThatHoldIt \(' <<<"$out" \
  && go test ./internal/apply/ ./cmd/mrw/ ./internal/mcp/ ./internal/writer/ -count=1 >/dev/null 2>&1 \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 338\. ' scripts/contract.sh \
  && grep -qF 'PASS  a name holding a backslash is not rewritten in a reason' <<<"$c" \
  && grep -qF 'PASS  the pair: a path in a directory is named with its slash' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/check internal/state internal/seen internal/writer internal/rooted internal/mcp cmd \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/check internal/state internal/seen internal/writer internal/rooted internal/mcp cmd)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAReasonSpellsAPlansPathAsItsRowDoes` | `internal/apply/reason198_test.go` | every converted site's path is spelled with `/`; typed paths, system paths, the read-only command and the path fields are kept | none | S1, S2 |
| `TestAnAnchorMissNamesTheLinesThatHoldIt` | `internal/apply/anchor197_test.go` | ADR-197's endings are unchanged | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | every refused hunk: the reason is built in `Apply`, which the CLI, `mrw_write` and `mrw skill --dir` call |
| 3 — the caller can discover it | it is the refusal itself |
| 4 — it is used | no telemetry (ADR-009); the evidence is issue #642's measurement |

## Mutation Log
- 2026-11-15 · 682b17a9* · mutant killed · exit 1 · `internal/apply/wirepath.go` · asPlan returns its argument unchanged, so every converted site prints the platform spelling and the anchor case must go red · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · covers:a path inside a reason is spelled as the row spells it
- 2026-11-15 · 682b17a9* · mutant killed · exit 1 · `internal/apply/apply.go` · the unread-lines site formats the bare path, so the SPAN of d/f.txt case must go red · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · covers:a path inside a reason is spelled as the row spells it
- 2026-11-15 · 682b17a9* · mutant killed · exit 1 · `internal/apply/pathop.go` · a rename-destination site formats the bare path, so the named-twice case must go red · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · covers:a path inside a reason is spelled as the row spells it
- 2026-11-15 · 682b17a9* · mutant killed · exit 1 · `internal/apply/fromcopy.go` · sourceGate formats the bare path inside its mrw read advice, so the unread-source case must go red · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · covers:a path inside a reason is spelled as the row spells it
- 2026-11-15 · 682b17a9* · mutant killed · exit 1 · `internal/apply/apply.go` · the read-only command is formatted through asPlan, so the kept chmod case must go red · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · covers:a path the caller typed, a system path and the attrib advice are kept
- 2026-11-15 · 682b17a9* · mutant killed · exit 1 · `internal/apply/apply.go` · a path field is converted in the engine, so the path-fields case must go red · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · covers:the path fields keep the platform's spelling
- 2026-11-15 · 682b17a9* · mutant killed · exit 1 · `internal/apply/wirepath.go` · the reason separator is a backslash where the platform separator is a slash, so the case that leaves the separator alone must go red · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · covers:a path the caller typed, a system path and the attrib advice are kept
- 2026-11-15 · 682b17a9* · mutant killed · exit 1 · `internal/apply/apply.go` · the ALREADY WRITTEN list joins the bare paths, so the stopped-commit case must go red · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · covers:a path inside a reason is spelled as the row spells it
- 2026-11-15 · 682b17a9* · mutant killed · exit 1 · `internal/apply/apply.go` · the staging refusal rename dest D: ERR formats the bare path, so the swapped-directory case must go red · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · covers:a path inside a reason is spelled as the row spells it

## Verification Log
- 2026-11-15 · 682b17a9* · exit 1 · `set -o pipefail …` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:1534 · test-lock-sha256:ba928dcc254a3b96a7f250dfaf49908d458bc7a9094f43a58b3a0ee2a6c2f25b · test-lock-file:docs/adr/locks/ba928dcc254a3b96a7f250dfaf49908d458bc7a9094f43a58b3a0ee2a6c2f25b.lock
  ```
  ```
- 2026-11-15 · 682b17a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:195006
- 2026-11-15 · 682b17a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:213444
- 2026-11-15 · 682b17a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:170663
- 2026-11-15 · 682b17a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:168187
- 2026-11-15 · 682b17a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:178426
- 2026-11-15 · 682b17a9* · exit 1 · `set -o pipefail …` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:166673
  ```
  ```
- 2026-11-15 · 682b17a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:164210
- 2026-11-15 · 682b17a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:168846
- 2026-11-15 · 682b17a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:167419
- 2026-11-15 · 682b17a9* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:0 · test-lock-sha256:b2f02863e160ae14a70e8a1bf210a2321f0bc9d79d45a9f865e35e56121b0562 · test-lock-file:docs/adr/locks/b2f02863e160ae14a70e8a1bf210a2321f0bc9d79d45a9f865e35e56121b0562.lock · test-lock-kind:replace
- 2026-11-15 · 682b17a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:247198
- 2026-11-15 · 682b17a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:196738
- 2026-11-15 · 682b17a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:8d9c8be8b8ce7290ffbc1e82213fbfbd5e1bc1ef3baf17b782c59e5621eea74f · ms:182113

## Invariants

- Where the separator is `/`, every reason is byte for byte what it is today.
- Every path field `Apply` returns, the ledger and the drift baseline keep the platform's spelling.

## Risks

- None beyond the record's.

## Stop Condition

The fence exits 0 and the Windows probe reads `/` in both places. Stop and ask Zy if a refusal site needs its path converted that the record's audit lists as kept, if an existing test pins a `\` inside a reason, or if code outside `internal/apply` needs a reason in the platform's spelling.

## Out of Scope

- The read side's refusals (permanent: boundary: the record leaves that surface out)
- `rooted`'s identity error (deferred: docs/adr/BACKLOG.md)
