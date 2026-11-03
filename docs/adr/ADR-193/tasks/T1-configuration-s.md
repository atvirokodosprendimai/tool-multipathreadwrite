# Task ADR-193-T1: configuration S and the stage8 profile

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** configuration S, profile `stage8`, `EditParagraph`, `run.json` `edit_paragraph_sha256`, the stage8 canary
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `S is B plus the paragraph`, `B under stage8 is stage5's B`, `stage8 runs only B and S`, `the paragraph cannot change between blocks`, `the canary runs B and S`, `the hook passes a SEARCH/REPLACE heredoc`, `the bench suite still passes`, `the usage text names stage8`

## Goal

`bench run --profile stage8 --configs B,S` and `bench canary --profile stage8` run a configuration that is B with
ADR-193 Decision 1's paragraph after `mrw instructions`, and nothing else differs from B (ADR-193 Decisions 1 and 2).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/profile.go` | edit | `ProfileStage8`; `EditParagraph`, ADR-193 Decision 1's paragraph as one line; its digest |
| `internal/bench/runner.go` | edit | `Configs["S"]` (`Bash, Glob`, `Mrw: true`, `EditFormat: true`); `invocation` appends a newline and `EditParagraph` after `mrw instructions` for S (`:1166`, beside H's); `run.json` `edit_paragraph_sha256`, refused when a later block names another (`:348`); a `stage8Problem` beside `stage7Problem` (`:562`): only B and S, a model that runs S runs B; S outside stage8 refused (`:518`); the profile list in the `--profile` refusal |
| `internal/bench/canary.go` | edit | S's init tools equal B's; one canary per model runs B and S under stage8 |
| `cmd/bench/main.go` | edit | `stage8` in `--profile`, `S` in `--configs`, the canary's configurations under stage8 (`canaryConfigs`, `:271`), the usage text |
| `internal/bench/look.go` | edit | `bench look` refuses a block whose `edit_paragraph_sha256` is not block 1's, beside the H sentence (`Look`, the cross-block comparison) |
| `internal/bench/runner_test.go` | edit | the runner tests |
| `internal/bench/hook_test.go` | edit | the hook test: S's commands are not refused by bench |
| `cmd/bench/main_test.go` | edit | the canary configurations test |
| `docs/bench/README.md` | edit | configuration S, profile stage8 |

## Ordered Steps

1. [S1] Write the five failing tests and see them red. [proof: acceptance]
2. [S2] `Configs["S"]` and its invocation: B's tools, B's hook and refusal text, the prompt file `mrw instructions`, a newline, then `EditParagraph` byte for byte as ADR-193 Decision 1 gives it. [proof: mutation]
3. [S3] `edit_paragraph_sha256` in `run.json`, refused when block 2 or later names another; `stage8Problem`; S outside stage8 refused. [proof: mutation]
4. [S4] The canary under stage8 runs B and S; S's init tools are B's. [proof: mutation]
5. [S5] Wire `stage8` and `S` in `cmd/bench/main.go` and its usage text; document them in `docs/bench/README.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ ./cmd/bench/ -count=1 -v -run '^(TestConfigurationSIsBPlusTheEditParagraph|TestBUnderStage8IsStage5sB|TestStage8RunsOnlyBAndS|TestTheStage8CanaryRunsBAndS|TestTheBashHookPassesASearchReplaceHeredoc)$' 2>&1) \
  && for t in TestConfigurationSIsBPlusTheEditParagraph TestBUnderStage8IsStage5sB TestStage8RunsOnlyBAndS TestTheStage8CanaryRunsBAndS TestTheBashHookPassesASearchReplaceHeredoc; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && go test ./internal/bench/ ./cmd/bench/ -count=1 \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q 'stage8'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestConfigurationSIsBPlusTheEditParagraph` | `internal/bench/runner_test.go` | S's `--tools` are B's; its prompt file is the instructions, a newline and `EditParagraph`; its settings and refusal text are B's; `edit_paragraph_sha256` in `run.json`, and a block 2 naming another sha refused | none | S2, S3 |
| `TestBUnderStage8IsStage5sB` | `internal/bench/runner_test.go` | under stage8, B's arguments, tools, prompt, settings and refusal text equal stage5's | none | S2 |
| `TestStage8RunsOnlyBAndS` | `internal/bench/runner_test.go` | under stage8 every list is a subset of B and S, and a model that runs S runs B; S under stage5 or stage7 refused | none | S3 |
| `TestTheStage8CanaryRunsBAndS` | `cmd/bench/main_test.go` | the canary command's configurations under stage8 are B and S, and under every other profile they are what they were | none | S4, S5 |
| `TestTheBashHookPassesASearchReplaceHeredoc` | `internal/bench/hook_test.go` | a SEARCH/REPLACE document by quoted and unquoted heredoc to `$TMPDIR` and to `mrw write --format=search_replace -` passes `ClassifyBash`, and the same document redirected into the tree is refused | none | S2 |
| `TestRunJSONNamesAConfigurationsTextOnlyUnderItsProfile` | `internal/bench/runner_test.go` | a Stage 5 run.json has neither `h_sentence_sha256` nor `edit_paragraph_sha256`; stage7 names the first alone, stage8 the second alone | none | S3 |
| `TestALookRefusesABlockThatRanAnotherEditParagraph` | `internal/bench/look_test.go` | a block whose run.json names another edit paragraph than block 1's is refused by name | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the five tests |
| 2 — something selects it | `Configs["S"]` read by `--configs` and by the canary's list; deleting either fails a named test |
| 3 — the caller can discover it | the usage text names `stage8`, the fence's last clause |
| 4 — it is used | T3's canary and smoke, T4's pilot |

## Mutation Log
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `internal/bench/runner.go` · S2: the newline between mrw instructions and the paragraph is dropped; S's prompt is not instructions, a newline, paragraph · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `internal/bench/hook.go` · S2: heredoc bodies are read as shell, so a SEARCH/REPLACE document is mistaken for a redirection and the hook refuses it · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `internal/bench/runner.go` · S3: a block 2 naming another paragraph than block 1 is no longer refused · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `internal/bench/runner.go` · S3: a model that runs S without B is no longer refused · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · covers:stage8 runs only B and S
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `internal/bench/canary.go` · S4: the stage8 canary runs A and B, not B and S · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · covers:the canary runs B and S
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `internal/bench/runner.go` · S2: B under stage8 gets H's hook flag, so it differs from stage5's B · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · covers:B under stage8 is stage5's B
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `internal/bench/profile.go` · S2: the paragraph is not ADR-193 Decision 1's, byte for byte · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · covers:S is B plus the paragraph
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `internal/bench/runner.go` · S3: a block 2 naming another paragraph than block 1 is no longer refused · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · covers:the paragraph cannot change between blocks
- 2026-11-03 · 5c2a308b* · mutant killed · exit 1 · `internal/bench/hook.go` · S2: heredoc bodies are read as shell, so a SEARCH/REPLACE document is mistaken for a redirection and the hook refuses it · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · covers:the hook passes a SEARCH/REPLACE heredoc
- 2026-11-03 · 8a9f8005* · mutant killed · exit 1 · `internal/bench/runner.go` · S3: every run.json names the edit paragraph, Stage 5's included · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · covers:the paragraph cannot change between blocks
- 2026-11-03 · 8a9f8005* · mutant killed · exit 1 · `internal/bench/runner.go` · S3: every run.json names H's sentence, Stage 5's included · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · covers:B under stage8 is stage5's B
- 2026-11-03 · 8a9f8005* · mutant killed · exit 1 · `internal/bench/runner.go` · S3: a stage8 run without --status-line is accepted, so its B is not Stage 5's B · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · covers:stage8 runs only B and S
- 2026-11-03 · 8a9f8005* · mutant killed · exit 1 · `internal/bench/look.go` · S3: bench look accepts a block that ran another edit paragraph · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · covers:the paragraph cannot change between blocks

## Invariants

- Without `--profile stage8`, every configuration's invocation and refusal text are byte-identical to today's.
- B's prompt under stage8 is `mrw instructions` alone, as in Stage 5.

## Risks

- The tests stand in for Claude Code's init line; T3's canary meets the real CLI.
- S's prompt ends without a final newline: `EditParagraph` is appended to `mrw instructions` byte for byte and has none. T5's `--host claude` output must end the same way.
- Stage8 refuses a `bench run` without `--status-line` so that B is Stage 5's B (its instruction carried the STATUS sentence); a canary has no such flag and is exempt.

## Stop Condition

The pinned Claude Code does not read `--append-system-prompt-file` for S as it does for B: stop to Zy.

## Out of Scope

- Counting the write calls (T2's job).

## Verification Log
- 2026-11-03 · 5c2a308b* · exit 1 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:141 · test-lock-sha256:4bf8279749b8ab11d06ec53373c4e511fe818153df678f61ca9b4627852226ca · test-lock-file:docs/adr/locks/4bf8279749b8ab11d06ec53373c4e511fe818153df678f61ca9b4627852226ca.lock
  ```
  ```
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:168714
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:167824
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:163402
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:194646
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:164582
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:168800
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:152780
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:156333
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:176426
- 2026-11-03 · 5c2a308b* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:161615
- 2026-11-03 · 8a9f8005* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:158719
- 2026-11-03 · 8a9f8005* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:166888
- 2026-11-03 · 8a9f8005* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:172328
- 2026-11-03 · 8a9f8005* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:163452
- 2026-11-03 · 8a9f8005* · exit 0 · `set -o pipefail …` · acceptance-sha256:dbd71c3e371a98a9c79c45866a0c1ed15cdaa984744e1dc3a8e7398412e0e6b7 · ms:159605
