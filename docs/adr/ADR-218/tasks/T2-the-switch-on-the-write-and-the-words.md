# Task ADR-218-T2: the switch on the write, and the words

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the engine option `NoShowing`; the widened number rule and the remedy words under the switch; the instructions' and `docs/agents.md`'s section; contract §386
**Consumes:** `shownOff` (T1); the switched read (T1); `internal/guide/guide.go`
**Data dependency:** the pilot of `docs/break/code-mode-790/pilot/README.md` needs live Claude Code sessions in Code Mode; the fence is hermetic and cannot see it, so its counts go into `pilot/results.md` and S5's sign-off
**Proof map:** v1
**Rests-on:** `the three named tests passed with every named case and none skipped`, `contract §386 exists, names the switch and the script parses`, `the engine packages this record does not own are unchanged`

## Goal

ADR-218 Decisions 4, 5 and 7.

- **A number alone.** Under the switch a hunk that names its lines by number alone is refused when one of them has no recorded showing in this version, on a whole-file licence and on a partial one. Without the switch a number alone on a partial licence applies, as on `main`.
- **The reasons.** Under the switch the number rule, the line-after rule, `occurrence=N` above 1 and a `from=` copy each name what works there and no read, and end `(reads here record no showing: MRW_SHOWN=off)`.
- **What stays.** `anchor=`, a pattern start, a `search_replace` block, `--force`, and a showing already in the ledger behave as today. `mrw stats` counts as today: ADR-200's predicate is not moved.
- **The words.** The full `mrw instructions`, `docs/agents.md` (the host section), README, `AGENTS.md`. `mrw instructions --core` is unchanged.
- **Contract §386**, each case that must pass beside the one that must fail.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | the write passes the switch to the engine |
| `internal/apply/apply.go`, `internal/apply/fromcopy.go` | edit | the option, the refusal under it, the remedy words |
| `internal/guide/guide.go`, `docs/agents.md`, `README.md`, `AGENTS.md` | edit | the words |
| `scripts/contract.sh` | edit | §386 |
| `cmd/mrw/shownwrite218_test.go`, `internal/apply/shown218_test.go`, `internal/guide/shown218_test.go` | new | the tests |

What selects it: the `write` action (`cmd/mrw/main.go:2334`, where the options of a CLI write are built). The CLI test runs the command in-process with the environment set and §386 runs the built binary, so an option that exists and is never set fails both.

## Ordered Steps

