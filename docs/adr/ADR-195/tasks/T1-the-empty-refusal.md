# Task ADR-195-T1: `--create` refuses empty standard input and names the `body=0` plan

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `CompileCreate`'s empty refusal; `TestCreateRefusesEmptyInputAndNamesTheEmptyFilePlan`; contract §336
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `zero bytes are refused`, `one byte or more is created as before`, `the built binary exits 2 and creates nothing`, `the README no longer promises an empty file`

## Goal

Decisions 1-4 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/create195_test.go` | new | the red test |
| `internal/ingest/create.go` | edit | `CompileCreate` refuses zero bytes; its doc comment (`create.go:102-103`, "empty content makes an empty file") says so |
| `internal/ingest/create139_test.go` | edit | its empty case pins ADR-139 Decision 3; it moves to the refusal. ADR-139's acceptance digest moves with it, so lint ADR-139 too |
| `scripts/contract.sh` | edit | §336, through the built binary |
| `README.md` | edit | line 326's "empty input makes an empty file" |
| `AGENTS.md` | edit | the `--create` paragraph says empty input is refused and names `body=0` |
| `internal/guide/guide.go` | edit | `mrw instructions`' `--create` sentence (`guide.go:99`), where a caller with only the binary learns the flag |
| `cmd/mrw/main.go` | none | selects the refusal already: `CompileCreate`'s error is a content refusal, exit 2, not counted (`main.go:2183-2186`) |

## Ordered Steps

1. [S1] Write the failing test `TestCreateRefusesEmptyInputAndNamesTheEmptyFilePlan` and see it RED. Cases, through `CompileCreate` and through `CreateContent` first, as `main.go` calls them:
   - zero bytes: an error naming `standard input was empty` and `body=0`;
   - zero bytes for `with space.txt` and for `q"b\\c.txt`: the suggested plan, cut from the error, parses with `plan.Parse` to one `create` of exactly that path with `body=0`;
   - `"\n"`: one empty line, compiled as now;
   - `"\r\n"` (what PowerShell pipes for an empty string): created as now, never trimmed to zero bytes;
   - `"x"`: compiled as now.
2. [S2] `CompileCreate` refuses zero bytes before it splits anything. Mutants, each killed:
   - the check removed, so the zero-byte case goes red;
   - the check written as "only whitespace" instead of "no bytes", so the `"\n"` case goes red;
   - the suggested plan's path left unquoted, so the `with space.txt` case goes red.
   [proof: mutation]
3. [S3] `TestCompileCreateMakesACreatePlanOfStdin`'s empty case becomes the refusal; the rest of it is unchanged. [proof: acceptance]
4. [S4] Contract §336, through the built binary: `: | mrw write --create e.txt` exits 2, `e.txt` does not exist, and the message names `body=0`; the pair: `printf '\n' | mrw write --create nl.txt` exits 0 and makes a 1-byte file, and `printf '@@ e.txt 0 create body=0\n' | mrw write -` exits 0 and makes the empty file; and an empty `--create` of a PATH that exists exits 2, the file unchanged (the refusal comes before the apply). [proof: acceptance]
5. [S5] README, AGENTS.md and `mrw instructions` say so. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/ingest/ -count=1 -v -run '^(TestCreateRefusesEmptyInputAndNamesTheEmptyFilePlan|TestCompileCreateMakesACreatePlanOfStdin)$' 2>&1) \
  && grep -qE '^--- PASS: TestCreateRefusesEmptyInputAndNamesTheEmptyFilePlan \(' <<<"$out" \
  && grep -qE '^--- PASS: TestCompileCreateMakesACreatePlanOfStdin \(' <<<"$out" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 336\. ' scripts/contract.sh \
  && grep -qF 'PASS  --create with empty standard input exits 2 and creates nothing' <<<"$c" \
  && grep -qF 'PASS  the pair: a lone newline is created, and the body=0 plan makes the empty file' <<<"$c" \
  && grep -qF 'standard input was empty' internal/ingest/create139_test.go \
  && ! grep -qF 'empty input makes an empty file' README.md \
  && grep -qF 'body=0' <(grep -F -- '--create' README.md) \
  && grep -qiE 'empty (standard )?input is refused' AGENTS.md \
  && grep -qiE 'empty (standard )?input is refused' internal/guide/guide.go \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/apply internal/plan internal/read internal/check internal/state internal/seen internal/writer \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/read internal/check internal/state internal/seen internal/writer)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestCreateRefusesEmptyInputAndNamesTheEmptyFilePlan` | `internal/ingest/create195_test.go` | zero bytes refused, naming `body=0`; a newline, a CRLF and a byte created as now | none | S1, S2 |
| `TestCompileCreateMakesACreatePlanOfStdin` | `internal/ingest/create139_test.go` | ADR-139's cases, its empty one now the refusal | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `mrw write --create`, through `CompileCreate` (`cmd/mrw/main.go:2183`) |
| 3 — the caller can discover it | the refusal names the plan; README and AGENTS.md |
| 4 — it is used | no telemetry (ADR-009); the evidence for building it is issue #606's run-9 crash |

## Mutation Log

## Verification Log

## Invariants

- Content of one byte or more compiles exactly as before.
- The refusal writes nothing and is not counted by the parse tally.

## Risks

- None beyond the record's.

## Stop Condition

The fence exits 0. Stop and ask Zy if any existing test or contract row other than `TestCompileCreateMakesACreatePlanOfStdin`'s empty case relies on empty standard input making a file.

## Out of Scope

- `body=@path` naming an empty file (deferred: docs/adr/BACKLOG.md)