1. [S1] §386 and the tests first, RED on `main`: after a switched ranged read a number alone applies there. The CLI test's cases, each a named subtest: `number_alone_whole_refused`, `number_alone_ranged_refused`, `ranged_quiet_without_switch_applies`, `switched_read_plain_write_applies`, `anchor_applies`, `pattern_applies`, `multi_refused_names_the_block`, `block_applies`, `earlier_showing_applies`, `force_applies`, `occurrence_reason`, `from_reason`.
2. [S2] The option and the number rule. Mutants, each killed: the option never set by the CLI (`number_alone_ranged_refused`); the widening applied with the switch off (`switched_read_plain_write_applies`, the same switched ranged read and a write without the switch: one variable against `number_alone_ranged_refused`; and `ranged_quiet_without_switch_applies`); the pricing marks moved by the option (the engine test's `pricing_unmoved`). [proof: mutation]
3. [S3] The reasons. Mutants: the line-after reason names a read under the switch (`multi_refused_names_the_block`); the `from=` reason does (`from_reason`). [proof: mutation]
4. [S4] The instructions, `docs/agents.md`, README, `AGENTS.md`; `bash -n scripts/contract.sh`; `./scripts/contract.sh` with §386 passing; `adr-lint` on every record whose test lock names an edited file. [proof: acceptance]
5. [S5] The pilot, by the protocol frozen in `docs/break/code-mode-790/pilot/README.md` before S1. [proof: human: `pilot/results.md` holds, per run, the digest verdict, the writes by form, the refusals by kind and the void checks, with the Claude Code version and model, against the bar; the sha256 of the protocol and the fixture equal the ones filed in memoryd before the build]

## Acceptance

```bash
out="$(mktemp)" \
  && go test -count=1 -v ./cmd/mrw ./internal/apply ./internal/guide -run '^(TestAWriteUnderShownOffWantsTheTextQuoted|TestANumberAloneUnderNoShowing|TestTheInstructionsSayWhatAScriptedCallerDoes)$' > "$out" \
  && [ "$(grep -c '^--- PASS: ' "$out")" -eq 3 ] && ! grep -q -- '--- SKIP' "$out" \
  && for c in TestAWriteUnderShownOffWantsTheTextQuoted/number_alone_whole_refused TestAWriteUnderShownOffWantsTheTextQuoted/number_alone_ranged_refused TestAWriteUnderShownOffWantsTheTextQuoted/ranged_quiet_without_switch_applies TestAWriteUnderShownOffWantsTheTextQuoted/switched_read_plain_write_applies TestAWriteUnderShownOffWantsTheTextQuoted/anchor_applies TestAWriteUnderShownOffWantsTheTextQuoted/pattern_applies TestAWriteUnderShownOffWantsTheTextQuoted/multi_refused_names_the_block TestAWriteUnderShownOffWantsTheTextQuoted/block_applies TestAWriteUnderShownOffWantsTheTextQuoted/earlier_showing_applies TestAWriteUnderShownOffWantsTheTextQuoted/force_applies TestAWriteUnderShownOffWantsTheTextQuoted/occurrence_reason TestAWriteUnderShownOffWantsTheTextQuoted/from_reason TestANumberAloneUnderNoShowing/pricing_unmoved; do grep -q "^    --- PASS: $c " "$out" || exit 1; done \
  && grep -q 'MRW_SHOWN' <(sed -n '/^# 386\. /,/^# [0-9][0-9]*\. /p' scripts/contract.sh) && bash -n scripts/contract.sh \
  && base="$(git merge-base HEAD origin/main)" && git diff --quiet "$base" -- internal/read internal/plan internal/seen internal/check internal/state && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state)" ]
```

What each clause holds. The three tests pass, each of the thirteen named cases passes, none skipped. §386 exists, names the switch, and the script parses; that the row RAN is `./scripts/contract.sh` (S4). The engine packages this record does not own are unchanged, test files included; `internal/apply` is the record's.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteUnderShownOffWantsTheTextQuoted` | `cmd/mrw/shownwrite218_test.go` | the twelve cases through the CLI, each refusal's exit, reason and unchanged bytes, each landing's bytes | none | S1, S2, S3 |
| `TestANumberAloneUnderNoShowing` | `internal/apply/shown218_test.go` | the engine option on whole and partial licences; ADR-200's marks equal with the option on and off | none | S2 |
| `TestTheInstructionsSayWhatAScriptedCallerDoes` | `internal/guide/shown218_test.go` | the section in the full text; the core unchanged in length class (under 300 words) | none | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §386 |
| 2 — something selects it | the `write` action of the CLI |
| 3 — the caller can discover it | the refusals, `mrw instructions`, `docs/agents.md` |
| 4 — it is used | telemetry is refused by ADR-009; the evidence is #790 and the pilot of S5 |

## Mutation Log
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the option never set by the CLI (number_alone_ranged_refused) · acceptance-sha256:9d191bc4fb21c13433f1b940003b2d5351f901a5fd4e22afb965fdfb19d6dc30
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: the widening applied with the switch off (ranged_quiet_without_switch_applies) · acceptance-sha256:9d191bc4fb21c13433f1b940003b2d5351f901a5fd4e22afb965fdfb19d6dc30
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: the pricing mark moved by the option (pricing_unmoved) · acceptance-sha256:9d191bc4fb21c13433f1b940003b2d5351f901a5fd4e22afb965fdfb19d6dc30
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `internal/apply/apply.go` · S3: the line-after reason names a read under the switch (multi_refused_names_the_block) · acceptance-sha256:9d191bc4fb21c13433f1b940003b2d5351f901a5fd4e22afb965fdfb19d6dc30
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `internal/apply/fromcopy.go` · S3: the from= reason names a read under the switch (from_reason) · acceptance-sha256:9d191bc4fb21c13433f1b940003b2d5351f901a5fd4e22afb965fdfb19d6dc30
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the option never set by the CLI (number_alone_ranged_refused) · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: the widening applied with the switch off (switched_read_plain_write_applies) · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: the pricing mark moved by the option (pricing_unmoved) · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `internal/apply/apply.go` · S3: the number reason names a read under the switch (number_alone_whole_refused) · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `internal/apply/apply.go` · S3: the occurrence reason names a read under the switch (occurrence_reason) · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `internal/apply/apply.go` · S3: the line-after reason names a read under the switch (multi_refused_names_the_block) · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b
- 2026-11-24 · c0e9f381* · mutant killed · exit 1 · `internal/apply/fromcopy.go` · S3: the from= reason names a read under the switch (from_reason) · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b

## Verification Log
- 2026-11-24 · c0e9f381* · exit 1 · `out="$(mktemp)" \ …` · acceptance-sha256:9d191bc4fb21c13433f1b940003b2d5351f901a5fd4e22afb965fdfb19d6dc30 · ms:744 · test-lock-sha256:cb677aa47d1da192044bdc8aef1bef5c96590335aacf7bb107ad6bf134e848e9 · test-lock-file:docs/adr/locks/cb677aa47d1da192044bdc8aef1bef5c96590335aacf7bb107ad6bf134e848e9.lock
  ```
  ```
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9d191bc4fb21c13433f1b940003b2d5351f901a5fd4e22afb965fdfb19d6dc30 · ms:3733
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9d191bc4fb21c13433f1b940003b2d5351f901a5fd4e22afb965fdfb19d6dc30 · ms:2713
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9d191bc4fb21c13433f1b940003b2d5351f901a5fd4e22afb965fdfb19d6dc30 · ms:2482
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9d191bc4fb21c13433f1b940003b2d5351f901a5fd4e22afb965fdfb19d6dc30 · ms:2534
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9d191bc4fb21c13433f1b940003b2d5351f901a5fd4e22afb965fdfb19d6dc30 · ms:2442
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:2606
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:2553
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:2319
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:2306
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:2463
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:2517
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:2521
- 2026-11-24 · c0e9f381* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:12 · test-lock-sha256:cafefe729f6b44b3bccffbd85cf51fe25560c8b59512245affd5588922ab52df · test-lock-file:docs/adr/locks/cafefe729f6b44b3bccffbd85cf51fe25560c8b59512245affd5588922ab52df.lock · test-lock-kind:replace
- 2026-11-24 · c0e9f381* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:1190
- 2026-11-24 · human-observed · Claude (session of 2026-11-24) observed S5: 30 runs, all ending with the task's digest; the switched arm had no refused write and landed a search_replace block in every run of tasks 3-5. Blocked for the owner: bar clause 1 is not met as written, 13 of 15 switched runs being void under two void rules the protocol worded badly (a host Write outside pkg/, and a note cut from the log); docs/break/code-mode-790/pilot/results.md holds both readings. · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b
- 2026-11-24 · 65764867* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:13 · test-lock-sha256:1b71cd18c808996ecf2767a99b50fecbb29988aa73f2f1a2ff8cc760c0b40faa · test-lock-file:docs/adr/locks/1b71cd18c808996ecf2767a99b50fecbb29988aa73f2f1a2ff8cc760c0b40faa.lock · test-lock-kind:replace
- 2026-11-24 · 65764867* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:1649
- 2026-11-24 · b54ff01f* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:993
- 2026-11-24 · c64de773* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:972
- 2026-11-24 · 11ee2a9f* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:9a9ef67a3e92e0e602118302d09fc201082c081eea9bb1fd6d0c462210bb4c3b · ms:955

## Invariants

- Without the switch a write behaves as on `main`: ADR-190's number on a `search_replace` write's lines included.
- ADR-200's predicate is not moved: for one ledger record, `numbered_candidates` counts the same with the option on and off. What a switched read records differs from a plain read's (a written-shaped record), so the same session can enter the count where it did not before.
- `mrw_write` does not read the switch. No receipt key is added; no exit code changes; no plan syntax changes.

## Risks

- A session that has the switch on loses a number alone on the lines its own `search_replace` write produced; the reason names `anchor=`.
- A model in Code Mode may not recover from the refusals: S5's pilot is the evidence, and its bar was written first.

## Stop Condition

The fence exits 0; `./scripts/contract.sh` exits 0 with §386 passing and not skipped; every gate in `AGENTS.md` passes; a `go-reviewer` pass on the engine diff has no open finding; S5's results are in `pilot/results.md`; `adr-debt` reports nothing broken or unreceipted. If S5 misses the bar this task stops `blocked` and does not release.

## Out of Scope

- The switch on `mrw_write` (permanent: boundary: MCP's showing is an acknowledgement, ADR-108; ADR-218 Out of Scope)
- Checkpoints and `ack` on the CLI (deferred: docs/adr/BACKLOG.md "From ADR-218")
- A release note (external: release owner: the served-path change of ADR-218)
